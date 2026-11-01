package proxy

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ziuus/goroute/pkg/providers"
)

// TruncatePreview truncates string cleanly to maxLen runes and collapses whitespace into single spaces.
func TruncatePreview(s string, maxLen int) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	fields := strings.Fields(s)
	joined := strings.Join(fields, " ")
	runes := []rune(joined)
	if len(runes) > maxLen {
		return string(runes[:maxLen]) + "..."
	}
	return joined
}

// ExtractPromptPreview extracts the most recent user prompt preview from chat messages.
func ExtractPromptPreview(messages []providers.ChatMessage) string {
	if len(messages) == 0 {
		return ""
	}
	// Walk backwards to find the last user message
	for i := len(messages) - 1; i >= 0; i-- {
		m := messages[i]
		if m.Role == "user" {
			switch c := m.Content.(type) {
			case string:
				return TruncatePreview(c, 200)
			case []interface{}:
				for _, part := range c {
					if pMap, ok := part.(map[string]interface{}); ok {
						if pMap["type"] == "text" {
							if txt, ok := pMap["text"].(string); ok {
								return TruncatePreview(txt, 200)
							}
						}
					}
				}
			}
		}
	}
	// Fallback to first message with text content
	for _, m := range messages {
		if str, ok := m.Content.(string); ok && str != "" {
			return TruncatePreview(str, 200)
		}
	}
	return ""
}

// ExtractResponsePreview extracts completion text or error preview from response JSON.
func ExtractResponsePreview(body []byte) string {
	if len(body) == 0 {
		return ""
	}
	var res struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
			Delta struct {
				Content string `json:"content"`
			} `json:"delta"`
		} `json:"choices"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &res); err == nil {
		if res.Error.Message != "" {
			return TruncatePreview(res.Error.Message, 200)
		}
		if len(res.Choices) > 0 {
			if res.Choices[0].Message.Content != "" {
				return TruncatePreview(res.Choices[0].Message.Content, 200)
			}
			if res.Choices[0].Delta.Content != "" {
				return TruncatePreview(res.Choices[0].Delta.Content, 200)
			}
		}
		if len(res.Content) > 0 {
			for _, part := range res.Content {
				if part.Type == "text" && part.Text != "" {
					return TruncatePreview(part.Text, 200)
				}
			}
		}
	}
	// Fallback to raw text preview
	return TruncatePreview(string(body), 200)
}

// ExtractSSEPreview parses text deltas out of an SSE chunk stream snippet.
func ExtractSSEPreview(raw string) string {
	lines := strings.Split(raw, "\n")
	var textBuilder strings.Builder
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if strings.HasPrefix(l, "data: ") {
			payload := strings.TrimPrefix(l, "data: ")
			if payload == "[DONE]" {
				continue
			}
			var chunk struct {
				Choices []struct {
					Delta struct {
						Content string `json:"content"`
					} `json:"delta"`
				} `json:"choices"`
				Delta struct {
					Text string `json:"text"`
				} `json:"delta"`
			}
			if err := json.Unmarshal([]byte(payload), &chunk); err == nil {
				if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
					textBuilder.WriteString(chunk.Choices[0].Delta.Content)
				}
				if chunk.Delta.Text != "" {
					textBuilder.WriteString(chunk.Delta.Text)
				}
				if textBuilder.Len() >= 200 {
					break
				}
			}
		}
	}
	if textBuilder.Len() > 0 {
		return TruncatePreview(textBuilder.String(), 200)
	}
	return ""
}

// ExtractErrorMessage formats upstream errors into concise, human-readable strings.
func ExtractErrorMessage(body []byte, statusCode int) string {
	if len(body) > 0 {
		var errObj struct {
			Error struct {
				Message string `json:"message"`
				Type    string `json:"type"`
				Code    interface{} `json:"code"`
			} `json:"error"`
			Detail  string `json:"detail"`
			Message string `json:"message"`
		}
		if err := json.Unmarshal(body, &errObj); err == nil {
			if errObj.Error.Message != "" {
				return TruncatePreview(errObj.Error.Message, 200)
			}
			if errObj.Message != "" {
				return TruncatePreview(errObj.Message, 200)
			}
			if errObj.Detail != "" {
				return TruncatePreview(errObj.Detail, 200)
			}
		}
		raw := strings.TrimSpace(string(body))
		if len(raw) > 0 && len(raw) < 200 {
			return TruncatePreview(raw, 200)
		}
	}
	return fmt.Sprintf("upstream returned HTTP %d", statusCode)
}
