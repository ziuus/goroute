package proxy

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/ziuus/goroute/pkg/config"
	"github.com/ziuus/goroute/pkg/providers"
	"github.com/ziuus/goroute/pkg/router"
)

func TestReloadRegistersMistralAndPreservesOpenAI(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	cfg := config.DefaultConfig()
	cfg.Providers = []config.ProviderConfig{
		{ID: "mistral-reload-test", Type: "mistral", APIKey: "mistral-key"},
		{ID: "openai-reload-test", Type: "openai", APIKey: "openai-key"},
	}
	if err := cfg.Save(configPath); err != nil {
		t.Fatalf("save config: %v", err)
	}

	r := router.NewRouter()
	srv := &Server{
		router:    r,
		cfgPath:   configPath,
		storePath: filepath.Join(tmpDir, "vault.json"),
	}
	if err := srv.Reload(); err != nil {
		t.Fatalf("Reload() error = %v", err)
	}

	mistralProvider, ok := r.GetProvider("mistral-reload-test")
	if !ok {
		t.Fatal("Reload() did not register the Mistral provider")
	}
	if got := mistralProvider.Type(); got != "mistral" {
		t.Fatalf("Mistral provider Type() = %q, want %q", got, "mistral")
	}
	openaiProvider, ok := r.GetProvider("openai-reload-test")
	if !ok {
		t.Fatal("Reload() did not register the OpenAI provider")
	}
	if got := openaiProvider.Type(); got != "openai" {
		t.Fatalf("explicit OpenAI provider Type() = %q, want %q", got, "openai")
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost || req.URL.Path != "/chat/completions" {
			t.Errorf("request = %s %s, want POST /chat/completions", req.Method, req.URL.Path)
		}
		if got := req.Header.Get("Authorization"); got != "Bearer mistral-key" {
			t.Errorf("Authorization = %q, want Bearer mistral-key", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"choices":[{"message":{"role":"assistant","content":"ok"}}]}`)
	}))
	defer server.Close()

	// Reload again with a test endpoint to verify the registered adapter can make
	// the same OpenAI-compatible request through the application reload path.
	cfg.Providers[0].BaseURL = server.URL
	if err := cfg.Save(configPath); err != nil {
		t.Fatalf("save endpoint config: %v", err)
	}
	if err := srv.Reload(); err != nil {
		t.Fatalf("Reload() with test endpoint error = %v", err)
	}
	mistralProvider, ok = r.GetProvider("mistral-reload-test")
	if !ok {
		t.Fatal("Reload() did not register the Mistral provider with the test endpoint")
	}
	body, isStream, statusCode, err := mistralProvider.ChatCompletion(context.Background(), &providers.ChatRequest{
		Model: "codestral-latest",
	})
	if err != nil {
		t.Fatalf("Mistral ChatCompletion() error = %v", err)
	}
	defer body.Close()
	if statusCode != http.StatusOK || isStream {
		t.Fatalf("ChatCompletion() = status %d, stream %t; want status 200, stream false", statusCode, isStream)
	}
}
