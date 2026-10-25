package antigravity

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

var (
	maskedClientID     = []byte{86, 95, 69, 94, 69, 68, 83, 29, 70, 69, 87, 85, 88, 78, 89, 27, 89, 20, 28, 27, 1, 71, 28, 87, 28, 28, 22, 16, 9, 91, 80, 24, 0, 69, 8, 3, 29, 5, 29, 64, 2, 25, 64, 70, 7, 28, 71, 2, 93, 6, 66, 73, 8, 29, 0, 18, 24, 0, 88, 3, 16, 16, 15, 6, 13, 89, 19, 95, 19, 65, 17, 0, 24}
	maskedClientSecret = []byte{32, 32, 49, 60, 37, 44, 72, 102, 69, 77, 36, 59, 59, 87, 21, 64, 125, 3, 35, 56, 94, 24, 56, 39, 21, 3, 45, 33, 88, 19, 85, 92, 50, 112, 1}
	maskKey            = "goroute-public-v1"
)

func unmask(b []byte) string {
	res := make([]byte, len(b))
	for i, c := range b {
		res[i] = c ^ maskKey[i%len(maskKey)]
	}
	return string(res)
}

func GetClientID() string {
	return unmask(maskedClientID)
}

func GetClientSecret() string {
	return unmask(maskedClientSecret)
}

const (
	AuthorizeURL        = "https://accounts.google.com/o/oauth2/v2/auth"
	DefaultTokenURL     = "https://oauth2.googleapis.com/token"
	UserInfoURL         = "https://www.googleapis.com/oauth2/v1/userinfo"
	CloudCodeAPIBaseURL = "https://daily-cloudcode-pa.googleapis.com"
	FallbackCloudCodeURL = "https://cloudcode-pa.googleapis.com"
)

var Scopes = []string{
	"https://www.googleapis.com/auth/cloud-platform",
	"https://www.googleapis.com/auth/userinfo.email",
	"https://www.googleapis.com/auth/userinfo.profile",
	"https://www.googleapis.com/auth/cclog",
	"https://www.googleapis.com/auth/experimentsandconfigs",
}

type Account struct {
	ID           string    `json:"id"`
	Pool         string    `json:"pool,omitempty"`
	Email        string    `json:"email"`
	RefreshToken string    `json:"refresh_token"`
	AccessToken  string    `json:"access_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	ProjectID    string    `json:"project_id"`
	TokenURL     string    `json:"token_url,omitempty"`
	mu           sync.Mutex
}

func (a *Account) GetValidAccessToken(ctx context.Context) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// If still valid for at least 60 seconds, reuse
	if a.AccessToken != "" && time.Now().Add(60*time.Second).Before(a.ExpiresAt) {
		return a.AccessToken, nil
	}

	tokenURL := a.TokenURL
	if tokenURL == "" {
		tokenURL = DefaultTokenURL
	}

	form := url.Values{
		"grant_type":    {"refresh_token"},
		"client_id":     {GetClientID()},
		"client_secret": {GetClientSecret()},
		"refresh_token": {a.RefreshToken},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("token refresh request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("refresh token failed with status %d", resp.StatusCode)
	}

	var payload struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", err
	}

	a.AccessToken = payload.AccessToken
	a.ExpiresAt = time.Now().Add(time.Duration(payload.ExpiresIn) * time.Second)

	return a.AccessToken, nil
}

func (a *Account) InvalidateToken() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.AccessToken = ""
	a.ExpiresAt = time.Time{}
}

// GeneratePKCE returns a random code verifier and its S256 code challenge
func GeneratePKCE() (verifier, challenge string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", err
	}
	verifier = base64.RawURLEncoding.EncodeToString(b)
	h := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(h[:])
	return verifier, challenge, nil
}

// BuildAuthURL creates the Google OAuth authorization URL with PKCE
func BuildAuthURL(redirectURI, state, codeChallenge string) string {
	params := url.Values{
		"client_id":             {GetClientID()},
		"response_type":         {"code"},
		"redirect_uri":          {redirectURI},
		"scope":                 {strings.Join(Scopes, " ")},
		"state":                 {state},
		"access_type":           {"offline"},
		"prompt":                {"consent"},
		"code_challenge":        {codeChallenge},
		"code_challenge_method": {"S256"},
	}
	return AuthorizeURL + "?" + params.Encode()
}

// ExchangeCode exchanges an authorization code for refresh & access tokens
func ExchangeCode(ctx context.Context, code, redirectURI, codeVerifier string) (*Account, error) {
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {GetClientID()},
		"client_secret": {GetClientSecret()},
		"code":          {code},
		"redirect_uri":  {redirectURI},
		"code_verifier": {codeVerifier},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", DefaultTokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("exchange code returned status %d", resp.StatusCode)
	}

	var payload struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	acc := &Account{
		AccessToken:  payload.AccessToken,
		RefreshToken: payload.RefreshToken,
		ExpiresAt:    time.Now().Add(time.Duration(payload.ExpiresIn) * time.Second),
	}

	// Fetch user email
	acc.FetchUserInfo(ctx)
	// Discover Cloud Code Project ID
	acc.DiscoverProjectID(ctx)

	return acc, nil
}

func (a *Account) FetchUserInfo(ctx context.Context) {
	req, err := http.NewRequestWithContext(ctx, "GET", UserInfoURL+"?alt=json", nil)
	if err != nil {
		return
	}
	req.Header.Set("Authorization", "Bearer "+a.AccessToken)

	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return
	}
	defer resp.Body.Close()

	var data struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err == nil {
		a.Email = data.Email
	}
}

func (a *Account) DiscoverProjectID(ctx context.Context) {
	body := `{"metadata":{"ideType":9,"platform":3,"pluginType":2}}`
	url := CloudCodeAPIBaseURL + "/v1internal:loadCodeAssist"

	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Authorization", "Bearer "+a.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "antigravity/cli/1.0.0 (darwin/arm64)")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return
	}
	defer resp.Body.Close()

	var data struct {
		CloudaicompanionProject struct {
			ID string `json:"id"`
		} `json:"cloudaicompanionProject"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err == nil && data.CloudaicompanionProject.ID != "" {
		a.ProjectID = data.CloudaicompanionProject.ID
	}
}

// AutoImportLocalAntigravity discovers logged-in Google Antigravity accounts from local IDE & CLI configs.
func AutoImportLocalAntigravity(ctx context.Context) ([]*Account, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	seen := make(map[string]bool)
	var accounts []*Account

	addAccount := func(refresh, access, email, source string) {
		refresh = strings.TrimSpace(refresh)
		if refresh == "" || seen[refresh] {
			return
		}
		seen[refresh] = true

		acc := &Account{
			ID:           fmt.Sprintf("agy-%s-%d", source, len(accounts)+1),
			Pool:         "pool1",
			RefreshToken: refresh,
			AccessToken:  access,
			Email:        email,
		}

		// Proactively refresh access token & discover project metadata
		if freshTok, err := acc.GetValidAccessToken(ctx); err == nil && freshTok != "" {
			if acc.Email == "" {
				acc.FetchUserInfo(ctx)
			}
			acc.DiscoverProjectID(ctx)
		}
		if acc.ProjectID == "" {
			acc.ProjectID = "aicode-consumers"
		}
		accounts = append(accounts, acc)
	}

	// 1. Antigravity CLI Token Files
	cliPatterns := []string{
		filepath.Join(home, ".gemini", "antigravity-cli", "antigravity-oauth-token"),
		filepath.Join(home, ".gemini", "antigravity", "antigravity-oauth-token"),
		filepath.Join(home, ".config", "agentcontrol", "profiles", "*", ".gemini", "antigravity-cli", "antigravity-oauth-token"),
	}

	for _, pat := range cliPatterns {
		matches, _ := filepath.Glob(pat)
		for _, file := range matches {
			data, err := os.ReadFile(file)
			if err != nil {
				continue
			}
			var tokPayload struct {
				Token struct {
					AccessToken  string `json:"access_token"`
					RefreshToken string `json:"refresh_token"`
				} `json:"token"`
				Email string `json:"id_token_email"`
			}
			if err := json.Unmarshal(data, &tokPayload); err == nil && tokPayload.Token.RefreshToken != "" {
				addAccount(tokPayload.Token.RefreshToken, tokPayload.Token.AccessToken, tokPayload.Email, "cli")
			}
		}
	}

	// 2. Linux Secret Service / Keyring (service: gemini, username: antigravity)
	if out, err := exec.Command("secret-tool", "lookup", "service", "gemini", "username", "antigravity").Output(); err == nil && len(out) > 0 {
		var tokPayload struct {
			Token struct {
				AccessToken  string `json:"access_token"`
				RefreshToken string `json:"refresh_token"`
			} `json:"token"`
			Email string `json:"id_token_email"`
		}
		if err := json.Unmarshal(out, &tokPayload); err == nil && tokPayload.Token.RefreshToken != "" {
			addAccount(tokPayload.Token.RefreshToken, tokPayload.Token.AccessToken, tokPayload.Email, "cli-keyring")
		}
	}

	// 3. Antigravity IDE SQLite State Database
	ideDbs := []string{
		filepath.Join(home, ".config", "Antigravity", "User", "globalStorage", "state.vscdb"),
		filepath.Join(home, ".config", "Antigravity IDE", "User", "globalStorage", "state.vscdb"),
	}

	reRefresh := regexp.MustCompile(`1//[a-zA-Z0-9_\-]+`)
	reAccess := regexp.MustCompile(`ya29\.[a-zA-Z0-9_\-]+`)

	for _, dbPath := range ideDbs {
		if _, err := os.Stat(dbPath); err != nil {
			continue
		}
		out, err := exec.Command("sqlite3", dbPath, "SELECT value FROM ItemTable WHERE key = 'antigravityUnifiedStateSync.oauthToken';").Output()
		if err != nil || len(out) == 0 {
			continue
		}

		rawStr := string(out)
		decoded, err := base64.StdEncoding.DecodeString(rawStr)
		searchSpace := string(decoded)
		if err != nil {
			searchSpace = rawStr
		}

		refreshMatches := reRefresh.FindAllString(searchSpace, -1)
		accessMatches := reAccess.FindAllString(searchSpace, -1)

		for i, ref := range refreshMatches {
			accTok := ""
			if i < len(accessMatches) {
				accTok = accessMatches[i]
			}
			addAccount(ref, accTok, "", "ide")
		}
	}

	if len(accounts) == 0 {
		return nil, fmt.Errorf("no locally logged-in Antigravity accounts detected in CLI or IDE paths")
	}

	return accounts, nil
}
