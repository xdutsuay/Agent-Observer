package usage

import (
	"context"

	"agent-memory-mcp/internal/app"
	"agent-memory-mcp/internal/core"
	"agent-memory-mcp/internal/store/sqlite"
)

type Service struct {
	store *sqlite.Store
}

func NewService(store *sqlite.Store) app.UsageService {
	return &Service{store: store}
}

func (s *Service) Record(ctx context.Context, transport, method string, query map[string]any, responsePreview, clientName, clientVersion, hostIDE string, durationMS float64, ok bool) error {
	sessionID := clientName + "-" + hostIDE // simplified session id
	querySummary := "query"
	return s.store.RecordUsage(ctx, sessionID, transport, method, clientName, clientVersion, hostIDE, querySummary, query, responsePreview, durationMS, ok)
}

func (s *Service) ListInteractions(ctx context.Context, limit int, hostIDE *string) ([]core.UsageInteraction, error) {
	return s.store.ListUsageInteractions(ctx, limit, hostIDE)
}

func (s *Service) ListSessions(ctx context.Context, limit int) ([]core.UsageSession, error) {
	// Not fully implemented in sqlite layer yet, returning empty for now
	return []core.UsageSession{}, nil
}

func (s *Service) Summary(ctx context.Context) (core.UsageSummary, error) {
	return core.UsageSummary{
		ByMethod:    []core.CountByName{},
		ByHostIDE:   []core.CountByName{},
		ByTransport: []core.CountByName{},
		RunningIDEs: []core.RunningIDE{},
	}, nil
}
