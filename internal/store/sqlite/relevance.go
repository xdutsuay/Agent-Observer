package sqlite

import (
	"context"
	"database/sql"
	"math"
	"time"

	"github.com/google/uuid"
)

var kindWeights = map[string]float64{
	"decision":   1.0,
	"fact":       0.9,
	"preference": 0.85,
	"failure":    0.7,
	"attempt":    0.3,
}

func (s *Store) RecordAccess(ctx context.Context, memoryID, accessType, query, sessionID, hostIDE string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	id := uuid.New().String()

	_, err := s.memoryDB.ExecContext(ctx, `
		INSERT INTO memory_access_log (id, memory_id, access_type, query_text, session_id, host_ide, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, id, memoryID, accessType, query, sessionID, hostIDE, now)
	if err != nil {
		return err
	}

	_, err = s.memoryDB.ExecContext(ctx, `
		UPDATE memories SET access_count = COALESCE(access_count, 0) + 1, last_accessed = ?
		WHERE id = ?
	`, now, memoryID)
	return err
}

func (s *Store) computeRelevanceScore(ctx context.Context, memoryID string) float64 {
	row := s.memoryDB.QueryRowContext(ctx, `
		SELECT kind, access_count, last_accessed, created_at, quality_tier 
		FROM memories WHERE id = ?
	`, memoryID)
	
	var kind string
	var created string
	var lastAcc, qualTier sql.NullString
	var accCount sql.NullInt64

	if err := row.Scan(&kind, &accCount, &lastAcc, &created, &qualTier); err != nil {
		return 0.0
	}

	now := time.Now().UTC()
	
	kindW, ok := kindWeights[kind]
	if !ok {
		kindW = 0.3
	}

	count := int64(0)
	if accCount.Valid { count = accCount.Int64 }
	accessFreq := math.Log1p(float64(count)) / math.Log1p(50.0)
	if accessFreq > 1.0 { accessFreq = 1.0 }

	createdT, _ := time.Parse(time.RFC3339, created)
	ageDays := now.Sub(createdT).Hours() / 24.0
	if ageDays < 0.01 { ageDays = 0.01 }
	recency := math.Pow(0.5, ageDays/14.0)

	accessRecency := 0.0
	if lastAcc.Valid {
		laT, _ := time.Parse(time.RFC3339, lastAcc.String)
		laAge := now.Sub(laT).Hours() / 24.0
		if laAge < 0.01 { laAge = 0.01 }
		accessRecency = math.Pow(0.5, laAge/14.0)
	}

	usefulness := 0.5
	if count > 0 {
		var total, positive int
		s.memoryDB.QueryRowContext(ctx, `
			SELECT COUNT(*), SUM(CASE WHEN was_useful = 1 THEN 1 ELSE 0 END)
			FROM memory_access_log WHERE memory_id = ? AND was_useful IS NOT NULL
		`, memoryID).Scan(&total, &positive)
		if total > 0 {
			usefulness = float64(positive) / float64(total)
		}
	}

	tier := "unrated"
	if qualTier.Valid { tier = qualTier.String }
	noisePenalty := 1.0
	if tier == "noise" {
		noisePenalty = 0.1
	}

	score := (0.20*kindW + 0.25*accessFreq + 0.20*recency + 0.15*accessRecency + 0.20*usefulness) * noisePenalty
	if score < 0.0 { score = 0.0 }
	if score > 1.0 { score = 1.0 }
	return score
}

func (s *Store) RefreshRelevance(ctx context.Context, repoID *string) (int, error) {
	query := "SELECT id FROM memories WHERE deleted = 0"
	var args []any
	if repoID != nil {
		query += " AND repo_id = ?"
		args = append(args, *repoID)
	}

	rows, err := s.memoryDB.QueryContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			ids = append(ids, id)
		}
	}
	rows.Close()

	count := 0
	for _, id := range ids {
		score := s.computeRelevanceScore(ctx, id)
		if _, err := s.memoryDB.ExecContext(ctx, "UPDATE memories SET relevance_score = ? WHERE id = ?", score, id); err == nil {
			count++
		}
	}
	return count, nil
}

func (s *Store) ClassifyNoise(ctx context.Context, maxAgeDays int) (int, error) {
	cutoff := time.Now().UTC().AddDate(0, 0, -maxAgeDays).Format(time.RFC3339)
	res, err := s.memoryDB.ExecContext(ctx, `
		UPDATE memories SET quality_tier = 'noise'
		WHERE kind = 'attempt' 
		AND COALESCE(access_count, 0) = 0
		AND created_at < ?
		AND COALESCE(quality_tier, 'unrated') = 'unrated'
		AND deleted = 0
	`, cutoff)
	if err != nil {
		return 0, err
	}
	c, _ := res.RowsAffected()
	return int(c), nil
}
