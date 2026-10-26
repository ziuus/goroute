package codex

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ziuus/goroute/pkg/affinity"
	"github.com/ziuus/goroute/pkg/providers"
	"github.com/ziuus/goroute/pkg/resilience"
)

const (
	DefaultCodexClientID   = "app_EMoamEEZ73f0CkXaXp7hrann"
	CodexTokenURL          = "https://auth.openai.com/oauth/token"
	CodexUserAgent         = "codex-cli/0.149.0 (Linux; x64)"
	CodexOriginator        = "codex_cli_rs"
	DefaultCodexBackendURL = "https://chatgpt.com/backend-api/codex/responses"
)

type Account struct {
	ID           string
	Pool         string
	RefreshToken string
	AccessToken  string
	AccountID    string
	ExpiresAt    time.Time
	mu           sync.RWMutex
}

func (a *Account) GetValidAccessToken(ctx context.Context, client *http.Client) (string, error) {
	a.mu.RLock()
	if a.AccessToken != "" && time.Now().Before(a.ExpiresAt.Add(-2*time.Minute)) {
		token := a.AccessToken
		a.mu.RUnlock()
		return token, nil
	}
	a.mu.RUnlock()

	a.mu.Lock()
	defer a.mu.Unlock()

	if a.AccessToken != "" && time.Now().Before(a.ExpiresAt.Add(-2*time.Minute)) {
		return a.AccessToken, nil
	}

	form := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {a.RefreshToken},
		"client_id":     {DefaultCodexClientID},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", CodexTokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("create codex refresh request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", CodexUserAgent)
	req.Header.Set("originator", CodexOriginator)

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("execute codex token refresh: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("codex token refresh failed with status %d: %s", resp.StatusCode, string(body))
	}

	var res struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", fmt.Errorf("decode codex token refresh response: %w", err)
	}

	if res.AccessToken == "" {
		return "", fmt.Errorf("empty access token returned from Codex refresh")
	}

	a.AccessToken = res.AccessToken
	if res.RefreshToken != "" {
		a.RefreshToken = res.RefreshToken
	}
	expiresIn := res.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = 86400
	}
	a.ExpiresAt = time.Now().Add(time.Duration(expiresIn) * time.Second)

	return a.AccessToken, nil
}

type Provider struct {
	id       string
	accounts []*Account
	idx      uint64
	client   *http.Client
	mu       sync.RWMutex
}

func NewProvider(id string, accounts []*Account) *Provider {
	return &Provider{
		id:       id,
		accounts: accounts,
		client: &http.Client{
			Transport: &http.Transport{
				ResponseHeaderTimeout: 300 * time.Second,
				IdleConnTimeout:       120 * time.Second,
			},
			Timeout: 0,
		},
	}
}

func (p *Provider) ID() string {
	return p.id
}

func (p *Provider) Type() string {
	return "codex"
}

func (p *Provider) IsHealthy() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.accounts) > 0
}

func (p *Provider) Accounts() []*Account {
	p.mu.RLock()
	defer p.mu.RUnlock()
	res := make([]*Account, len(p.accounts))
	copy(res, p.accounts)
	return res
}

func (p *Provider) nextAccount(pinnedID string) *Account {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if len(p.accounts) == 0 {
		return nil
	}
	if pinnedID != "" {
		for _, a := range p.accounts {
			if a.ID == pinnedID || a.AccountID == pinnedID {
				return a
			}
		}
	}
	next := atomic.AddUint64(&p.idx, 1)
	return p.accounts[next%uint64(len(p.accounts))]
}

func (p *Provider) TestAccount(ctx context.Context, accountID string) error {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if len(p.accounts) == 0 {
		return fmt.Errorf("no accounts configured for codex provider")
	}
	if accountID == "" {
		_, err := p.accounts[0].GetValidAccessToken(ctx, p.client)
		return err
	}
	for _, a := range p.accounts {
		if a.ID == accountID {
			_, err := a.GetValidAccessToken(ctx, p.client)
			return err
		}
	}
	return fmt.Errorf("account %q not found in codex provider", accountID)
}

func (p *Provider) TestConnection(ctx context.Context) error {
	return p.TestAccount(ctx, "")
}

func (p *Provider) Models(ctx context.Context) ([]providers.ModelInfo, error) {
	return []providers.ModelInfo{
		{ID: "gpt-5.6-sol", Object: "model", OwnedBy: "codex"},
		{ID: "gpt-5.6-sol-ultra", Object: "model", OwnedBy: "codex"},
		{ID: "gpt-5.6-sol-high", Object: "model", OwnedBy: "codex"},
		{ID: "gpt-5.6-sol-medium", Object: "model", OwnedBy: "codex"},
		{ID: "gpt-5.5-sol", Object: "model", OwnedBy: "codex"},
		{ID: "o3-mini", Object: "model", OwnedBy: "codex"},
		{ID: "o1", Object: "model", OwnedBy: "codex"},
		{ID: "gpt-4o", Object: "model", OwnedBy: "codex"},
	}, nil
}

func (p *Provider) Embeddings(ctx context.Context, req *providers.EmbeddingRequest) (*providers.EmbeddingResponse, error) {
	return nil, fmt.Errorf("embeddings not supported on codex provider")
}

func (p *Provider) ChatCompletion(ctx context.Context, req *providers.ChatRequest) (io.ReadCloser, bool, int, error) {
	attempts := len(p.accounts)
	if attempts == 0 {
		return nil, false, http.StatusServiceUnavailable, fmt.Errorf("no codex accounts available")
	}

	sessionKey := affinity.DeriveSessionKey(nil, req)
	pinnedAcc := affinity.GlobalManager.GetAffinity(sessionKey, p.id)

	var lastErr error
	var lastStatus int

	for i := 0; i < attempts; i++ {
		var acc *Account
		if i == 0 && pinnedAcc != "" {
			acc = p.nextAccount(pinnedAcc)
		} else {
			acc = p.nextAccount("")
		}
		if acc == nil {
			break
		}

		token, err := acc.GetValidAccessToken(ctx, p.client)
		if err != nil {
			lastErr = fmt.Errorf("account %s token refresh error: %w", acc.ID, err)
			lastStatus = http.StatusUnauthorized
			continue
		}

		// Prepare OpenAI-format completion request
		payloadBytes, err := json.Marshal(req)
		if err != nil {
			return nil, false, http.StatusBadRequest, fmt.Errorf("marshal codex request: %w", err)
		}

		// Upstream endpoint: Codex responses or OpenAI completions
		targetURL := "https://api.openai.com/v1/chat/completions"
		httpReq, err := http.NewRequestWithContext(ctx, "POST", targetURL, bytes.NewReader(payloadBytes))
		if err != nil {
			return nil, false, http.StatusInternalServerError, err
		}

		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Authorization", "Bearer "+token)
		httpReq.Header.Set("User-Agent", CodexUserAgent)
		httpReq.Header.Set("originator", CodexOriginator)
		if acc.AccountID != "" {
			httpReq.Header.Set("ChatGPT-Account-ID", acc.AccountID)
		}

		resp, err := p.client.Do(httpReq)
		if err != nil {
			lastErr = err
			lastStatus = http.StatusBadGateway
			continue
		}

		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusUnauthorized || resp.StatusCode >= 500 {
			resp.Body.Close()
			lastErr = fmt.Errorf("codex account %s returned HTTP %d", acc.ID, resp.StatusCode)
			lastStatus = resp.StatusCode
			log.Printf("[codex] Account %s returned %d, rotating to next account", acc.ID, resp.StatusCode)

			// Evict session affinity if this pinned account failed
			if sessionKey != "" && pinnedAcc == acc.ID {
				affinity.GlobalManager.EvictAffinity(sessionKey, p.id, acc.ID)
			}
			if resp.StatusCode == http.StatusTooManyRequests {
				resilience.GlobalCooldown.MarkCooldown(p.id, req.Model, 429, "rate_limit", 60*time.Second)
			}
			continue
		}

		// Success: record session affinity for prompt cache continuity
		if sessionKey != "" && acc.ID != "" {
			affinity.GlobalManager.SetAffinity(sessionKey, p.id, acc.ID)
		}

		isStream := strings.Contains(resp.Header.Get("Content-Type"), "text/event-stream")
		return resp.Body, isStream, resp.StatusCode, nil
	}

	if lastStatus == 0 {
		lastStatus = http.StatusBadGateway
	}
	return nil, false, lastStatus, lastErr
}

// ReadCodexCLIAuth parses ~/.codex/auth.json
func ReadCodexCLIAuth() (*Account, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	authPath := filepath.Join(home, ".codex", "auth.json")
	data, err := os.ReadFile(authPath)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", authPath, err)
	}

	var parsed struct {
		Tokens struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
			AccountID    string `json:"account_id"`
		} `json:"tokens"`
	}

	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, fmt.Errorf("parse %s: %w", authPath, err)
	}

	if parsed.Tokens.RefreshToken == "" && parsed.Tokens.AccessToken == "" {
		return nil, fmt.Errorf("no tokens found in %s", authPath)
	}

	return &Account{
		ID:           "codex-cli",
		RefreshToken: parsed.Tokens.RefreshToken,
		AccessToken:  parsed.Tokens.AccessToken,
		AccountID:    parsed.Tokens.AccountID,
	}, nil
}
