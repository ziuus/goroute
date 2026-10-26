package proxy

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ziuus/goroute/pkg/quota"
	"github.com/ziuus/goroute/pkg/router"
	"github.com/ziuus/goroute/pkg/storage"
)

func TestCliToolsEndpoints(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_clitools.db")
	store, err := storage.InitDB(dbPath)
	if err != nil {
		t.Fatalf("failed to init db: %v", err)
	}
	defer store.Close()

	r := router.NewRouter()
	limiter := quota.NewLimiter()
	server := NewServer(r, limiter, false)

	// 1. GET /api/cli-tools/all-statuses
	req := httptest.NewRequest(http.MethodGet, "/api/cli-tools/all-statuses", nil)
	rr := httptest.NewRecorder()
	server.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var statuses map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &statuses); err != nil {
		t.Fatalf("failed to unmarshal statuses: %v", err)
	}
	if _, ok := statuses["claude"]; !ok {
		t.Fatalf("expected claude in statuses")
	}

	// 2. GET /api/cli-tools/detect
	req = httptest.NewRequest(http.MethodGet, "/api/cli-tools/detect", nil)
	rr = httptest.NewRecorder()
	server.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for detect, got %d", rr.Code)
	}

	// 3. GET /api/cli-tools/claude-settings
	req = httptest.NewRequest(http.MethodGet, "/api/cli-tools/claude-settings", nil)
	rr = httptest.NewRecorder()
	server.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for claude-settings, got %d", rr.Code)
	}

	// 4. POST /api/cli-tools/apply (dry-run)
	body := `{"toolId":"claude","baseUrl":"http://127.0.0.1:8128","dryRun":true}`
	req = httptest.NewRequest(http.MethodPost, "/api/cli-tools/apply", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	server.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for apply dry-run, got %d", rr.Code)
	}

	// 5. GET /api/cli-tools/openclaw/auto-order
	req = httptest.NewRequest(http.MethodGet, "/api/cli-tools/openclaw/auto-order", nil)
	rr = httptest.NewRecorder()
	server.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for openclaw auto-order, got %d", rr.Code)
	}

	// 6. GET /api/cli-tools/antigravity-mitm
	req = httptest.NewRequest(http.MethodGet, "/api/cli-tools/antigravity-mitm", nil)
	rr = httptest.NewRecorder()
	server.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for mitm get, got %d", rr.Code)
	}
}
