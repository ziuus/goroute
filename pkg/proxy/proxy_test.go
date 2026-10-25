package proxy

import (
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

type dummyProvider struct {
	id string
}

func (d *dummyProvider) ID() string   { return d.id }
func (d *dummyProvider) Type() string { return "dummy" }
func (d *dummyProvider) IsHealthy() bool {
	return true
}

func (d *dummyProvider) ChatCompletion(ctx context.Context, req *providers.ChatRequest) (io.ReadCloser, bool, int, error) {
	if req.Stream {
		sseData := "data: {\"id\":\"chatcmpl-test\",\"choices\":[{\"delta\":{\"content\":\"streamed content\"}}]}\n\ndata: [DONE]\n\n"
		return io.NopCloser(strings.NewReader(sseData)), true, http.StatusOK, nil
	}
	resp := `{"id":"chatcmpl-test","choices":[{"message":{"role":"assistant","content":"json content"}}]}`
	return io.NopCloser(strings.NewReader(resp)), false, http.StatusOK, nil
}

func (d *dummyProvider) Models(ctx context.Context) ([]providers.ModelInfo, error) {
	return []providers.ModelInfo{{ID: "dummy-model", Object: "model"}}, nil
}

func (d *dummyProvider) Embeddings(ctx context.Context, req *providers.EmbeddingRequest) (*providers.EmbeddingResponse, error) {
	return &providers.EmbeddingResponse{
		Object: "list",
		Model:  req.Model,
		Data:   []providers.EmbeddingItem{{Object: "embedding", Index: 0, Embedding: []float64{0.1, 0.2, 0.3}}},
	}, nil
}

func TestProxyServerChatCompletions(t *testing.T) {
	r := router.NewRouter()
	r.RegisterProvider(&dummyProvider{id: "dummy"})
	limiter := quota.NewLimiter()

	srv := NewServer(r, limiter, true)
	srv.SetConfigPaths("", "")
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	// 1. Test non-streaming (first call: Cache MISS)
	reqBody := `{"model":"dummy/dummy-model","messages":[{"role":"user","content":"hi"}]}`
	resp, err := http.Post(ts.URL+"/v1/chat/completions", "application/json", strings.NewReader(reqBody))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if resp.Header.Get("X-Cache") != "MISS" {
		t.Errorf("expected X-Cache: MISS on first call, got %s", resp.Header.Get("X-Cache"))
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "json content") {
		t.Errorf("expected json content, got: %s", string(body))
	}

	// 1b. Test second call: Cache HIT!
	resp2, err := http.Post(ts.URL+"/v1/chat/completions", "application/json", strings.NewReader(reqBody))
	if err != nil {
		t.Fatalf("request 2 failed: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.Header.Get("X-Cache") != "HIT" {
		t.Errorf("expected X-Cache: HIT on second call, got %s", resp2.Header.Get("X-Cache"))
	}

	// 2. Test streaming
	respStream, err := http.Post(ts.URL+"/v1/chat/completions", "application/json", strings.NewReader(`{"model":"dummy/dummy-model","messages":[{"role":"user","content":"stream"}],"stream":true}`))
	if err != nil {
		t.Fatalf("stream request failed: %v", err)
	}
	defer respStream.Body.Close()

	if !strings.Contains(respStream.Header.Get("Content-Type"), "text/event-stream") {
		t.Errorf("expected text/event-stream, got %s", respStream.Header.Get("Content-Type"))
	}
	streamBytes, _ := io.ReadAll(respStream.Body)
	if !strings.Contains(string(streamBytes), "streamed content") {
		t.Errorf("expected streamed content, got: %s", string(streamBytes))
	}

	// 3. Test /v1/models
	modelsResp, err := http.Get(ts.URL + "/v1/models")
	if err != nil {
		t.Fatalf("models request failed: %v", err)
	}
	defer modelsResp.Body.Close()

	var modelsData struct {
		Data []providers.ModelInfo `json:"data"`
	}
	json.NewDecoder(modelsResp.Body).Decode(&modelsData)
	if len(modelsData.Data) == 0 {
		t.Errorf("expected at least 1 model in /v1/models")
	}

	// 4. Test /v1/embeddings
	embReqBody := `{"model":"dummy/dummy-model","input":"test string"}`
	embResp, err := http.Post(ts.URL+"/v1/embeddings", "application/json", strings.NewReader(embReqBody))
	if err != nil {
		t.Fatalf("embeddings request failed: %v", err)
	}
	defer embResp.Body.Close()
	if embResp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK from embeddings, got %d", embResp.StatusCode)
	}
	var embData providers.EmbeddingResponse
	json.NewDecoder(embResp.Body).Decode(&embData)
	if len(embData.Data) == 0 {
		t.Errorf("expected embeddings data in response")
	}
}
