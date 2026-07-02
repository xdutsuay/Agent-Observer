package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"math"
	"regexp"
	"sort"
	"strings"
	"time"

	"agent-memory-mcp/internal/core"
)

func (s *Store) embed(text string) []float64 {
	// Simple deterministic bag-of-words vector (legacy fallback)
	dim := 64
	vec := make([]float64, dim)
	
	re := regexp.MustCompile(`[a-z0-9]{3,}`)
	tokens := re.FindAllString(strings.ToLower(text), -1)
	if len(tokens) == 0 {
		return vec
	}

	for _, tok := range tokens {
		var h uint32
		for i := 0; i < len(tok); i++ {
			h = h*31 + uint32(tok[i])
		}
		vec[h%uint32(dim)] += 1.0
	}

	sumSq := 0.0
	for _, v := range vec {
		sumSq += v * v
	}
	norm := math.Sqrt(sumSq)
	if norm == 0 {
		norm = 1.0
	}
	for i := range vec {
		vec[i] /= norm
	}
	return vec
}

func cosine(a, b []float64) float64 {
	if len(a) != len(b) {
		return 0.0
	}
	dot := 0.0
	naSq := 0.0
	nbSq := 0.0
	for i := range a {
		dot += a[i] * b[i]
		naSq += a[i] * a[i]
		nbSq += b[i] * b[i]
	}
	na := math.Sqrt(naSq)
	nb := math.Sqrt(nbSq)
	if na == 0 { na = 1.0 }
	if nb == 0 { nb = 1.0 }
	return dot / (na * nb)
}

func (s *Store) Search(ctx context.Context, query string, repoID *string, kinds []string, limit int) ([]core.Memory, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		// fallback to list
		var k *string
		if len(kinds) > 0 { k = &kinds[0] }
		return s.ListMemories(ctx, repoID, k, limit)
	}

	results := make(map[string]*core.Memory)
	qEmb := s.embed(query)

	// 1. FTS5 Keyword Search
	ftsTokens := strings.Fields(query)
	for i, t := range ftsTokens {
		ftsTokens[i] = `"` + strings.ReplaceAll(t, `"`, `""`) + `"`
	}
	ftsQuery := strings.Join(ftsTokens, " ")

	ftsSQL := `
		SELECT m.id, m.repo_id, m.kind, m.content, m.source, m.metadata_json,
			   m.session_id, m.summary, m.created_at,
			   bm25(memories_fts) AS rank
		FROM memories_fts f
		JOIN memories m ON m.rowid = f.rowid
		WHERE memories_fts MATCH ? AND m.deleted = 0
	`
	var args []any
	args = append(args, ftsQuery)
	if repoID != nil {
		ftsSQL += " AND m.repo_id = ?"
		args = append(args, *repoID)
	}
	if len(kinds) > 0 {
		ftsSQL += " AND m.kind IN (" + strings.Repeat("?,", len(kinds)-1) + "?)"
		for _, k := range kinds { args = append(args, normalizeKind(k)) }
	}
	ftsSQL += " ORDER BY rank LIMIT ?"
	args = append(args, limit*2)

	rows, err := s.memoryDB.QueryContext(ctx, ftsSQL, args...)
	if err != nil {
		return nil, err
	}
	if rows != nil {
		for rows.Next() {
			var m core.Memory
			var rank float64
			var metaJSON, sessionID, summary sql.NullString
			if err := rows.Scan(&m.ID, &m.RepoID, &m.Kind, &m.Content, &m.Source, &metaJSON, &sessionID, &summary, &m.CreatedAt, &rank); err == nil {
				if metaJSON.Valid { json.Unmarshal([]byte(metaJSON.String), &m.Metadata) }
				if sessionID.Valid { m.SessionID = sessionID.String }
				if summary.Valid { m.Summary = summary.String }
				m.Score = -rank // bm25 returns negative values for better ranks
				m.Match = "keyword"
				results[m.ID] = &m
			}
		}
		rows.Close()
	}

	// 2. Vector Search
	vecSQL := `
		SELECT m.id, m.repo_id, m.kind, m.content, m.source, m.metadata_json,
			   m.session_id, m.summary, m.created_at,
			   m.access_count, m.last_accessed, m.relevance_score, m.quality_tier,
			   e.embedding_json
		FROM memories m
		JOIN memory_embeddings e ON e.memory_id = m.id
		WHERE m.deleted = 0
	`
	var vecArgs []any
	if repoID != nil {
		vecSQL += " AND m.repo_id = ?"
		vecArgs = append(vecArgs, *repoID)
	}
	if len(kinds) > 0 {
		vecSQL += " AND m.kind IN (" + strings.Repeat("?,", len(kinds)-1) + "?)"
		for _, k := range kinds { vecArgs = append(vecArgs, normalizeKind(k)) }
	}
	vecSQL += " ORDER BY m.created_at DESC LIMIT ?"
	vecArgs = append(vecArgs, limit*5)

	vRows, err := s.memoryDB.QueryContext(ctx, vecSQL, vecArgs...)
	if err != nil {
		return nil, err
	}
	if vRows != nil {
		for vRows.Next() {
			var m core.Memory
			var metaJSON, sessionID, summary, lastAcc, qualTier sql.NullString
			var accCount sql.NullInt64
			var relScore sql.NullFloat64
			var embJSON string

			if err := vRows.Scan(&m.ID, &m.RepoID, &m.Kind, &m.Content, &m.Source, &metaJSON, &sessionID, &summary, &m.CreatedAt, &accCount, &lastAcc, &relScore, &qualTier, &embJSON); err == nil {
				var emb []float64
				json.Unmarshal([]byte(embJSON), &emb)
				sim := cosine(qEmb, emb)
				if sim < 0.05 { continue }

				if metaJSON.Valid { json.Unmarshal([]byte(metaJSON.String), &m.Metadata) }
				if sessionID.Valid { m.SessionID = sessionID.String }
				if summary.Valid { m.Summary = summary.String }
				if lastAcc.Valid { m.LastAccessed = lastAcc.String }
				if qualTier.Valid { m.QualityTier = qualTier.String }
				m.AccessCount = int(accCount.Int64)
				m.RelevanceScore = relScore.Float64
				
				m.Score = sim
				m.Match = "vector"

				if prev, exists := results[m.ID]; !exists || m.Score > prev.Score {
					results[m.ID] = &m
				}
			}
		}
		vRows.Close()
	}

	// 3. Blending and Ranking
	ranked := []core.Memory{}
	for _, m := range results {
		if m.QualityTier == "noise" {
			continue // skip noise
		}
		
		t, err := time.Parse(time.RFC3339, m.CreatedAt)
		ageDays := 0.01
		if err == nil {
			dur := time.Since(t).Hours() / 24.0
			if dur > 0.01 { ageDays = dur }
		}
		recency := math.Pow(0.5, ageDays/14.0)
		
		raw := m.Score
		if raw > 1.0 { raw = 1.0 }

		m.Score = (0.45 * raw) + (0.30 * m.RelevanceScore) + (0.25 * recency)
		if m.Score > 0 {
			ranked = append(ranked, *m)
		}
	}

	sort.Slice(ranked, func(i, j int) bool {
		return ranked[i].Score > ranked[j].Score
	})

	if len(ranked) > limit {
		ranked = ranked[:limit]
	}

	return ranked, nil
}
