package infra

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// SSEEvent represents a single Server-Sent Event received over a stream.
type SSEEvent struct {
	Event string
	Data  string
	ID    string
	Raw   string
}

// SSEResult aggregates all events and timing metrics from an SSE stream.
type SSEResult struct {
	TTFT          time.Duration // Time to first non-empty data token chunk
	TotalDuration time.Duration // Complete stream duration
	Events        []SSEEvent    // All received SSE events in chronological order
	RawLines      []string      // All raw stream lines
	FullText      string        // Concatenated text content extracted from chunks
	StatusCode    int           // HTTP response status code
}

// ClientOption configures an HTTP client instance.
type ClientOption func(*Client)

// Client is a high-level opaque HTTP/SSE client for interacting with the GoRoute daemon.
type Client struct {
	baseURL        string
	httpClient     *http.Client
	defaultHeaders map[string]string
	virtualKey     string
	mu             sync.RWMutex
}

// NewClient creates a new opaque-box HTTP client pointed at the daemon's base URL.
func NewClient(baseURL string, opts ...ClientOption) *Client {
	c := &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		defaultHeaders: make(map[string]string),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// WithVirtualKey configures an Authorization: Bearer <key> header on all requests.
func WithVirtualKey(key string) ClientOption {
	return func(c *Client) {
		c.virtualKey = key
	}
}

// WithHeader sets a persistent default header.
func WithHeader(key, val string) ClientOption {
	return func(c *Client) {
		c.defaultHeaders[key] = val
	}
}

// WithTimeout sets custom HTTP client timeout.
func WithTimeout(d time.Duration) ClientOption {
	return func(c *Client) {
		c.httpClient.Timeout = d
	}
}

// WithHTTPClient overrides the underlying net/http.Client.
func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = httpClient
	}
}

// SetVirtualKey updates the client's virtual key.
func (c *Client) SetVirtualKey(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.virtualKey = key
}

// SetHeader sets or updates a persistent default header.
func (c *Client) SetHeader(key, val string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.defaultHeaders[key] = val
}

// BaseURL returns the configured base URL.
func (c *Client) BaseURL() string {
	return c.baseURL
}

// buildRequest constructs an http.Request with base URL, headers, and authentication.
func (c *Client) buildRequest(ctx context.Context, method, path string, body io.Reader, headers map[string]string) (*http.Request, error) {
	url := c.baseURL
	if !strings.HasPrefix(path, "/") {
		url += "/" + path
	} else {
		url += path
	}

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.mu.RLock()
	// Apply default headers
	for k, v := range c.defaultHeaders {
		req.Header.Set(k, v)
	}
	// Apply virtual key
	if c.virtualKey != "" && req.Header.Get("Authorization") == "" {
		req.Header.Set("Authorization", "Bearer "+c.virtualKey)
	}
	c.mu.RUnlock()

	// Apply request-specific headers
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	return req, nil
}

// Do executes an HTTP request and reads the full response body into memory.
func (c *Client) Do(req *http.Request) (*http.Response, []byte, error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp, nil, fmt.Errorf("failed to read response body: %w", err)
	}
	return resp, bodyBytes, nil
}

// DoRaw executes an HTTP request and returns the active response stream without closing Body.
func (c *Client) DoRaw(req *http.Request) (*http.Response, error) {
	return c.httpClient.Do(req)
}

// Get performs an HTTP GET request against the daemon.
func (c *Client) Get(ctx context.Context, path string, headers map[string]string) (*http.Response, []byte, error) {
	req, err := c.buildRequest(ctx, http.MethodGet, path, nil, headers)
	if err != nil {
		return nil, nil, err
	}
	return c.Do(req)
}

// Post performs an HTTP POST request with an arbitrary reader body and content type.
func (c *Client) Post(ctx context.Context, path, contentType string, body io.Reader, headers map[string]string) (*http.Response, []byte, error) {
	req, err := c.buildRequest(ctx, http.MethodPost, path, body, headers)
	if err != nil {
		return nil, nil, err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	return c.Do(req)
}

// PostJSON performs an HTTP POST request with a JSON-encoded body.
func (c *Client) PostJSON(ctx context.Context, path string, body interface{}, headers map[string]string) (*http.Response, []byte, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonBytes, err := json.Marshal(body)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to marshal json body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBytes)
	}

	if headers == nil {
		headers = make(map[string]string)
	}
	if _, ok := headers["Content-Type"]; !ok {
		headers["Content-Type"] = "application/json"
	}

	req, err := c.buildRequest(ctx, http.MethodPost, path, bodyReader, headers)
	if err != nil {
		return nil, nil, err
	}
	return c.Do(req)
}

// PutJSON performs an HTTP PUT request with a JSON-encoded body.
func (c *Client) PutJSON(ctx context.Context, path string, body interface{}, headers map[string]string) (*http.Response, []byte, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonBytes, err := json.Marshal(body)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to marshal json body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBytes)
	}

	if headers == nil {
		headers = make(map[string]string)
	}
	if _, ok := headers["Content-Type"]; !ok {
		headers["Content-Type"] = "application/json"
	}

	req, err := c.buildRequest(ctx, http.MethodPut, path, bodyReader, headers)
	if err != nil {
		return nil, nil, err
	}
	return c.Do(req)
}

// Delete performs an HTTP DELETE request.
func (c *Client) Delete(ctx context.Context, path string, headers map[string]string) (*http.Response, []byte, error) {
	req, err := c.buildRequest(ctx, http.MethodDelete, path, nil, headers)
	if err != nil {
		return nil, nil, err
	}
	return c.Do(req)
}

// ChatCompletion posts an OpenAI-compatible /v1/chat/completions request.
func (c *Client) ChatCompletion(ctx context.Context, payload interface{}, headers map[string]string) (*http.Response, []byte, error) {
	return c.PostJSON(ctx, "/v1/chat/completions", payload, headers)
}

// Messages posts an Anthropic-compatible /v1/messages request.
func (c *Client) Messages(ctx context.Context, payload interface{}, headers map[string]string) (*http.Response, []byte, error) {
	return c.PostJSON(ctx, "/v1/messages", payload, headers)
}

// Health checks the /health endpoint and returns the parsed health status.
func (c *Client) Health(ctx context.Context) (int, map[string]interface{}, error) {
	resp, body, err := c.Get(ctx, "/health", nil)
	if err != nil {
		return 0, nil, err
	}
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return resp.StatusCode, nil, fmt.Errorf("failed to decode health JSON: %w (body: %s)", err, string(body))
	}
	return resp.StatusCode, data, nil
}

// Status checks the /api/status endpoint.
func (c *Client) Status(ctx context.Context) (int, map[string]interface{}, error) {
	resp, body, err := c.Get(ctx, "/api/status", nil)
	if err != nil {
		return 0, nil, err
	}
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return resp.StatusCode, nil, fmt.Errorf("failed to decode status JSON: %w", err)
	}
	return resp.StatusCode, data, nil
}

// Models fetches the active model list from /v1/models.
func (c *Client) Models(ctx context.Context) (int, map[string]interface{}, error) {
	resp, body, err := c.Get(ctx, "/v1/models", nil)
	if err != nil {
		return 0, nil, err
	}
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return resp.StatusCode, nil, fmt.Errorf("failed to decode models JSON: %w", err)
	}
	return resp.StatusCode, data, nil
}

// Reload triggers a daemon reload via POST /api/reload.
func (c *Client) Reload(ctx context.Context) (int, map[string]interface{}, error) {
	resp, body, err := c.PostJSON(ctx, "/api/reload", nil, nil)
	if err != nil {
		return 0, nil, err
	}
	var data map[string]interface{}
	_ = json.Unmarshal(body, &data)
	return resp.StatusCode, data, nil
}

// Shutdown initiates a daemon shutdown via POST /api/shutdown.
func (c *Client) Shutdown(ctx context.Context) (int, map[string]interface{}, error) {
	resp, body, err := c.PostJSON(ctx, "/api/shutdown", nil, nil)
	if err != nil {
		return 0, nil, err
	}
	var data map[string]interface{}
	_ = json.Unmarshal(body, &data)
	return resp.StatusCode, data, nil
}

// StreamSSE executes an SSE streaming request, reading line-by-line while recording Time-To-First-Token (TTFT).
func (c *Client) StreamSSE(
	ctx context.Context,
	path string,
	body interface{},
	headers map[string]string,
	onEvent func(event SSEEvent) error,
) (*SSEResult, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonBytes, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal json stream body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBytes)
	}

	reqHeaders := make(map[string]string)
	for k, v := range headers {
		reqHeaders[k] = v
	}
	if _, ok := reqHeaders["Accept"]; !ok {
		reqHeaders["Accept"] = "text/event-stream"
	}
	if _, ok := reqHeaders["Content-Type"]; !ok && body != nil {
		reqHeaders["Content-Type"] = "application/json"
	}

	req, err := c.buildRequest(ctx, http.MethodPost, path, bodyReader, reqHeaders)
	if err != nil {
		return nil, err
	}

	// Disable client-level timeout for streaming; respect ctx timeout
	streamClient := *c.httpClient
	streamClient.Timeout = 0

	startTime := time.Now()
	resp, err := streamClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("stream connection failed: %w", err)
	}
	defer resp.Body.Close()

	result := &SSEResult{
		StatusCode: resp.StatusCode,
		Events:     make([]SSEEvent, 0),
		RawLines:   make([]string, 0),
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return result, fmt.Errorf("stream returned non-2xx status code %d: %s", resp.StatusCode, string(bodyBytes))
	}

	scanner := bufio.NewScanner(resp.Body)
	// Support up to 1MB per SSE line (e.g. large tool calls or thinking blocks)
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	var currentEvent SSEEvent
	var fullTextBuilder strings.Builder

	for scanner.Scan() {
		line := scanner.Text()
		result.RawLines = append(result.RawLines, line)

		trimmed := strings.TrimRight(line, "\r")
		if trimmed == "" {
			// Blank line dispatches the current event
			if currentEvent.Data != "" || currentEvent.Event != "" {
				result.Events = append(result.Events, currentEvent)
				if onEvent != nil {
					if err := onEvent(currentEvent); err != nil {
						return result, fmt.Errorf("onEvent callback error: %w", err)
					}
				}
				currentEvent = SSEEvent{}
			}
			continue
		}

		// Comment line
		if strings.HasPrefix(trimmed, ":") {
			continue
		}

		if strings.HasPrefix(trimmed, "event:") {
			currentEvent.Event = strings.TrimSpace(strings.TrimPrefix(trimmed, "event:"))
		} else if strings.HasPrefix(trimmed, "data:") {
			dataContent := strings.TrimPrefix(trimmed, "data:")
			if strings.HasPrefix(dataContent, " ") {
				dataContent = dataContent[1:]
			}

			// Record Time To First Token on first data payload
			if result.TTFT == 0 && dataContent != "" && dataContent != "[DONE]" {
				result.TTFT = time.Since(startTime)
			}

			if currentEvent.Data == "" {
				currentEvent.Data = dataContent
			} else {
				currentEvent.Data += "\n" + dataContent
			}

			// Attempt extracting text from OpenAI or Anthropic chunks
			if dataContent != "[DONE]" {
				extracted := extractTextFromChunk(dataContent)
				if extracted != "" {
					fullTextBuilder.WriteString(extracted)
				}
			}
		} else if strings.HasPrefix(trimmed, "id:") {
			currentEvent.ID = strings.TrimSpace(strings.TrimPrefix(trimmed, "id:"))
		}
	}

	// Dispatch trailing event if not followed by a blank line
	if currentEvent.Data != "" || currentEvent.Event != "" {
		result.Events = append(result.Events, currentEvent)
		if onEvent != nil {
			_ = onEvent(currentEvent)
		}
	}

	result.TotalDuration = time.Since(startTime)
	result.FullText = fullTextBuilder.String()

	if err := scanner.Err(); err != nil && err != io.EOF && ctx.Err() == nil {
		return result, fmt.Errorf("error reading SSE stream: %w", err)
	}

	return result, nil
}

// extractTextFromChunk parses chunk JSON to extract incremental text tokens.
func extractTextFromChunk(data string) string {
	var chunk struct {
		Choices []struct {
			Delta struct {
				Content string `json:"content"`
			} `json:"delta"`
		} `json:"choices"`
		// Anthropic streaming format
		Delta struct {
			Text string `json:"text"`
		} `json:"delta"`
		ContentBlock struct {
			Text string `json:"text"`
		} `json:"content_block"`
	}

	if err := json.Unmarshal([]byte(data), &chunk); err != nil {
		return ""
	}

	if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
		return chunk.Choices[0].Delta.Content
	}
	if chunk.Delta.Text != "" {
		return chunk.Delta.Text
	}
	if chunk.ContentBlock.Text != "" {
		return chunk.ContentBlock.Text
	}
	return ""
}
