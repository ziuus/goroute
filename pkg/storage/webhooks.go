package storage

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// WebhookSubscription defines an automated outbound notification target.
type WebhookSubscription struct {
	ID              string    `json:"id"`
	URL             string    `json:"url"`
	Events          []string  `json:"events"`
	Secret          string    `json:"secret,omitempty"`
	Enabled         bool      `json:"enabled"`
	Description     string    `json:"description,omitempty"`
	Kind            string    `json:"kind"` // "slack", "telegram", "discord", "custom"
	LastTriggeredAt *time.Time `json:"lastTriggeredAt,omitempty"`
	LastStatus      *int      `json:"lastStatus,omitempty"`
	FailureCount    int       `json:"failureCount"`
	CreatedAt       time.Time `json:"createdAt"`
}

// WebhookDelivery records historical dispatch attempts and outcomes for a webhook.
type WebhookDelivery struct {
	ID              int64      `json:"id"`
	WebhookID       string     `json:"webhookId"`
	EventType       string     `json:"eventType"`
	Status          string     `json:"status"` // "success" | "failed"
	HTTPStatus      *int       `json:"httpStatus,omitempty"`
	LatencyMs       *int64     `json:"latencyMs,omitempty"`
	Error           string     `json:"error,omitempty"`
	PayloadSnapshot string     `json:"payloadSnapshot,omitempty"`
	CreatedAt       time.Time  `json:"createdAt"`
}

// WebhookStore manages webhooks table in SQLite.
type WebhookStore struct {
	db *DBStore
	mu sync.RWMutex
}

var (
	globalWebhookStore *WebhookStore
	webhookOnce        sync.Once
)

func GetWebhookStore() *WebhookStore {
	webhookOnce.Do(func() {
		globalWebhookStore = &WebhookStore{
			db: GetDB(),
		}
		globalWebhookStore.ensureTable()
	})
	return globalWebhookStore
}

func (s *WebhookStore) ensureTable() {
	if s.db == nil || s.db.db == nil {
		return
	}
	_, _ = s.db.db.Exec(`
		CREATE TABLE IF NOT EXISTS webhooks (
			id TEXT PRIMARY KEY,
			url TEXT NOT NULL,
			events TEXT NOT NULL DEFAULT '["*"]',
			secret TEXT,
			enabled INTEGER DEFAULT 1,
			description TEXT,
			kind TEXT DEFAULT 'custom',
			last_triggered_at DATETIME,
			last_status INTEGER,
			failure_count INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`)
}

func (s *WebhookStore) ListWebhooks() ([]WebhookSubscription, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var list []WebhookSubscription
	if s.db == nil || s.db.db == nil {
		return list, nil
	}

	rows, err := s.db.db.Query(`
		SELECT id, url, events, secret, enabled, description, kind, last_triggered_at, last_status, failure_count, created_at
		FROM webhooks
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var w WebhookSubscription
		var evStr string
		var sec, desc, kind *string
		var lastTrig *string
		var lastStat *int
		var enInt int
		var crStr string

		if err := rows.Scan(
			&w.ID,
			&w.URL,
			&evStr,
			&sec,
			&enInt,
			&desc,
			&kind,
			&lastTrig,
			&lastStat,
			&w.FailureCount,
			&crStr,
		); err == nil {
			_ = json.Unmarshal([]byte(evStr), &w.Events)
			w.Enabled = (enInt == 1)
			if sec != nil {
				w.Secret = *sec
			}
			if desc != nil {
				w.Description = *desc
			}
			if kind != nil {
				w.Kind = *kind
			} else {
				w.Kind = "custom"
			}
			if lastTrig != nil {
				t, _ := time.Parse(time.RFC3339, *lastTrig)
				w.LastTriggeredAt = &t
			}
			w.LastStatus = lastStat
			w.CreatedAt, _ = time.Parse(time.RFC3339, crStr)
			list = append(list, w)
		}
	}
	return list, nil
}

func (s *WebhookStore) CreateWebhook(url, secret, description, kind string, events []string) (WebhookSubscription, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(events) == 0 {
		events = []string{"*"}
	}
	if kind == "" {
		kind = "custom"
	}

	w := WebhookSubscription{
		ID:          fmt.Sprintf("wh_%d", time.Now().UnixNano()),
		URL:         url,
		Events:      events,
		Secret:      secret,
		Enabled:     true,
		Description: description,
		Kind:        kind,
		CreatedAt:   time.Now().UTC(),
	}

	if s.db == nil || s.db.db == nil {
		return w, fmt.Errorf("database not initialized")
	}

	evBytes, _ := json.Marshal(events)

	_, err := s.db.db.Exec(`
		INSERT INTO webhooks (id, url, events, secret, enabled, description, kind, created_at)
		VALUES (?, ?, ?, ?, 1, ?, ?, ?)
	`, w.ID, w.URL, string(evBytes), w.Secret, w.Description, w.Kind, w.CreatedAt.Format(time.RFC3339))

	return w, err
}

func (s *WebhookStore) DeleteWebhook(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.db == nil || s.db.db == nil {
		return fmt.Errorf("database not initialized")
	}

	_, err := s.db.db.Exec(`DELETE FROM webhooks WHERE id = ?`, id)
	return err
}

// RecordDelivery persists a webhook delivery attempt outcome.
func (s *WebhookStore) RecordDelivery(del *WebhookDelivery) error {
	if s.db == nil {
		return fmt.Errorf("database not initialized")
	}
	return s.db.RecordWebhookDelivery(del)
}

// GetDeliveries returns historical deliveries for a webhook.
func (s *WebhookStore) GetDeliveries(webhookID string, limit int) ([]WebhookDelivery, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	return s.db.GetWebhookDeliveries(webhookID, limit)
}

// UpdateWebhookTriggerStatus updates last_triggered_at, last_status, and failure_count.
func (s *WebhookStore) UpdateWebhookTriggerStatus(id string, lastStatus int, failure bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.db == nil || s.db.db == nil {
		return fmt.Errorf("database not initialized")
	}

	now := time.Now().UTC().Format(time.RFC3339)
	if failure {
		_, err := s.db.db.Exec(`
			UPDATE webhooks
			SET last_triggered_at = ?, last_status = ?, failure_count = failure_count + 1
			WHERE id = ?
		`, now, lastStatus, id)
		return err
	}

	_, err := s.db.db.Exec(`
		UPDATE webhooks
		SET last_triggered_at = ?, last_status = ?, failure_count = 0
		WHERE id = ?
	`, now, lastStatus, id)
	return err
}
