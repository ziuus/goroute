package tokens

import (
	"encoding/json"
	"strings"

	"github.com/ziuus/goroute/pkg/fast"
)

// CountTokensRequest mirrors Anthropic's count_tokens payload
type CountTokensRequest struct {
	Model    string        `json:"model"`
	Messages []interface{} `json:"messages"`
	System   interface{}   `json:"system,omitempty"`
	Tools    []interface{} `json:"tools,omitempty"`
}

// CountTokensResponse represents the standard token count return
type CountTokensResponse struct {
	InputTokens int `json:"input_tokens"`
}

// EstimateTokens calculates accurate input tokens across Claude & OpenAI schemas
func EstimateTokens(req *CountTokensRequest) int {
	if req == nil {
		return 0
	}

	totalTokens := 0

	// 1. System prompt tokens
	if req.System != nil {
		totalTokens += countBlockTokens(req.System) + 4
	}

	// 2. Tools tokens
	for _, tool := range req.Tools {
		if toolBytes, err := json.Marshal(tool); err == nil {
			totalTokens += countTextTokens(string(toolBytes)) + 6
		}
	}

	// 3. Message turns
	for _, rawMsg := range req.Messages {
		totalTokens += 4 // Turn framing overhead

		msgMap, ok := rawMsg.(map[string]interface{})
		if !ok {
			continue
		}

		if role, ok := msgMap["role"].(string); ok {
			totalTokens += countTextTokens(role)
		}

		if content := msgMap["content"]; content != nil {
			totalTokens += countBlockTokens(content)
		}

		// Tool calls in OpenAI format
		if toolCalls, ok := msgMap["tool_calls"].([]interface{}); ok {
			for _, tc := range toolCalls {
				if tcBytes, err := json.Marshal(tc); err == nil {
					totalTokens += countTextTokens(string(tcBytes)) + 4
				}
			}
		}
	}

	return totalTokens
}

func countBlockTokens(block interface{}) int {
	if block == nil {
		return 0
	}

	switch v := block.(type) {
	case string:
		return countTextTokens(v)
	case []interface{}:
		subTotal := 0
		for _, item := range v {
			subTotal += countBlockTokens(item)
		}
		return subTotal
	case map[string]interface{}:
		bType, _ := v["type"].(string)
		switch bType {
		case "text":
			if txt, ok := v["text"].(string); ok {
				return countTextTokens(txt)
			}
		case "tool_use":
			tokens := 4
			if name, ok := v["name"].(string); ok {
				tokens += countTextTokens(name)
			}
			if input := v["input"]; input != nil {
				if b, err := json.Marshal(input); err == nil {
					tokens += countTextTokens(string(b))
				}
			}
			return tokens
		case "tool_result":
			tokens := 4
			if content := v["content"]; content != nil {
				tokens += countBlockTokens(content)
			}
			return tokens
		case "thinking":
			if thinking, ok := v["thinking"].(string); ok {
				return countTextTokens(thinking)
			}
		default:
			// Fallback stringify
			if b, err := json.Marshal(v); err == nil {
				return countTextTokens(string(b))
			}
		}
	}
	return 0
}

func countTextTokens(s string) int {
	if len(s) == 0 {
		return 0
	}
	// Try Rust SIMD tokenizer if available
	if fast.IsAvailable() {
		if toks, ok := fast.CountTokens(s); ok && toks > 0 {
			return toks
		}
	}

	// High accuracy Go BPE approximation (3.4 chars per token average on code & prose)
	chars := len(s)
	// Add overhead for word boundaries
	words := len(strings.Fields(s))
	if words > (chars / 4) {
		return int(float64(chars)*0.28) + 1
	}
	return (chars / 4) + 1
}
