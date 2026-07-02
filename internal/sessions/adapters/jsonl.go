package adapters

import (
	"context"
	"os"
	"strings"

	"agent-memory-mcp/internal/core"
	"agent-memory-mcp/internal/sessions/parsers"
)

type JsonlAdapter struct{}

func (a *JsonlAdapter) Supports(path string) bool {
	return strings.HasSuffix(path, ".jsonl")
}

func (a *JsonlAdapter) Parse(ctx context.Context, path string) ([]core.SessionTurn, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	contentBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	content := string(contentBytes)

	// Use the file path as the session ID
	sessionID := path

	return parsers.ParseJSONL(sessionID, content, info.ModTime()), nil
}
