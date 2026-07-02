package sqlite

import (
	"context"
	"time"

	"agent-memory-mcp/internal/core"
)

// GetIndexState returns the last modified time for a given file path.
// If the file is not found, it returns a zero time.
func (s *Store) GetIndexState(ctx context.Context, filePath string) (time.Time, error) {
	var ts string
	err := s.memoryDB.QueryRowContext(ctx, "SELECT last_modified FROM index_state WHERE file_path = ?", filePath).Scan(&ts)
	if err != nil {
		return time.Time{}, nil // Treat as not found
	}
	t, _ := time.Parse(time.RFC3339, ts)
	return t, nil
}

// UpdateIndexState updates the last modified time for a file and overwrites its turns.
func (s *Store) UpdateIndexState(ctx context.Context, filePath string, mtime time.Time, turns []core.SessionTurn) error {
	tx, err := s.memoryDB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Update index state
	_, err = tx.ExecContext(ctx, `
		INSERT INTO index_state (file_path, last_modified) 
		VALUES (?, ?) 
		ON CONFLICT(file_path) DO UPDATE SET last_modified = excluded.last_modified
	`, filePath, mtime.Format(time.RFC3339))
	if err != nil {
		return err
	}

	// Delete old turns for this session (filePath is used as session_id for file-based logs)
	_, err = tx.ExecContext(ctx, "DELETE FROM session_turns_fts WHERE rowid IN (SELECT rowid FROM session_turns WHERE session_id = ?)", filePath)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, "DELETE FROM session_turns WHERE session_id = ?", filePath)
	if err != nil {
		return err
	}

	// Insert new turns
	insertTurn, err := tx.PrepareContext(ctx, `
		INSERT INTO session_turns (id, session_id, turn_number, user_input, agent_response, timestamp)
		VALUES (?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer insertTurn.Close()

	insertFTS, err := tx.PrepareContext(ctx, `
		INSERT INTO session_turns_fts (rowid, user_input, agent_response)
		VALUES (?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer insertFTS.Close()

	for _, turn := range turns {
		tsStr := turn.Timestamp.Format(time.RFC3339)
		res, err := insertTurn.ExecContext(ctx, turn.ID, turn.SessionID, turn.TurnNumber, turn.UserInput, turn.AgentResponse, tsStr)
		if err != nil {
			return err
		}
		
		rowID, err := res.LastInsertId()
		if err == nil {
			insertFTS.ExecContext(ctx, rowID, turn.UserInput, turn.AgentResponse)
		}
	}

	return tx.Commit()
}

// SearchSessions searches the session turns using FTS5.
func (s *Store) SearchSessions(ctx context.Context, query string, limit int) ([]core.SessionTurn, error) {
	rows, err := s.memoryDB.QueryContext(ctx, `
		SELECT s.id, s.session_id, s.turn_number, s.user_input, s.agent_response, s.timestamp
		FROM session_turns_fts f
		JOIN session_turns s ON s.rowid = f.rowid
		WHERE session_turns_fts MATCH ?
		ORDER BY rank
		LIMIT ?
	`, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := []core.SessionTurn{}
	for rows.Next() {
		var turn core.SessionTurn
		var ts string
		if err := rows.Scan(&turn.ID, &turn.SessionID, &turn.TurnNumber, &turn.UserInput, &turn.AgentResponse, &ts); err != nil {
			return nil, err
		}
		turn.Timestamp, _ = time.Parse(time.RFC3339, ts)
		results = append(results, turn)
	}
	return results, nil
}
