package proxy

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/ziuus/goroute/pkg/providers"
	"github.com/ziuus/goroute/pkg/storage"
)

// AnthropicMessageRequest represents an incoming /v1/messages POST request
type AnthropicMessageRequest struct {
	Model       string                 `json:"model"`
	Messages    []AnthropicMessage     `json:"messages"`
	System      interface{}            `json:"system,omitempty"` // string or []contentBlock
	MaxTokens   int                    `json:"max_tokens,omitempty"`
	Stream      bool                   `json:"stream,omitempty"`
	Temperature *float64               `json:"temperature,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Tools       []interface{}          `json:"tools,omitempty"`
	ToolChoice  interface{}            `json:"tool_choice,omitempty"`
}

type AnthropicMessage struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"` // string or []AnthropicContentBlock
}

type AnthropicContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

func (s *Server) handleMessages(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// AnthropicNativeProvider is an optional interface for providers that can natively handle
	// Anthropic /v1/messages POST payloads without protocol translation (e.g. agentrouter, Anthropic upstream).
	type AnthropicNativeProvider interface {
		SupportsAnthropic() bool
		AnthropicMessages(ctx context.Context, body []byte, incomingHeaders http.Header) (io.ReadCloser, bool, int, error)
	}

	var reqMap map[string]interface{}
	var modelStr string
	if err := json.Unmarshal(bodyBytes, &reqMap); err == nil {
		modelStr, _ = reqMap["model"].(string)
	}

	// Validate virtual key if keys are registered
	if s.keyMgr != nil && s.keyMgr.HasKeys() {
		ok, _, errMsg := s.keyMgr.ValidateRequest(r, modelStr)
		if !ok {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"type": "error",
				"error": map[string]interface{}{
					"type":    "authentication_error",
					"message": errMsg,
				},
			})
			return
		}
	}

	if reqMap != nil {
		if target, ok := s.router.ResolveTarget(modelStr); ok {
			if prov, ok := s.router.GetProvider(target.ProviderID); ok {
				if aProv, ok := prov.(AnthropicNativeProvider); ok && aProv.SupportsAnthropic() {
					reqMap["model"] = target.Model
					if newBody, err := json.Marshal(reqMap); err == nil {
						bodyBytes = newBody
					}
					respBody, isStream, statusCode, err := aProv.AnthropicMessages(r.Context(), bodyBytes, r.Header)
					if err == nil && statusCode >= 200 && statusCode < 400 {
						defer respBody.Close()

						if isStream {
							w.Header().Set("Content-Type", "text/event-stream")
							w.Header().Set("Cache-Control", "no-cache")
							w.Header().Set("Connection", "keep-alive")
							w.WriteHeader(statusCode)
							flusher, _ := w.(http.Flusher)
							buf := make([]byte, 4096)
							for {
								n, err := respBody.Read(buf)
								if n > 0 {
									w.Write(buf[:n])
									if flusher != nil {
										flusher.Flush()
									}
								}
								if err != nil {
									break
								}
							}
							RecordRequest(RequestLog{
								ID:         fmt.Sprintf("req_%d", time.Now().UnixNano()),
								Timestamp:  start,
								Endpoint:   "/v1/messages",
								Model:      modelStr,
								Provider:   target.ProviderID,
								StatusCode: statusCode,
								DurationMs: time.Since(start).Milliseconds(),
							})
							return
						}

						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(statusCode)
						io.Copy(w, respBody)
						RecordRequest(RequestLog{
							ID:         fmt.Sprintf("req_%d", time.Now().UnixNano()),
							Timestamp:  start,
							Endpoint:   "/v1/messages",
							Model:      modelStr,
							Provider:   target.ProviderID,
							StatusCode: statusCode,
							DurationMs: time.Since(start).Milliseconds(),
						})
						return
					}
					if respBody != nil {
						respBody.Close()
					}
				}
			}
		}
	}

	var req AnthropicMessageRequest
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		http.Error(w, "invalid Anthropic messages JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	var maxTokensPtr *int
	if req.MaxTokens > 0 {
		maxTokensPtr = &req.MaxTokens
	}

	req.Model = storage.ResolveModelAlias(req.Model)

	// Translate Anthropic format to OpenAI ChatRequest
	chatReq := providers.ChatRequest{
		Model:       req.Model,
		Stream:      req.Stream,
		MaxTokens:   maxTokensPtr,
		Temperature: req.Temperature,
		Tools:       translateAnthropicToolsToOpenAI(req.Tools),
		ToolChoice:  req.ToolChoice,
		Messages:    make([]providers.ChatMessage, 0, len(req.Messages)+1),
	}

	// Extract system message
	if req.System != nil {
		var sysText string
		switch v := req.System.(type) {
		case string:
			sysText = v
		case []interface{}:
			for _, item := range v {
				if m, ok := item.(map[string]interface{}); ok {
					if txt, ok := m["text"].(string); ok {
						sysText += txt + "\n"
					}
				}
			}
		}
		if sysText != "" {
			chatReq.Messages = append(chatReq.Messages, providers.ChatMessage{
				Role:    "system",
				Content: strings.TrimSpace(sysText),
			})
		}
	}

	// Extract messages with proper tool_use and tool_result translation
	translatedMsgs := translateAnthropicMessagesToOpenAI(req.Messages)
	chatReq.Messages = append(chatReq.Messages, translatedMsgs...)

	// Guardrail messages to prevent context overflow 400 errors
	chatReq.Messages = SanitizeAndGuardrailMessages(chatReq.Messages, 0)

	// Rate limiting / quota
	if !s.limiter.Allow(chatReq.Model) {
		http.Error(w, "Rate limit exceeded (cooling down)", http.StatusTooManyRequests)
		return
	}

	respBody, isStream, statusCode, err := s.router.Execute(r.Context(), &chatReq)
	if err != nil {
		RecordRequest(RequestLog{
			ID:         fmt.Sprintf("req_%d", time.Now().UnixNano()),
			Timestamp:  start,
			Endpoint:   "/v1/messages",
			Model:      req.Model,
			StatusCode: statusCode,
			DurationMs: time.Since(start).Milliseconds(),
			Error:      err.Error(),
		})
		http.Error(w, err.Error(), statusCode)
		return
	}
	defer respBody.Close()

	if statusCode >= 400 {
		RecordRequest(RequestLog{
			ID:         fmt.Sprintf("req_%d", time.Now().UnixNano()),
			Timestamp:  start,
			Endpoint:   "/v1/messages",
			Model:      req.Model,
			StatusCode: statusCode,
			DurationMs: time.Since(start).Milliseconds(),
			Error:      fmt.Sprintf("upstream returned %d", statusCode),
		})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		io.Copy(w, respBody)
		return
	}

	if isStream {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.WriteHeader(statusCode)

		flusher, _ := w.(http.Flusher)
		translateOpenAIToAnthropicSSE(respBody, w, flusher, req.Model)
		RecordRequest(RequestLog{
			ID:         fmt.Sprintf("req_%d", time.Now().UnixNano()),
			Timestamp:  start,
			Endpoint:   "/v1/messages",
			Model:      req.Model,
			StatusCode: statusCode,
			DurationMs: time.Since(start).Milliseconds(),
		})
		return
	}

	// Non-streaming response translation
	var openAIResp struct {
		ID      string `json:"id"`
		Choices []struct {
			Message struct {
				Role      string `json:"role"`
				Content   string `json:"content"`
				ToolCalls []struct {
					ID       string `json:"id"`
					Type     string `json:"type"`
					Function struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
	}

	respRaw, err := io.ReadAll(respBody)
	if err != nil {
		http.Error(w, "failed to read upstream response: "+err.Error(), http.StatusBadGateway)
		return
	}

	if err := json.Unmarshal(respRaw, &openAIResp); err != nil {
		// Pass-through as fallback
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		w.Write(respRaw)
		return
	}

	replyText := ""
	if len(openAIResp.Choices) > 0 {
		replyText = openAIResp.Choices[0].Message.Content
	}

	cleanedText, xmlTools := ExtractXmlToolCalls(replyText)
	var contentBlocks []interface{}
	stopReason := "end_turn"

	// Native structured tool calls from OpenAI response
	if len(openAIResp.Choices) > 0 && len(openAIResp.Choices[0].Message.ToolCalls) > 0 {
		stopReason = "tool_use"
		for _, tc := range openAIResp.Choices[0].Message.ToolCalls {
			var inputObj map[string]interface{}
			_ = json.Unmarshal([]byte(tc.Function.Arguments), &inputObj)
			if inputObj == nil {
				inputObj = map[string]interface{}{}
			}
			contentBlocks = append(contentBlocks, map[string]interface{}{
				"type":  "tool_use",
				"id":    tc.ID,
				"name":  tc.Function.Name,
				"input": inputObj,
			})
		}
	}

	if len(xmlTools) > 0 {
		stopReason = "tool_use"
		for _, tc := range xmlTools {
			contentBlocks = append(contentBlocks, map[string]interface{}{
				"type":  "tool_use",
				"id":    tc.ID,
				"name":  tc.Name,
				"input": tc.Input,
			})
		}
	}

	if cleanedText != "" || len(contentBlocks) == 0 {
		contentBlocks = append([]interface{}{
			map[string]interface{}{
				"type": "text",
				"text": cleanedText,
			},
		}, contentBlocks...)
	}

	anthropicResp := map[string]interface{}{
		"id":            "msg_" + fmt.Sprintf("%d", time.Now().UnixNano()),
		"type":          "message",
		"role":          "assistant",
		"model":         req.Model,
		"content":       contentBlocks,
		"stop_reason":   stopReason,
		"stop_sequence": nil,
		"usage": map[string]int{
			"input_tokens":  openAIResp.Usage.PromptTokens,
			"output_tokens": openAIResp.Usage.CompletionTokens,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(anthropicResp)
	inTok := openAIResp.Usage.PromptTokens
	outTok := openAIResp.Usage.CompletionTokens
	RecordRequest(RequestLog{
		ID:           fmt.Sprintf("req_%d", time.Now().UnixNano()),
		Timestamp:    start,
		Endpoint:     "/v1/messages",
		Model:        req.Model,
		StatusCode:   statusCode,
		DurationMs:   time.Since(start).Milliseconds(),
		InputTokens:  inTok,
		OutputTokens: outTok,
		Tokens:       inTok + outTok,
	})
}

type StreamToolCall struct {
	ID           string
	Name         string
	BlockIndex   int
	StartEmitted bool
	ArgsBuf      strings.Builder
	TotalArgs    strings.Builder
}

func translateOpenAIToAnthropicSSE(upstream io.Reader, w io.Writer, flusher http.Flusher, model string) {
	msgID := fmt.Sprintf("msg_%d", time.Now().UnixNano())

	// Send message_start
	fmt.Fprintf(w, "event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"id\":\"%s\",\"type\":\"message\",\"role\":\"assistant\",\"model\":\"%s\",\"content\":[],\"stop_reason\":null,\"stop_sequence\":null,\"usage\":{\"input_tokens\":1,\"output_tokens\":1}}}\n\n", msgID, model)
	if flusher != nil {
		flusher.Flush()
	}

	scanner := bufio.NewScanner(upstream)
	outputTokens := 0
	inTextBlock := false
	blockIndex := 0
	stopReason := "end_turn"
	seenTools := make(map[int]*StreamToolCall)
	xmlParser := NewXmlStreamParser()

	for scanner.Scan() {
		line := scanner.Bytes()
		if !bytes.HasPrefix(line, []byte("data: ")) {
			continue
		}
		data := bytes.TrimSpace(bytes.TrimPrefix(line, []byte("data: ")))
		if bytes.Equal(data, []byte("[DONE]")) {
			break
		}

		var chunk struct {
			Choices []struct {
				Delta struct {
					Content   string `json:"content"`
					ToolCalls []struct {
						Index    int    `json:"index"`
						ID       string `json:"id"`
						Type     string `json:"type"`
						Function struct {
							Name      string `json:"name"`
							Arguments string `json:"arguments"`
						} `json:"function"`
					} `json:"tool_calls"`
				} `json:"delta"`
				FinishReason string `json:"finish_reason"`
			} `json:"choices"`
		}

		if err := json.Unmarshal(data, &chunk); err == nil && len(chunk.Choices) > 0 {
			choice := chunk.Choices[0]

			// Handle text content via xmlParser
			if choice.Delta.Content != "" {
				cleanText, xmlCalls := xmlParser.Feed(choice.Delta.Content)

				if cleanText != "" {
					if !inTextBlock {
						fmt.Fprintf(w, "event: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":%d,\"content_block\":{\"type\":\"text\",\"text\":\"\"}}\n\n", blockIndex)
						inTextBlock = true
					}
					outputTokens++
					deltaJSON, _ := json.Marshal(map[string]interface{}{
						"type":  "content_block_delta",
						"index": blockIndex,
						"delta": map[string]string{
							"type": "text_delta",
							"text": cleanText,
						},
					})
					fmt.Fprintf(w, "event: content_block_delta\ndata: %s\n\n", string(deltaJSON))
					if flusher != nil {
						flusher.Flush()
					}
				}

				if len(xmlCalls) > 0 {
					for _, tc := range xmlCalls {
						stopReason = "tool_use"
						if inTextBlock {
							fmt.Fprintf(w, "event: content_block_stop\ndata: {\"type\":\"content_block_stop\",\"index\":%d}\n\n", blockIndex)
							inTextBlock = false
							blockIndex++
						}

						toolBlockIdx := blockIndex
						blockIndex++

						toolStartJSON, _ := json.Marshal(map[string]interface{}{
							"type":  "content_block_start",
							"index": toolBlockIdx,
							"content_block": map[string]interface{}{
								"type":  "tool_use",
								"id":    tc.ID,
								"name":  tc.Name,
								"input": map[string]interface{}{},
							},
						})
						fmt.Fprintf(w, "event: content_block_start\ndata: %s\n\n", string(toolStartJSON))

						argsRaw, _ := json.Marshal(tc.Input)
						argsJSON, _ := json.Marshal(map[string]interface{}{
							"type":  "content_block_delta",
							"index": toolBlockIdx,
							"delta": map[string]string{
								"type":         "input_json_delta",
								"partial_json": string(argsRaw),
							},
						})
						fmt.Fprintf(w, "event: content_block_delta\ndata: %s\n\n", string(argsJSON))
						fmt.Fprintf(w, "event: content_block_stop\ndata: {\"type\":\"content_block_stop\",\"index\":%d}\n\n", toolBlockIdx)
						if flusher != nil {
							flusher.Flush()
						}
					}
				}
			}

			// Handle native structured tool calls
			if len(choice.Delta.ToolCalls) > 0 {
				for _, tc := range choice.Delta.ToolCalls {
					stopReason = "tool_use"
					toolState, exists := seenTools[tc.Index]
					if !exists {
						toolState = &StreamToolCall{
							BlockIndex: -1,
						}
						seenTools[tc.Index] = toolState
					}

					if tc.ID != "" && toolState.ID == "" {
						toolState.ID = tc.ID
					}
					if tc.Function.Name != "" && toolState.Name == "" {
						toolState.Name = tc.Function.Name
					}

					if tc.Function.Arguments != "" && tc.Function.Arguments != "null" {
						if !toolState.StartEmitted {
							toolState.ArgsBuf.WriteString(tc.Function.Arguments)
						}
					}

					// Only emit start block once we have the real tool name
					if !toolState.StartEmitted && toolState.Name != "" {
						if inTextBlock {
							fmt.Fprintf(w, "event: content_block_stop\ndata: {\"type\":\"content_block_stop\",\"index\":%d}\n\n", blockIndex)
							inTextBlock = false
							blockIndex++
						}
						toolState.BlockIndex = blockIndex
						blockIndex++

						if toolState.ID == "" {
							toolState.ID = fmt.Sprintf("toolu_%d", toolState.BlockIndex)
						}

						toolStartJSON, _ := json.Marshal(map[string]interface{}{
							"type":  "content_block_start",
							"index": toolState.BlockIndex,
							"content_block": map[string]interface{}{
								"type":  "tool_use",
								"id":    toolState.ID,
								"name":  toolState.Name,
								"input": map[string]interface{}{},
							},
						})
						fmt.Fprintf(w, "event: content_block_start\ndata: %s\n\n", string(toolStartJSON))
						if flusher != nil {
							flusher.Flush()
						}
						toolState.StartEmitted = true

						// Flush any arguments buffered before tool name was known
						if toolState.ArgsBuf.Len() > 0 {
							bufferedArgs := toolState.ArgsBuf.String()
							toolState.TotalArgs.WriteString(bufferedArgs)
							toolState.ArgsBuf.Reset()
							argsJSON, _ := json.Marshal(map[string]interface{}{
								"type":  "content_block_delta",
								"index": toolState.BlockIndex,
								"delta": map[string]string{
									"type":         "input_json_delta",
									"partial_json": bufferedArgs,
								},
							})
							fmt.Fprintf(w, "event: content_block_delta\ndata: %s\n\n", string(argsJSON))
							if flusher != nil {
								flusher.Flush()
							}
						}
					} else if toolState.StartEmitted && tc.Function.Arguments != "" && tc.Function.Arguments != "null" {
						toolState.TotalArgs.WriteString(tc.Function.Arguments)
						argsJSON, _ := json.Marshal(map[string]interface{}{
							"type":  "content_block_delta",
							"index": toolState.BlockIndex,
							"delta": map[string]string{
								"type":         "input_json_delta",
								"partial_json": tc.Function.Arguments,
							},
						})
						fmt.Fprintf(w, "event: content_block_delta\ndata: %s\n\n", string(argsJSON))
						if flusher != nil {
							flusher.Flush()
						}
					}
				}
			}
		}
	}

	leftover := xmlParser.Flush()
	if leftover != "" {
		if !inTextBlock {
			fmt.Fprintf(w, "event: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":%d,\"content_block\":{\"type\":\"text\",\"text\":\"\"}}\n\n", blockIndex)
			inTextBlock = true
		}
		deltaJSON, _ := json.Marshal(map[string]interface{}{
			"type":  "content_block_delta",
			"index": blockIndex,
			"delta": map[string]string{
				"type": "text_delta",
				"text": leftover,
			},
		})
		fmt.Fprintf(w, "event: content_block_delta\ndata: %s\n\n", string(deltaJSON))
	}

	if inTextBlock {
		fmt.Fprintf(w, "event: content_block_stop\ndata: {\"type\":\"content_block_stop\",\"index\":%d}\n\n", blockIndex)
	}
	var activeTools []*StreamToolCall
	for _, toolState := range seenTools {
		if toolState.StartEmitted {
			activeTools = append(activeTools, toolState)
		}
	}
	sort.Slice(activeTools, func(i, j int) bool {
		return activeTools[i].BlockIndex < activeTools[j].BlockIndex
	})
	for _, toolState := range activeTools {
		// Guard against empty input that violates Claude Code schema for critical tools
		totalArgsStr := strings.TrimSpace(toolState.TotalArgs.String())
		if totalArgsStr == "" || totalArgsStr == "{}" {
			var fallbackJSON string
			switch toolState.Name {
			case "AskUserQuestion":
				fallbackJSON = `{"questions":[{"question":"How would you like to proceed?","options":["Proceed","Cancel"]}]}`
			case "Bash":
				fallbackJSON = `{"command":"pwd"}`
			}
			if fallbackJSON != "" {
				argsJSON, _ := json.Marshal(map[string]interface{}{
					"type":  "content_block_delta",
					"index": toolState.BlockIndex,
					"delta": map[string]string{
						"type":         "input_json_delta",
						"partial_json": fallbackJSON,
					},
				})
				fmt.Fprintf(w, "event: content_block_delta\ndata: %s\n\n", string(argsJSON))
				if flusher != nil {
					flusher.Flush()
				}
			}
		}
		fmt.Fprintf(w, "event: content_block_stop\ndata: {\"type\":\"content_block_stop\",\"index\":%d}\n\n", toolState.BlockIndex)
	}
	// Send message_delta
	fmt.Fprintf(w, "event: message_delta\ndata: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"%s\",\"stop_sequence\":null},\"usage\":{\"output_tokens\":%d}}\n\n", stopReason, outputTokens)
	// Send message_stop
	fmt.Fprintf(w, "event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n")
	if flusher != nil {
		flusher.Flush()
	}
}

// translateAnthropicToolsToOpenAI converts Anthropic tool specs (input_schema) into OpenAI function specs
func translateAnthropicToolsToOpenAI(tools []interface{}) []interface{} {
	if len(tools) == 0 {
		return nil
	}
	res := make([]interface{}, 0, len(tools))
	for _, t := range tools {
		m, ok := t.(map[string]interface{})
		if !ok {
			res = append(res, t)
			continue
		}
		if m["type"] == "function" && m["function"] != nil {
			res = append(res, t)
			continue
		}
		name, _ := m["name"].(string)
		desc, _ := m["description"].(string)
		schema := m["input_schema"]
		if schema == nil {
			schema = m["parameters"]
		}
		if schema == nil {
			schema = map[string]interface{}{"type": "object", "properties": map[string]interface{}{}}
		}
		res = append(res, map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name":        name,
				"description": desc,
				"parameters":  schema,
			},
		})
	}
	return res
}

// translateAnthropicMessagesToOpenAI parses tool_use and tool_result blocks into standard OpenAI ChatMessages
func translateAnthropicMessagesToOpenAI(messages []AnthropicMessage) []providers.ChatMessage {
	var result []providers.ChatMessage

	// Pre-index tool_use names by ID across all messages so tool_results always know their tool name
	toolNamesByID := make(map[string]string)
	for _, m := range messages {
		if blocks, ok := m.Content.([]interface{}); ok {
			for _, b := range blocks {
				if bMap, ok := b.(map[string]interface{}); ok {
					if bMap["type"] == "tool_use" {
						if id, ok := bMap["id"].(string); ok {
							if name, ok := bMap["name"].(string); ok && name != "" {
								toolNamesByID[id] = name
							}
						}
					}
				}
			}
		}
	}

	for _, m := range messages {
		blocks, ok := m.Content.([]interface{})
		if !ok {
			result = append(result, providers.ChatMessage{
				Role:    m.Role,
				Content: m.Content,
			})
			continue
		}

		var textParts []string
		var toolCalls []interface{}
		var toolResults []providers.ChatMessage
		var hasMultimodal bool

		for _, b := range blocks {
			bMap, ok := b.(map[string]interface{})
			if !ok {
				continue
			}
			bType, _ := bMap["type"].(string)
			switch bType {
			case "text":
				if txt, ok := bMap["text"].(string); ok {
					textParts = append(textParts, txt)
				}
			case "tool_use":
				id, _ := bMap["id"].(string)
				name, _ := bMap["name"].(string)
				input := bMap["input"]
				argsStr := "{}"
				if input != nil {
					if argBytes, err := json.Marshal(input); err == nil {
						argsStr = string(argBytes)
					}
				}
				toolCalls = append(toolCalls, map[string]interface{}{
					"id":   id,
					"type": "function",
					"function": map[string]interface{}{
						"name":      name,
						"arguments": argsStr,
					},
				})
			case "tool_result":
				id, _ := bMap["tool_use_id"].(string)
				var contentStr string
				switch cv := bMap["content"].(type) {
				case string:
					contentStr = cv
				default:
					cBytes, _ := json.Marshal(cv)
					contentStr = string(cBytes)
				}
				toolName := toolNamesByID[id]
				if toolName == "" {
					if n, ok := bMap["name"].(string); ok {
						toolName = n
					}
				}
				toolResults = append(toolResults, providers.ChatMessage{
					Role:       "tool",
					Name:       toolName,
					ToolCallID: id,
					Content:    contentStr,
				})
			case "image":
				hasMultimodal = true
			}
		}

		if hasMultimodal {
			result = append(result, providers.ChatMessage{
				Role:    m.Role,
				Content: m.Content,
			})
			continue
		}

		if len(toolResults) > 0 {
			// Anthropic / Claude requires tool results to immediately follow the tool_use turn.
			// Any accompanying user text must come after the tool results (#17256).
			result = append(result, toolResults...)
			if len(textParts) > 0 {
				result = append(result, providers.ChatMessage{
					Role:    m.Role,
					Content: strings.Join(textParts, "\n"),
				})
			}
			continue
		}

		msg := providers.ChatMessage{
			Role:    m.Role,
			Content: strings.Join(textParts, "\n"),
		}
		if len(toolCalls) > 0 {
			msg.ToolCalls = toolCalls
		}
		result = append(result, msg)
	}

	return result
}
