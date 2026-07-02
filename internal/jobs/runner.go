package jobs

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"time"

	"agent-memory-mcp/internal/app"
	"agent-memory-mcp/internal/patterns"
	"agent-memory-mcp/internal/sessions"
	"agent-memory-mcp/internal/sessions/adapters"
	"agent-memory-mcp/internal/store/sqlite"
)

// Runner manages periodic background intelligence jobs.
type Runner struct {
	store    *sqlite.Store
	detector *patterns.Detector
	indexer  *sessions.Indexer
	stop     chan struct{}
}

func NewRunner(store *sqlite.Store, detector *patterns.Detector, sessionStore app.SessionService) *Runner {
	indexer := sessions.NewIndexer(sessionStore, []adapters.Adapter{
		&adapters.JsonlAdapter{},
		&adapters.MarkdownAdapter{},
	})
	return &Runner{
		store:    store,
		detector: detector,
		indexer:  indexer,
		stop:     make(chan struct{}),
	}
}

// Start launches the background job loop.
// - Relevance refresh runs every hour.
// - Noise classification runs every 6 hours.
func (r *Runner) Start(ctx context.Context) {
	go r.loop(ctx)
	log.Println("Background intelligence jobs started")
}

// Stop gracefully stops the background loop.
func (r *Runner) Stop() {
	close(r.stop)
}

func (r *Runner) loop(ctx context.Context) {
	// Run once on startup
	r.runRelevanceRefresh(ctx)
	r.runNoiseClassification(ctx)
	r.runSessionIndexing(ctx)

	relevanceTicker := time.NewTicker(1 * time.Hour)
	noiseTicker := time.NewTicker(6 * time.Hour)
	sessionTicker := time.NewTicker(5 * time.Minute)
	
	defer relevanceTicker.Stop()
	defer noiseTicker.Stop()
	defer sessionTicker.Stop()

	for {
		select {
		case <-r.stop:
			log.Println("Background jobs stopped")
			return
		case <-ctx.Done():
			return
		case <-relevanceTicker.C:
			r.runRelevanceRefresh(ctx)
		case <-noiseTicker.C:
			r.runNoiseClassification(ctx)
		case <-sessionTicker.C:
			r.runSessionIndexing(ctx)
		}
	}
}

func (r *Runner) runRelevanceRefresh(ctx context.Context) {
	count, err := r.store.RefreshRelevance(ctx, nil)
	if err != nil {
		log.Printf("[jobs] relevance refresh error: %v", err)
		return
	}
	if count > 0 {
		log.Printf("[jobs] refreshed relevance for %d memories", count)
	}
}

func (r *Runner) runNoiseClassification(ctx context.Context) {
	count, err := r.store.ClassifyNoise(ctx, 7)
	if err != nil {
		log.Printf("[jobs] noise classification error: %v", err)
		return
	}
	if count > 0 {
		log.Printf("[jobs] classified %d memories as noise", count)
	}
}

func (r *Runner) runSessionIndexing(ctx context.Context) {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Printf("[jobs] session indexing: cannot determine home dir: %v", err)
		return
	}

	// Cross-tool session transcript paths
	sessionPaths := []string{
		filepath.Join(home, ".claude", "projects"),          // Claude Code sessions
		filepath.Join(home, ".codex", "sessions"),           // Codex CLI sessions
		filepath.Join(home, ".cursor", "projects"),          // Cursor agent transcripts
		filepath.Join(home, "agent_companion_data", "sessions"), // Our own sessions
	}

	for _, p := range sessionPaths {
		if _, err := os.Stat(p); err == nil {
			if syncErr := r.indexer.Sync(ctx, p); syncErr != nil {
				log.Printf("[jobs] session sync error for %s: %v", p, syncErr)
			}
		}
	}
}
