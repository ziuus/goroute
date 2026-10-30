package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ziuus/goroute/pkg/providers"
	"github.com/ziuus/goroute/pkg/quota"
	"github.com/ziuus/goroute/pkg/router"
)

type mockAnthropicProvider struct{}

func (m *mockAnthropicProvider) ID() string   { return "mock" }
func (m *mockAnthropicProvider) Type() string { return "openai" }
func (m *mockAnthropicProvider) IsHealthy() bool { return true }
func (m *mockAnthropicProvider) Models(ctx context.Context) ([]providers.ModelInfo, error) {
	return []providers.ModelInfo{{ID: "claude-3-7-sonnet"}}, nil
}
func (m *mockAnthropicProvider) Embeddings(ctx context.Context, req *providers.EmbeddingRequest) (*providers.EmbeddingResponse, error) {
	return &providers.EmbeddingResponse{
		Object: "list",
		Model:  req.Model,
		Data:   []providers.EmbeddingItem{{Object: "embedding", Index: 0, Embedding: []float64{0.1, 0.2}}},
	}, nil
}
func (m *mockAnthropicProvider) ChatCompletion(ctx context.Context, req *providers.ChatRequest) (io.ReadCloser, bool, int, error) {
	if req.Stream {
		sseData := "data: {\"choices\":[{\"delta\":{\"content\":\"Hello world\"}}]}\n\ndata: [DONE]\n\n"
		return io.NopCloser(strings.NewReader(sseData)), true, http.StatusOK, nil
	}
	resp := map[string]interface{}{
		"id": "chatcmpl-test",
		"choices": []map[string]interface{}{
			{
				"message": map[string]string{
					"role":    "assistant",
					"content": "Hello from mock",
				},
				"finish_reason": "stop",
			},
		},
		"usage": map[string]int{
			"prompt_tokens":     10,
			"completion_tokens": 15,
		},
	}
	b, _ := json.Marshal(resp)
	return io.NopCloser(bytes.NewReader(b)), false, http.StatusOK, nil
}

func TestAnthropicMessagesHandler(t *testing.T) {
	r := router.NewRouter()
	r.RegisterProvider(&mockAnthropicProvider{})
	limiter := quota.NewLimiter()
	srv := NewServer(r, limiter, false)
	srv.SetConfigPaths("", "")

	// Test Non-streaming request
	body := `{"model":"claude-3-7-sonnet","messages":[{"role":"user","content":"Hi"}],"stream":false}`
	req := httptest.NewRequest("POST", "/v1/messages", strings.NewReader(body))
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
	}

	var anthropicResp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &anthropicResp); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}

	if anthropicResp["type"] != "message" {
		t.Errorf("expected type 'message', got %v", anthropicResp["type"])
	}

	// Test Streaming request
	streamBody := `{"model":"claude-3-7-sonnet","messages":[{"role":"user","content":"Hi"}],"stream":true}`
	reqStream := httptest.NewRequest("POST", "/v1/messages", strings.NewReader(streamBody))
	wStream := httptest.NewRecorder()

	srv.Handler().ServeHTTP(wStream, reqStream)

	if wStream.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", wStream.Code)
	}
	respStr := wStream.Body.String()
	if !strings.Contains(respStr, "event: message_start") {
		t.Errorf("expected message_start SSE event, got: %s", respStr)
	}
	if !strings.Contains(respStr, "content_block_delta") {
		t.Errorf("expected content_block_delta SSE event, got: %s", respStr)
	}
}

func TestAnthropicToolTranslation(t *testing.T) {
	// Test Tool Spec conversion
	anthropicTools := []interface{}{
		map[string]interface{}{
			"name":        "search_code",
			"description": "Searches codebase",
			"input_schema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{"type": "string"},
				},
				"required": []string{"query"},
			},
		},
	}

	openAITools := translateAnthropicToolsToOpenAI(anthropicTools)
	if len(openAITools) != 1 {
		t.Fatalf("expected 1 translated tool, got %d", len(openAITools))
	}
	tMap, ok := openAITools[0].(map[string]interface{})
	if !ok || tMap["type"] != "function" {
		t.Fatalf("expected tool type 'function', got: %v", openAITools[0])
	}
	fnMap, ok := tMap["function"].(map[string]interface{})
	if !ok || fnMap["name"] != "search_code" || fnMap["parameters"] == nil {
		t.Fatalf("invalid translated function spec: %v", fnMap)
	}

	// Test Messages conversion (tool_use and tool_result blocks)
	messages := []AnthropicMessage{
		{
			Role: "assistant",
			Content: []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": "Checking files",
				},
				map[string]interface{}{
					"type": "tool_use",
					"id":   "toolu_01",
					"name": "search_code",
					"input": map[string]interface{}{
						"query": "router",
					},
				},
			},
		},
		{
			Role: "user",
			Content: []interface{}{
				map[string]interface{}{
					"type":         "tool_result",
					"tool_use_id": "toolu_01",
					"content":      "router.go found",
				},
			},
		},
	}

	converted := translateAnthropicMessagesToOpenAI(messages)
	if len(converted) != 2 {
		t.Fatalf("expected 2 converted messages, got %d", len(converted))
	}
	// Assistant message has tool_calls
	if converted[0].Role != "assistant" || len(converted[0].ToolCalls) != 1 {
		t.Fatalf("expected assistant message with 1 tool call, got: %+v", converted[0])
	}
	// User message converted to role: tool
	if converted[1].Role != "tool" || converted[1].ToolCallID != "toolu_01" || converted[1].Content != "router.go found" {
		t.Fatalf("expected tool result message, got: %+v", converted[1])
	}
}

func TestSynthesizedSSE(t *testing.T) {
	jsonBody := []byte(`{
		"id": "chatcmpl-test-syn",
		"created": 1729000000,
		"model": "gpt-4o",
		"choices": [
			{
				"index": 0,
				"message": {
					"role": "assistant",
					"content": "This was non-streaming"
				},
				"finish_reason": "stop"
			}
		]
	}`)

	var buf bytes.Buffer
	writeSynthesizedSSE(&buf, nil, jsonBody, "gpt-4o")

	output := buf.String()
	if !strings.Contains(output, "chat.completion.chunk") {
		t.Errorf("expected chunk in SSE output, got: %s", output)
	}
	if !strings.Contains(output, "This was non-streaming") {
		t.Errorf("expected synthesized content in SSE output, got: %s", output)
	}
	if !strings.Contains(output, "data: [DONE]") {
		t.Errorf("expected [DONE] delimiter in SSE output, got: %s", output)
	}
}

func TestTranslateOpenAIToAnthropicSSE_StreamingToolCalls(t *testing.T) {
	// Simulate OpenAI SSE stream where chunk 1 provides ID + Name, and subsequent chunks provide argument fragments
	openAISSE := `data: {"choices": [{"index": 0, "delta": {"role": "assistant", "content": null, "tool_calls": [{"index": 0, "id": "call_test_123", "type": "function", "function": {"name": "search_files", "arguments": ""}}]}, "finish_reason": null}]}

data: {"choices": [{"index": 0, "delta": {"tool_calls": [{"index": 0, "function": {"arguments": "{\"query\":"}}]}, "finish_reason": null}]}

data: {"choices": [{"index": 0, "delta": {"tool_calls": [{"index": 0, "function": {"arguments": "\"main.go\"}"}}]}, "finish_reason": "tool_calls"}]}

data: [DONE]
`
	var buf bytes.Buffer
	translateOpenAIToAnthropicSSE(strings.NewReader(openAISSE), &buf, nil, "claude-test")

	out := buf.String()
	t.Logf("OUTPUT:\n%s", out)

	// 1. Should have content_block_start with name "search_files" and id "call_test_123"
	if !strings.Contains(out, `"name":"search_files"`) {
		t.Errorf("expected tool name 'search_files', got:\n%s", out)
	}
	if !strings.Contains(out, `"id":"call_test_123"`) {
		t.Errorf("expected tool id 'call_test_123', got:\n%s", out)
	}

	// 2. MUST NOT emit generic name: "tool"
	if strings.Contains(out, `"name":"tool"`) {
		t.Errorf("regression: emitted generic name 'tool':\n%s", out)
	}

	// 3. Count content_block_start occurrences - must be exactly 1
	startCount := strings.Count(out, `"type":"content_block_start"`)
	if startCount != 1 {
		t.Errorf("expected exactly 1 content_block_start, got %d:\n%s", startCount, out)
	}

	// 4. Input json deltas should be streamed
	if !strings.Contains(out, `main.go`) {
		t.Errorf("expected arguments chunk with main.go, got:\n%s", out)
	}

	// 5. Must have stop_reason: "tool_use"
	if !strings.Contains(out, `"stop_reason":"tool_use"`) {
		t.Errorf("expected stop_reason 'tool_use', got:\n%s", out)
	}
}
