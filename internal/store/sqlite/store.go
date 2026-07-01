package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	"agent-memory-mcp/internal/core"
	"github.com/google/uuid"
)

var kindAliases = map[string]string{
	"failures":    "failure",
	"failure":     "failure",
	"decisions":   "decision",
	"decision":    "decision",
	"attempts":    "attempt",
	"attempt":     "attempt",
	"facts":       "fact",
	"fact":        "fact",
	"preferences": "preference",
	"preference":  "preference",
}

func normalizeKind(kind string) string {
	k := strings.ToLower(strings.TrimSpace(kind))
	if alias, ok := kindAliases[k]; ok {
		return alias
	}
	return k
}

func (s *Store) UpsertRepo(ctx context.Context, repoID, path string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.memoryDB.ExecContext(ctx, `
		INSERT INTO repos (id, path, created_at) VALUES (?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET path = excluded.path
	`, repoID, path, now)
	return err
}

func (s *Store) InsertMemory(ctx context.Context, repoID, kind, content, source string, metadata map[string]any, skipDedup bool) (string, bool, error) {
	kindN := normalizeKind(kind)
	metaJSON, _ := json.Marshal(metadata)
	now := time.Now().UTC().Format(time.RFC3339)

	var sig string
	if content != "" {
		lines := strings.Split(content, "\n")
		sig = lines[0]
		if len(sig) > 100 {
			sig = sig[:100]
		}
		sig = strings.TrimSpace(sig)
	}

	if kindN == "failure" && !skipDedup {
		var memID sql.NullString
		err := s.memoryDB.QueryRowContext(ctx, `
			SELECT memory_id FROM failure_signatures 
			WHERE repo_id = ? AND signature = ?
		`, repoID, sig).Scan(&memID)
		
		if err == nil {
			// Exists
			_, err = s.memoryDB.ExecContext(ctx, `
				UPDATE failure_signatures SET count = count + 1, last_seen = ? 
				WHERE repo_id = ? AND signature = ?
			`, now, repoID, sig)
			return memID.String, false, err
		} else if err != sql.ErrNoRows {
			return "", false, err
		}
	}

	memID := uuid.New().String()
	_, err := s.memoryDB.ExecContext(ctx, `
		INSERT INTO memories (id, repo_id, kind, content, source, metadata_json, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, memID, repoID, kindN, content, source, string(metaJSON), now)
	if err != nil {
		return "", false, err
	}

	var rowid int64
	err = s.memoryDB.QueryRowContext(ctx, "SELECT rowid FROM memories WHERE id = ?", memID).Scan(&rowid)
	if err == nil {
		s.memoryDB.ExecContext(ctx, "INSERT INTO memories_fts(rowid, content, kind, repo_id) VALUES (?, ?, ?, ?)", rowid, content, kindN, repoID)
	}

	emb := s.embed(content)
	embJSON, _ := json.Marshal(emb)
	s.memoryDB.ExecContext(ctx, "INSERT OR REPLACE INTO memory_embeddings (memory_id, embedding_json) VALUES (?, ?)", memID, string(embJSON))

	if kindN == "failure" {
		s.memoryDB.ExecContext(ctx, `
			INSERT INTO failure_signatures (repo_id, signature, count, first_seen, last_seen, resolved, memory_id)
			VALUES (?, ?, 1, ?, ?, 0, ?)
			ON CONFLICT(repo_id, signature) DO UPDATE SET 
				count = count + 1, last_seen = excluded.last_seen, memory_id = excluded.memory_id
		`, repoID, sig, now, now, memID)
	}

	return memID, true, nil
}

func (s *Store) ListMemories(ctx context.Context, repoID *string, kind *string, limit int) ([]core.Memory, error) {
	query := "SELECT id, repo_id, kind, content, source, metadata_json, session_id, summary, created_at, access_count, last_accessed, relevance_score, quality_tier FROM memories WHERE deleted = 0"
	var args []any

	if repoID != nil {
		query += " AND repo_id = ?"
		args = append(args, *repoID)
	}
	if kind != nil {
		query += " AND kind = ?"
		args = append(args, normalizeKind(*kind))
	}
	query += " ORDER BY created_at DESC LIMIT ?"
	args = append(args, limit)

	rows, err := s.memoryDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []core.Memory
	for rows.Next() {
		var m core.Memory
		var metaJSON, sessionID, summary, lastAccessed, qualityTier sql.NullString
		var accessCount sql.NullInt64
		var relScore sql.NullFloat64

		if err := rows.Scan(&m.ID, &m.RepoID, &m.Kind, &m.Content, &m.Source, &metaJSON, &sessionID, &summary, &m.CreatedAt, &accessCount, &lastAccessed, &relScore, &qualityTier); err != nil {
			return nil, err
		}

		if metaJSON.Valid {
			json.Unmarshal([]byte(metaJSON.String), &m.Metadata)
		} else {
			m.Metadata = make(map[string]any)
		}
		if sessionID.Valid { m.SessionID = sessionID.String }
		if summary.Valid { m.Summary = summary.String }
		if lastAccessed.Valid { m.LastAccessed = lastAccessed.String }
		if qualityTier.Valid { m.QualityTier = qualityTier.String }
		m.AccessCount = int(accessCount.Int64)
		m.RelevanceScore = relScore.Float64
		
		results = append(results, m)
	}
	return results, nil
}

func (s *Store) Forget(ctx context.Context, memoryID, signature, repoID string) (int, error) {
	count := 0
	if memoryID != "" {
		res, err := s.memoryDB.ExecContext(ctx, "UPDATE memories SET deleted = 1 WHERE id = ?", memoryID)
		if err != nil {
			return count, err
		}
		c, _ := res.RowsAffected()
		count += int(c)
	}
	if signature != "" && repoID != "" {
		res, err := s.memoryDB.ExecContext(ctx, `
			UPDATE memories SET deleted = 1 WHERE id IN (
				SELECT memory_id FROM failure_signatures WHERE repo_id = ? AND signature = ?
			)
		`, repoID, signature)
		if err != nil {
			return count, err
		}
		c, _ := res.RowsAffected()
		count += int(c)
	}
	return count, nil
}

func (s *Store) MarkFailureResolved(ctx context.Context, repoID, signature string) (bool, error) {
	res, err := s.memoryDB.ExecContext(ctx, "UPDATE failure_signatures SET resolved = 1 WHERE repo_id = ? AND signature = ?", repoID, signature)
	if err != nil {
		return false, err
	}
	c, _ := res.RowsAffected()
	return c > 0, nil
}
