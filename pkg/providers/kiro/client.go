package kiro

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ziuus/goroute/pkg/affinity"
	"github.com/ziuus/goroute/pkg/pool"
	"github.com/ziuus/goroute/pkg/providers"
	"github.com/ziuus/goroute/pkg/rtk"
)

type Account struct {
	ID           string
	Pool         string
	APIKey       string
	RefreshToken string
	ProfileARN   string
	accessToken  string
	expiresAt    time.Time
	mu           sync.RWMutex
}

func (a *Account) GetValidAccessToken(ctx context.Context, client *http.Client) (string, error) {
	a.mu.RLock()
	if a.APIKey != "" {
		token := a.APIKey
		a.mu.RUnlock()
		return token, nil
	}
	if strings.HasPrefix(a.RefreshToken, "ey") {
		token := a.RefreshToken
		a.mu.RUnlock()
		return token, nil
	}
	if a.accessToken != "" && time.Now().Before(a.expiresAt.Add(-2*time.Minute)) {
		token := a.accessToken
		a.mu.RUnlock()
		return token, nil
	}
	a.mu.RUnlock()

	a.mu.Lock()
	defer a.mu.Unlock()

	if a.APIKey != "" {
		return a.APIKey, nil
	}
	if strings.HasPrefix(a.RefreshToken, "ey") {
		return a.RefreshToken, nil
	}
	if a.accessToken != "" && time.Now().Before(a.expiresAt.Add(-2*time.Minute)) {
		return a.accessToken, nil
	}

	refreshPayload := map[string]string{
		"refreshToken": a.RefreshToken,
	}
	reqBytes, err := json.Marshal(refreshPayload)
	if err != nil {
		return "", fmt.Errorf("marshal refresh payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://prod.us-east-1.auth.desktop.kiro.dev/refreshToken", bytes.NewReader(reqBytes))
	if err != nil {
		return "", fmt.Errorf("create refresh request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("execute token refresh: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("token refresh failed with status %d: %s", resp.StatusCode, string(body))
	}

	var res struct {
		AccessToken string `json:"accessToken"`
		ExpiresIn   int    `json:"expiresIn"`
		ProfileArn  string `json:"profileArn"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", fmt.Errorf("decode token refresh response: %w", err)
	}

	if res.AccessToken == "" {
		return "", fmt.Errorf("empty access token returned from Kiro refresh")
	}

	a.accessToken = res.AccessToken
	expiresIn := res.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = 3600
	}
	a.expiresAt = time.Now().Add(time.Duration(expiresIn) * time.Second)
	if res.ProfileArn != "" {
		a.ProfileARN = res.ProfileArn
	}

	return a.accessToken, nil
}

type Provider struct {
	id       string
	accounts []*Account
	strategy string
	counter  uint64
	mu       sync.RWMutex
	client   *http.Client
}

func (p *Provider) SetStrategy(s string) {
	p.strategy = s
}

func NewProvider(id, refreshToken, profileARN string) *Provider {
	if profileARN == "" {
		profileARN = "arn:aws:codewhisperer:us-east-1:699475941385:profile/EHGA3GRVQMUK"
	}
	var accs []*Account
	if refreshToken != "" {
		accs = append(accs, &Account{
			ID:           id,
			RefreshToken: refreshToken,
			ProfileARN:   profileARN,
		})
	}
	return NewProviderWithAccounts(id, accs)
}

func NewProviderWithAccounts(id string, accounts []*Account) *Provider {
	for _, a := range accounts {
		if a.ProfileARN == "" {
			a.ProfileARN = "arn:aws:codewhisperer:us-east-1:699475941385:profile/EHGA3GRVQMUK"
		}
	}
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
	return "kiro"
}

func (p *Provider) IsHealthy() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.accounts) > 0
}

func (p *Provider) nextAccount(ctx context.Context, pinnedID string) *Account {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.accounts) == 0 {
		return nil
	}
	if pinnedID != "" {
		for _, a := range p.accounts {
			if a.ID == pinnedID {
				return a
			}
		}
	}

	targetPool := "pool1"
	if ctx != nil {
		if pVal, ok := ctx.Value("goroute-pool").(string); ok && pVal != "" {
			targetPool = pVal
		}
	}

	var poolAccounts []*Account
	for _, a := range p.accounts {
		accPool := a.Pool
		if accPool == "" {
			accPool = "pool1"
		}
		if accPool == targetPool {
			poolAccounts = append(poolAccounts, a)
		}
	}
	if len(poolAccounts) == 0 {
		poolAccounts = p.accounts
	}

	var candidateIDs []string
	for _, a := range poolAccounts {
		candidateIDs = append(candidateIDs, a.ID)
	}

	strat := p.strategy
	if strat == "" {
		strat = "priority"
	}
	pl := pool.GlobalPool.GetOrCreatePool(targetPool, strat)
	bestID := pl.SelectBestAccount(candidateIDs)
	if bestID != "" {
		for _, a := range poolAccounts {
			if a.ID == bestID {
				return a
			}
		}
	}

	acc := poolAccounts[p.counter%uint64(len(poolAccounts))]
	p.counter++
	return acc
}

func (p *Provider) Accounts() []*Account {
	p.mu.RLock()
	defer p.mu.RUnlock()
	res := make([]*Account, len(p.accounts))
	copy(res, p.accounts)
	return res
}

func (p *Provider) TestAccount(ctx context.Context, accountID string) error {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if len(p.accounts) == 0 {
		return fmt.Errorf("no accounts configured for kiro provider")
	}
	var targetAcc *Account
	if accountID == "" {
		targetAcc = p.accounts[0]
	} else {
		for _, a := range p.accounts {
			if a.ID == accountID {
				targetAcc = a
				break
			}
		}
	}
	if targetAcc == nil {
		return fmt.Errorf("account %q not found in kiro provider", accountID)
	}

	token, err := targetAcc.GetValidAccessToken(ctx, p.client)
	if err != nil {
		return fmt.Errorf("token validation failed: %w", err)
	}

	// Runtime probe to verify credentials against AWS
	probeURL := "https://codewhisperer.us-east-1.amazonaws.com/generateAssistantResponse"
	if targetAcc.APIKey != "" {
		probeURL = "https://q.us-east-1.amazonaws.com/generateAssistantResponse"
	}
	profileArn := targetAcc.ProfileARN
	if profileArn == "" {
		profileArn = "arn:aws:codewhisperer:us-east-1:699475941385:profile/EHGA3GRVQMUK"
	}
	probeBody := map[string]interface{}{
		"profileArn": profileArn,
		"conversationState": map[string]interface{}{
			"chatTriggerType": "MANUAL",
			"conversationId": fmt.Sprintf("probe-%d", time.Now().UnixNano()),
			"currentMessage": map[string]interface{}{
				"userInputMessage": map[string]interface{}{
					"content": "ping",
					"modelId": "auto",
					"origin":  "AI_EDITOR",
				},
			},
			"history": []interface{}{},
		},
		"inferenceConfig": map[string]interface{}{
			"maxTokens": 1,
		},
	}
	b, _ := json.Marshal(probeBody)
	probeReq, err := http.NewRequestWithContext(ctx, "POST", probeURL, bytes.NewReader(b))
	if err != nil {
		return nil
	}
	probeReq.Header.Set("Content-Type", "application/x-amz-json-1.0")
	probeReq.Header.Set("X-Amz-Target", "AmazonCodeWhispererStreamingService.GenerateAssistantResponse")
	probeReq.Header.Set("Accept", "application/vnd.amazon.eventstream")
	probeReq.Header.Set("Authorization", "Bearer "+token)
	if targetAcc.APIKey != "" {
		probeReq.Header.Set("tokentype", "API_KEY")
	}

	resp, err := p.client.Do(probeReq)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("AWS Kiro rejected credential (HTTP %d): %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

func (p *Provider) TestConnection(ctx context.Context) error {
	return p.TestAccount(ctx, "")
}

type kiroToolSpec struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

type kiroToolItem struct {
	ToolSpecification kiroToolSpec `json:"toolSpecification"`
}

type kiroToolUse struct {
	ToolUseId string      `json:"toolUseId"`
	Name      string      `json:"name"`
	Input     interface{} `json:"input"`
}

type kiroToolResultText struct {
	Text string `json:"text"`
}

type kiroToolResult struct {
	ToolUseId string               `json:"toolUseId"`
	Status    string               `json:"status"` // "success" or "error"
	Content   []kiroToolResultText `json:"content"`
}

type kiroUserInputContext struct {
	Tools       []kiroToolItem   `json:"tools,omitempty"`
	ToolResults []kiroToolResult `json:"toolResults,omitempty"`
}

type kiroUserMsg struct {
	Content                 string                `json:"content"`
	ModelID                 string                `json:"modelId"`
	Origin                  string                `json:"origin"`
	UserInputMessageContext *kiroUserInputContext `json:"userInputMessageContext,omitempty"`
}

type kiroAssistMsg struct {
	Content  string        `json:"content"`
	ToolUses []kiroToolUse `json:"toolUses,omitempty"`
}

type kiroHistoryTurn struct {
	UserInputMessage        *kiroUserMsg   `json:"userInputMessage,omitempty"`
	AssistantResponseMessage *kiroAssistMsg `json:"assistantResponseMessage,omitempty"`
}

type kiroCurrentMsg struct {
	UserInputMessage kiroUserMsg `json:"userInputMessage"`
}

type kiroConversationState struct {
	CurrentMessage  kiroCurrentMsg    `json:"currentMessage"`
	ChatTriggerType string            `json:"chatTriggerType"`
	ConversationID  string            `json:"conversationId"`
	History         []kiroHistoryTurn `json:"history,omitempty"`
}

type kiroRequestPayload struct {
	ConversationState kiroConversationState `json:"conversationState"`
	ProfileArn        string                `json:"profileArn"`
}

func randomUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

func (p *Provider) ChatCompletion(ctx context.Context, req *providers.ChatRequest) (io.ReadCloser, bool, int, error) {
	p.mu.RLock()
	accCount := len(p.accounts)
	p.mu.RUnlock()

	if accCount == 0 {
		return nil, false, http.StatusServiceUnavailable, fmt.Errorf("no active accounts configured for kiro provider %s", p.id)
	}

	modelID := "claude-sonnet-4.5"
	lowerModel := strings.ToLower(req.Model)
	if strings.Contains(lowerModel, "haiku") {
		modelID = "claude-haiku-4.5"
	} else if strings.Contains(lowerModel, "sonnet-5") {
		modelID = "claude-sonnet-5"
	} else if strings.Contains(lowerModel, "deepseek") {
		modelID = "deepseek-3.2"
	} else if strings.Contains(lowerModel, "luna") {
		modelID = "gpt-5.6-luna"
	} else if strings.Contains(lowerModel, "qwen") {
		modelID = "qwen3-coder-next"
	}

	// Prepare messages
	var userContent string
	var currentToolResults []kiroToolResult
	var history []kiroHistoryTurn

	lastAssistantToolUseIDs := make(map[string]bool)
	lastAssistantToolUseCount := 0

	for i, m := range req.Messages {
		isLast := i == len(req.Messages)-1

		if m.Role == "assistant" {
			text := ""
			var toolUses []kiroToolUse
			assistantToolUseIDs := make(map[string]bool)

			// 1. Parse OpenAI format m.ToolCalls
			for _, tc := range m.ToolCalls {
				if tcMap, ok := tc.(map[string]interface{}); ok {
					id, _ := tcMap["id"].(string)
					name := ""
					var input interface{} = map[string]interface{}{}
					if fn, ok := tcMap["function"].(map[string]interface{}); ok {
						name, _ = fn["name"].(string)
						if argsStr, ok := fn["arguments"].(string); ok {
							var parsedArgs interface{}
							if err := json.Unmarshal([]byte(argsStr), &parsedArgs); err == nil {
								input = parsedArgs
							}
						} else if fn["arguments"] != nil {
							input = fn["arguments"]
						}
					}
					if id != "" {
						toolUses = append(toolUses, kiroToolUse{
							ToolUseId: id,
							Name:      name,
							Input:     input,
						})
						assistantToolUseIDs[id] = true
					}
				}
			}

			// 2. Parse Anthropic format content blocks
			switch v := m.Content.(type) {
			case string:
				text = v
			case []interface{}:
				for _, item := range v {
					if bm, ok := item.(map[string]interface{}); ok {
						if bm["type"] == "text" {
							if tStr, ok := bm["text"].(string); ok {
								text += tStr + "\n"
							}
						} else if bm["type"] == "tool_use" {
							id, _ := bm["id"].(string)
							name, _ := bm["name"].(string)
							input := bm["input"]
							if id != "" && !assistantToolUseIDs[id] {
								toolUses = append(toolUses, kiroToolUse{
									ToolUseId: id,
									Name:      name,
									Input:     input,
								})
								assistantToolUseIDs[id] = true
							}
						}
					}
				}
			}

			lastAssistantToolUseIDs = assistantToolUseIDs
			lastAssistantToolUseCount = len(toolUses)

			history = append(history, kiroHistoryTurn{
				AssistantResponseMessage: &kiroAssistMsg{
					Content:  strings.TrimSpace(text),
					ToolUses: toolUses,
				},
			})
		} else {
			// user, system, or tool
			text := ""
			var rawToolResults []kiroToolResult

			// 1. Direct tool role message (OpenAI format)
			if m.Role == "tool" && m.ToolCallID != "" {
				resContent := ""
				switch c := m.Content.(type) {
				case string:
					resContent = c
				default:
					b, _ := json.Marshal(c)
					resContent = string(b)
				}
				if resContent == "" {
					resContent = "(no output)"
				} else if len(resContent) > 200 {
					filtered := rtk.GlobalEngine.FilterCommandOutput("", resContent)
					resContent = filtered.Output
				}
				rawToolResults = append(rawToolResults, kiroToolResult{
					ToolUseId: m.ToolCallID,
					Status:    "success",
					Content:   []kiroToolResultText{{Text: resContent}},
				})
			}

			// 2. Content blocks (Anthropic format or text)
			switch v := m.Content.(type) {
			case string:
				if m.Role != "tool" {
					text = v
				}
			case []interface{}:
				for _, item := range v {
					if bm, ok := item.(map[string]interface{}); ok {
						if bm["type"] == "text" {
							if tStr, ok := bm["text"].(string); ok {
								text += tStr + "\n"
							}
						} else if bm["type"] == "tool_result" {
							toolUseId, _ := bm["tool_use_id"].(string)
							isErr, _ := bm["is_error"].(bool)
							status := "success"
							if isErr {
								status = "error"
							}
							resContent := ""
							switch c := bm["content"].(type) {
							case string:
								resContent = c
							default:
								b, _ := json.Marshal(c)
								resContent = string(b)
							}
							if resContent == "" {
								resContent = "(no output)"
							} else if len(resContent) > 200 {
								filtered := rtk.GlobalEngine.FilterCommandOutput("", resContent)
								resContent = filtered.Output
							}
							rawToolResults = append(rawToolResults, kiroToolResult{
								ToolUseId: toolUseId,
								Status:    status,
								Content:   []kiroToolResultText{{Text: resContent}},
							})
						}
					}
				}
			}

			// 3. Strict Bedrock Invariant Guard:
			// Prevents: "The number of toolResult blocks at messages.N.content exceeds the number of toolUse blocks of previous turn."
			// Any toolResult whose ID does not match the previous assistant turn, or exceeds the count,
			// is safely converted into plain text context inside Content.
			var validToolResults []kiroToolResult
			for _, tr := range rawToolResults {
				trText := ""
				if len(tr.Content) > 0 {
					trText = tr.Content[0].Text
				}

				if lastAssistantToolUseIDs[tr.ToolUseId] && len(validToolResults) < lastAssistantToolUseCount {
					validToolResults = append(validToolResults, tr)
					delete(lastAssistantToolUseIDs, tr.ToolUseId)
				} else {
					if text != "" {
						text += "\n"
					}
					text += fmt.Sprintf("[Tool Result %s]:\n%s", tr.ToolUseId, trText)
				}
			}

			// Once consumed by a user turn, reset the counter
			lastAssistantToolUseCount = 0
			lastAssistantToolUseIDs = make(map[string]bool)

			text = strings.TrimSpace(text)
			if isLast {
				userContent = text
				currentToolResults = validToolResults
			} else {
				var uCtx *kiroUserInputContext
				if len(validToolResults) > 0 {
					uCtx = &kiroUserInputContext{ToolResults: validToolResults}
				}
				history = append(history, kiroHistoryTurn{
					UserInputMessage: &kiroUserMsg{
						Content:                 text,
						ModelID:                 modelID,
						Origin:                  "AI_EDITOR",
						UserInputMessageContext: uCtx,
					},
				})
			}
		}
	}

	if userContent == "" && len(currentToolResults) == 0 {
		userContent = "(empty prompt)"
	}

	var kiroTools []kiroToolItem
	for _, t := range req.Tools {
		m, ok := t.(map[string]interface{})
		if !ok {
			continue
		}
		name, _ := m["name"].(string)
		desc, _ := m["description"].(string)
		schema := m["input_schema"]
		if schema == nil {
			if fn, ok := m["function"].(map[string]interface{}); ok {
				name, _ = fn["name"].(string)
				desc, _ = fn["description"].(string)
				schema = fn["parameters"]
			}
		}
		schemaMap, ok := NormalizeKiroToolSchema(schema).(map[string]interface{})
		if !ok {
			schemaMap = map[string]interface{}{"type": "object", "properties": map[string]interface{}{}}
		}
		kiroTools = append(kiroTools, kiroToolItem{
			ToolSpecification: kiroToolSpec{
				Name:        name,
				Description: desc,
				InputSchema: map[string]interface{}{
					"json": schemaMap,
				},
			},
		})
	}

	var userCtx *kiroUserInputContext
	if len(kiroTools) > 0 || len(currentToolResults) > 0 {
		userCtx = &kiroUserInputContext{
			Tools:       kiroTools,
			ToolResults: currentToolResults,
		}
	}

	convState := kiroConversationState{
		CurrentMessage: kiroCurrentMsg{
			UserInputMessage: kiroUserMsg{
				Content:                 userContent,
				ModelID:                 modelID,
				Origin:                  "AI_EDITOR",
				UserInputMessageContext: userCtx,
			},
		},
		ChatTriggerType: "MANUAL",
		ConversationID:  randomUUID(),
		History:         history,
	}

	// Full fidelity passthrough by default; auto-recovery only triggers if AWS returns CONTENT_LENGTH_EXCEEDS_THRESHOLD

	sessionKey := affinity.DeriveSessionKey(nil, req)
	pinnedAcc := affinity.GlobalManager.GetAffinity(sessionKey, p.id)

	var lastErr error
	var lastStatusCode int
	thresholdRetries := 0

	for attempt := 0; attempt < accCount; attempt++ {
		var acc *Account
		if attempt == 0 && pinnedAcc != "" {
			acc = p.nextAccount(ctx, pinnedAcc)
		} else {
			acc = p.nextAccount(ctx, "")
		}
		if acc == nil {
			break
		}

		token, err := acc.GetValidAccessToken(ctx, p.client)
		if err != nil {
			lastErr = fmt.Errorf("kiro account %s auth failed: %w", acc.ID, err)
			lastStatusCode = http.StatusUnauthorized
			log.Printf("[kiro] Account %s auth error: %v", acc.ID, err)
			continue
		}

		payload := kiroRequestPayload{
			ConversationState: convState,
			ProfileArn:        acc.ProfileARN,
		}

		bodyBytes, err := json.Marshal(payload)
		if err != nil {
			return nil, false, 0, fmt.Errorf("marshal kiro payload: %w", err)
		}

		url := "https://codewhisperer.us-east-1.amazonaws.com/generateAssistantResponse"
		if acc.APIKey != "" {
			url = "https://q.us-east-1.amazonaws.com/generateAssistantResponse"
		}
		httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
		if err != nil {
			return nil, false, 0, fmt.Errorf("create kiro request: %w", err)
		}

		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Accept", "application/vnd.amazon.eventstream")
		httpReq.Header.Set("X-Amz-Target", "AmazonCodeWhispererStreamingService.GenerateAssistantResponse")
		httpReq.Header.Set("Authorization", "Bearer "+token)
		if acc.APIKey != "" {
			httpReq.Header.Set("tokentype", "API_KEY")
		}
		httpReq.Header.Set("User-Agent", "AWS-SDK-JS/3.0.0 kiro-ide/1.0.0")
		httpReq.Header.Set("X-Amz-User-Agent", "aws-sdk-js/3.0.0 kiro-ide/1.0.0")

		resp, err := p.client.Do(httpReq)
		if err != nil {
			lastErr = fmt.Errorf("kiro upstream call failed on account %s: %w", acc.ID, err)
			lastStatusCode = http.StatusBadGateway
			log.Printf("[kiro] Account %s network call failed: %v", acc.ID, err)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			raw, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			lastErr = fmt.Errorf("kiro upstream error %d: %s", resp.StatusCode, string(raw))
			lastStatusCode = resp.StatusCode

			// Auto-recovery for AWS CodeWhisperer CONTENT_LENGTH_EXCEEDS_THRESHOLD
			if resp.StatusCode == 400 && strings.Contains(string(raw), "CONTENT_LENGTH_EXCEEDS_THRESHOLD") && thresholdRetries < 3 {
				thresholdRetries++
				targetBytes := 120000 / thresholdRetries
				log.Printf("[kiro] CONTENT_LENGTH_EXCEEDS_THRESHOLD detected (attempt %d/3). Pruning context window to %d bytes and retrying...", thresholdRetries, targetBytes)
				pruneKiroConversationState(&convState, targetBytes)
				attempt-- // Retry without consuming account attempt
				continue
			}

			if resp.StatusCode == 429 || resp.StatusCode == 402 || resp.StatusCode == 401 || resp.StatusCode == 403 || resp.StatusCode == 404 || resp.StatusCode >= 500 {
				log.Printf("[kiro] Account %s returned %d (%s), cooling down and failover to next account (%d/%d)", acc.ID, resp.StatusCode, string(raw), attempt+1, accCount)
				affinity.GlobalManager.InvalidateAffinity(sessionKey, p.id)
				cooldownDuration := 5 * time.Minute
				if resp.StatusCode == 402 {
					cooldownDuration = 6 * time.Hour
				}
				pool.GlobalPool.SetCooldown(acc.Pool, acc.ID, cooldownDuration)
				continue
			}
			return nil, false, resp.StatusCode, lastErr
		}

		if sessionKey != "" {
			affinity.GlobalManager.SetAffinity(sessionKey, p.id, acc.ID)
		}

		// Record token / request consumption in Fair-Share Pool
		pool.GlobalPool.RecordUsage(acc.Pool, acc.ID, 100)

		pr, pw := io.Pipe()

		go func() {
		defer resp.Body.Close()
		defer pw.Close()

		created := time.Now().Unix()
		respID := fmt.Sprintf("chatcmpl-kiro-%d", time.Now().UnixNano())

		// Read buffer for eventstream
		buf := make([]byte, 0, 64*1024)
		readBuf := make([]byte, 4096)

		for {
			n, rErr := resp.Body.Read(readBuf)
			if n > 0 {
				buf = append(buf, readBuf[:n]...)

				// Process complete event frames
				for len(buf) >= 16 {
					totalLen := binary.BigEndian.Uint32(buf[0:4])
					if totalLen < 16 || totalLen > 10*1024*1024 {
						// Malformed, skip
						buf = buf[1:]
						continue
					}
					if uint32(len(buf)) < totalLen {
						// Wait for complete frame
						break
					}

					headersLen := binary.BigEndian.Uint32(buf[4:8])
					frame := buf[:totalLen]
					buf = buf[totalLen:]

					// Parse headers
					var eventType string
					offset := 12
					headerEnd := int(12 + headersLen)
					for offset < headerEnd && offset < len(frame) {
						nameLen := int(frame[offset])
						offset++
						if offset+nameLen > len(frame) {
							break
						}
						headerName := string(frame[offset : offset+nameLen])
						offset += nameLen
						if offset >= len(frame) {
							break
						}
						headerType := frame[offset]
						offset++
						if headerType == 7 { // String type
							if offset+2 > len(frame) {
								break
							}
							valLen := int(binary.BigEndian.Uint16(frame[offset : offset+2]))
							offset += 2
							if offset+valLen > len(frame) {
								break
							}
							val := string(frame[offset : offset+valLen])
							offset += valLen
							if headerName == ":event-type" {
								eventType = val
							}
						} else {
							break
						}
					}

					// Parse payload
					payloadStart := 12 + int(headersLen)
					payloadEnd := int(totalLen) - 4
					if payloadEnd > payloadStart && payloadEnd <= len(frame) {
						payloadBytes := frame[payloadStart:payloadEnd]
						if eventType == "assistantResponseEvent" {
							var evt struct {
								Content string `json:"content"`
							}
							if err := json.Unmarshal(payloadBytes, &evt); err == nil && evt.Content != "" {
								chunk := map[string]interface{}{
									"id":      respID,
									"object":  "chat.completion.chunk",
									"created": created,
									"model":   modelID,
									"choices": []map[string]interface{}{
										{
											"index": 0,
											"delta": map[string]string{
												"content": evt.Content,
											},
											"finish_reason": nil,
										},
									},
								}
								chunkBytes, _ := json.Marshal(chunk)
								fmt.Fprintf(pw, "data: %s\n\n", string(chunkBytes))
							}
						} else if eventType == "toolUseEvent" {
							var evt struct {
								ToolUseId string      `json:"toolUseId"`
								Name      string      `json:"name"`
								Input     interface{} `json:"input"`
							}
							if err := json.Unmarshal(payloadBytes, &evt); err == nil && (evt.Name != "" || evt.ToolUseId != "") {
								argsStr := ""
								switch v := evt.Input.(type) {
								case string:
									if v != "null" {
										argsStr = v
									}
								case nil:
									argsStr = ""
								default:
									b, _ := json.Marshal(v)
									s := string(b)
									if s != "null" {
										argsStr = s
									}
								}
								chunk := map[string]interface{}{
									"id":      respID,
									"object":  "chat.completion.chunk",
									"created": created,
									"model":   modelID,
									"choices": []map[string]interface{}{
										{
											"index": 0,
											"delta": map[string]interface{}{
												"tool_calls": []map[string]interface{}{
													{
														"index": 0,
														"id":    evt.ToolUseId,
														"type":  "function",
														"function": map[string]string{
															"name":      evt.Name,
															"arguments": argsStr,
														},
													},
												},
											},
											"finish_reason": "tool_calls",
										},
									},
								}
								chunkBytes, _ := json.Marshal(chunk)
								fmt.Fprintf(pw, "data: %s\n\n", string(chunkBytes))
							}
						}
					}
				}
			}
			if rErr != nil {
				break
			}
		}

		// Emit final finish chunk and DONE
		finishChunk := map[string]interface{}{
			"id":      respID,
			"object":  "chat.completion.chunk",
			"created": created,
			"model":   modelID,
			"choices": []map[string]interface{}{
				{
					"index":         0,
					"delta":         map[string]string{},
					"finish_reason": "stop",
				},
			},
		}
		chunkBytes, _ := json.Marshal(finishChunk)
		fmt.Fprintf(pw, "data: %s\n\n", string(chunkBytes))
		fmt.Fprintf(pw, "data: [DONE]\n\n")
	}()

		return pr, true, http.StatusOK, nil
	}

	return nil, false, lastStatusCode, lastErr
}

func (p *Provider) Embeddings(ctx context.Context, req *providers.EmbeddingRequest) (*providers.EmbeddingResponse, error) {
	return nil, fmt.Errorf("embeddings not supported by Kiro")
}

func (p *Provider) Models(ctx context.Context) ([]providers.ModelInfo, error) {
	return []providers.ModelInfo{
		{ID: "claude-sonnet-4.5", Object: "model", OwnedBy: "kiro"},
		{ID: "claude-haiku-4.5", Object: "model", OwnedBy: "kiro"},
		{ID: "deepseek-3.2", Object: "model", OwnedBy: "kiro"},
		{ID: "qwen3-coder-next", Object: "model", OwnedBy: "kiro"},
		{ID: "gpt-5.6-luna", Object: "model", OwnedBy: "kiro"},
	}, nil
}

// pruneKiroConversationState dynamically compresses and sliding-windows conversation history to prevent AWS CodeWhisperer CONTENT_LENGTH_EXCEEDS_THRESHOLD
func pruneKiroConversationState(state *kiroConversationState, maxBytes int) {
	if state == nil {
		return
	}

	// 0. Compact tool descriptions first to free up massive space (e.g. 200+ MCP tools)
	if state.CurrentMessage.UserInputMessage.UserInputMessageContext != nil {
		tools := state.CurrentMessage.UserInputMessage.UserInputMessageContext.Tools
		for i := range tools {
			if len(tools[i].ToolSpecification.Description) > 100 {
				tools[i].ToolSpecification.Description = tools[i].ToolSpecification.Description[:100] + "..."
			}
		}
	}

	// 1. Truncate tool results in history turns to compact size
	for i := range state.History {
		turn := &state.History[i]
		if turn.UserInputMessage != nil && turn.UserInputMessage.UserInputMessageContext != nil {
			for j := range turn.UserInputMessage.UserInputMessageContext.ToolResults {
				tr := &turn.UserInputMessage.UserInputMessageContext.ToolResults[j]
				for k := range tr.Content {
					if len(tr.Content[k].Text) > 1024 {
						tr.Content[k].Text = tr.Content[k].Text[:1024] + "\n... [truncated to fit AWS limit]"
					}
				}
			}
		}
	}

	// 2. Truncate current message tool results if excessively huge
	if state.CurrentMessage.UserInputMessage.UserInputMessageContext != nil {
		for j := range state.CurrentMessage.UserInputMessage.UserInputMessageContext.ToolResults {
			tr := &state.CurrentMessage.UserInputMessage.UserInputMessageContext.ToolResults[j]
			for k := range tr.Content {
				if len(tr.Content[k].Text) > 4096 {
					tr.Content[k].Text = tr.Content[k].Text[:4096] + "\n... [truncated to fit AWS limit]"
				}
			}
		}
	}

	// 3. Sliding window on history turns: drop in assistant+user pairs (2 turns) to prevent TOOL_USE_RESULT_MISMATCH
	for len(state.History) >= 3 {
		b, err := json.Marshal(state)
		if err == nil && len(b) <= maxBytes {
			break
		}
		// Drop a full pair (2 turns: assistant + user)
		if len(state.History) >= 4 {
			state.History = append(state.History[:1], state.History[3:]...)
		} else {
			break
		}
	}
}
