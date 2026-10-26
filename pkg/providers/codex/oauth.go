package codex

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	CodexDeviceUserCodeURL = "https://auth.openai.com/api/accounts/deviceauth/usercode"
	CodexDeviceTokenURL    = "https://auth.openai.com/api/accounts/deviceauth/token"
	CodexDeviceVerifyURI   = "https://auth.openai.com/codex/device"
	CodexDeviceCallbackURI = "https://auth.openai.com/deviceauth/callback"
)

type DeviceCodeResponse struct {
	DeviceAuthID    string `json:"device_auth_id"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	Interval        int    `json:"interval"`
	ExpiresIn       int    `json:"expires_in"`
}

type DeviceTokenPollResponse struct {
	AuthorizationCode string `json:"authorization_code"`
	CodeVerifier      string `json:"code_verifier"`
	Status            string `json:"status"` // "pending", "approved"
}

type TokenExchangeResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

// StartDeviceAuth initiates the OpenAI Codex device flow
func StartDeviceAuth(ctx context.Context) (*DeviceCodeResponse, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	reqBody, _ := json.Marshal(map[string]string{
		"client_id": DefaultCodexClientID,
	})

	req, err := http.NewRequestWithContext(ctx, "POST", CodexDeviceUserCodeURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", CodexUserAgent)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("device code request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("device code request returned status %d", resp.StatusCode)
	}

	var data struct {
		DeviceAuthID string      `json:"device_auth_id"`
		UserCode     string      `json:"user_code"`
		Interval     interface{} `json:"interval"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	interval := 5
	switch v := data.Interval.(type) {
	case string:
		fmt.Sscanf(v, "%d", &interval)
	case float64:
		interval = int(v)
	case int:
		interval = v
	}
	if interval <= 0 {
		interval = 5
	}

	return &DeviceCodeResponse{
		DeviceAuthID:    data.DeviceAuthID,
		UserCode:        data.UserCode,
		VerificationURI: CodexDeviceVerifyURI,
		Interval:        interval,
		ExpiresIn:       900,
	}, nil
}

// PollAndExchangeDeviceToken polls the device auth endpoint and exchanges for tokens
func PollAndExchangeDeviceToken(ctx context.Context, deviceAuthID, userCode string) (*TokenExchangeResponse, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	reqBody, _ := json.Marshal(map[string]string{
		"device_auth_id": deviceAuthID,
		"user_code":     userCode,
	})

	req, err := http.NewRequestWithContext(ctx, "POST", CodexDeviceTokenURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", CodexUserAgent)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("poll request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("authorization_pending")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("poll returned status %d", resp.StatusCode)
	}

	var pollData struct {
		AuthorizationCode string `json:"authorization_code"`
		CodeVerifier      string `json:"code_verifier"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&pollData); err != nil {
		return nil, err
	}

	if pollData.AuthorizationCode == "" || pollData.CodeVerifier == "" {
		return nil, fmt.Errorf("authorization_pending")
	}

	// Step 3: Exchange authorization_code + code_verifier for tokens
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {DefaultCodexClientID},
		"code":          {pollData.AuthorizationCode},
		"code_verifier": {pollData.CodeVerifier},
		"redirect_uri":  {CodexDeviceCallbackURI},
	}

	tokenReq, err := http.NewRequestWithContext(ctx, "POST", CodexTokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	tokenReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	tokenReq.Header.Set("User-Agent", CodexUserAgent)

	tokenResp, err := client.Do(tokenReq)
	if err != nil {
		return nil, fmt.Errorf("token exchange failed: %w", err)
	}
	defer tokenResp.Body.Close()

	if tokenResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange returned status %d", tokenResp.StatusCode)
	}

	var tokenData TokenExchangeResponse
	if err := json.NewDecoder(tokenResp.Body).Decode(&tokenData); err != nil {
		return nil, err
	}

	return &tokenData, nil
}
