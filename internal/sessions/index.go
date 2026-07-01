package sessions

import (
	"context"
	"log"
	"os"
	"path/filepath"

	"agent-memory-mcp/internal/app"
	"agent-memory-mcp/internal/sessions/adapters"
)

type Indexer struct {
	store    app.SessionService
	adapters []adapters.Adapter
}

func NewIndexer(store app.SessionService, adps []adapters.Adapter) *Indexer {
	return &Indexer{
		store:    store,
		adapters: adps,
	}
}

// Sync scans the given root directory recursively, parsing and indexing supported session files.
func (idx *Indexer) Sync(ctx context.Context, root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip unreadable
		}
		if info.IsDir() {
			if isIgnoredDir(path) {
				return filepath.SkipDir
			}
			return nil
		}

		for _, adapter := range idx.adapters {
			if adapter.Supports(path) {
				if err := idx.syncFile(ctx, path, info, adapter); err != nil {
					log.Printf("failed to sync session file %s: %v", path, err)
				}
				break // Only use the first matching adapter
			}
		}
		return nil
	})
}

func (idx *Indexer) syncFile(ctx context.Context, path string, info os.FileInfo, adapter adapters.Adapter) error {
	lastMod, err := idx.store.GetIndexState(ctx, path)
	if err != nil {
		return err
	}

	// Skip if it hasn't changed since last index
	if !info.ModTime().After(lastMod) {
		return nil
	}

	turns, err := adapter.Parse(ctx, path)
	if err != nil {
		return err
	}

	return idx.store.UpdateIndexState(ctx, path, info.ModTime(), turns)
}

func isIgnoredDir(path string) bool {
	base := filepath.Base(path)
	switch base {
	case ".git", "node_modules", "vendor", ".venv", "__pycache__":
		return true
	}
	return false
}
