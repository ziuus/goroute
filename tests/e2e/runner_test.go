package e2e_test

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/ziuus/goroute/tests/e2e/infra"
)

func TestMain(m *testing.M) {
	// Pre-build GoRoute gateway binary once for the entire test run
	if _, err := infra.EnsureBinary(); err != nil {
		fmt.Fprintf(os.Stderr, "Error pre-building GoRoute binary: %v\n", err)
		os.Exit(1)
	}

	code := m.Run()

	infra.CleanupBuiltBinary()
	os.Exit(code)
}

// TestE2E_Runner establishes the master E2E test harness sanity verification.
func TestE2E_Runner(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 1. Spin up lightweight mock OpenAI upstream
	mockOpenAI := infra.NewOpenAIMock()
	defer mockOpenAI.Close()
	mockOpenAI.SetCustomResponse("Opaque-box E2E test pipeline verified!")

	// 2. Prepare ephemeral config.yaml routing an 'openai' provider to our local mock
	configYAML := fmt.Sprintf(`
server:
  host: "127.0.0.1"
  enable_dashboard: false
  disable_dashboard: true

providers:
  - id: "openai"
    type: "openai"
    base_url: "%s/v1"
    api_key: "mock-sk-12345"
    accounts:
      - id: "acc-mock-1"
        api_key: "mock-sk-12345"

combos:
  - name: "gpt-4o"
    strategy: "failover"
    targets:
      - provider_id: "openai"
        model: "gpt-4o"
keys: []
`, mockOpenAI.URL())

	// 3. Initialize isolated test harness with ephemeral DB and custom port
	harness := infra.NewHarness(t, infra.WithCustomConfig(configYAML))
	if err := harness.Start(); err != nil {
		t.Fatalf("Failed to start GoRoute test harness: %v", err)
	}

	client := harness.Client()

	// ------------------------------------------------------------------------
	// Test Sub-case 1: Daemon Health & Status Check
	// ------------------------------------------------------------------------
	t.Run("DaemonHealthAndStatus", func(t *testing.T) {
		status, healthData, err := client.Health(ctx)
		if err != nil {
			t.Fatalf("Health check failed: %v", err)
		}
		if status != http.StatusOK {
			t.Fatalf("Expected HTTP 200 from /health, got %d", status)
		}

		if s, ok := healthData["status"].(string); !ok || s != "ok" {
			t.Logf("Notice: health status value: %v", healthData["status"])
		}

		if !harness.IsRunning() {
			t.Fatal("Expected harness daemon to be running")
		}
		if harness.PID() <= 0 {
			t.Fatalf("Invalid daemon process PID: %d", harness.PID())
		}
	})

	// ------------------------------------------------------------------------
	// Test Sub-case 2: Non-Streaming Chat Completion Proxying
	// ------------------------------------------------------------------------
	t.Run("ChatCompletionProxying", func(t *testing.T) {
		reqPayload := map[string]interface{}{
			"model": "gpt-4o",
			"messages": []map[string]string{
				{"role": "user", "content": "Ping GoRoute opaque-box test"},
			},
		}

		resp, body, err := client.ChatCompletion(ctx, reqPayload, nil)
		if err != nil {
			t.Fatalf("ChatCompletion request failed: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected HTTP 200, got %d. Body: %s", resp.StatusCode, string(body))
		}

		// Assert mock upstream received the proxy request
		if mockOpenAI.Recorder.Count() == 0 {
			t.Fatal("Expected mock upstream to have recorded at least 1 request")
		}
		lastReq := mockOpenAI.Recorder.LastRequest()
		if lastReq == nil || lastReq.Path != "/v1/chat/completions" {
			t.Fatalf("Expected upstream path /v1/chat/completions, got %v", lastReq)
		}
	})

	// ------------------------------------------------------------------------
	// Test Sub-case 3: SSE Streaming & Time-To-First-Token (TTFT) Tracking
	// ------------------------------------------------------------------------
	t.Run("SSEStreamingWithTTFT", func(t *testing.T) {
		streamPayload := map[string]interface{}{
			"model":  "gpt-4o",
			"stream": true,
			"messages": []map[string]string{
				{"role": "user", "content": "Stream tokens to client"},
			},
		}

		var eventCount int
		streamResult, err := client.StreamSSE(ctx, "/v1/chat/completions", streamPayload, nil, func(evt infra.SSEEvent) error {
			eventCount++
			return nil
		})

		if err != nil {
			t.Fatalf("StreamSSE error: %v", err)
		}
		if streamResult.StatusCode != http.StatusOK {
			t.Fatalf("Expected HTTP 200, got %d", streamResult.StatusCode)
		}
		if streamResult.TTFT <= 0 {
			t.Fatalf("Expected positive TTFT duration, got %v", streamResult.TTFT)
		}
		if len(streamResult.Events) == 0 {
			t.Fatal("Expected at least one SSE event to be collected")
		}
		if streamResult.FullText == "" {
			t.Fatal("Expected non-empty full text extracted from stream chunks")
		}
		t.Logf("Verified SSE stream: TTFT=%v, Events=%d, Text=%q",
			streamResult.TTFT, len(streamResult.Events), streamResult.FullText)
	})

	// ------------------------------------------------------------------------
	// Test Sub-case 4: SQLite Persistence Direct Verification
	// ------------------------------------------------------------------------
	t.Run("SQLiteDirectVerification", func(t *testing.T) {
		verifier, err := harness.Verifier()
		if err != nil {
			t.Fatalf("Failed to open SQLite verifier: %v", err)
		}

		// Verify table exists
		tableExists, err := verifier.TableExists("call_logs")
		if err != nil {
			t.Fatalf("Failed checking call_logs table existence: %v", err)
		}
		if !tableExists {
			t.Fatal("Expected call_logs table to exist in SQLite database")
		}

		// Wait briefly for asynchronous SQLite flush if needed
		time.Sleep(100 * time.Millisecond)

		logCount, err := verifier.GetCallLogCount()
		if err != nil {
			t.Fatalf("Failed counting call logs: %v", err)
		}
		if logCount < 1 {
			t.Fatalf("Expected at least 1 call log entry in SQLite, found %d", logCount)
		}

		lastLog, err := verifier.GetLastCallLog()
		if err != nil {
			t.Fatalf("Failed getting last call log: %v", err)
		}
		if lastLog == nil {
			t.Fatal("Expected non-nil last call log record")
		}
		if lastLog.Model != "gpt-4o" {
			t.Errorf("Expected logged model gpt-4o, got %s", lastLog.Model)
		}
		t.Logf("Verified SQLite record: ID=%s, Model=%s, Status=%d, InputTokens=%d, OutputTokens=%d",
			lastLog.ID, lastLog.Model, lastLog.StatusCode, lastLog.InputTokens, lastLog.OutputTokens)
	})

	// ------------------------------------------------------------------------
	// Test Sub-case 5: Graceful Shutdown
	// ------------------------------------------------------------------------
	t.Run("GracefulShutdown", func(t *testing.T) {
		if err := harness.Stop(); err != nil {
			t.Fatalf("Harness graceful stop failed: %v", err)
		}
		if harness.IsRunning() {
			t.Fatal("Expected harness daemon to no longer be running")
		}
	})
}
