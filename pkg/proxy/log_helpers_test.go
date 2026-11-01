package proxy

import (
	"testing"

	"github.com/ziuus/goroute/pkg/providers"
)

func TestExtractPromptPreview(t *testing.T) {
	msgs := []providers.ChatMessage{
		{Role: "system", Content: "You are an AI assistant."},
		{Role: "user", Content: "First prompt"},
		{Role: "assistant", Content: "First answer"},
		{Role: "user", Content: "How do I use goroute logs in terminal?"},
	}

	got := ExtractPromptPreview(msgs)
	expected := "How do I use goroute logs in terminal?"
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}

	// Truncation check
	longPrompt := "This is a very long prompt "
	for i := 0; i < 20; i++ {
		longPrompt += "with repeating words "
	}
	msgs[3].Content = longPrompt
	gotLong := ExtractPromptPreview(msgs)
	if len(gotLong) > 203 || len(gotLong) == 0 {
		t.Errorf("expected truncated preview under 203 chars, got length %d", len(gotLong))
	}
}

func TestExtractResponsePreview(t *testing.T) {
	// OpenAI Chat response JSON
	openAIJSON := []byte(`{
		"choices": [{
			"message": {
				"role": "assistant",
				"content": "GoRoute supports live streaming terminal logs via SSE."
			}
		}]
	}`)

	got := ExtractResponsePreview(openAIJSON)
	expected := "GoRoute supports live streaming terminal logs via SSE."
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}

	// Anthropic response JSON
	anthropicJSON := []byte(`{
		"content": [
			{"type": "text", "text": "Anthropic format response parsed successfully."}
		]
	}`)
	gotAnthropic := ExtractResponsePreview(anthropicJSON)
	if gotAnthropic != "Anthropic format response parsed successfully." {
		t.Errorf("expected Anthropic text parsed, got %q", gotAnthropic)
	}
}

func TestExtractSSEPreview(t *testing.T) {
	sseRaw := "data: {\"choices\":[{\"delta\":{\"content\":\"Hello \"}}]}\n\ndata: {\"choices\":[{\"delta\":{\"content\":\"world!\"}}]}\n\ndata: [DONE]\n\n"
	got := ExtractSSEPreview(sseRaw)
	if got != "Hello world!" {
		t.Errorf("expected %q, got %q", "Hello world!", got)
	}
}

func TestExtractErrorMessage(t *testing.T) {
	errJSON := []byte(`{"error": {"message": "Insufficient balance on account", "type": "insufficient_quota"}}`)
	got := ExtractErrorMessage(errJSON, 402)
	if got != "Insufficient balance on account" {
		t.Errorf("expected %q, got %q", "Insufficient balance on account", got)
	}

	emptyErr := ExtractErrorMessage(nil, 503)
	if emptyErr != "upstream returned HTTP 503" {
		t.Errorf("expected %q, got %q", "upstream returned HTTP 503", emptyErr)
	}
}
