package infra

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"time"

	_ "modernc.org/sqlite"
)

// CallLogRecord captures a query record from the SQLite call_logs table.
type CallLogRecord struct {
	ID              string
	Timestamp       time.Time
	Model           string
	Provider        string
	AccountID       string
	InputTokens     int
	OutputTokens    int
	CostUSD         float64
	LatencyMs       int64
	StatusCode      int
	ErrorMessage    string
	Endpoint        string
	PromptPreview   string
	ResponsePreview string
}

// WebhookDeliveryRecord captures a delivery attempt from SQLite webhook_deliveries.
type WebhookDeliveryRecord struct {
	ID          string
	WebhookID   string
	Event       string
	URL         string
	StatusCode  int
	DurationMs  int64
	Attempt     int
	Error       string
	DeliveredAt time.Time
}

// ComboRecord captures a configured combo rule in SQLite combos table.
type ComboRecord struct {
	Name        string
	Strategy    string
	TargetsJSON string
}

// SQLiteVerifier opens the daemon's SQLite database directly in read-only mode to verify persistence.
type SQLiteVerifier struct {
	dbPath string
	db     *sql.DB
}

// NewSQLiteVerifier opens an existing GoRoute SQLite database file.
func NewSQLiteVerifier(dbPath string) (*SQLiteVerifier, error) {
	if _, err := os.Stat(dbPath); err != nil {
		return nil, fmt.Errorf("database file does not exist at %s: %w", dbPath, err)
	}

	// Connect to SQLite with WAL concurrency & read-only enforcement
	dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=query_only(ON)", dbPath)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite verifier: %w", err)
	}

	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("sqlite ping failed: %w", err)
	}

	return &SQLiteVerifier{
		dbPath: dbPath,
		db:     db,
	}, nil
}

// Close closes the underlying SQLite database connection.
func (v *SQLiteVerifier) Close() error {
	if v.db != nil {
		return v.db.Close()
	}
	return nil
}

// DBPath returns the verified database file path.
func (v *SQLiteVerifier) DBPath() string {
	return v.dbPath
}

// TableExists checks if a specific table exists in sqlite_master.
func (v *SQLiteVerifier) TableExists(tableName string) (bool, error) {
	query := `SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`
	var count int
	if err := v.db.QueryRow(query, tableName).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetCallLogCount returns total number of records in call_logs.
func (v *SQLiteVerifier) GetCallLogCount() (int, error) {
	exists, err := v.TableExists("call_logs")
	if err != nil || !exists {
		return 0, err
	}
	var count int
	err = v.db.QueryRow(`SELECT COUNT(*) FROM call_logs`).Scan(&count)
	return count, err
}

// GetCallLogs retrieves the most recent call logs up to limit.
func (v *SQLiteVerifier) GetCallLogs(limit int) ([]CallLogRecord, error) {
	exists, err := v.TableExists("call_logs")
	if err != nil || !exists {
		return nil, err
	}

	query := `
	SELECT id, timestamp, model, provider, coalesce(account_id, ''),
	       input_tokens, output_tokens, cost_usd, latency_ms, status_code,
	       coalesce(error_message, ''), coalesce(endpoint, ''),
	       coalesce(prompt_preview, ''), coalesce(response_preview, '')
	FROM call_logs
	ORDER BY timestamp DESC
	LIMIT ?
	`
	rows, err := v.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []CallLogRecord
	for rows.Next() {
		var r CallLogRecord
		var rawTime interface{}
		err := rows.Scan(
			&r.ID, &rawTime, &r.Model, &r.Provider, &r.AccountID,
			&r.InputTokens, &r.OutputTokens, &r.CostUSD, &r.LatencyMs, &r.StatusCode,
			&r.ErrorMessage, &r.Endpoint, &r.PromptPreview, &r.ResponsePreview,
		)
		if err != nil {
			return nil, err
		}
		r.Timestamp = parseDBTime(rawTime)
		records = append(records, r)
	}
	return records, rows.Err()
}

// GetLastCallLog returns the single most recent call log record or nil.
func (v *SQLiteVerifier) GetLastCallLog() (*CallLogRecord, error) {
	logs, err := v.GetCallLogs(1)
	if err != nil {
		return nil, err
	}
	if len(logs) == 0 {
		return nil, nil
	}
	return &logs[0], nil
}

// GetCallLogsByModel filters call logs by exact model match.
func (v *SQLiteVerifier) GetCallLogsByModel(model string) ([]CallLogRecord, error) {
	query := `
	SELECT id, timestamp, model, provider, coalesce(account_id, ''),
	       input_tokens, output_tokens, cost_usd, latency_ms, status_code,
	       coalesce(error_message, ''), coalesce(endpoint, ''),
	       coalesce(prompt_preview, ''), coalesce(response_preview, '')
	FROM call_logs
	WHERE model = ?
	ORDER BY timestamp DESC
	`
	rows, err := v.db.Query(query, model)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []CallLogRecord
	for rows.Next() {
		var r CallLogRecord
		var rawTime interface{}
		if err := rows.Scan(
			&r.ID, &rawTime, &r.Model, &r.Provider, &r.AccountID,
			&r.InputTokens, &r.OutputTokens, &r.CostUSD, &r.LatencyMs, &r.StatusCode,
			&r.ErrorMessage, &r.Endpoint, &r.PromptPreview, &r.ResponsePreview,
		); err != nil {
			return nil, err
		}
		r.Timestamp = parseDBTime(rawTime)
		records = append(records, r)
	}
	return records, rows.Err()
}

// GetCallLogsByProvider filters call logs by provider identifier.
func (v *SQLiteVerifier) GetCallLogsByProvider(provider string) ([]CallLogRecord, error) {
	query := `
	SELECT id, timestamp, model, provider, coalesce(account_id, ''),
	       input_tokens, output_tokens, cost_usd, latency_ms, status_code,
	       coalesce(error_message, ''), coalesce(endpoint, ''),
	       coalesce(prompt_preview, ''), coalesce(response_preview, '')
	FROM call_logs
	WHERE provider = ?
	ORDER BY timestamp DESC
	`
	rows, err := v.db.Query(query, provider)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []CallLogRecord
	for rows.Next() {
		var r CallLogRecord
		var rawTime interface{}
		if err := rows.Scan(
			&r.ID, &rawTime, &r.Model, &r.Provider, &r.AccountID,
			&r.InputTokens, &r.OutputTokens, &r.CostUSD, &r.LatencyMs, &r.StatusCode,
			&r.ErrorMessage, &r.Endpoint, &r.PromptPreview, &r.ResponsePreview,
		); err != nil {
			return nil, err
		}
		r.Timestamp = parseDBTime(rawTime)
		records = append(records, r)
	}
	return records, rows.Err()
}

// GetWebhookDeliveries inspects webhook delivery attempts in SQLite.
func (v *SQLiteVerifier) GetWebhookDeliveries(limit int) ([]WebhookDeliveryRecord, error) {
	exists, err := v.TableExists("webhook_deliveries")
	if err != nil || !exists {
		return nil, nil
	}

	query := `
	SELECT id, coalesce(webhook_id, ''), coalesce(event, ''), coalesce(url, ''),
	       coalesce(status_code, 0), coalesce(duration_ms, 0), coalesce(attempt, 1),
	       coalesce(error, ''), coalesce(delivered_at, CURRENT_TIMESTAMP)
	FROM webhook_deliveries
	ORDER BY delivered_at DESC
	LIMIT ?
	`
	rows, err := v.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []WebhookDeliveryRecord
	for rows.Next() {
		var r WebhookDeliveryRecord
		var rawTime interface{}
		if err := rows.Scan(
			&r.ID, &r.WebhookID, &r.Event, &r.URL,
			&r.StatusCode, &r.DurationMs, &r.Attempt,
			&r.Error, &rawTime,
		); err != nil {
			return nil, err
		}
		r.DeliveredAt = parseDBTime(rawTime)
		records = append(records, r)
	}
	return records, rows.Err()
}

// GetKeyValue reads a string value from key_value table by namespace and key.
func (v *SQLiteVerifier) GetKeyValue(namespace, key string) (string, error) {
	exists, err := v.TableExists("key_value")
	if err != nil || !exists {
		return "", fmt.Errorf("key_value table does not exist")
	}

	var val string
	err = v.db.QueryRow(`SELECT value FROM key_value WHERE namespace = ? AND key = ?`, namespace, key).Scan(&val)
	if err != nil {
		return "", err
	}
	return val, nil
}

// GetAllKeyValues reads all key-value pairs in a specific namespace.
func (v *SQLiteVerifier) GetAllKeyValues(namespace string) (map[string]string, error) {
	exists, err := v.TableExists("key_value")
	if err != nil || !exists {
		return make(map[string]string), nil
	}

	rows, err := v.db.Query(`SELECT key, value FROM key_value WHERE namespace = ?`, namespace)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make(map[string]string)
	for rows.Next() {
		var k, val string
		if err := rows.Scan(&k, &val); err != nil {
			continue
		}
		results[k] = val
	}
	return results, rows.Err()
}

// GetSetting reads a specific setting from namespace='settings'.
func (v *SQLiteVerifier) GetSetting(key string) (string, error) {
	return v.GetKeyValue("settings", key)
}

// GetCombos loads all combo definitions from the combos table.
func (v *SQLiteVerifier) GetCombos() ([]ComboRecord, error) {
	exists, err := v.TableExists("combos")
	if err != nil || !exists {
		return nil, nil
	}

	rows, err := v.db.Query(`SELECT name, strategy, targets_json FROM combos ORDER BY name ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []ComboRecord
	for rows.Next() {
		var r ComboRecord
		if err := rows.Scan(&r.Name, &r.Strategy, &r.TargetsJSON); err != nil {
			continue
		}
		records = append(records, r)
	}
	return records, rows.Err()
}

// GetWALSize returns the current byte size of the SQLite WAL journal file if present.
func (v *SQLiteVerifier) GetWALSize() (int64, error) {
	walPath := v.dbPath + "-wal"
	info, err := os.Stat(walPath)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	return info.Size(), nil
}

// RawQuery allows executing arbitrary SELECT queries for adversarial verification.
func (v *SQLiteVerifier) RawQuery(query string, args ...interface{}) (*sql.Rows, error) {
	return v.db.Query(query, args...)
}

func parseDBTime(raw interface{}) time.Time {
	switch v := raw.(type) {
	case time.Time:
		return v
	case string:
		for _, layout := range []string{
			time.RFC3339Nano,
			time.RFC3339,
			"2006-01-02 15:04:05.999999999-07:00",
			"2006-01-02 15:04:05",
			"2006-01-02T15:04:05Z",
		} {
			if t, err := time.Parse(layout, v); err == nil {
				return t
			}
		}
	case []byte:
		return parseDBTime(string(v))
	}
	return time.Time{}
}

// ParseTargetsJSON unmarshals targets_json column into slice of map.
func (c *ComboRecord) ParseTargets() ([]map[string]string, error) {
	var targets []map[string]string
	if err := json.Unmarshal([]byte(c.TargetsJSON), &targets); err != nil {
		return nil, err
	}
	return targets, nil
}
