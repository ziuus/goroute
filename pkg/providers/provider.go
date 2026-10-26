package providers

import (
	"context"
	"io"
)

type ChatMessage struct {
	Role       string        `json:"role"`
	Content    interface{}   `json:"content"`
	Name       string        `json:"name,omitempty"`
	ToolCalls  []interface{} `json:"tool_calls,omitempty"`
	ToolCallID string        `json:"tool_call_id,omitempty"`
}

type ChatRequest struct {
	Model            string        `json:"model"`
	Messages         []ChatMessage `json:"messages"`
	Stream           bool          `json:"stream,omitempty"`
	Temperature      *float64      `json:"temperature,omitempty"`
	TopP             *float64      `json:"top_p,omitempty"`
	MaxTokens        *int          `json:"max_tokens,omitempty"`
	Tools            []interface{} `json:"tools,omitempty"`
	ToolChoice       interface{}   `json:"tool_choice,omitempty"`
	ReasoningEffort  string        `json:"reasoning_effort,omitempty"`
	Thinking         interface{}   `json:"thinking,omitempty"`
	ResponseFormat   interface{}   `json:"response_format,omitempty"`
	Stop             interface{}   `json:"stop,omitempty"`
	Seed             *int          `json:"seed,omitempty"`
	PresencePenalty  *float64      `json:"presence_penalty,omitempty"`
	FrequencyPenalty *float64      `json:"frequency_penalty,omitempty"`
	User             string        `json:"user,omitempty"`
}

type ModelInfo struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	OwnedBy string `json:"owned_by"`
}

type EmbeddingRequest struct {
	Model string      `json:"model"`
	Input interface{} `json:"input"` // string or []string
	User  string      `json:"user,omitempty"`
}

type EmbeddingItem struct {
	Object    string    `json:"object"`
	Index     int       `json:"index"`
	Embedding []float64 `json:"embedding"`
}

type EmbeddingResponse struct {
	Object string          `json:"object"`
	Data   []EmbeddingItem `json:"data"`
	Model  string          `json:"model"`
	Usage  map[string]int  `json:"usage"`
}

type Provider interface {
	ID() string
	Type() string
	ChatCompletion(ctx context.Context, req *ChatRequest) (body io.ReadCloser, isStream bool, statusCode int, err error)
	Embeddings(ctx context.Context, req *EmbeddingRequest) (*EmbeddingResponse, error)
	Models(ctx context.Context) ([]ModelInfo, error)
	IsHealthy() bool
}
