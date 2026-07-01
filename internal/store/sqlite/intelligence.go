package sqlite

import (
	"context"
	"time"

	"agent-memory-mcp/internal/core"
)

func (s *Store) GlobalSearch(ctx context.Context, query string, kinds []string, limit int) ([]core.Memory, error) {
	// Search with repoID = nil implies cross-repo
	return s.Search(ctx, query, nil, kinds, limit)
}

func (s *Store) GetPatternReport(ctx context.Context, repoID *string) (map[string]any, error) {
	// Minimal pattern report stub matching Python dictionary shape
	report := map[string]any{
		"repo_id": "global",
		"total_memories": 0,
		"health_score": map[string]any{
			"score": 100,
			"grade": "A",
			"reasons": []string{"Placeholder for Go Port"},
		},
		"breakdown": map[string]int{
			"failures": 0,
			"decisions": 0,
			"attempts": 0,
		},
		"recent_unresolved_signatures": []string{},
	}
	if repoID != nil {
		report["repo_id"] = *repoID
	}
	return report, nil
}

func (s *Store) FailureHotspots(ctx context.Context, limit int) ([]map[string]any, error) {
	rows, err := s.memoryDB.QueryContext(ctx, `
		SELECT repo_id, SUM(count) as unresolved 
		FROM failure_signatures 
		WHERE resolved = 0 
		GROUP BY repo_id 
		ORDER BY unresolved DESC 
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hotspots []map[string]any
	for rows.Next() {
		var repoID string
		var count int
		if err := rows.Scan(&repoID, &count); err == nil {
			hotspots = append(hotspots, map[string]any{
				"repo_id": repoID,
				"unresolved_failures": count,
				"path": "/placeholder", // could fetch from repos table
			})
		}
	}
	return hotspots, nil
}

func (s *Store) GetRelatedMemories(ctx context.Context, memoryID string, limit int) ([]core.Memory, error) {
	// Placeholder: just returns latest memories to satisfy API
	return s.ListMemories(ctx, nil, nil, limit)
}

func (s *Store) RecordFeedback(ctx context.Context, memoryID string, useful bool, contextStr string) error {
	usefulInt := 0
	if useful {
		usefulInt = 1
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.memoryDB.ExecContext(ctx, `
		UPDATE memory_access_log 
		SET was_useful = ?, context = ?
		WHERE memory_id = ? AND was_useful IS NULL
	`, usefulInt, contextStr, memoryID)
	if err != nil {
		return err
	}
	// The Python version inserts a new record if none exists.
	// For simplicity, we just insert a feedback log entry.
	s.memoryDB.ExecContext(ctx, `
		INSERT INTO memory_access_log (id, memory_id, access_type, query_text, was_useful, context, created_at)
		VALUES (lower(hex(randomblob(16))), ?, 'feedback', '', ?, ?, ?)
	`, memoryID, usefulInt, contextStr, now)
	return nil
}
