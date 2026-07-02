package tenant

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"agent-memory-mcp/internal/app"
	"agent-memory-mcp/internal/store/sqlite"
	"agent-memory-mcp/internal/memory"
	"agent-memory-mcp/internal/usage"
	"agent-memory-mcp/internal/watcher"
)

type TenantServices struct {
	MemoryService app.MemoryService
	UsageService  app.UsageService
	WatcherService app.WatcherService
	Store         *sqlite.Store
}

type Provider interface {
	Get(tenantID string) (*TenantServices, error)
}

type TenantManager struct {
	mu       sync.Mutex
	rootPath string
	tenants  map[string]*TenantServices
}

func NewTenantManager(rootPath string) *TenantManager {
	return &TenantManager{
		rootPath: rootPath,
		tenants:  make(map[string]*TenantServices),
	}
}

func (tm *TenantManager) Get(tenantID string) (*TenantServices, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if ts, exists := tm.tenants[tenantID]; exists {
		return ts, nil
	}

	// Create new tenant DB
	tenantDir := tm.rootPath
	if tenantID != "local" {
		tenantDir = filepath.Join(tm.rootPath, "tenants", tenantID)
		if err := os.MkdirAll(tenantDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create tenant dir: %w", err)
		}
	}

	store, err := sqlite.Open(tenantDir)
	if err != nil {
		return nil, fmt.Errorf("failed to open tenant db: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := sqlite.RunLegacyMigration(ctx, store, tenantDir); err != nil {
		// Log migration error but continue
	}

	ts := &TenantServices{
		Store:         store,
		MemoryService: memory.NewService(store),
		UsageService:  usage.NewService(store),
		WatcherService: watcher.NewService(),
	}

	tm.tenants[tenantID] = ts
	return ts, nil
}

func (tm *TenantManager) CloseAll() {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	for _, ts := range tm.tenants {
		ts.Store.Close()
	}
}
