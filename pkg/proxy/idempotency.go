package proxy

import (
	"bytes"
	"net/http"
	"sync"
	"time"
)

type idempotentEntry struct {
	statusCode int
	headers    http.Header
	body       []byte
	expiresAt  time.Time
}

type IdempotencyManager struct {
	mu      sync.RWMutex
	entries map[string]*idempotentEntry
	ttl     time.Duration
}

var globalIdempotency = NewIdempotencyManager(10 * time.Second)

func NewIdempotencyManager(ttl time.Duration) *IdempotencyManager {
	m := &IdempotencyManager{
		entries: make(map[string]*idempotentEntry),
		ttl:     ttl,
	}
	go m.cleanupLoop()
	return m
}

func (m *IdempotencyManager) cleanupLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		m.mu.Lock()
		now := time.Now()
		for k, v := range m.entries {
			if now.After(v.expiresAt) {
				delete(m.entries, k)
			}
		}
		m.mu.Unlock()
	}
}

func (m *IdempotencyManager) GetKey(r *http.Request) string {
	if k := r.Header.Get("Idempotency-Key"); k != "" {
		return k
	}
	if k := r.Header.Get("X-Request-Id"); k != "" {
		return k
	}
	return ""
}

func (m *IdempotencyManager) Lookup(key string) (*idempotentEntry, bool) {
	if key == "" {
		return nil, false
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	entry, ok := m.entries[key]
	if !ok || time.Now().After(entry.expiresAt) {
		return nil, false
	}
	return entry, true
}

func (m *IdempotencyManager) Store(key string, statusCode int, headers http.Header, body []byte) {
	if key == "" || len(body) == 0 {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	cleanHeaders := make(http.Header)
	for k, v := range headers {
		if k == "Content-Type" || k == "Content-Length" {
			cleanHeaders[k] = v
		}
	}

	m.entries[key] = &idempotentEntry{
		statusCode: statusCode,
		headers:    cleanHeaders,
		body:       body,
		expiresAt:  time.Now().Add(m.ttl),
	}
}

type responseRecorder struct {
	http.ResponseWriter
	statusCode int
	body       bytes.Buffer
}

func (rec *responseRecorder) WriteHeader(code int) {
	rec.statusCode = code
	rec.ResponseWriter.WriteHeader(code)
}

func (rec *responseRecorder) Write(b []byte) (int, error) {
	if rec.statusCode == 0 {
		rec.statusCode = http.StatusOK
	}
	rec.body.Write(b)
	return rec.ResponseWriter.Write(b)
}
