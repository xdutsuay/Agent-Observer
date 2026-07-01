package ingest

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strings"

	"agent-memory-mcp/internal/app"
	"agent-memory-mcp/internal/ingest/logs"
	"agent-memory-mcp/internal/ingest/watcher"
)

type Service struct {
	watcher       *watcher.Watcher
	memoryService app.MemoryService
}

func NewService(w *watcher.Watcher, ms app.MemoryService) *Service {
	return &Service{
		watcher:       w,
		memoryService: ms,
	}
}

func (s *Service) Start(ctx context.Context) {
	s.watcher.Start(ctx)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case evt := <-s.watcher.Events():
				s.processEvent(ctx, evt)
			case err := <-s.watcher.Errors():
				log.Printf("watcher error: %v", err)
			}
		}
	}()
}

func (s *Service) processEvent(ctx context.Context, evt watcher.FileEvent) {
	// Skip processing non-files
	info, err := os.Stat(evt.Path)
	if err != nil || info.IsDir() {
		return
	}

	content, err := os.ReadFile(evt.Path)
	if err != nil {
		return
	}
	contentStr := string(content)

	repoID, err := s.memoryService.ResolveRepo(filepath.Dir(evt.Path))
	if err != nil {
		return
	}

	// Rule 1: Log file errors
	if isLogFile(evt.Path) {
		if extErr := logs.ExtractError(contentStr); extErr != nil {
			meta := map[string]any{
				"file": evt.Path,
				"lang": extErr.Language,
			}
			s.memoryService.Remember(repoID, "failure", extErr.Message, "watcher", meta)
			return
		}
	}

	// Rule 2: Basic classification (stub)
	// Instead of saving EVERY file change as 'attempt', we drop it unless it has significant changes.
	// For this port, we consider "significant changes" any file over 100 bytes (for testing)
	// that isn't a log.
	if len(contentStr) > 100 && !isLogFile(evt.Path) {
		meta := map[string]any{
			"file": evt.Path,
		}
		// In a real system, we might only log summary diffs. We'll store a snippet.
		snippet := contentStr
		if len(snippet) > 200 {
			snippet = snippet[:200] + "..."
		}
		s.memoryService.Remember(repoID, "attempt", "File modified: "+filepath.Base(evt.Path)+"\n"+snippet, "watcher", meta)
	}
}

func isLogFile(path string) bool {
	return strings.HasSuffix(path, ".log") || strings.Contains(path, "stdout") || strings.Contains(path, "stderr")
}
