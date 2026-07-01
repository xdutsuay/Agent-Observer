package sqlite

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"agent-memory-mcp/internal/store/migrations"
	_ "modernc.org/sqlite"
)

type Store struct {
	memoryDB *sql.DB
	usageDB  *sql.DB
	root     string
}

func Open(root string) (*Store, error) {
	if err := os.MkdirAll(root, 0755); err != nil {
		return nil, fmt.Errorf("create root dir: %w", err)
	}

	memoryPath := filepath.Join(root, "memory.db")
	memoryDB, err := sql.Open("sqlite", memoryPath)
	if err != nil {
		return nil, fmt.Errorf("open memory.db: %w", err)
	}
	// Avoid database is locked errors in WAL mode
	memoryDB.SetMaxOpenConns(1)

	usagePath := filepath.Join(root, "usage.db")
	usageDB, err := sql.Open("sqlite", usagePath)
	if err != nil {
		memoryDB.Close()
		return nil, fmt.Errorf("open usage.db: %w", err)
	}
	usageDB.SetMaxOpenConns(1)

	// Apply migrations
	if err := applyMigrations(memoryDB, "001_memory.sql"); err != nil {
		memoryDB.Close()
		usageDB.Close()
		return nil, fmt.Errorf("migrate memory: %w", err)
	}
	if err := applyMigrations(usageDB, "001_usage.sql"); err != nil {
		memoryDB.Close()
		usageDB.Close()
		return nil, fmt.Errorf("migrate usage: %w", err)
	}

	return &Store{
		memoryDB: memoryDB,
		usageDB:  usageDB,
		root:     root,
	}, nil
}

func applyMigrations(db *sql.DB, filename string) error {
	script := migrations.MustLoad(filename)
	_, err := db.Exec(script)
	return err
}

func (s *Store) Close() error {
	var errs []error
	if err := s.memoryDB.Close(); err != nil {
		errs = append(errs, fmt.Errorf("memory db: %w", err))
	}
	if err := s.usageDB.Close(); err != nil {
		errs = append(errs, fmt.Errorf("usage db: %w", err))
	}
	if len(errs) > 0 {
		return fmt.Errorf("close errors: %v", errs)
	}
	return nil
}
