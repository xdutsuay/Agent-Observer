package sqlite

import (
	"context"
	"encoding/json"
	"time"

	"agent-memory-mcp/internal/core"
	"github.com/google/uuid"
)

func (s *Store) RecordUsage(ctx context.Context, sessionID, transport, method, clientName, clientVersion, hostIDE, querySummary string, query map[string]any, responsePreview string, durationMS float64, ok bool) error {
	now := time.Now().UTC().Format(time.RFC3339)

	s.usageDB.ExecContext(ctx, `
		INSERT INTO usage_sessions (id, client_name, client_version, host_ide, transport, connected_at, last_seen_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET last_seen_at = excluded.last_seen_at
	`, sessionID, clientName, clientVersion, hostIDE, transport, now, now)

	queryJSON, _ := json.Marshal(query)
	okInt := 0
	if ok { okInt = 1 }

	interactionID := uuid.New().String()
	_, err := s.usageDB.ExecContext(ctx, `
		INSERT INTO usage_interactions (
			id, session_id, transport, method, client_name, client_version, host_ide,
			query_summary, query_json, response_preview, duration_ms, ok, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, interactionID, sessionID, transport, method, clientName, clientVersion, hostIDE,
		querySummary, string(queryJSON), responsePreview, durationMS, okInt, now)

	return err
}

func (s *Store) ListUsageInteractions(ctx context.Context, limit int, hostIDE *string) ([]core.UsageInteraction, error) {
	query := `SELECT id, session_id, transport, method, client_name, client_version, host_ide, query_summary, query_json, response_preview, duration_ms, ok, created_at FROM usage_interactions`
	var args []any
	if hostIDE != nil {
		query += " WHERE host_ide = ?"
		args = append(args, *hostIDE)
	}
	query += " ORDER BY created_at DESC LIMIT ?"
	args = append(args, limit)

	rows, err := s.usageDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	res := []core.UsageInteraction{}
	for rows.Next() {
		var u core.UsageInteraction
		var okInt int
		if err := rows.Scan(&u.ID, &u.SessionID, &u.Transport, &u.Method, &u.ClientName, &u.ClientVersion, &u.HostIDE, &u.QuerySummary, &u.QueryJSON, &u.ResponsePreview, &u.DurationMS, &okInt, &u.CreatedAt); err == nil {
			u.OK = okInt == 1
			res = append(res, u)
		}
	}
	return res, nil
}
