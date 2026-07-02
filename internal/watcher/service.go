package watcher

import (
	"context"
	"agent-memory-mcp/internal/app"
)

type Service struct{}

func NewService() app.WatcherService {
	return &Service{}
}

func (s *Service) Start(ctx context.Context, repoID string) error {
	return nil
}

func (s *Service) Stop(ctx context.Context, repoID string) error {
	return nil
}
