package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"agent-memory-mcp/internal/core"
)

func (s *Store) GetRepoContext(ctx context.Context, repoID string) (core.RepoContext, error) {
	failures, _ := s.ListMemories(ctx, &repoID, strPtr("failure"), 10)
	decisions, _ := s.ListMemories(ctx, &repoID, strPtr("decision"), 10)
	attempts, _ := s.ListMemories(ctx, &repoID, strPtr("attempt"), 15)
	facts, _ := s.ListMemories(ctx, &repoID, strPtr("fact"), 10)

	format := func(mems []core.Memory) string {
		if len(mems) == 0 {
			return "(none)"
		}
		var parts []string
		for _, m := range mems {
			c := m.Content
			if len(c) > 500 {
				c = c[:500] + "..."
			}
			parts = append(parts, fmt.Sprintf("- [%s] %s", m.CreatedAt, c))
		}
		return strings.Join(parts, "\n")
	}

	sigs, _ := s.GetFailureSignatures(ctx, repoID)
	var unresolved []map[string]any
	for _, sig := range sigs {
		if res, ok := sig["resolved"].(bool); ok && !res {
			unresolved = append(unresolved, sig)
			if len(unresolved) >= 20 {
				break
			}
		}
	}
	sigBytes, _ := json.MarshalIndent(unresolved, "", "  ")

	return core.RepoContext{
		Failures:          format(failures),
		Decisions:         format(decisions),
		Attempts:          format(attempts),
		Facts:             format(facts),
		FailureSignatures: string(sigBytes),
	}, nil
}

func (s *Store) GetFailureSignatures(ctx context.Context, repoID string) ([]map[string]any, error) {
	rows, err := s.memoryDB.QueryContext(ctx, `
		SELECT signature, count, first_seen, last_seen, resolved, memory_id
		FROM failure_signatures WHERE repo_id = ?
		ORDER BY count DESC
	`, repoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]any
	for rows.Next() {
		var sig, first, last string
		var count int
		var resolved int
		var memID sql.NullString
		if err := rows.Scan(&sig, &count, &first, &last, &resolved, &memID); err == nil {
			m := map[string]any{
				"signature":  sig,
				"count":      count,
				"first_seen": first,
				"last_seen":  last,
				"resolved":   resolved == 1,
			}
			if memID.Valid {
				m["memory_id"] = memID.String
			}
			results = append(results, m)
		}
	}
	return results, nil
}

func (s *Store) ListReposDB(ctx context.Context) ([]core.Repo, error) {
	rows, err := s.memoryDB.QueryContext(ctx, "SELECT id, path, created_at FROM repos ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var repos []core.Repo
	for rows.Next() {
		var r core.Repo
		if err := rows.Scan(&r.ID, &r.Path, &r.CreatedAt); err == nil {
			repos = append(repos, r)
		}
	}
	return repos, nil
}

func (s *Store) FailureErrorCount(ctx context.Context, repoID string) int {
	var count int
	err := s.memoryDB.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(count), 0) FROM failure_signatures
		WHERE repo_id = ? AND resolved = 0
	`, repoID).Scan(&count)
	if err != nil {
		return 0
	}
	return count
}

func strPtr(s string) *string {
	return &s
}
