package router

import (
	"bytes"
	"context"
	"io"
	"math"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/ziuus/goroute/pkg/providers"
)

type mockProvider struct {
	id         string
	shouldFail bool
	failCode   int
	callCount  int
	delay      time.Duration
}

func (m *mockProvider) ID() string   { return m.id }
func (m *mockProvider) Type() string { return "mock" }
func (m *mockProvider) IsHealthy() bool {
	return true
}

func (m *mockProvider) ChatCompletion(ctx context.Context, req *providers.ChatRequest) (io.ReadCloser, bool, int, error) {
	m.callCount++
	if m.delay > 0 {
		time.Sleep(m.delay)
	}
	if m.shouldFail {
		return io.NopCloser(bytes.NewReader([]byte("rate limited"))), false, m.failCode, nil
	}
	return io.NopCloser(bytes.NewReader([]byte("success from " + m.id))), false, http.StatusOK, nil
}

func (m *mockProvider) Models(ctx context.Context) ([]providers.ModelInfo, error) {
	return []providers.ModelInfo{{ID: "claude-3-7-sonnet", Object: "model"}}, nil
}

func (m *mockProvider) Embeddings(ctx context.Context, req *providers.EmbeddingRequest) (*providers.EmbeddingResponse, error) {
	return &providers.EmbeddingResponse{
		Object: "list",
		Model:  req.Model,
		Data:   []providers.EmbeddingItem{{Object: "embedding", Index: 0, Embedding: []float64{0.1, 0.2}}},
	}, nil
}

func TestRouterFailoverCascade(t *testing.T) {
	primary := &mockProvider{id: "primary", shouldFail: true, failCode: http.StatusTooManyRequests} // 429
	backup := &mockProvider{id: "backup", shouldFail: false}

	r := NewRouter()
	r.RegisterProvider(primary)
	r.RegisterProvider(backup)

	r.RegisterCombo("smart-model", "failover", []Target{
		{ProviderID: "primary", Model: "claude-3-7-sonnet"},
		{ProviderID: "backup", Model: "claude-3-7-sonnet"},
	})

	req := &providers.ChatRequest{
		Model: "smart-model",
		Messages: []providers.ChatMessage{
			{Role: "user", Content: "hi"},
		},
	}

	body, _, status, err := r.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer body.Close()

	if status != http.StatusOK {
		t.Errorf("expected status 200, got %d", status)
	}

	resp, _ := io.ReadAll(body)
	if !strings.Contains(string(resp), "success from backup") {
		t.Errorf("expected response from backup provider, got: %s", string(resp))
	}

	if primary.callCount != 1 || backup.callCount != 1 {
		t.Errorf("expected both providers called, primary=%d, backup=%d", primary.callCount, backup.callCount)
	}
}

func TestRouterLowestLatencyStrategy(t *testing.T) {
	fastProv := &mockProvider{id: "fast-prov"}
	slowProv := &mockProvider{id: "slow-prov"}

	r := NewRouter()
	r.RegisterProvider(fastProv)
	r.RegisterProvider(slowProv)

	// Simulate latency telemetry: fast = 45ms, slow = 320ms
	r.RecordTargetLatency("fast-prov", "gpt-4o", 45)
	r.RecordTargetLatency("slow-prov", "gpt-4o", 320)

	r.RegisterCombo("low-latency-combo", "lowest_latency", []Target{
		{ProviderID: "slow-prov", Model: "gpt-4o"},
		{ProviderID: "fast-prov", Model: "gpt-4o"},
	})

	req := &providers.ChatRequest{
		Model: "low-latency-combo",
		Messages: []providers.ChatMessage{
			{Role: "user", Content: "test"},
		},
	}

	body, _, status, err := r.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer body.Close()

	if status != http.StatusOK {
		t.Errorf("expected status 200, got %d", status)
	}

	resp, _ := io.ReadAll(body)
	if !strings.Contains(string(resp), "success from fast-prov") {
		t.Errorf("expected fast-prov to be chosen first due to lowest latency, got: %s", string(resp))
	}

	if fastProv.callCount != 1 || slowProv.callCount != 0 {
		t.Errorf("expected only fast-prov called, fast=%d, slow=%d", fastProv.callCount, slowProv.callCount)
	}
}

func TestRouterQuotaWeightedStrategy(t *testing.T) {
	depletedProv := &mockProvider{id: "depleted-prov"}
	highQuotaProv := &mockProvider{id: "high-quota-prov"}

	r := NewRouter()
	r.RegisterProvider(depletedProv)
	r.RegisterProvider(highQuotaProv)

	r.RegisterCombo("quota-combo", "quota_weighted", []Target{
		{ProviderID: "depleted-prov", Model: "claude-3-5-sonnet", QuotaPercent: 5.0},
		{ProviderID: "high-quota-prov", Model: "claude-3-5-sonnet", QuotaPercent: 95.0},
	})

	req := &providers.ChatRequest{
		Model: "quota-combo",
		Messages: []providers.ChatMessage{
			{Role: "user", Content: "test"},
		},
	}

	body, _, status, err := r.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer body.Close()

	if status != http.StatusOK {
		t.Errorf("expected status 200, got %d", status)
	}

	resp, _ := io.ReadAll(body)
	if !strings.Contains(string(resp), "success from high-quota-prov") {
		t.Errorf("expected high-quota-prov to be selected first, got: %s", string(resp))
	}

	if highQuotaProv.callCount != 1 || depletedProv.callCount != 0 {
		t.Errorf("expected high-quota-prov called first, high=%d, depleted=%d", highQuotaProv.callCount, depletedProv.callCount)
	}
}

func TestRouterEWMAAccuracy(t *testing.T) {
	r := NewRouter()

	// Initial sample: 100ms -> EWMA = 100
	r.RecordTargetLatency("provA", "model1", 100)
	m1 := r.GetTargetMetrics("provA", "model1")
	if m1.EWMALatencyMs != 100.0 || m1.SampleCount != 1 {
		t.Fatalf("expected initial EWMA 100.0, got %f", m1.EWMALatencyMs)
	}

	// Second sample: 200ms -> EWMA = 0.25*200 + 0.75*100 = 50 + 75 = 125
	r.RecordTargetLatency("provA", "model1", 200)
	m2 := r.GetTargetMetrics("provA", "model1")
	if math.Abs(m2.EWMALatencyMs-125.0) > 0.001 || m2.SampleCount != 2 {
		t.Fatalf("expected EWMA 125.0, got %f", m2.EWMALatencyMs)
	}

	// Quota setter check
	r.SetTargetQuota("provA", "model1", 72.5)
	m3 := r.GetTargetMetrics("provA", "model1")
	if m3.QuotaPercent != 72.5 {
		t.Fatalf("expected QuotaPercent 72.5, got %f", m3.QuotaPercent)
	}
}

func TestNestedCombosCascade(t *testing.T) {
	provA := &mockProvider{id: "prov-a", shouldFail: true, failCode: http.StatusTooManyRequests}
	provB := &mockProvider{id: "prov-b", shouldFail: false}

	r := NewRouter()
	r.RegisterProvider(provA)
	r.RegisterProvider(provB)

	// Sub-combo 1: fast-tier (fails)
	r.RegisterCombo("fast-tier", "failover", []Target{
		{ProviderID: "prov-a", Model: "claude-haiku"},
	})

	// Sub-combo 2: heavy-tier (succeeds)
	r.RegisterCombo("heavy-tier", "failover", []Target{
		{ProviderID: "prov-b", Model: "claude-sonnet"},
	})

	// Master combo referencing both sub-combos
	r.RegisterCombo("master-resilient", "failover", []Target{
		{ProviderID: "combo", Model: "fast-tier"},
		{ProviderID: "combo", Model: "heavy-tier"},
	})

	req := &providers.ChatRequest{
		Model: "master-resilient",
		Messages: []providers.ChatMessage{
			{Role: "user", Content: "hello"},
		},
	}

	body, _, status, err := r.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer body.Close()

	if status != http.StatusOK {
		t.Errorf("expected status 200, got %d", status)
	}

	resp, _ := io.ReadAll(body)
	if !strings.Contains(string(resp), "success from prov-b") {
		t.Errorf("expected cascade into heavy-tier prov-b, got: %s", string(resp))
	}
}

func TestNestedCombosCycleDetection(t *testing.T) {
	r := NewRouter()

	// Mutual recursive cycle: combo-a -> combo-b -> combo-a
	r.RegisterCombo("combo-a", "failover", []Target{
		{ProviderID: "combo", Model: "combo-b"},
	})
	r.RegisterCombo("combo-b", "failover", []Target{
		{ProviderID: "combo", Model: "combo-a"},
	})

	req := &providers.ChatRequest{
		Model: "combo-a",
		Messages: []providers.ChatMessage{
			{Role: "user", Content: "hello"},
		},
	}

	_, _, status, err := r.Execute(context.Background(), req)
	if err == nil {
		t.Fatalf("expected loop error, got nil")
	}
	if status != http.StatusLoopDetected && status != http.StatusServiceUnavailable {
		t.Errorf("expected 508 Loop Detected, got %d", status)
	}
}
