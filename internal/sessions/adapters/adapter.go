package adapters

import (
	"context"

	"agent-memory-mcp/internal/core"
)

// Adapter interface for parsing sessions from files.
type Adapter interface {
	// Parse reads a file at the given path and extracts SessionTurns.
	Parse(ctx context.Context, path string) ([]core.SessionTurn, error)
	// Supports checks if this adapter can handle the given file path.
	Supports(path string) bool
}
