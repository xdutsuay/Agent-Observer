package smartctx

import (
	"testing"

	"agent-memory-mcp/internal/core"
)

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"", 0},
		{"hello world", 2},       // 11 chars / 4 = 2
		{"a longer text here", 4}, // 18 chars / 4 = 4
	}

	for _, tt := range tests {
		got := estimateTokens(tt.input)
		if got != tt.expected {
			t.Errorf("estimateTokens(%q) = %d, want %d", tt.input, got, tt.expected)
		}
	}
}

func TestScoreMemory(t *testing.T) {
	decision := core.Memory{Kind: "decision", Content: "use Go for the rewrite", RelevanceScore: 0.8}
	attempt := core.Memory{Kind: "attempt", Content: "tried python but failed", RelevanceScore: 0.1}

	dScore := scoreMemory(decision, "Go rewrite")
	aScore := scoreMemory(attempt, "Go rewrite")

	if dScore <= aScore {
		t.Errorf("decision score (%f) should be higher than attempt score (%f)", dScore, aScore)
	}
}

func TestBuildPromptFragment_Empty(t *testing.T) {
	frag := buildPromptFragment([]core.Memory{}, "test-repo")
	if frag == "" {
		t.Error("expected non-empty fragment for empty memories")
	}
}

func TestBuildPromptFragment_WithMemories(t *testing.T) {
	mems := []core.Memory{
		{Kind: "decision", Content: "Use SQLite for storage"},
		{Kind: "fact", Content: "The repo has 10k lines"},
	}
	frag := buildPromptFragment(mems, "my-repo")

	if len(frag) == 0 {
		t.Error("expected non-empty fragment")
	}
	if !contains(frag, "Key Decisions") {
		t.Error("expected 'Key Decisions' section")
	}
	if !contains(frag, "Known Facts") {
		t.Error("expected 'Known Facts' section")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && len(substr) > 0 && indexOf(s, substr) >= 0
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
