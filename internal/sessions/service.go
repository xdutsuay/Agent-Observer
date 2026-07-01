package sessions

import (
	"context"
	
	"agent-memory-mcp/internal/core"
	"agent-memory-mcp/internal/app"
)

// Service provides a high-level API over session indexing and searching.
type Service struct {
	store   app.SessionService
	indexer *Indexer
}

func NewService(store app.SessionService, indexer *Indexer) *Service {
	return &Service{
		store:   store,
		indexer: indexer,
	}
}

// Search queries the indexed session turns.
func (s *Service) Search(ctx context.Context, query string, limit int) ([]core.SessionTurn, error) {
	return s.store.SearchSessions(ctx, query, limit)
}

// Sync forces an indexer sync over a root path.
func (s *Service) Sync(ctx context.Context, root string) error {
	return s.indexer.Sync(ctx, root)
}
