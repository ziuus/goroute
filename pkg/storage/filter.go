package storage

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// CallLogFilter defines server-side search and pagination criteria for call logs.
type CallLogFilter struct {
	Status        string `json:"status,omitempty"`
	Model         string `json:"model,omitempty"`
	Provider      string `json:"provider,omitempty"`
	Account       string `json:"account,omitempty"`
	Search        string `json:"search,omitempty"`
	ExcludeTests  bool   `json:"exclude_tests,omitempty"`
	Limit         int    `json:"limit,omitempty"`
	Offset        int    `json:"offset,omitempty"`
}

// QueryFilteredLogs executes a SQL query on call_logs matching the exact OmniRoute filter semantics.
func (s *DBStore) QueryFilteredLogs(f CallLogFilter) ([]CallLog, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var whereClauses []string
	var args []any

	if f.Status == "error" {
		whereClauses = append(whereClauses, "(status_code >= 400 OR error_message IS NOT NULL AND error_message != '')")
	} else if f.Status == "ok" {
		whereClauses = append(whereClauses, "status_code >= 200 AND status_code < 300")
	} else if f.Status != "" {
		whereClauses = append(whereClauses, "status_code = ?")
		args = append(args, f.Status)
	}

	if f.Model != "" {
		whereClauses = append(whereClauses, "model LIKE ?")
		args = append(args, "%"+f.Model+"%")
	}

	if f.Provider != "" {
		whereClauses = append(whereClauses, "provider LIKE ?")
		args = append(args, "%"+f.Provider+"%")
	}

	if f.Account != "" {
		whereClauses = append(whereClauses, "account_id LIKE ?")
		args = append(args, "%"+f.Account+"%")
	}

	if f.ExcludeTests {
		whereClauses = append(whereClauses, "model NOT LIKE '%test%' AND model NOT LIKE '%probe%'")
	}

	if f.Search != "" {
		whereClauses = append(whereClauses, "(model LIKE ? OR provider LIKE ? OR error_message LIKE ? OR id LIKE ?)")
		searchTerm := "%" + f.Search + "%"
		args = append(args, searchTerm, searchTerm, searchTerm, searchTerm)
	}

	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	limit := f.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	offset := f.Offset
	if offset < 0 {
		offset = 0
	}

	query := fmt.Sprintf(`
		SELECT id, timestamp, model, provider, account_id, input_tokens, output_tokens, cost_usd, latency_ms, status_code, error_message, COALESCE(endpoint, '/v1/chat/completions'), COALESCE(prompt_preview, ''), COALESCE(response_preview, '')
		FROM call_logs
		%s
		ORDER BY timestamp DESC
		LIMIT ? OFFSET ?
	`, whereSQL)

	args = append(args, limit, offset)

	rows, err := s.db.Query(query, args...)
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
