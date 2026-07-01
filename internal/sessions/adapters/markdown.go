package adapters

import (
	"context"
	"os"
	"strings"

	"agent-memory-mcp/internal/core"
	"agent-memory-mcp/internal/sessions/parsers"
)

type MarkdownAdapter struct{}

func (m *MarkdownAdapter) Supports(path string) bool {
	return strings.HasSuffix(strings.ToLower(path), ".md") || strings.Contains(strings.ToLower(path), "transcript")
}

func (m *MarkdownAdapter) Parse(ctx context.Context, path string) ([]core.SessionTurn, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return parsers.ParseMarkdown(path, string(content), info.ModTime()), nil
}
