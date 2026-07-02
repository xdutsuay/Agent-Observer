package sqlite

import (
	"context"
	"time"

	"agent-memory-mcp/internal/core"
)
func (s *Store) ListUsageSessions(ctx context.Context, limit int) ([]core.UsageSession, error) {
	query := `
		SELECT s.id, s.client_name, s.client_version, s.host_ide, s.transport, s.connected_at, s.last_seen_at,
		       (SELECT COUNT(*) FROM usage_interactions WHERE session_id = s.id) as call_count,
		       (SELECT created_at FROM usage_interactions WHERE session_id = s.id ORDER BY created_at DESC LIMIT 1) as last_call
		FROM usage_sessions s
		ORDER BY s.last_seen_at DESC
		LIMIT ?
	`
	rows, err := s.usageDB.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []core.UsageSession
	for rows.Next() {
		var u core.UsageSession
		var lastCall *string
		if err := rows.Scan(&u.ID, &u.ClientName, &u.ClientVersion, &u.HostIDE, &u.Transport, &u.ConnectedAt, &u.LastSeenAt, &u.CallCount, &lastCall); err == nil {
			if lastCall != nil {
				u.LastCall = *lastCall
			}
			res = append(res, u)
		}
	}
	return res, nil
}

func (s *Store) UsageSummary(ctx context.Context) (core.UsageSummary, error) {
	var sum core.UsageSummary
	// total interactions
	s.usageDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM usage_interactions").Scan(&sum.TotalInteractions)
	
	// last 24h
	yesterday := time.Now().Add(-24 * time.Hour).UTC().Format(time.RFC3339)
	s.usageDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM usage_interactions WHERE created_at >= ?", yesterday).Scan(&sum.Last24H)
	
	// reads vs writes
	// basic heuristic: write methods start with memory_, except memory_search, memory_list, etc.
	// let's do a simple count for specific methods
	s.usageDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM usage_interactions WHERE method IN ('memory_search', 'memory_list', 'global_search', 'get_related_memories', 'find_similar_failures', 'failure_hotspots', 'get_pattern_report')").Scan(&sum.Reads)
	
	s.usageDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM usage_interactions WHERE method IN ('memory_create', 'memory_feedback', 'memory_mark_resolved', 'memory_forget')").Scan(&sum.Writes)

	// by method
	rows, err := s.usageDB.QueryContext(ctx, "SELECT method, COUNT(*) FROM usage_interactions GROUP BY method ORDER BY COUNT(*) DESC")
	if err == nil {
		for rows.Next() {
			var n core.CountByName
			if err := rows.Scan(&n.Name, &n.Count); err == nil {
				sum.ByMethod = append(sum.ByMethod, n)
			}
		}
		rows.Close()
	}

	// by host IDE
	rows, err = s.usageDB.QueryContext(ctx, "SELECT host_ide, COUNT(*) FROM usage_interactions GROUP BY host_ide ORDER BY COUNT(*) DESC")
	if err == nil {
		for rows.Next() {
			var n core.CountByName
			if err := rows.Scan(&n.Name, &n.Count); err == nil {
				sum.ByHostIDE = append(sum.ByHostIDE, n)
			}
		}
		rows.Close()
	}

	// by transport
	rows, err = s.usageDB.QueryContext(ctx, "SELECT transport, COUNT(*) FROM usage_interactions GROUP BY transport ORDER BY COUNT(*) DESC")
	if err == nil {
		for rows.Next() {
			var n core.CountByName
			if err := rows.Scan(&n.Name, &n.Count); err == nil {
				sum.ByTransport = append(sum.ByTransport, n)
			}
		}
		rows.Close()
	}

	// running ides
	fifteenMinsAgo := time.Now().Add(-15 * time.Minute).UTC().Format(time.RFC3339)
	rows, err = s.usageDB.QueryContext(ctx, "SELECT host_ide, COUNT(*) FROM usage_sessions WHERE last_seen_at >= ? GROUP BY host_ide", fifteenMinsAgo)
	if err == nil {
		for rows.Next() {
			var r core.RunningIDE
			if err := rows.Scan(&r.Label, &r.ProcessCount); err == nil {
				sum.RunningIDEs = append(sum.RunningIDEs, r)
			}
		}
		rows.Close()
	}

	if sum.ByMethod == nil { sum.ByMethod = []core.CountByName{} }
	if sum.ByHostIDE == nil { sum.ByHostIDE = []core.CountByName{} }
	if sum.ByTransport == nil { sum.ByTransport = []core.CountByName{} }
	if sum.RunningIDEs == nil { sum.RunningIDEs = []core.RunningIDE{} }

	return sum, nil
}
