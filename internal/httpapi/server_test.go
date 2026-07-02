package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"agent-memory-mcp/internal/core"
	"agent-memory-mcp/internal/tenant"
	"agent-memory-mcp/internal/config"
)

type mockMemoryService struct {}

func (m *mockMemoryService) ResolveRepo(path string) (string, error) {
	return "test-repo", nil
}
func (m *mockMemoryService) Remember(repoID, kind, content, source string, metadata map[string]any) (string, bool, error) {
	return "mem-123", true, nil
}
func (m *mockMemoryService) Search(ctx context.Context, query string, repoID *string, kinds []string, limit int) ([]core.Memory, error) {
	return nil, nil
}
func (m *mockMemoryService) ListMemories(ctx context.Context, repoID *string, kind *string, limit int) ([]core.Memory, error) {
	return nil, nil
}
func (m *mockMemoryService) ListRepos(ctx context.Context) ([]core.RepoListing, error) {
	return []core.RepoListing{
		{ID: "repo1", LastModified: "2023-01-01"},
	}, nil
}
func (m *mockMemoryService) GetRepoContext(ctx context.Context, repoID string) (core.RepoContext, error) {
	return core.RepoContext{}, nil
}
func (m *mockMemoryService) MarkFailureResolved(ctx context.Context, repoID, signature string) (bool, error) {
	return true, nil
}
func (m *mockMemoryService) Forget(ctx context.Context, memoryID, signature, repoID string) (int, error) {
	return 1, nil
}
func (m *mockMemoryService) RefreshRelevance(ctx context.Context, repoID *string) (int, int, error) {
	return 0, 0, nil
}
func (m *mockMemoryService) GenerateContextFile(ctx context.Context, repoID string, projectPath string) error {
	return nil
}
func (m *mockMemoryService) BuildContextFile(ctx context.Context, repoID string, projectPath string) (string, error) {
	return "", nil
}
func (m *mockMemoryService) SmartContext(ctx context.Context, repoID, task string, maxTokens int) (core.SmartContext, error) {
	return core.SmartContext{}, nil
}
func (m *mockMemoryService) GlobalSearch(ctx context.Context, query string, kinds []string, limit int) ([]core.Memory, error) {
	return nil, nil
}
func (m *mockMemoryService) GetPatternReport(ctx context.Context, repoID *string) (map[string]any, error) {
	return nil, nil
}
func (m *mockMemoryService) FailureHotspots(ctx context.Context, limit int) ([]map[string]any, error) {
	return nil, nil
}
func (m *mockMemoryService) GetRelatedMemories(ctx context.Context, memoryID string, limit int) ([]core.Memory, error) {
	return nil, nil
}
func (m *mockMemoryService) RecordFeedback(ctx context.Context, memoryID string, useful bool, comment string) error {
	return nil
}

func (m *mockMemoryService) Export(ctx context.Context, repoID *string) ([]core.Memory, error) {
	return nil, nil
}
func (m *mockMemoryService) Import(ctx context.Context, memories []core.Memory) (int, error) {
	return 0, nil
}

type mockUsageService struct {}

func (m *mockUsageService) Record(ctx context.Context, transport, method string, query map[string]any, responsePreview, clientName, clientVersion, hostIDE string, durationMS float64, ok bool) error {
	return nil
}
func (m *mockUsageService) ListInteractions(ctx context.Context, limit int, hostIDE *string) ([]core.UsageInteraction, error) {
	return nil, nil
}
func (m *mockUsageService) ListSessions(ctx context.Context, limit int) ([]core.UsageSession, error) {
	return nil, nil
}
func (m *mockUsageService) Summary(ctx context.Context) (core.UsageSummary, error) {
	return core.UsageSummary{}, nil
}

type mockTenantProvider struct {}
func (m *mockTenantProvider) Get(tenantID string) (*tenant.TenantServices, error) {
	return &tenant.TenantServices{
		MemoryService: &mockMemoryService{},
		UsageService:  &mockUsageService{},
	}, nil
}

func TestHealthRoute(t *testing.T) {
	srv := NewServer(&mockTenantProvider{}, config.Config{})
	h := srv.Handler()

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", w.Result().StatusCode)
	}
	
	var res map[string]string
	json.NewDecoder(w.Body).Decode(&res)
	if res["status"] != "ok" {
		t.Errorf("expected status ok, got %s", res["status"])
	}
}

func TestListProjectsRoute(t *testing.T) {
	srv := NewServer(&mockTenantProvider{}, config.Config{})
	h := srv.Handler()

	req := httptest.NewRequest("GET", "/api/projects", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", w.Result().StatusCode)
	}

	var repos []core.RepoListing
	json.NewDecoder(w.Body).Decode(&repos)
	
	if len(repos) != 1 || repos[0].ID != "repo1" {
		t.Errorf("expected 1 repo with ID repo1, got %v", repos)
	}
}
