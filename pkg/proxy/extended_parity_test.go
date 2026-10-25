package proxy

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ziuus/goroute/pkg/quota"
	"github.com/ziuus/goroute/pkg/router"
	"github.com/ziuus/goroute/pkg/storage"
)

func TestExtendedOmniRouteParitySubsystems(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/test_parity.db"
	store, err := storage.InitDB(dbPath)
	if err != nil {
		t.Fatalf("failed to init db: %v", err)
	}
	defer store.Close()

	r := router.NewRouter()
	limiter := quota.NewLimiter()
	server := NewServer(r, limiter, false)
	handler := server.Handler()

	// 1. Agent-to-Agent Agent Card
	t.Run("A2A Agent Card", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/.well-known/agent.json", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}
		var card map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &card); err != nil {
			t.Fatalf("failed to parse card: %v", err)
		}
		if card["name"] == "" {
			t.Fatalf("expected non-empty agent card name")
		}
	})

	// 2. A2A Status & Tasks
	t.Run("A2A Tasks Lifecycle", func(t *testing.T) {
		// Status
		req := httptest.NewRequest(http.MethodGet, "/api/a2a/status", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}

		// Create Task
		createBody := `{"skill":"smart-routing","input":"route to fastest"}`
		req = httptest.NewRequest(http.MethodPost, "/api/a2a/tasks", strings.NewReader(createBody))
		req.Header.Set("Content-Type", "application/json")
		rr = httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d", rr.Code)
		}
		var task map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &task)
		taskId := task["id"].(string)

		// Get Task
		req = httptest.NewRequest(http.MethodGet, "/api/a2a/tasks/"+taskId, nil)
		rr = httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200 for get task, got %d", rr.Code)
		}

		// Cancel Task
		req = httptest.NewRequest(http.MethodPost, "/api/a2a/tasks/"+taskId+"/cancel", nil)
		rr = httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200 for cancel task, got %d", rr.Code)
		}
	})

	// 3. Agent Skills Catalog & Coverage
	t.Run("Agent Skills", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/agent-skills", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200 for skills, got %d", rr.Code)
		}

		req = httptest.NewRequest(http.MethodGet, "/api/agent-skills/coverage", nil)
		rr = httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200 for coverage, got %d", rr.Code)
		}

		req = httptest.NewRequest(http.MethodGet, "/api/agent-skills/goroute-auth/raw", nil)
		rr = httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200 for raw skill, got %d", rr.Code)
		}
	})

	// 4. VNC Sessions
	t.Run("VNC Sessions", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/vnc-session", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}

		// Start session
		req = httptest.NewRequest(http.MethodPost, "/api/vnc-session/claude/start", nil)
		rr = httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d", rr.Code)
		}
		var sess map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &sess)
		sessId := sess["sessionId"].(string)

		// Harvest
		req = httptest.NewRequest(http.MethodPost, "/api/vnc-session/claude/"+sessId+"/harvest", nil)
		rr = httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200 for harvest, got %d", rr.Code)
		}

		// Delete
		req = httptest.NewRequest(http.MethodDelete, "/api/vnc-session/claude/"+sessId, nil)
		rr = httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200 for delete, got %d", rr.Code)
		}
	})

	// 5. Auth Login & Logout
	t.Run("Auth Lifecycle", func(t *testing.T) {
		body := `{"password":"admin"}`
		req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200 for login, got %d", rr.Code)
		}

		cookie := rr.Result().Cookies()
		foundToken := false
		for _, c := range cookie {
			if c.Name == "auth_token" && c.Value != "" {
				foundToken = true
			}
		}
		if !foundToken {
			t.Fatalf("expected auth_token cookie to be set")
		}

		// Logout
		req = httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
		rr = httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200 for logout, got %d", rr.Code)
		}
	})

	// 6. Web Search
	t.Run("Web Search Endpoints", func(t *testing.T) {
		searchBody := `{"query":"GoRoute high performance","max_results":3}`
		req := httptest.NewRequest(http.MethodPost, "/v1/search", strings.NewReader(searchBody))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		// Should succeed (or return graceful response)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200 for search, got %d", rr.Code)
		}
	})
}
