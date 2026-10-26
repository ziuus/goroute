package keys

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ziuus/goroute/pkg/config"
)

type Manager struct {
	mu      sync.RWMutex
	cfgPath string
	keys    map[string]*config.VirtualKey // by raw key string
	byID    map[string]*config.VirtualKey // by ID
}

func NewManager(cfgPath string) *Manager {
	m := &Manager{
		cfgPath: cfgPath,
		keys:    make(map[string]*config.VirtualKey),
		byID:    make(map[string]*config.VirtualKey),
	}
	m.Reload()
	return m
}

func (m *Manager) Reload() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	cfg, err := config.Load(m.cfgPath)
	if err != nil {
		return err
	}

	m.keys = make(map[string]*config.VirtualKey)
	m.byID = make(map[string]*config.VirtualKey)
	for i := range cfg.Keys {
		k := &cfg.Keys[i]
		m.keys[k.Key] = k
		m.byID[k.ID] = k
	}
	return nil
}

func (m *Manager) HasKeys() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.byID) > 0
}

func (m *Manager) ListKeys() []config.VirtualKey {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var res []config.VirtualKey
	for _, k := range m.byID {
		sanitized := *k
		// Redact raw key for listing, show prefix + suffix
		if len(sanitized.Key) > 16 {
			sanitized.Key = sanitized.Key[:10] + "..." + sanitized.Key[len(sanitized.Key)-4:]
		}
		res = append(res, sanitized)
	}
	return res
}

func (m *Manager) CreateKey(name string, allowedModels []string, maxRPM int) (*config.VirtualKey, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	cfg, err := config.Load(m.cfgPath)
	if err != nil {
		return nil, err
	}

	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return nil, err
	}
	rawKey := "sk-goroute-" + hex.EncodeToString(bytes)
	keyID := fmt.Sprintf("key_%d", time.Now().UnixNano())

	newKey := config.VirtualKey{
		ID:            keyID,
		Name:          name,
		Key:           rawKey,
		IsActive:      true,
		AllowedModels: allowedModels,
		MaxRPM:        maxRPM,
		CreatedAt:     time.Now().UTC(),
	}

	cfg.Keys = append(cfg.Keys, newKey)
	if err := cfg.Save(m.cfgPath); err != nil {
		return nil, err
	}

	m.keys[newKey.Key] = &newKey
	m.byID[newKey.ID] = &newKey
	return &newKey, nil
}

func (m *Manager) RevokeKey(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	cfg, err := config.Load(m.cfgPath)
	if err != nil {
		return err
	}

	var newKeys []config.VirtualKey
	for _, k := range cfg.Keys {
		if k.ID != id {
			newKeys = append(newKeys, k)
		}
	}
	cfg.Keys = newKeys
	if err := cfg.Save(m.cfgPath); err != nil {
		return err
	}

	if k, ok := m.byID[id]; ok {
		delete(m.keys, k.Key)
		delete(m.byID, id)
	}
	return nil
}

// ExtractToken extracts auth token from standard headers (Authorization, x-api-key, x-goog-api-key) or query (?key=)
func ExtractToken(r *http.Request) string {
	if r == nil {
		return ""
	}
	if auth := r.Header.Get("Authorization"); auth != "" {
		trimmed := strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
		if trimmed != "" {
			return trimmed
		}
	}
	if xKey := r.Header.Get("x-api-key"); xKey != "" {
		return strings.TrimSpace(xKey)
	}
	if gKey := r.Header.Get("x-goog-api-key"); gKey != "" {
		return strings.TrimSpace(gKey)
	}
	if r.URL != nil {
		if qKey := r.URL.Query().Get("key"); qKey != "" {
			return strings.TrimSpace(qKey)
		}
	}
	return ""
}

// ValidateRequest extracts token across headers and validates against virtual key store
func (m *Manager) ValidateRequest(r *http.Request, model string) (bool, *config.VirtualKey, string) {
	if !m.HasKeys() {
		return true, nil, ""
	}
	token := ExtractToken(r)
	if token == "" {
		return false, nil, "Missing API key (provide via Authorization: Bearer, x-api-key, or x-goog-api-key)"
	}
	return m.Validate(token, model)
}

func (m *Manager) Validate(authHeader string, model string) (bool, *config.VirtualKey, string) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// If no virtual keys are registered, open proxy access is permitted
	if len(m.byID) == 0 {
		return true, nil, ""
	}

	token := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
	if token == "" {
		return false, nil, "Missing Authorization Bearer token"
	}

	// Always allow standard local master development key "goroute"
	if token == "goroute" {
		return true, nil, ""
	}

	vk, ok := m.keys[token]
	if !ok || !vk.IsActive {
		return false, nil, "Invalid or revoked API key"
	}

	if len(vk.AllowedModels) > 0 && model != "" {
		allowed := false
		for _, pattern := range vk.AllowedModels {
			if pattern == "*" || pattern == model || matchGlob(pattern, model) {
				allowed = true
				break
			}
		}
		if !allowed {
			return false, vk, fmt.Sprintf("API key not permitted to access model '%s'", model)
		}
	}

	return true, vk, ""
}

func (m *Manager) TrackUsage(keyID string, tokens int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if k, ok := m.byID[keyID]; ok {
		k.TotalTokensUsed += tokens
	}
}

func matchGlob(pattern, val string) bool {
	matched, _ := filepath.Match(pattern, val)
	return matched
}
