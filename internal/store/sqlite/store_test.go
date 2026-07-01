package sqlite

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestStoreBootstrapAndDedup(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := Open(tmpDir)
	if err != nil {
		t.Fatalf("Failed to open store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	store.UpsertRepo(ctx, "test-repo", "/tmp/test")

	// Insert a failure memory
	content := "Error: something went wrong\nat file.go:10"
	memID1, inserted, err := store.InsertMemory(ctx, "test-repo", "failure", content, "test", nil, false)
	if err != nil || !inserted {
		t.Fatalf("Expected memory to be inserted: err=%v, inserted=%v", err, inserted)
	}

	// Insert duplicate failure memory
	memID2, inserted2, err := store.InsertMemory(ctx, "test-repo", "failure", content, "test", nil, false)
	if err != nil || inserted2 {
		t.Fatalf("Expected duplicate memory to NOT be inserted: err=%v, inserted=%v", err, inserted2)
	}

	if memID1 != memID2 {
		t.Fatalf("Expected returned ID to match original for deduplicated failure: got %s, want %s", memID2, memID1)
	}

	errCount := store.FailureErrorCount(ctx, "test-repo")
	if errCount != 2 {
		t.Fatalf("Expected error count 2, got %d", errCount)
	}
}

func TestStoreSearch(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := Open(tmpDir)
	if err != nil {
		t.Fatalf("Failed to open store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	store.UpsertRepo(ctx, "repo1", "/test1")
	store.InsertMemory(ctx, "repo1", "fact", "The sky is very blue today", "test", nil, false)
	store.InsertMemory(ctx, "repo1", "decision", "We decided to use golang for performance", "test", nil, false)

	// Search for keyword
	mems, err := store.Search(ctx, "golang", nil, nil, 10)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(mems) == 0 {
		t.Fatalf("Expected to find 'golang' memory")
	}
	if mems[0].Kind != "decision" {
		t.Fatalf("Expected decision memory, got %s", mems[0].Kind)
	}
}

func TestLegacyMigration(t *testing.T) {
	tmpDir := t.TempDir()
	
	// Create mock legacy structure
	memDir := filepath.Join(tmpDir, "agent-memory", "test-repo", "memory")
	os.MkdirAll(memDir, 0755)
	os.WriteFile(filepath.Join(memDir, "failures.md"), []byte("### 2026-07-01T12:00:00Z\nSome failure content\n\n### 2026-07-01T12:05:00Z\nAnother failure\n"), 0644)

	store, _ := Open(tmpDir)
	defer store.Close()

	ctx := context.Background()
	err := RunLegacyMigration(ctx, store, tmpDir)
	if err != nil {
		t.Fatalf("Migration failed: %v", err)
	}

	// Verify imported
	mems, err := store.ListMemories(ctx, nil, nil, 10)
	if err != nil {
		t.Fatalf("ListMemories failed: %v", err)
	}
	if len(mems) != 2 {
		t.Fatalf("Expected 2 imported memories, got %d", len(mems))
	}
}
