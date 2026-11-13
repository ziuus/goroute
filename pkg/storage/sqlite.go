package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	_ "modernc.org/sqlite"
)

// CallLog represents a persistent historical proxy query record.
type CallLog struct {
	ID              string    `json:"id"`
	Timestamp       time.Time `json:"timestamp"`
	Endpoint        string    `json:"endpoint"`
	Model           string    `json:"model"`
	Provider        string    `json:"provider"`
	AccountID       string    `json:"account_id"`
	InputTokens     int       `json:"input_tokens"`
	OutputTokens    int       `json:"output_tokens"`
	CostUSD         float64   `json:"cost_usd"`
	LatencyMs       int64     `json:"latency_ms"`
	StatusCode      int       `json:"status_code"`
	ErrorMessage    string    `json:"error_message,omitempty"`
	PromptPreview   string    `json:"prompt_preview,omitempty"`
	ResponsePreview string    `json:"response_preview,omitempty"`
}

// DBStore wraps SQLite database operations for GoRoute.
type DBStore struct {
	db *sql.DB
	mu sync.RWMutex
}

var (
	globalDB atomic.Pointer[DBStore]
	initDBMu sync.Mutex
)

// InitDB initializes SQLite storage in ~/.goroute/goroute.db
func InitDB(dbPath string) (*DBStore, error) {
	if dbPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		dir := filepath.Join(home, ".goroute")
		if err := os.MkdirAll(dir, 0700); err != nil {
			return nil, err
		}
		dbPath = filepath.Join(dir, "goroute.db")
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite: %w", err)
	}

	// High concurrency tuning for SQLite in WAL mode
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(10 * time.Minute)

	if _, err := db.Exec(`
		PRAGMA journal_mode = WAL;
		PRAGMA synchronous = NORMAL;
		PRAGMA foreign_keys = ON;
		PRAGMA busy_timeout = 5000;
	`); err != nil {
		return nil, fmt.Errorf("failed to execute pragma: %w", err)
	}

	schema := `
	CREATE TABLE IF NOT EXISTS call_logs (
		id TEXT PRIMARY KEY,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		model TEXT NOT NULL,
		provider TEXT NOT NULL,
		account_id TEXT,
		input_tokens INTEGER DEFAULT 0,
		output_tokens INTEGER DEFAULT 0,
		cost_usd REAL DEFAULT 0.0,
		latency_ms INTEGER DEFAULT 0,
		status_code INTEGER DEFAULT 200,
		error_message TEXT,
		endpoint TEXT DEFAULT '/v1/chat/completions',
		prompt_preview TEXT DEFAULT '',
		response_preview TEXT DEFAULT ''
	);

	CREATE INDEX IF NOT EXISTS idx_call_logs_timestamp ON call_logs(timestamp DESC);
	CREATE INDEX IF NOT EXISTS idx_call_logs_provider ON call_logs(provider);
	CREATE INDEX IF NOT EXISTS idx_call_logs_model ON call_logs(model);

	CREATE TABLE IF NOT EXISTS domain_policies (
		domain TEXT PRIMARY KEY,
		action TEXT NOT NULL,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS cli_tool_state (
		tool_id TEXT PRIMARY KEY,
		last_configured_at INTEGER NOT NULL,
		config_path TEXT,
		endpoint TEXT
	);

	CREATE TABLE IF NOT EXISTS a2a_tasks (
		id TEXT PRIMARY KEY,
		skill TEXT,
		state TEXT NOT NULL,
		input TEXT,
		result TEXT,
		error TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS combos (
		name TEXT PRIMARY KEY,
		strategy TEXT NOT NULL DEFAULT 'failover',
		targets_json TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS key_value (
		namespace TEXT NOT NULL,
		key TEXT NOT NULL,
		value TEXT NOT NULL,
		PRIMARY KEY (namespace, key)
	);

	CREATE TABLE IF NOT EXISTS webhook_deliveries (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		webhook_id TEXT NOT NULL,
		event_type TEXT NOT NULL,
		status TEXT NOT NULL,
		http_status INTEGER,
		latency_ms INTEGER,
		error TEXT,
		payload_snapshot TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_webhook_created ON webhook_deliveries(webhook_id, created_at DESC);
	`

	if _, err := db.Exec(schema); err != nil {
		return nil, fmt.Errorf("failed to execute schema: %w", err)
	}

	// Dynamic column migrations for existing SQLite databases
	_, _ = db.Exec(`ALTER TABLE call_logs ADD COLUMN endpoint TEXT DEFAULT '/v1/chat/completions';`)
	_, _ = db.Exec(`ALTER TABLE call_logs ADD COLUMN prompt_preview TEXT DEFAULT '';`)
	_, _ = db.Exec(`ALTER TABLE call_logs ADD COLUMN response_preview TEXT DEFAULT '';`)

	store := &DBStore{db: db}
	globalDB.Store(store)
	return store, nil
}

// GetDB returns the initialized global DB store.
func GetDB() *DBStore {
	if store := globalDB.Load(); store != nil {
		return store
	}
	initDBMu.Lock()
	defer initDBMu.Unlock()
	if store := globalDB.Load(); store != nil {
		return store
	}
	store, _ := InitDB("")
	return store
}

// SaveCombo saves or updates a combo in SQLite with ACID guarantee.
func (s *DBStore) SaveCombo(name, strategy, targetsJSON string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	query := `
	INSERT INTO combos (name, strategy, targets_json, updated_at)
	VALUES (?, ?, ?, CURRENT_TIMESTAMP)
	ON CONFLICT(name) DO UPDATE SET
		strategy = excluded.strategy,
		targets_json = excluded.targets_json,
		updated_at = CURRENT_TIMESTAMP
	`
	_, err := s.db.Exec(query, name, strategy, targetsJSON)
	return err
}

// GetCombos loads all combos from SQLite.
func (s *DBStore) GetCombos() ([]map[string]interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query(`SELECT name, strategy, targets_json FROM combos ORDER BY name ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []map[string]interface{}
	for rows.Next() {
		var name, strategy, targetsJSON string
		if err := rows.Scan(&name, &strategy, &targetsJSON); err != nil {
			continue
		}
		var targets interface{}
		_ = json.Unmarshal([]byte(targetsJSON), &targets)
		result = append(result, map[string]interface{}{
			"name":     name,
			"strategy": strategy,
			"targets":  targets,
		})
	}
	return result, nil
}

// DeleteCombo removes a combo by name from SQLite.
func (s *DBStore) DeleteCombo(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec(`DELETE FROM combos WHERE name = ?`, name)
	return err
}
func (s *DBStore) RecordCall(log CallLog) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	endpoint := log.Endpoint
	if endpoint == "" {
		endpoint = "/v1/chat/completions"
	}

	query := `
	INSERT INTO call_logs (id, timestamp, model, provider, account_id, input_tokens, output_tokens, cost_usd, latency_ms, status_code, error_message, endpoint, prompt_preview, response_preview)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := s.db.Exec(query,
		log.ID,
		log.Timestamp.Format(time.RFC3339),
		log.Model,
		log.Provider,
		log.AccountID,
		log.InputTokens,
		log.OutputTokens,
		log.CostUSD,
		log.LatencyMs,
		log.StatusCode,
		log.ErrorMessage,
		endpoint,
		log.PromptPreview,
		log.ResponsePreview,
	)
	return err
}

// QueryRecentLogs retrieves recent call logs with pagination.
func (s *DBStore) QueryRecentLogs(limit, offset int) ([]CallLog, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 {
		limit = 50
	}
	query := `
	SELECT id, timestamp, model, provider, account_id, input_tokens, output_tokens, cost_usd, latency_ms, status_code, error_message, COALESCE(endpoint, '/v1/chat/completions'), COALESCE(prompt_preview, ''), COALESCE(response_preview, '')
	FROM call_logs
	ORDER BY timestamp DESC
	LIMIT ? OFFSET ?
	`
	rows, err := s.db.Query(query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []CallLog
	for rows.Next() {
		var l CallLog
		var tsStr string
		var errMsg sql.NullString
		var accID sql.NullString

		if err := rows.Scan(
			&l.ID,
			&tsStr,
			&l.Model,
			&l.Provider,
			&accID,
			&l.InputTokens,
			&l.OutputTokens,
			&l.CostUSD,
			&l.LatencyMs,
			&l.StatusCode,
			&errMsg,
			&l.Endpoint,
			&l.PromptPreview,
			&l.ResponsePreview,
		); err != nil {
			return nil, err
		}

		l.Timestamp, _ = time.Parse(time.RFC3339, tsStr)
		if errMsg.Valid {
			l.ErrorMessage = errMsg.String
		}
		if accID.Valid {
			l.AccountID = accID.String
		}
		logs = append(logs, l)
	}

	return logs, nil
}

// WindowUsage holds aggregated consumption statistics for a time window.
type WindowUsage struct {
	Requests     int
	InputTokens  int
	OutputTokens int
	TotalTokens  int
	OldestCall   time.Time
	AvgLatencyMs int64
}

// GetAccountWindowUsage returns rolling usage stats for an account or provider within window.
func (s *DBStore) GetAccountWindowUsage(provider, accountID string, window time.Duration) (WindowUsage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var u WindowUsage
	if s.db == nil {
		return u, fmt.Errorf("db not initialized")
	}

	since := time.Now().Add(-window).Format(time.RFC3339)
	var query string
	var rows *sql.Rows
	var err error

	if accountID != "" {
		query = `
			SELECT COUNT(*), COALESCE(SUM(input_tokens), 0), COALESCE(SUM(output_tokens), 0),
			       COALESCE(MIN(timestamp), ''), COALESCE(AVG(latency_ms), 0)
			FROM call_logs
			WHERE timestamp >= ? AND (account_id = ? OR (account_id = '' AND (provider = ? OR ? = '')))
		`
		rows, err = s.db.Query(query, since, accountID, provider, provider)
	} else if provider != "" {
		query = `
			SELECT COUNT(*), COALESCE(SUM(input_tokens), 0), COALESCE(SUM(output_tokens), 0),
			       COALESCE(MIN(timestamp), ''), COALESCE(AVG(latency_ms), 0)
			FROM call_logs
			WHERE timestamp >= ? AND (provider = ? OR provider = '')
		`
		rows, err = s.db.Query(query, since, provider)
	} else {
		query = `
			SELECT COUNT(*), COALESCE(SUM(input_tokens), 0), COALESCE(SUM(output_tokens), 0),
			       COALESCE(MIN(timestamp), ''), COALESCE(AVG(latency_ms), 0)
			FROM call_logs
			WHERE timestamp >= ?
		`
		rows, err = s.db.Query(query, since)
	}

	if err != nil {
		return u, err
	}
	defer rows.Close()

	if rows.Next() {
		var minTs string
		var avgLat float64
		if err := rows.Scan(&u.Requests, &u.InputTokens, &u.OutputTokens, &minTs, &avgLat); err == nil {
			u.TotalTokens = u.InputTokens + u.OutputTokens
			u.AvgLatencyMs = int64(avgLat)
			if minTs != "" {
				u.OldestCall, _ = time.Parse(time.RFC3339, minTs)
			}
		}
	}

	return u, nil
}

// CliToolState records configuration state for a developer CLI tool.
type CliToolState struct {
	ToolID           string `json:"tool_id"`
	LastConfiguredAt int64  `json:"last_configured_at"`
	ConfigPath       string `json:"config_path"`
	Endpoint         string `json:"endpoint"`
}

// SaveCliToolState persists or updates the CLI tool's last configured timestamp and metadata.
func (s *DBStore) SaveCliToolState(toolID string, lastConfiguredAt int64, configPath, endpoint string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	query := `
	INSERT INTO cli_tool_state (tool_id, last_configured_at, config_path, endpoint)
	VALUES (?, ?, ?, ?)
	ON CONFLICT(tool_id) DO UPDATE SET
		last_configured_at = excluded.last_configured_at,
		config_path = excluded.config_path,
		endpoint = excluded.endpoint;
	`
	_, err := s.db.Exec(query, toolID, lastConfiguredAt, configPath, endpoint)
	return err
}

// GetAllCliToolLastConfigured returns a map of toolID -> lastConfiguredAt Unix timestamp (ms or sec).
func (s *DBStore) GetAllCliToolLastConfigured() (map[string]int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rows, err := s.db.Query("SELECT tool_id, last_configured_at FROM cli_tool_state")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	res := make(map[string]int64)
	for rows.Next() {
		var toolID string
		var lastConfigured int64
		if err := rows.Scan(&toolID, &lastConfigured); err == nil {
			res[toolID] = lastConfigured
		}
	}
	return res, nil
}

// GetCliToolState retrieves the persisted state for a specific tool.
func (s *DBStore) GetCliToolState(toolID string) (*CliToolState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var state CliToolState
	query := "SELECT tool_id, last_configured_at, config_path, endpoint FROM cli_tool_state WHERE tool_id = ?"
	err := s.db.QueryRow(query, toolID).Scan(&state.ToolID, &state.LastConfiguredAt, &state.ConfigPath, &state.Endpoint)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &state, nil
}

// DeleteCliToolState removes the configured state entry for a tool.
func (s *DBStore) DeleteCliToolState(toolID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec("DELETE FROM cli_tool_state WHERE tool_id = ?", toolID)
	return err
}

// A2ATask represents an Agent-to-Agent task execution unit.
type A2ATask struct {
	ID        string `json:"id"`
	Skill     string `json:"skill"`
	State     string `json:"state"` // submitted, working, completed, failed, cancelled
	Input     string `json:"input"`
	Result    string `json:"result,omitempty"`
	Error     string `json:"error,omitempty"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

// UpsertA2ATask persists or updates an A2A task in SQLite.
func (s *DBStore) UpsertA2ATask(task *A2ATask) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	query := `
	INSERT INTO a2a_tasks (id, skill, state, input, result, error, updated_at)
	VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
	ON CONFLICT(id) DO UPDATE SET
		skill = excluded.skill,
		state = excluded.state,
		input = excluded.input,
		result = excluded.result,
		error = excluded.error,
		updated_at = CURRENT_TIMESTAMP;
	`
	_, err := s.db.Exec(query, task.ID, task.Skill, task.State, task.Input, task.Result, task.Error)
	return err
}

// GetA2ATask retrieves a single A2A task by ID.
func (s *DBStore) GetA2ATask(id string) (*A2ATask, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var t A2ATask
	query := "SELECT id, skill, state, input, COALESCE(result, ''), COALESCE(error, ''), created_at, updated_at FROM a2a_tasks WHERE id = ?"
	err := s.db.QueryRow(query, id).Scan(&t.ID, &t.Skill, &t.State, &t.Input, &t.Result, &t.Error, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &t, nil
}

// ListA2ATasks returns matching tasks with pagination and total count.
func (s *DBStore) ListA2ATasks(state, skill string, limit, offset int) ([]A2ATask, int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	where := "WHERE 1=1"
	var args []interface{}
	if state != "" {
		where += " AND state = ?"
		args = append(args, state)
	}
	if skill != "" {
		where += " AND skill = ?"
		args = append(args, skill)
	}

	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM a2a_tasks %s", where)
	if err := s.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	if limit <= 0 {
		limit = 50
	}
	listQuery := fmt.Sprintf("SELECT id, skill, state, input, COALESCE(result, ''), COALESCE(error, ''), created_at, updated_at FROM a2a_tasks %s ORDER BY created_at DESC LIMIT ? OFFSET ?", where)
	args = append(args, limit, offset)

	rows, err := s.db.Query(listQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var tasks []A2ATask
	for rows.Next() {
		var t A2ATask
		if err := rows.Scan(&t.ID, &t.Skill, &t.State, &t.Input, &t.Result, &t.Error, &t.CreatedAt, &t.UpdatedAt); err == nil {
			tasks = append(tasks, t)
		}
	}
	return tasks, total, nil
}

// GetA2ATaskStats returns the count of tasks grouped by state.
func (s *DBStore) GetA2ATaskStats() (map[string]int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rows, err := s.db.Query("SELECT state, COUNT(*) FROM a2a_tasks GROUP BY state")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := map[string]int{
		"submitted": 0,
		"working":   0,
		"completed": 0,
		"failed":    0,
		"cancelled": 0,
	}
	for rows.Next() {
		var state string
		var count int
		if err := rows.Scan(&state, &count); err == nil {
			stats[state] = count
		}
	}
	return stats, nil
}

// Checkpoint executes SQLite WAL checkpointing (e.g. TRUNCATE, PASSIVE, FULL, RESTART).
func (s *DBStore) Checkpoint(mode string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db == nil {
		return fmt.Errorf("database not initialized")
	}
	if mode == "" {
		mode = "TRUNCATE"
	}
	_, err := s.db.Exec(fmt.Sprintf("PRAGMA wal_checkpoint(%s);", mode))
	return err
}

// Vacuum executes full SQLite database vacuuming to reclaim fragmented pages.
func (s *DBStore) Vacuum() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db == nil {
		return fmt.Errorf("database not initialized")
	}
	_, err := s.db.Exec("VACUUM;")
	return err
}

// StartWALMaintenance starts a recurring background worker for WAL checkpointing and vacuuming.
func (s *DBStore) StartWALMaintenance(interval time.Duration) (stop func()) {
	if interval <= 0 {
		interval = 15 * time.Minute
	}
	ticker := time.NewTicker(interval)
	stopChan := make(chan struct{})
	var once sync.Once
	go func() {
		for {
			select {
			case <-ticker.C:
				_ = s.Checkpoint("TRUNCATE")
				_ = s.Vacuum()
			case <-stopChan:
				ticker.Stop()
				return
			}
		}
	}()
	return func() {
		once.Do(func() {
			close(stopChan)
		})
	}
}

// SetKeyValue persists an entry into the key_value table.
func (s *DBStore) SetKeyValue(namespace, key, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db == nil {
		return fmt.Errorf("database not initialized")
	}
	query := `
	INSERT INTO key_value (namespace, key, value)
	VALUES (?, ?, ?)
	ON CONFLICT(namespace, key) DO UPDATE SET value = excluded.value
	`
	_, err := s.db.Exec(query, namespace, key, value)
	return err
}

// GetKeyValue retrieves an entry from the key_value table.
func (s *DBStore) GetKeyValue(namespace, key string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.db == nil {
		return "", fmt.Errorf("database not initialized")
	}
	var val string
	err := s.db.QueryRow(`SELECT value FROM key_value WHERE namespace = ? AND key = ?`, namespace, key).Scan(&val)
	if err != nil {
		return "", err
	}
	return val, nil
}

// SetVirtualKeyUsage records accumulated token usage for a virtual key.
func (s *DBStore) SetVirtualKeyUsage(keyID string, tokens int) error {
	return s.SetKeyValue("virtual_keys", keyID, fmt.Sprintf("%d", tokens))
}

// GetVirtualKeyUsage returns accumulated token usage for a virtual key.
func (s *DBStore) GetVirtualKeyUsage(keyID string) (int, error) {
	val, err := s.GetKeyValue("virtual_keys", keyID)
	if err != nil {
		return 0, err
	}
	var tokens int
	fmt.Sscanf(val, "%d", &tokens)
	return tokens, nil
}

// RecordWebhookDelivery stores a delivery attempt into webhook_deliveries table.
func (s *DBStore) RecordWebhookDelivery(del *WebhookDelivery) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db == nil {
		return fmt.Errorf("database not initialized")
	}
	query := `
	INSERT INTO webhook_deliveries (webhook_id, event_type, status, http_status, latency_ms, error, payload_snapshot, created_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	crStr := time.Now().UTC().Format(time.RFC3339)
	if !del.CreatedAt.IsZero() {
		crStr = del.CreatedAt.Format(time.RFC3339)
	}
	res, err := s.db.Exec(query, del.WebhookID, del.EventType, del.Status, del.HTTPStatus, del.LatencyMs, del.Error, del.PayloadSnapshot, crStr)
	if err != nil {
		return err
	}
	if id, err := res.LastInsertId(); err == nil {
		del.ID = id
	}
	return nil
}

// GetWebhookDeliveries queries historical deliveries for a webhook ID.
func (s *DBStore) GetWebhookDeliveries(webhookID string, limit int) ([]WebhookDelivery, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	if limit <= 0 {
		limit = 50
	}
	query := `
	SELECT id, webhook_id, event_type, status, http_status, latency_ms, error, payload_snapshot, created_at
	FROM webhook_deliveries
	WHERE webhook_id = ?
	ORDER BY created_at DESC, id DESC
	LIMIT ?
	`
	rows, err := s.db.Query(query, webhookID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var deliveries []WebhookDelivery
	for rows.Next() {
		var d WebhookDelivery
		var httpStatus *int
		var latencyMs *int64
		var errStr, payloadSnap *string
		var crStr string
		if err := rows.Scan(&d.ID, &d.WebhookID, &d.EventType, &d.Status, &httpStatus, &latencyMs, &errStr, &payloadSnap, &crStr); err != nil {
			continue
		}
		d.HTTPStatus = httpStatus
		d.LatencyMs = latencyMs
		if errStr != nil {
			d.Error = *errStr
		}
		if payloadSnap != nil {
			d.PayloadSnapshot = *payloadSnap
		}
		d.CreatedAt, _ = time.Parse(time.RFC3339, crStr)
		deliveries = append(deliveries, d)
	}
	return deliveries, nil
}

// Close closes the database connection, ensuring WAL is cleanly flushed and truncated.
func (s *DBStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	globalDB.CompareAndSwap(s, nil)
	if s.db != nil {
		_, _ = s.db.Exec("PRAGMA wal_checkpoint(TRUNCATE);")
		err := s.db.Close()
		s.db = nil
		return err
	}
	return nil
}
