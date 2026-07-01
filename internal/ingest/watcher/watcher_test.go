package watcher

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWatcher_Debounce(t *testing.T) {
	dir, err := os.MkdirTemp("", "watcher_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	// Short debounce for test
	w, err := New(50 * time.Millisecond)
	if err != nil {
		t.Fatalf("failed to create watcher: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	w.Start(ctx)
	w.AddRecursive(dir)

	file := filepath.Join(dir, "test.txt")
	
	// Create multiple writes in quick succession
	for i := 0; i < 5; i++ {
		os.WriteFile(file, []byte("data"), 0644)
		time.Sleep(10 * time.Millisecond)
	}

	// We should only receive one event due to debouncing
	select {
	case evt := <-w.Events():
		if evt.Path != file {
			t.Errorf("expected %s, got %s", file, evt.Path)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for event")
	}

	// Ensure no more events trickle in for that burst
	select {
	case evt := <-w.Events():
		t.Fatalf("expected no more events, got one for %s", evt.Path)
	case <-time.After(100 * time.Millisecond):
		// Success
	}
}

func TestIsIgnoredDir(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{".git", true},
		{"node_modules", true},
		{".venv", true},
		{"src", false},
		{"src/components", false},
		{"./.git", true},
		{"myproject/vendor", true},
		{".hidden", true},
	}

	for _, tc := range tests {
		if got := isIgnoredDir(tc.path); got != tc.want {
			t.Errorf("isIgnoredDir(%s) = %v, want %v", tc.path, got, tc.want)
		}
	}
}
