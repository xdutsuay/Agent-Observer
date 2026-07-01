package mcp

import (
	"context"
	"testing"
	
	"agent-memory-mcp/internal/core"
	"github.com/mark3labs/mcp-go/mcp"
)

type mockMemoryService struct {}

func (m *mockMemoryService) ResolveRepo(path string) (string, error) {
	return "test-repo", nil
}
func (m *mockMemoryService) Remember(repoID, kind, content, source string, metadata map[string]any) (string, bool, error) {
	return "mem-123", true, nil
}
func (m *mockMemoryService) Search(ctx context.Context, query string, repoID *string, kinds []string, limit int) ([]core.Memory, error) {
	return []core.Memory{
		{ID: "mem-123", Kind: "fact", Content: "this is a mock result"},
	}, nil
}
func (m *mockMemoryService) ListMemories(ctx context.Context, repoID *string, kind *string, limit int) ([]core.Memory, error) {
	return nil, nil
}
func (m *mockMemoryService) ListRepos(ctx context.Context) ([]core.RepoListing, error) {
	return nil, nil
}
func (m *mockMemoryService) GetRepoContext(ctx context.Context, repoID string) (core.RepoContext, error) {
	return core.RepoContext{Failures: "none", Decisions: "none"}, nil
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
func (m *mockMemoryService) SmartContext(ctx context.Context, repoID, task string, maxTokens int) (core.SmartContext, error) {
	return core.SmartContext{}, nil
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

func TestServerRememberTool(t *testing.T) {
	srv := NewServer(&mockMemoryService{}, &mockUsageService{})

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "remember",
			Arguments: map[string]interface{}{
				"path": "/test/path",
				"kind": "fact",
				"content": "test content",
			},
		},
	}

	res, err := srv.handleRemember(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.IsError {
		t.Fatalf("expected IsError false, got true")
	}

	tc, ok := res.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent")
	}

	if tc.Text != "Memory stored in test-repo (id=mem-123)" {
		t.Errorf("unexpected text output: %s", tc.Text)
	}
}

func TestServerSearchTool(t *testing.T) {
	srv := NewServer(&mockMemoryService{}, &mockUsageService{})

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "search_memory",
			Arguments: map[string]interface{}{
				"query": "test",
			},
		},
	}

	res, err := srv.handleSearchMemory(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.IsError {
		t.Fatalf("expected IsError false, got true")
	}

	tc, ok := res.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent")
	}

	if tc.Text == "No memories found." || tc.Text == "" {
		t.Errorf("unexpected text output: %s", tc.Text)
	}
}
