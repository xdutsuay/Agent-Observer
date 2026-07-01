package app

import (
	"context"

	"time"
	"agent-memory-mcp/internal/core"
)

type MemoryService interface {
	ResolveRepo(path string) (string, error)
	Remember(repoID, kind, content, source string, metadata map[string]any) (string, bool, error)
	Search(ctx context.Context, query string, repoID *string, kinds []string, limit int) ([]core.Memory, error)
	ListMemories(ctx context.Context, repoID *string, kind *string, limit int) ([]core.Memory, error)
	ListRepos(ctx context.Context) ([]core.RepoListing, error)
	GetRepoContext(ctx context.Context, repoID string) (core.RepoContext, error)
	MarkFailureResolved(ctx context.Context, repoID, signature string) (bool, error)
	Forget(ctx context.Context, memoryID, signature, repoID string) (int, error)
	RefreshRelevance(ctx context.Context, repoID *string) (int, int, error)
	GenerateContextFile(ctx context.Context, repoID string, projectPath string) error
	BuildContextFile(ctx context.Context, repoID string, projectPath string) (string, error)
	SmartContext(ctx context.Context, repoID, task string, maxTokens int) (core.SmartContext, error)
	GlobalSearch(ctx context.Context, query string, kinds []string, limit int) ([]core.Memory, error)
	GetPatternReport(ctx context.Context, repoID *string) (map[string]any, error)
	FailureHotspots(ctx context.Context, limit int) ([]map[string]any, error)
	GetRelatedMemories(ctx context.Context, memoryID string, limit int) ([]core.Memory, error)
	RecordFeedback(ctx context.Context, memoryID string, useful bool, comment string) error
}

type SearchService interface {
	Search(ctx context.Context, query string, repoID *string, kinds []string, limit int) ([]core.Memory, error)
}

type SessionService interface {
	GetIndexState(ctx context.Context, filePath string) (time.Time, error)
	UpdateIndexState(ctx context.Context, filePath string, mtime time.Time, turns []core.SessionTurn) error
	SearchSessions(ctx context.Context, query string, limit int) ([]core.SessionTurn, error)
}

type UsageService interface {
	Record(ctx context.Context, transport, method string, query map[string]any, responsePreview, clientName, clientVersion, hostIDE string, durationMS float64, ok bool) error
	ListInteractions(ctx context.Context, limit int, hostIDE *string) ([]core.UsageInteraction, error)
	ListSessions(ctx context.Context, limit int) ([]core.UsageSession, error)
	Summary(ctx context.Context) (core.UsageSummary, error)
}

type WatcherService interface{}
