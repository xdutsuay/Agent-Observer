package parsers

import (
	"testing"
	"time"
)

func TestParseJSONL_ClaudeCodeFormat(t *testing.T) {
	// Claude Code uses nested message.content with content blocks
	input := `{"type":"user","message":{"role":"user","content":[{"type":"text","text":"Fix the auth bug"}],"created_at":"2025-01-15T10:30:00Z"}}
{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"I'll look at the auth module."},{"type":"tool_use","name":"read_file","input":{"path":"auth.go"}}],"created_at":"2025-01-15T10:30:05Z"}}`

	mtime := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	turns := ParseJSONL("test-session", input, mtime)

	if len(turns) != 1 {
		t.Fatalf("expected 1 turn, got %d", len(turns))
	}

	turn := turns[0]
	if turn.UserInput != "Fix the auth bug" {
		t.Errorf("unexpected user input: %q", turn.UserInput)
	}
	if turn.AgentResponse == "" {
		t.Error("agent response should not be empty")
	}
	// Should contain both text and tool_use
	if turn.TurnNumber != 1 {
		t.Errorf("expected turn number 1, got %d", turn.TurnNumber)
	}
}

func TestParseJSONL_FlatFormat(t *testing.T) {
	// Cursor/generic flat format
	input := `{"role":"user","content":"How do I test this?"}
{"role":"assistant","content":"You can use go test."}`

	mtime := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	turns := ParseJSONL("flat-session", input, mtime)

	if len(turns) != 1 {
		t.Fatalf("expected 1 turn, got %d", len(turns))
	}
	if turns[0].UserInput != "How do I test this?" {
		t.Errorf("unexpected user input: %q", turns[0].UserInput)
	}
	if turns[0].AgentResponse != "You can use go test." {
		t.Errorf("unexpected agent response: %q", turns[0].AgentResponse)
	}
}

func TestParseJSONL_ContentBlocks_ToolResult(t *testing.T) {
	input := `{"type":"user","message":{"role":"user","content":"Check the file"}}
{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"Reading file..."},{"type":"tool_result","content":"file contents here"}]}}`

	turns := ParseJSONL("tool-session", input, time.Now())

	if len(turns) != 1 {
		t.Fatalf("expected 1 turn, got %d", len(turns))
	}
	// Agent response should include both text and tool_result
	if turns[0].AgentResponse == "" {
		t.Error("agent response should not be empty")
	}
}

func TestParseJSONL_SkipsMalformed(t *testing.T) {
	input := `not json at all
{"type":"user","message":{"role":"user","content":"hello"}}
{broken json
{"type":"assistant","message":{"role":"assistant","content":"world"}}`

	turns := ParseJSONL("skip-session", input, time.Now())

	if len(turns) != 1 {
		t.Fatalf("expected 1 turn (malformed lines skipped), got %d", len(turns))
	}
}

func TestParseJSONL_StringContent(t *testing.T) {
	// message.content as plain string instead of array
	input := `{"type":"user","message":{"role":"user","content":"plain string user"}}
{"type":"assistant","message":{"role":"assistant","content":"plain string response"}}`

	turns := ParseJSONL("string-session", input, time.Now())

	if len(turns) != 1 {
		t.Fatalf("expected 1 turn, got %d", len(turns))
	}
	if turns[0].UserInput != "plain string user" {
		t.Errorf("unexpected user input: %q", turns[0].UserInput)
	}
}
