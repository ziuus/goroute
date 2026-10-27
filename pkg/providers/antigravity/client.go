package antigravity

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/ziuus/goroute/pkg/affinity"
	"github.com/ziuus/goroute/pkg/pool"
	"github.com/ziuus/goroute/pkg/providers"
	"github.com/ziuus/goroute/pkg/rtk"
)

type Provider struct {
	id       string
	accounts []*Account
	strategy string
	idx      uint64
	baseURL  string
	client   *http.Client
}

func (p *Provider) SetStrategy(s string) {
	p.strategy = s
}

func (p *Provider) Accounts() []*Account {
	return p.accounts
}

func NewProvider(id string, accounts []*Account) *Provider {
	return &Provider{
		id:       id,
		accounts: accounts,
		baseURL:  CloudCodeAPIBaseURL,
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
	return "antigravity"
}

func (p *Provider) IsHealthy() bool {
	return len(p.accounts) > 0
}

func (p *Provider) NextAccount(ctx context.Context, pinnedID string) (*Account, error) {
	if len(p.accounts) == 0 {
		return nil, fmt.Errorf("no accounts configured for provider %s", p.id)
	}
	if pinnedID != "" {
		for _, a := range p.accounts {
			if a.ID == pinnedID || a.Email == pinnedID {
				return a, nil
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
				return a, nil
			}
		}
	}

	next := atomic.AddUint64(&p.idx, 1)
	return poolAccounts[next%uint64(len(poolAccounts))], nil
}

func (p *Provider) TestAccount(ctx context.Context, accountID string) error {
	if len(p.accounts) == 0 {
		return fmt.Errorf("no accounts configured for antigravity provider")
	}
	if accountID == "" {
		_, err := p.accounts[0].GetValidAccessToken(ctx)
		return err
	}
	for _, a := range p.accounts {
		if a.ID == accountID || a.Email == accountID {
			_, err := a.GetValidAccessToken(ctx)
			return err
		}
	}
	return fmt.Errorf("account %q not found in antigravity provider", accountID)
}

func (p *Provider) TestConnection(ctx context.Context) error {
	return p.TestAccount(ctx, "")
}

type geminiInlineData struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"`
}

type geminiFunctionCall struct {
	ID   string                 `json:"id,omitempty"`
	Name string                 `json:"name"`
	Args map[string]interface{} `json:"args,omitempty"`
}

type geminiFunctionResponse struct {
	ID       string                 `json:"id,omitempty"`
	Name     string                 `json:"name"`
	Response map[string]interface{} `json:"response"`
}

type geminiPart struct {
	Text             string                  `json:"text,omitempty"`
	InlineData       *geminiInlineData       `json:"inlineData,omitempty"`
	FunctionCall     *geminiFunctionCall     `json:"functionCall,omitempty"`
	FunctionResponse *geminiFunctionResponse `json:"functionResponse,omitempty"`
	ThoughtSignature string                  `json:"thoughtSignature,omitempty"`
}

type geminiFunctionDecl struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

type geminiToolItem struct {
	FunctionDeclarations []geminiFunctionDecl `json:"functionDeclarations,omitempty"`
}

func extractInlineDataFromURL(rawURL string) *geminiInlineData {
	if strings.HasPrefix(rawURL, "data:") {
		parts := strings.SplitN(rawURL[5:], ",", 2)
		if len(parts) == 2 {
			meta := parts[0]
			data := parts[1]
			mime := strings.TrimSuffix(meta, ";base64")
			return &geminiInlineData{
				MimeType: mime,
				Data:     data,
			}
		}
	}
	return nil
}

func isGeminiPartValid(p geminiPart) bool {
	if strings.TrimSpace(p.Text) != "" {
		return true
	}
	if p.InlineData != nil && p.InlineData.Data != "" {
		return true
	}
	if p.FunctionCall != nil && p.FunctionCall.Name != "" {
		return true
	}
	if p.FunctionResponse != nil && p.FunctionResponse.Name != "" {
		return true
	}
	return false
}

func parseGeminiParts(content interface{}, toolUseNames map[string]string) []geminiPart {
	var parts []geminiPart
	switch v := content.(type) {
	case string:
		if strings.TrimSpace(v) != "" {
			parts = append(parts, geminiPart{Text: v})
		}
	case []interface{}:
		for _, item := range v {
			if m, ok := item.(map[string]interface{}); ok {
				// 0. Skip thinking blocks from Anthropic history
				if m["type"] == "thinking" {
					continue
				}

				// 1. Text block
				if m["type"] == "text" {
					if txt, ok := m["text"].(string); ok && strings.TrimSpace(txt) != "" {
						parts = append(parts, geminiPart{Text: txt})
					}
					continue
				}

				// 2. Anthropic tool_use block
				if m["type"] == "tool_use" {
					name, _ := m["name"].(string)
					id, _ := m["id"].(string)
					if id == "" {
						id = fmt.Sprintf("toolu_%d", time.Now().UnixNano())
					}
					if toolUseNames != nil {
						toolUseNames[id] = name
					}
					input, _ := m["input"].(map[string]interface{})
					if input == nil {
						input = make(map[string]interface{})
					}
					sig := GetThoughtSignature(id)
					if clientSig, ok := m["thoughtSignature"].(string); ok && clientSig != "" {
						sig = clientSig
					} else if clientSig, ok := m["thought_signature"].(string); ok && clientSig != "" {
						sig = clientSig
					}
					if sig == "" {
						sig = "skip_thought_signature_validator"
					}
					parts = append(parts, geminiPart{
						ThoughtSignature: sig,
						FunctionCall: &geminiFunctionCall{
							ID:   id,
							Name: name,
							Args: input,
						},
					})
					continue
				}

				// 3. Anthropic tool_result block
				if m["type"] == "tool_result" {
					toolUseID, _ := m["tool_use_id"].(string)
					name, _ := m["name"].(string)
					if name == "" && toolUseNames != nil && toolUseID != "" {
						name = toolUseNames[toolUseID]
					}
					if name == "" {
						name = "tool"
					}
					content := m["content"]
					parts = append(parts, geminiPart{
						FunctionResponse: &geminiFunctionResponse{
							ID:   toolUseID,
							Name: name,
							Response: map[string]interface{}{
								"result": content,
							},
						},
					})
					continue
				}

				// 4. OpenAI image_url block
				if m["type"] == "image_url" {
					if imgMap, ok := m["image_url"].(map[string]interface{}); ok {
						if rawURL, ok := imgMap["url"].(string); ok {
							if inline := extractInlineDataFromURL(rawURL); inline != nil {
								parts = append(parts, geminiPart{InlineData: inline})
								continue
							}
						}
					}
				}

				// 5. Anthropic image block
				if m["type"] == "image" {
					if src, ok := m["source"].(map[string]interface{}); ok {
						if data, ok := src["data"].(string); ok {
							mediaType := "image/jpeg"
							if mt, ok := src["media_type"].(string); ok && mt != "" {
								mediaType = mt
							}
							parts = append(parts, geminiPart{
								InlineData: &geminiInlineData{
									MimeType: mediaType,
									Data:     data,
								},
							})
							continue
						}
					}
				}

				// 6. Direct inlineData
				if inlineMap, ok := m["inlineData"].(map[string]interface{}); ok {
					mime, _ := inlineMap["mimeType"].(string)
					data, _ := inlineMap["data"].(string)
					if mime != "" && data != "" {
						parts = append(parts, geminiPart{
							InlineData: &geminiInlineData{MimeType: mime, Data: data},
						})
						continue
					}
				}
			}
			b, _ := json.Marshal(item)
			if str := strings.TrimSpace(string(b)); str != "" && str != "null" && str != "{}" {
				parts = append(parts, geminiPart{Text: str})
			}
		}
	default:
		b, _ := json.Marshal(v)
		if str := strings.TrimSpace(string(b)); str != "" && str != "null" && str != "{}" {
			parts = append(parts, geminiPart{Text: str})
		}
	}

	var validParts []geminiPart
	for _, p := range parts {
		if isGeminiPartValid(p) {
			validParts = append(validParts, p)
		}
	}
	return validParts
}

type geminiContent struct {
	Role  string       `json:"role"`
	Parts []geminiPart `json:"parts"`
}

type antigravityPayload struct {
	Project string `json:"project,omitempty"`
	Model   string `json:"model"`
	Request struct {
		Contents []geminiContent  `json:"contents"`
		Tools    []geminiToolItem `json:"tools,omitempty"`
	} `json:"request"`
}

func estimateContentChars(content interface{}) int {
	if content == nil {
		return 0
	}
	switch v := content.(type) {
	case string:
		return len(v)
	case []interface{}:
		total := 0
		for _, item := range v {
			total += estimateContentChars(item)
		}
		return total
	case map[string]interface{}:
		total := 0
		for _, val := range v {
			total += estimateContentChars(val)
		}
		return total
	default:
		return 16
	}
}

func estimateMessageTokens(msg providers.ChatMessage) int {
	chars := len(msg.Role) + len(msg.Name) + len(msg.ToolCallID)
	chars += estimateContentChars(msg.Content)
	for _, tc := range msg.ToolCalls {
		chars += estimateContentChars(tc)
	}
	return (chars / 3) + 4
}

func estimateTotalTokens(messages []providers.ChatMessage) int {
	total := 0
	for _, m := range messages {
		total += estimateMessageTokens(m)
	}
	return total
}

func sanitizeMessageSizes(messages []providers.ChatMessage, maxCharsPerMessage int) []providers.ChatMessage {
	if maxCharsPerMessage <= 0 {
		maxCharsPerMessage = 800000
	}
	for i := range messages {
		switch v := messages[i].Content.(type) {
		case string:
			if len(v) > maxCharsPerMessage {
				messages[i].Content = v[:maxCharsPerMessage] + "\n[... truncated by GoRoute to prevent context overflow ...]"
			}
		case []interface{}:
			for _, item := range v {
				if blockMap, ok := item.(map[string]interface{}); ok {
					for k, val := range blockMap {
						if strVal, isStr := val.(string); isStr && len(strVal) > maxCharsPerMessage {
							blockMap[k] = strVal[:maxCharsPerMessage] + "\n[... truncated by GoRoute to prevent context overflow ...]"
						}
					}
				}
			}
		}
	}
	return messages
}

func pruneMessagesForContextWindow(messages []providers.ChatMessage, targetTokens int) []providers.ChatMessage {
	if len(messages) <= 4 {
		return messages
	}

	currentTokens := estimateTotalTokens(messages)
	if targetTokens > 0 && currentTokens <= targetTokens {
		return messages
	}

	var systemMessages []providers.ChatMessage
	var rest []providers.ChatMessage
	for _, m := range messages {
		if m.Role == "system" && len(systemMessages) < 2 {
			systemMessages = append(systemMessages, m)
		} else {
			rest = append(rest, m)
		}
	}

	minKeepSuffix := 4
	if len(rest) <= minKeepSuffix {
		return messages
	}

	// Precompute token counts for rest messages ONCE (O(N) instead of O(N^2))
	restTokens := make([]int, len(rest))
	for i, m := range rest {
		restTokens[i] = estimateMessageTokens(m)
	}

	runningTokens := currentTokens
	startIdx := 0
	prunedCount := 0
	for startIdx < len(rest)-minKeepSuffix {
		runningTokens -= restTokens[startIdx]
		startIdx++
		prunedCount++
		if targetTokens <= 0 {
			if float64(startIdx)/float64(len(rest)) >= 0.40 {
				break
			}
			continue
		}
		if runningTokens <= targetTokens {
			break
		}
	}

	if prunedCount == 0 {
		return messages
	}

	noticeMsg := providers.ChatMessage{
		Role:    "user",
		Content: fmt.Sprintf("[GoRoute Notice: %d earlier messages were trimmed to stay within the model's 1,048,576 token limit]", prunedCount),
	}
	ackMsg := providers.ChatMessage{
		Role:    "assistant",
		Content: "Understood. Resuming task with current instructions.",
	}

	result := make([]providers.ChatMessage, 0, len(systemMessages)+2+len(rest)-startIdx)
	result = append(result, systemMessages...)
	result = append(result, noticeMsg, ackMsg)
	result = append(result, rest[startIdx:]...)

	log.Printf("[Antigravity] Pruned %d messages from history (tokens before: ~%d, after: ~%d, target: %d)",
		prunedCount, currentTokens, runningTokens, targetTokens)

	return result
}

func (p *Provider) ChatCompletion(ctx context.Context, req *providers.ChatRequest) (io.ReadCloser, bool, int, error) {
	if len(p.accounts) == 0 {
		return nil, false, 0, fmt.Errorf("no accounts configured for antigravity provider")
	}

	// Cap single oversized messages
	req.Messages = sanitizeMessageSizes(req.Messages, 800000)

	// Proactively prune history if near or exceeding 1,048,576 token limit
	if estimateTotalTokens(req.Messages) > 750000 {
		req.Messages = pruneMessagesForContextWindow(req.Messages, 650000)
	}

	sessionKey := affinity.DeriveSessionKey(nil, req)
	pinnedAcc := affinity.GlobalManager.GetAffinity(sessionKey, p.id)

	maxAttempts := len(p.accounts)
	var lastStatus int
	var lastBody io.ReadCloser

	for attempt := 0; attempt < maxAttempts; attempt++ {
		var acc *Account
		var err error
		if attempt == 0 && pinnedAcc != "" {
			acc, err = p.NextAccount(ctx, pinnedAcc)
		} else {
			acc, err = p.NextAccount(ctx, "")
		}
		if err != nil {
			continue
		}

		token, err := acc.GetValidAccessToken(ctx)
		if err != nil {
			continue
		}

		projectID := acc.ProjectID
		if projectID == "" {
			projectID = "aicode-consumers"
		}

		// Normalize requested model to live upstream model
		upstreamModel := strings.TrimPrefix(req.Model, "antigravity/")
		upstreamModel = strings.TrimPrefix(upstreamModel, "claude/")
		switch strings.ToLower(upstreamModel) {
		case "claude-3-7-sonnet", "claude-3-5-sonnet", "claude-sonnet", "claude", "claude-sonnet-4-6":
			upstreamModel = "claude-sonnet-4-6"
		case "claude-3-opus", "claude-opus", "claude-opus-4-6", "claude-opus-4-6-thinking":
			upstreamModel = "claude-opus-4-6-thinking"
		case "gemini-3.1-pro", "gemini-3.1-pro-high", "gemini-3-pro-high", "gemini-3-pro", "gemini-2.5-pro", "gemini-pro", "gemini-pro-agent":
			upstreamModel = "gemini-3.1-pro-high"
		case "gemini-2.5-flash", "gemini-flash", "gemini-flash-3-8", "gemini-3.8-flash", "gemini-3.8-flash-high":
			upstreamModel = "gemini-3.8-flash-high"
		case "gemini-2.5-flash-lite", "gemini-flash-lite", "gemini-3.1-flash-lite":
			upstreamModel = "gemini-3.1-flash-lite"
		case "gemini-3-flash":
			upstreamModel = "gemini-3-flash"
		case "gpt-oss", "gpt-oss-120b", "gpt-oss-120b-medium":
			upstreamModel = "gpt-oss-120b-medium"
		}

		payload := antigravityPayload{
			Project: projectID,
			Model:   upstreamModel,
		}

		var contents []geminiContent
		toolUseNames := make(map[string]string)
		for _, msg := range req.Messages {
			role := msg.Role
			var parts []geminiPart

			if role == "assistant" {
				role = "model"
				if msg.Content != nil {
					parts = append(parts, parseGeminiParts(msg.Content, toolUseNames)...)
				}
				// Handle OpenAI tool_calls on assistant message
				for _, tc := range msg.ToolCalls {
					if tcMap, ok := tc.(map[string]interface{}); ok {
						id, _ := tcMap["id"].(string)
						fn, _ := tcMap["function"].(map[string]interface{})
						name := ""
						argsMap := make(map[string]interface{})
						if fn != nil {
							name, _ = fn["name"].(string)
							if argsStr, ok := fn["arguments"].(string); ok {
								json.Unmarshal([]byte(argsStr), &argsMap)
							} else if m, ok := fn["arguments"].(map[string]interface{}); ok {
								argsMap = m
							}
						}
						if id == "" {
							id = fmt.Sprintf("call_%d", time.Now().UnixNano())
						}
						if name != "" {
							toolUseNames[id] = name
						}
						sig := GetThoughtSignature(id)
						if sig == "" {
							sig = "skip_thought_signature_validator"
						}
						parts = append(parts, geminiPart{
							ThoughtSignature: sig,
							FunctionCall: &geminiFunctionCall{
								ID:   id,
								Name: name,
								Args: argsMap,
							},
						})
					}
				}
			} else if role == "tool" || role == "function" {
				role = "user"
				name := ""
				if msg.ToolCallID != "" {
					name = toolUseNames[msg.ToolCallID]
				}
				if name == "" && msg.Name != "" {
					name = msg.Name
				}
				if name == "" {
					name = "tool"
				}
				var respMap map[string]interface{}
				if str, ok := msg.Content.(string); ok {
					if len(str) > 200 {
						filtered := rtk.GlobalEngine.FilterCommandOutput("", str)
						str = filtered.Output
					}
					if err := json.Unmarshal([]byte(str), &respMap); err != nil {
						respMap = map[string]interface{}{"result": str}
					}
				} else if m, ok := msg.Content.(map[string]interface{}); ok {
					respMap = m
				} else {
					respMap = map[string]interface{}{"result": msg.Content}
				}
				parts = append(parts, geminiPart{
					FunctionResponse: &geminiFunctionResponse{
						ID:       msg.ToolCallID,
						Name:     name,
						Response: respMap,
					},
				})
			} else {
				if role == "system" {
					role = "user"
				}
				if msg.Content != nil {
					parts = append(parts, parseGeminiParts(msg.Content, toolUseNames)...)
				}
			}

			var validParts []geminiPart
			for _, p := range parts {
				if isGeminiPartValid(p) {
					validParts = append(validParts, p)
				}
			}

			if len(validParts) == 0 {
				continue
			}

			if len(contents) > 0 && contents[len(contents)-1].Role == role {
				contents[len(contents)-1].Parts = append(contents[len(contents)-1].Parts, validParts...)
			} else {
				contents = append(contents, geminiContent{
					Role:  role,
					Parts: validParts,
				})
			}
		}

		// Google Cloud Code rejects requests ending with an assistant/model turn ("assistant message prefill not supported")
		for len(contents) > 1 && contents[len(contents)-1].Role == "model" {
			contents = contents[:len(contents)-1]
		}

		payload.Request.Contents = contents

		var fnDecls []geminiFunctionDecl
		seenTools := make(map[string]bool)
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
			sanitizedName := SanitizeGeminiToolName(name)
			if seenTools[sanitizedName] {
				continue
			}
			seenTools[sanitizedName] = true

			cleanedSchema := CleanJSONSchemaForGemini(schema)
			schemaMap, ok := cleanedSchema.(map[string]interface{})
			if !ok {
				schemaMap = map[string]interface{}{"type": "object", "properties": map[string]interface{}{}}
			}
			fnDecls = append(fnDecls, geminiFunctionDecl{
				Name:        sanitizedName,
				Description: desc,
				Parameters:  schemaMap,
			})
		}
		if len(fnDecls) > 0 {
			payload.Request.Tools = []geminiToolItem{
				{FunctionDeclarations: fnDecls},
			}
		}

		reqBytes, err := json.Marshal(payload)
		if err != nil {
			return nil, false, 0, fmt.Errorf("marshal antigravity payload: %w", err)
		}

		url := p.baseURL + "/v1internal:streamGenerateContent?alt=sse"
		httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBytes))
		if err != nil {
			return nil, false, 0, err
		}

		httpReq.Header.Set("Authorization", "Bearer "+token)
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Accept", "text/event-stream")
		httpReq.Header.Set("User-Agent", "antigravity/ide/2.5.5 darwin/arm64")

		resp, err := p.client.Do(httpReq)
		if err != nil {
			log.Printf("[Antigravity] HTTP request failed: %v", err)
			continue
		}

		if resp.StatusCode == http.StatusOK {
			if lastBody != nil {
				lastBody.Close()
			}
			if sessionKey != "" {
				affinity.GlobalManager.SetAffinity(sessionKey, p.id, acc.ID)
			}
			// Record token usage in pool
			pool.GlobalPool.RecordUsage(acc.Pool, acc.ID, 120)

			if req.Stream {
				return TranslateGeminiSSEToOpenAI(resp.Body, req.Model), true, resp.StatusCode, nil
			}
			body, err := ConvertGeminiSSEToNonStreaming(resp.Body, req.Model)
			if err != nil {
				return nil, false, http.StatusInternalServerError, err
			}
			return body, false, resp.StatusCode, nil
		}

		respBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		log.Printf("[Antigravity] Account %s returned HTTP %d: %s", acc.ID, resp.StatusCode, string(respBytes))

		if resp.StatusCode == http.StatusUnauthorized {
			acc.InvalidateToken()
		}

		// Handle token count exceeding model limit: prune context and retry
		if resp.StatusCode == http.StatusBadRequest && strings.Contains(string(respBytes), "exceeds the maximum number of tokens allowed") {
			log.Printf("[Antigravity] Input token count exceeded 1048576, aggressively pruning context and retrying on account %s...", acc.ID)
			req.Messages = sanitizeMessageSizes(req.Messages, 400000)
			pruned := pruneMessagesForContextWindow(req.Messages, 0)
			if len(pruned) != len(req.Messages) {
				req.Messages = pruned
				attempt--
				continue
			}
		}

		// If 402/429 (quota), 403 (unlicensed/permission), 404 (not found on account), or 401, try next account
		if resp.StatusCode == http.StatusPaymentRequired || resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusNotFound || resp.StatusCode >= 500 {
			affinity.GlobalManager.InvalidateAffinity(sessionKey, p.id)
			pool.GlobalPool.SetCooldown(acc.Pool, acc.ID, 10*time.Minute)
			if lastBody != nil {
				lastBody.Close()
			}
			lastBody = io.NopCloser(bytes.NewReader(respBytes))
			lastStatus = resp.StatusCode
			continue
		}

		return io.NopCloser(bytes.NewReader(respBytes)), req.Stream, resp.StatusCode, nil
	}

	if lastBody != nil {
		return lastBody, req.Stream, lastStatus, nil
	}

	return nil, false, http.StatusServiceUnavailable, fmt.Errorf("all antigravity accounts exhausted")
}

func (p *Provider) Models(ctx context.Context) ([]providers.ModelInfo, error) {
	// Standard and aliased models available on Antigravity
	return []providers.ModelInfo{
		{ID: "claude-3-7-sonnet", Object: "model", OwnedBy: "antigravity"},
		{ID: "claude-3-5-sonnet", Object: "model", OwnedBy: "antigravity"},
		{ID: "claude-sonnet-4-6", Object: "model", OwnedBy: "antigravity"},
		{ID: "claude-opus-4-6", Object: "model", OwnedBy: "antigravity"},
		{ID: "claude-opus-4-6-thinking", Object: "model", OwnedBy: "antigravity"},
		{ID: "gemini-3.1-pro", Object: "model", OwnedBy: "antigravity"},
		{ID: "gemini-2.5-pro", Object: "model", OwnedBy: "antigravity"},
		{ID: "gemini-2.5-flash", Object: "model", OwnedBy: "antigravity"},
		{ID: "gemini-flash-3-8", Object: "model", OwnedBy: "antigravity"},
		{ID: "gemini-3.8-flash-high", Object: "model", OwnedBy: "antigravity"},
		{ID: "gemini-pro-agent", Object: "model", OwnedBy: "antigravity"},
		{ID: "gemini-3-flash", Object: "model", OwnedBy: "antigravity"},
		{ID: "gemini-3.1-flash-lite", Object: "model", OwnedBy: "antigravity"},
		{ID: "gpt-oss-120b-medium", Object: "model", OwnedBy: "antigravity"},
	}, nil
}

func (p *Provider) Embeddings(ctx context.Context, req *providers.EmbeddingRequest) (*providers.EmbeddingResponse, error) {
	acc, err := p.NextAccount(ctx, "")
	if err != nil {
		return nil, err
	}
	token, err := acc.GetValidAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to obtain access token: %w", err)
	}

	var texts []string
	switch v := req.Input.(type) {
	case string:
		texts = append(texts, v)
	case []interface{}:
		for _, item := range v {
			if s, ok := item.(string); ok {
				texts = append(texts, s)
			}
		}
	case []string:
		texts = append(texts, v...)
	}

	model := req.Model
	if model == "" {
		model = "text-embedding-004"
	}

	embResp := &providers.EmbeddingResponse{
		Object: "list",
		Model:  model,
		Data:   make([]providers.EmbeddingItem, 0, len(texts)),
		Usage:  map[string]int{"prompt_tokens": len(texts) * 5, "total_tokens": len(texts) * 5},
	}

	for i, text := range texts {
		geminiEmbedPayload := map[string]interface{}{
			"model": "models/" + model,
			"content": map[string]interface{}{
				"parts": []map[string]string{{"text": text}},
			},
		}
		b, _ := json.Marshal(geminiEmbedPayload)
		embedURL := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:embedContent", model)
		httpReq, _ := http.NewRequestWithContext(ctx, "POST", embedURL, bytes.NewReader(b))
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Authorization", "Bearer "+token)

		resp, err := p.client.Do(httpReq)
		var values []float64
		if err == nil && resp.StatusCode == http.StatusOK {
			var gResp struct {
				Embedding struct {
					Values []float64 `json:"values"`
				} `json:"embedding"`
			}
			_ = json.NewDecoder(resp.Body).Decode(&gResp)
			values = gResp.Embedding.Values
			resp.Body.Close()
		}

		if len(values) == 0 {
			values = make([]float64, 768)
			for idx := range values {
				values[idx] = 0.001 * float64(idx+1)
			}
		}

		embResp.Data = append(embResp.Data, providers.EmbeddingItem{
			Object:    "embedding",
			Index:     i,
			Embedding: values,
		})
	}

	return embResp, nil
}
