package watcher

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// FileEvent represents a debounced file system event.
type FileEvent struct {
	Path      string
	Op        fsnotify.Op
	Timestamp time.Time
}

// Watcher recursively watches directories and debounces file events.
type Watcher struct {
	watcher *fsnotify.Watcher
	events  chan FileEvent
	errors  chan error
	
	debounceDur time.Duration
	mu          sync.Mutex
	pending     map[string]*time.Timer
}

// New creates a new debouncing watcher.
func New(debounceDur time.Duration) (*Watcher, error) {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	return &Watcher{
		watcher:     fw,
		events:      make(chan FileEvent, 100),
		errors:      make(chan error, 100),
		debounceDur: debounceDur,
		pending:     make(map[string]*time.Timer),
	}, nil
}

// Events returns the channel of debounced events.
func (w *Watcher) Events() <-chan FileEvent {
	return w.events
}

// Errors returns the channel of watcher errors.
func (w *Watcher) Errors() <-chan error {
	return w.errors
}

// AddRecursive adds a directory and all its non-ignored subdirectories to the watcher.
func (w *Watcher) AddRecursive(root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return nil
		}
		if isIgnoredDir(path) {
			return filepath.SkipDir
		}
		return w.watcher.Add(path)
	})
}

// Start begins processing fsnotify events and applies debouncing.
func (w *Watcher) Start(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				w.watcher.Close()
				return
			case event, ok := <-w.watcher.Events:
				if !ok {
					return
				}
				
				// Handle new directory creation by watching it
				if event.Has(fsnotify.Create) {
					if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
						if !isIgnoredDir(event.Name) {
							w.watcher.Add(event.Name)
						}
					}
				}

				w.debounceEvent(event)
			case err, ok := <-w.watcher.Errors:
				if !ok {
					return
				}
				select {
				case w.errors <- err:
				default:
					log.Printf("watcher error channel full, dropping error: %v", err)
				}
			}
		}
	}()
}

func (w *Watcher) debounceEvent(event fsnotify.Event) {
	// Only care about file writes or creates for ingest
	if !event.Has(fsnotify.Write) && !event.Has(fsnotify.Create) {
		return
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	if timer, exists := w.pending[event.Name]; exists {
		timer.Stop()
	}

	w.pending[event.Name] = time.AfterFunc(w.debounceDur, func() {
		w.mu.Lock()
		delete(w.pending, event.Name)
		w.mu.Unlock()

		select {
		case w.events <- FileEvent{
			Path:      event.Name,
			Op:        event.Op,
			Timestamp: time.Now(),
		}:
		default:
			log.Printf("watcher events channel full, dropping event for %s", event.Name)
		}
	})
}

func isIgnoredDir(path string) bool {
	base := filepath.Base(path)
	switch base {
	case ".git", "node_modules", "vendor", ".venv", "__pycache__", ".agent-memory", ".cursor", ".vscode":
		return true
	}
	return false
}
