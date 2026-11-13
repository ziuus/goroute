package infra

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// RecordedRequest stores details of an HTTP request received by a mock upstream.
type RecordedRequest struct {
	Method    string
	Path      string
	URL       string
	Headers   http.Header
	Body      []byte
	Timestamp time.Time
}

// RequestRecorder thread-safely records all inbound HTTP requests for test assertions.
type RequestRecorder struct {
	mu       sync.RWMutex
	requests []RecordedRequest
}

// NewRequestRecorder creates an empty request recorder.
func NewRequestRecorder() *RequestRecorder {
	return &RequestRecorder{
		requests: make([]RecordedRequest, 0),
	}
}

// Record captures an inbound HTTP request and stores a clone of headers and body.
func (r *RequestRecorder) Record(req *http.Request, body []byte) {
	r.mu.Lock()
	defer r.mu.Unlock()

	headerCopy := make(http.Header)
	for k, v := range req.Header {
		headerCopy[k] = append([]string(nil), v...)
	}

	bodyCopy := make([]byte, len(body))
	copy(bodyCopy, body)

	r.requests = append(r.requests, RecordedRequest{
		Method:    req.Method,
		Path:      req.URL.Path,
		URL:       req.URL.String(),
		Headers:   headerCopy,
		Body:      bodyCopy,
		Timestamp: time.Now(),
	})
}

// GetRequests returns a snapshot of all recorded requests.
func (r *RequestRecorder) GetRequests() []RecordedRequest {
	r.mu.RLock()
	defer r.mu.RUnlock()
	copied := make([]RecordedRequest, len(r.requests))
	copy(copied, r.requests)
	return copied
}

// GetRequestsByPath returns recorded requests matching the given path prefix.
func (r *RequestRecorder) GetRequestsByPath(pathPrefix string) []RecordedRequest {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var matched []RecordedRequest
	for _, req := range r.requests {
		if strings.HasPrefix(req.Path, pathPrefix) {
			matched = append(matched, req)
		}
	}
	return matched
}

// LastRequest returns the most recently recorded request or nil if empty.
func (r *RequestRecorder) LastRequest() *RecordedRequest {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if len(r.requests) == 0 {
		return nil
	}
	req := r.requests[len(r.requests)-1]
	return &req
}

// Count returns the number of recorded requests.
func (r *RequestRecorder) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.requests)
}

// Clear resets recorded requests.
func (r *RequestRecorder) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.requests = r.requests[:0]
}

// BaseMockServer provides common mocking utilities (recording, delay, fault injection).
type BaseMockServer struct {
	Server   *httptest.Server
	Recorder *RequestRecorder

	mu             sync.RWMutex
	delay          time.Duration
	injectedStatus int
	injectedBody   string
	injectedCount  int // if negative, permanent; if > 0, decremented per request
	customHandlers map[string]http.HandlerFunc
}

func newBaseMockServer() *BaseMockServer {
	return &BaseMockServer{
		Recorder:       NewRequestRecorder(),
		customHandlers: make(map[string]http.HandlerFunc),
	}
}

// SetDelay configures an artificial latency delay added before sending responses.
func (b *BaseMockServer) SetDelay(d time.Duration) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.delay = d
}

// SetErrorInjection configures the server to return an error status code.
// If count > 0, returns the error for `count` requests then clears. If count < 0, permanent.
func (b *BaseMockServer) SetErrorInjection(statusCode int, body string, count int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.injectedStatus = statusCode
	b.injectedBody = body
	b.injectedCount = count
}

// ResetInjection clears any active fault injection and artificial delay.
func (b *BaseMockServer) ResetInjection() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.injectedStatus = 0
	b.injectedBody = ""
	b.injectedCount = 0
	b.delay = 0
}

// SetCustomHandler registers a path-specific override.
func (b *BaseMockServer) SetCustomHandler(path string, h http.HandlerFunc) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.customHandlers[path] = h
}

// checkInjection returns true if an error was injected and served.
func (b *BaseMockServer) checkInjection(w http.ResponseWriter, r *http.Request) bool {
	b.mu.Lock()
	delay := b.delay
	status := b.injectedStatus
	body := b.injectedBody

	if b.injectedCount > 0 {
		b.injectedCount--
		if b.injectedCount == 0 {
			b.injectedStatus = 0
			b.injectedBody = ""
		}
	}
	b.mu.Unlock()

	if delay > 0 {
		time.Sleep(delay)
	}

	if status > 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if body != "" {
			_, _ = w.Write([]byte(body))
		} else {
			_, _ = fmt.Fprintf(w, `{"error":{"message":"Injected mock error","type":"server_error","code":%d}}`, status)
		}
		return true
	}
	return false
}

// URL returns the mock server's listener URL.
func (b *BaseMockServer) URL() string {
	if b.Server == nil {
		return ""
	}
	return b.Server.URL
}

// Close shuts down the mock server.
func (b *BaseMockServer) Close() {
	if b.Server != nil {
		b.Server.Close()
	}
}

// ============================================================================
// OpenAI Mock Upstream
// ============================================================================

// OpenAIMock simulates an OpenAI API upstream server.
type OpenAIMock struct {
	BaseMockServer
	customResponse string
	completionSeq  int64
}

// NewOpenAIMock starts a new OpenAI mock upstream server.
func NewOpenAIMock() *OpenAIMock {
	mock := &OpenAIMock{
		BaseMockServer: *newBaseMockServer(),
		customResponse: "Hello! This is a mock response from OpenAI upstream.",
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/chat/completions", mock.handleChatCompletions)
	mux.HandleFunc("/chat/completions", mock.handleChatCompletions)
	mux.HandleFunc("/v1/models", mock.handleModels)
	mux.HandleFunc("/v1/embeddings", mock.handleEmbeddings)

	mock.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		r.Body = io.NopCloser(bytes.NewReader(body))
		mock.Recorder.Record(r, body)

		// Check custom handlers
		mock.mu.RLock()
		custom, hasCustom := mock.customHandlers[r.URL.Path]
		mock.mu.RUnlock()
		if hasCustom {
			custom(w, r)
			return
		}

		if mock.checkInjection(w, r) {
			return
		}

		mux.ServeHTTP(w, r)
	}))

	return mock
}

// SetCustomResponse configures text returned by chat completion endpoints.
func (o *OpenAIMock) SetCustomResponse(text string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.customResponse = text
}

func (o *OpenAIMock) handleChatCompletions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, _ := io.ReadAll(r.Body)
	var req struct {
		Model    string `json:"model"`
		Stream   bool   `json:"stream"`
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
	}
	_ = json.Unmarshal(body, &req)

	if req.Model == "" {
		req.Model = "gpt-4o"
	}

	o.mu.RLock()
	respText := o.customResponse
	o.mu.RUnlock()

	seq := atomic.AddInt64(&o.completionSeq, 1)
	completionID := fmt.Sprintf("chatcmpl-mock-%d", seq)

	if req.Stream {
		// Server-Sent Events streaming chunks
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.WriteHeader(http.StatusOK)

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
			return
		}

		// Chunk 1: Role delta
		c1, _ := json.Marshal(map[string]interface{}{
			"id":      completionID,
			"object":  "chat.completion.chunk",
			"created": time.Now().Unix(),
			"model":   req.Model,
			"choices": []map[string]interface{}{
				{
					"index": 0,
					"delta": map[string]string{"role": "assistant"},
				},
			},
		})
		_, _ = fmt.Fprintf(w, "data: %s\n\n", c1)
		flusher.Flush()
		time.Sleep(10 * time.Millisecond)

		// Chunk 2: Content payload
		c2, _ := json.Marshal(map[string]interface{}{
			"id":      completionID,
			"object":  "chat.completion.chunk",
			"created": time.Now().Unix(),
			"model":   req.Model,
			"choices": []map[string]interface{}{
				{
					"index": 0,
					"delta": map[string]string{"content": respText},
				},
			},
		})
		_, _ = fmt.Fprintf(w, "data: %s\n\n", c2)
		flusher.Flush()
		time.Sleep(10 * time.Millisecond)

		// Chunk 3: Finish reason and usage
		c3, _ := json.Marshal(map[string]interface{}{
			"id":      completionID,
			"object":  "chat.completion.chunk",
			"created": time.Now().Unix(),
			"model":   req.Model,
			"choices": []map[string]interface{}{
				{
					"index":         0,
					"delta":         map[string]string{},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]int{
				"prompt_tokens":     15,
				"completion_tokens": 8,
				"total_tokens":      23,
			},
		})
		_, _ = fmt.Fprintf(w, "data: %s\n\n", c3)
		_, _ = fmt.Fprintf(w, "data: [DONE]\n\n")
		flusher.Flush()
		return
	}

	// Non-streaming JSON response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"id":      completionID,
		"object":  "chat.completion",
		"created": time.Now().Unix(),
		"model":   req.Model,
		"choices": []map[string]interface{}{
			{
				"index": 0,
				"message": map[string]string{
					"role":    "assistant",
					"content": respText,
				},
				"finish_reason": "stop",
			},
		},
		"usage": map[string]int{
			"prompt_tokens":     15,
			"completion_tokens": 8,
			"total_tokens":      23,
		},
	})
}

func (o *OpenAIMock) handleModels(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"object": "list",
		"data": []map[string]interface{}{
			{"id": "gpt-4o", "object": "model", "owned_by": "openai"},
			{"id": "gpt-4o-mini", "object": "model", "owned_by": "openai"},
			{"id": "o1-preview", "object": "model", "owned_by": "openai"},
			{"id": "text-embedding-3-small", "object": "model", "owned_by": "openai"},
		},
	})
}

func (o *OpenAIMock) handleEmbeddings(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"object": "list",
		"data": []map[string]interface{}{
			{
				"object":    "embedding",
				"index":     0,
				"embedding": []float64{0.0123, -0.0456, 0.0789, 0.1011},
			},
		},
		"model": "text-embedding-3-small",
		"usage": map[string]int{
			"prompt_tokens": 5,
			"total_tokens":  5,
		},
	})
}

// ============================================================================
// Anthropic Mock Upstream
// ============================================================================

// AnthropicMock simulates an Anthropic API upstream server.
type AnthropicMock struct {
	BaseMockServer
	customResponse string
	RequireAuth    bool
}

// NewAnthropicMock starts a new Anthropic mock upstream server.
func NewAnthropicMock() *AnthropicMock {
	mock := &AnthropicMock{
		BaseMockServer: *newBaseMockServer(),
		customResponse: "Hello! This is a mock response from Anthropic upstream.",
		RequireAuth:    true,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/messages", mock.handleMessages)
	mux.HandleFunc("/messages", mock.handleMessages)

	mock.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		r.Body = io.NopCloser(bytes.NewReader(body))
		mock.Recorder.Record(r, body)

		mock.mu.RLock()
		custom, hasCustom := mock.customHandlers[r.URL.Path]
		mock.mu.RUnlock()
		if hasCustom {
			custom(w, r)
			return
		}

		if mock.checkInjection(w, r) {
			return
		}

		mux.ServeHTTP(w, r)
	}))

	return mock
}

// SetCustomResponse configures text returned by /v1/messages.
func (a *AnthropicMock) SetCustomResponse(text string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.customResponse = text
}

func (a *AnthropicMock) handleMessages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Verify required headers per Anthropic protocol specification
	apiKey := r.Header.Get("x-api-key")
	if a.RequireAuth && apiKey == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"type": "error",
			"error": map[string]string{
				"type":    "authentication_error",
				"message": "x-api-key header is required",
			},
		})
		return
	}

	body, _ := io.ReadAll(r.Body)
	var req struct {
		Model     string `json:"model"`
		Stream    bool   `json:"stream"`
		MaxTokens int    `json:"max_tokens"`
	}
	_ = json.Unmarshal(body, &req)

	if req.Model == "" {
		req.Model = "claude-3-5-sonnet-20241022"
	}

	a.mu.RLock()
	respText := a.customResponse
	a.mu.RUnlock()

	msgID := fmt.Sprintf("msg_mock_%d", time.Now().UnixNano())

	if req.Stream {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.WriteHeader(http.StatusOK)

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
			return
		}

		// Anthropic SSE Protocol events sequence
		// 1. message_start
		mStart, _ := json.Marshal(map[string]interface{}{
			"type": "message_start",
			"message": map[string]interface{}{
				"id":           msgID,
				"type":         "message",
				"role":         "assistant",
				"model":        req.Model,
				"content":      []interface{}{},
				"stop_reason":  nil,
				"stop_sequence": nil,
				"usage": map[string]int{
					"input_tokens":  20,
					"output_tokens": 1,
				},
			},
		})
		_, _ = fmt.Fprintf(w, "event: message_start\ndata: %s\n\n", mStart)
		flusher.Flush()
		time.Sleep(10 * time.Millisecond)

		// 2. content_block_start
		cbStart, _ := json.Marshal(map[string]interface{}{
			"type":  "content_block_start",
			"index": 0,
			"content_block": map[string]string{
				"type": "text",
				"text": "",
			},
		})
		_, _ = fmt.Fprintf(w, "event: content_block_start\ndata: %s\n\n", cbStart)
		flusher.Flush()
		time.Sleep(10 * time.Millisecond)

		// 3. content_block_delta
		cbDelta, _ := json.Marshal(map[string]interface{}{
			"type":  "content_block_delta",
			"index": 0,
			"delta": map[string]string{
				"type": "text_delta",
				"text": respText,
			},
		})
		_, _ = fmt.Fprintf(w, "event: content_block_delta\ndata: %s\n\n", cbDelta)
		flusher.Flush()
		time.Sleep(10 * time.Millisecond)

		// 4. content_block_stop
		cbStop, _ := json.Marshal(map[string]interface{}{
			"type":  "content_block_stop",
			"index": 0,
		})
		_, _ = fmt.Fprintf(w, "event: content_block_stop\ndata: %s\n\n", cbStop)
		flusher.Flush()

		// 5. message_delta
		mDelta, _ := json.Marshal(map[string]interface{}{
			"type": "message_delta",
			"delta": map[string]interface{}{
				"stop_reason":   "end_turn",
				"stop_sequence": nil,
			},
			"usage": map[string]int{
				"output_tokens": 12,
			},
		})
		_, _ = fmt.Fprintf(w, "event: message_delta\ndata: %s\n\n", mDelta)
		flusher.Flush()

		// 6. message_stop
		mStop, _ := json.Marshal(map[string]interface{}{
			"type": "message_stop",
		})
		_, _ = fmt.Fprintf(w, "event: message_stop\ndata: %s\n\n", mStop)
		flusher.Flush()
		return
	}

	// Non-streaming JSON response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"id":      msgID,
		"type":    "message",
		"role":    "assistant",
		"model":   req.Model,
		"content": []map[string]string{{"type": "text", "text": respText}},
		"stop_reason": "end_turn",
		"usage": map[string]int{
			"input_tokens":  20,
			"output_tokens": 12,
		},
	})
}

// ============================================================================
// Codex Mock Upstream
// ============================================================================

// CodexMock simulates OpenAI's Codex backend OAuth endpoint.
type CodexMock struct {
	BaseMockServer
	customResponse string
}

// NewCodexMock starts a new Codex mock upstream server.
func NewCodexMock() *CodexMock {
	mock := &CodexMock{
		BaseMockServer: *newBaseMockServer(),
		customResponse: "Codex completion generated successfully.",
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/backend-api/codex/responses", mock.handleCodexResponses)
	mux.HandleFunc("/oauth/token", mock.handleOAuthToken)

	mock.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		r.Body = io.NopCloser(bytes.NewReader(body))
		mock.Recorder.Record(r, body)

		mock.mu.RLock()
		custom, hasCustom := mock.customHandlers[r.URL.Path]
		mock.mu.RUnlock()
		if hasCustom {
			custom(w, r)
			return
		}

		if mock.checkInjection(w, r) {
			return
		}

		mux.ServeHTTP(w, r)
	}))

	return mock
}

func (c *CodexMock) handleCodexResponses(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"id":      fmt.Sprintf("codex_resp_%d", time.Now().UnixNano()),
		"object":  "response",
		"created": time.Now().Unix(),
		"output":  c.customResponse,
		"usage": map[string]int{
			"prompt_tokens":     30,
			"completion_tokens": 15,
			"total_tokens":      45,
		},
	})
}

func (c *CodexMock) handleOAuthToken(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"access_token":  "mock-codex-access-token",
		"token_type":    "Bearer",
		"expires_in":    3600,
		"refresh_token": "mock-codex-refresh-token",
	})
}

// ============================================================================
// Cursor Mock Upstream (ConnectRPC Protobuf Framing)
// ============================================================================

// CursorMock simulates the Cursor ConnectRPC Protobuf service.
type CursorMock struct {
	BaseMockServer
}

// EncodeConnectRPCFrame encodes payload into ConnectRPC 5-byte framing envelope:
// Byte 0: Flags (0x00 for data, 0x02 for end-of-stream trailer)
// Bytes 1-4: Big-endian 32-bit payload length
func EncodeConnectRPCFrame(payload []byte, flags byte) []byte {
	envelope := make([]byte, 5+len(payload))
	envelope[0] = flags
	binary.BigEndian.PutUint32(envelope[1:5], uint32(len(payload)))
	copy(envelope[5:], payload)
	return envelope
}

// DecodeConnectRPCFrames decodes framed bytes into individual payload chunks.
func DecodeConnectRPCFrames(data []byte) ([][]byte, error) {
	var frames [][]byte
	offset := 0
	for offset < len(data) {
		if len(data)-offset < 5 {
			return nil, fmt.Errorf("truncated ConnectRPC header at offset %d", offset)
		}
		// flags := data[offset]
		length := binary.BigEndian.Uint32(data[offset+1 : offset+5])
		frameStart := offset + 5
		frameEnd := frameStart + int(length)
		if frameEnd > len(data) {
			return nil, fmt.Errorf("truncated ConnectRPC payload at offset %d (need %d, have %d)", offset, length, len(data)-frameStart)
		}
		framePayload := make([]byte, length)
		copy(framePayload, data[frameStart:frameEnd])
		frames = append(frames, framePayload)
		offset = frameEnd
	}
	return frames, nil
}

// NewCursorMock starts a new Cursor ConnectRPC mock upstream server.
func NewCursorMock() *CursorMock {
	mock := &CursorMock{
		BaseMockServer: *newBaseMockServer(),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/agent.v1.AgentService/Run", mock.handleAgentRun)
	mux.HandleFunc("/v1/chat/completions", mock.handleCursorChat)
	mux.HandleFunc("/aiserver.v1.DashboardService/GetCurrentPeriodUsage", mock.handlePeriodUsage)

	mock.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		r.Body = io.NopCloser(bytes.NewReader(body))
		mock.Recorder.Record(r, body)

		mock.mu.RLock()
		custom, hasCustom := mock.customHandlers[r.URL.Path]
		mock.mu.RUnlock()
		if hasCustom {
			custom(w, r)
			return
		}

		if mock.checkInjection(w, r) {
			return
		}

		mux.ServeHTTP(w, r)
	}))

	return mock
}

func (c *CursorMock) handleAgentRun(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/connect+proto")
	w.Header().Set("connect-protocol-version", "1")
	w.WriteHeader(http.StatusOK)

	// Simulated protobuf payload framed with 5-byte ConnectRPC header
	dataPayload := []byte(`{"text":"Cursor ConnectRPC simulation successful","finish":true}`)
	frame := EncodeConnectRPCFrame(dataPayload, 0x00)

	// Trailer frame flag 0x02
	trailerPayload := []byte(`{}`)
	trailer := EncodeConnectRPCFrame(trailerPayload, 0x02)

	_, _ = w.Write(frame)
	_, _ = w.Write(trailer)
}

func (c *CursorMock) handleCursorChat(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"id":      fmt.Sprintf("cursor_chat_%d", time.Now().UnixNano()),
		"object":  "chat.completion",
		"created": time.Now().Unix(),
		"model":   "cursor-fast",
		"choices": []map[string]interface{}{
			{
				"index": 0,
				"message": map[string]string{
					"role":    "assistant",
					"content": "Cursor completion OK",
				},
				"finish_reason": "stop",
			},
		},
		"usage": map[string]int{
			"prompt_tokens":     10,
			"completion_tokens": 5,
			"total_tokens":      15,
		},
	})
}

func (c *CursorMock) handlePeriodUsage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"totalUsage": 120,
		"limit":      1000,
	})
}
