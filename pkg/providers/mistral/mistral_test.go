package mistral

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ziuus/goroute/pkg/providers"
)

func TestProviderIdentity(t *testing.T) {
	var _ providers.Provider = (*Provider)(nil)

	provider := NewProvider("mistral-test", "", "test-key", nil)
	if provider.ID() != "mistral-test" {
		t.Fatalf("ID() = %q, want %q", provider.ID(), "mistral-test")
	}
	if provider.Type() != "mistral" {
		t.Fatalf("Type() = %q, want %q", provider.Type(), "mistral")
	}
	if DefaultBaseURL != "https://api.mistral.ai/v1" {
		t.Fatalf("DefaultBaseURL = %q, want %q", DefaultBaseURL, "https://api.mistral.ai/v1")
	}
}

func TestChatCompletionUsesOpenAICompatibleMistralAPI(t *testing.T) {
	models := []string{"codestral-latest", "mistral-large-latest", "pixtral-large-latest"}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}
		if r.URL.Path != "/chat/completions" {
			t.Errorf("path = %q, want /chat/completions", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Errorf("Authorization = %q, want Bearer test-key", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", got)
		}

		var got providers.ChatRequest
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Errorf("decode chat request: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if !containsModel(models, got.Model) {
			t.Errorf("unexpected model %q", got.Model)
		}
		if len(got.Messages) != 1 || got.Messages[0].Role != "user" || got.Messages[0].Content != "hello" {
			t.Errorf("messages = %#v, want one user message containing hello", got.Messages)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"choices":[{"message":{"role":"assistant","content":"hi"}}]}`)
	}))
	defer server.Close()

	provider := NewProvider("mistral-test", server.URL, "test-key", nil)
	for _, model := range models {
		t.Run(model, func(t *testing.T) {
			body, isStream, statusCode, err := provider.ChatCompletion(context.Background(), &providers.ChatRequest{
				Model:    model,
				Messages: []providers.ChatMessage{{Role: "user", Content: "hello"}},
			})
			if err != nil {
				t.Fatalf("ChatCompletion() error = %v", err)
			}
			defer body.Close()
			if statusCode != http.StatusOK {
				t.Fatalf("status = %d, want %d", statusCode, http.StatusOK)
			}
			if isStream {
				t.Fatal("isStream = true, want false")
			}
			response, err := io.ReadAll(body)
			if err != nil {
				t.Fatalf("read response: %v", err)
			}
			if !strings.Contains(string(response), `"content":"hi"`) {
				t.Fatalf("response = %s, want assistant content hi", response)
			}
		})
	}
}

func TestChatCompletionDetectsStreamingResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: {\"choices\":[]}\n\n")
	}))
	defer server.Close()

	provider := NewProvider("mistral-test", server.URL, "test-key", nil)
	body, isStream, statusCode, err := provider.ChatCompletion(context.Background(), &providers.ChatRequest{
		Model:  "codestral-latest",
		Stream: true,
	})
	if err != nil {
		t.Fatalf("ChatCompletion() error = %v", err)
	}
	defer body.Close()
	if statusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", statusCode, http.StatusOK)
	}
	if !isStream {
		t.Fatal("isStream = false, want true for text/event-stream")
	}
}

func TestChatCompletionPropagatesUpstreamErrorResponse(t *testing.T) {
	const upstreamBody = `{"error":{"message":"rate limit exceeded"}}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = io.WriteString(w, upstreamBody)
	}))
	defer server.Close()

	provider := NewProvider("mistral-test", server.URL, "test-key", nil)
	body, _, statusCode, err := provider.ChatCompletion(context.Background(), &providers.ChatRequest{
		Model: "mistral-large-latest",
	})
	if err != nil {
		t.Fatalf("ChatCompletion() error = %v, want HTTP response passthrough", err)
	}
	defer body.Close()
	if statusCode != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want %d", statusCode, http.StatusTooManyRequests)
	}
	response, err := io.ReadAll(body)
	if err != nil {
		t.Fatalf("read error response: %v", err)
	}
	if string(response) != upstreamBody {
		t.Fatalf("response = %q, want %q", response, upstreamBody)
	}
}

func containsModel(models []string, model string) bool {
	for _, candidate := range models {
		if candidate == model {
			return true
		}
	}
	return false
}
