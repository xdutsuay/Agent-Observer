package parsers

import (
	"testing"
	"time"
)

func TestParseMarkdown(t *testing.T) {
	content := `
# Project Update

## User
Can you help me fix the memory leak?

## Assistant
Sure, let's look at the watcher code.

**User:**
Here is the code.
Line 2

**Claude:**
I see the issue.
`

	mtime := time.Now()
	turns := ParseMarkdown("sess-1", content, mtime)

	if len(turns) != 2 {
		t.Fatalf("expected 2 turns, got %d", len(turns))
	}

	if turns[0].UserInput != "Can you help me fix the memory leak?" {
		t.Errorf("unexpected user input 1: %q", turns[0].UserInput)
	}
	if turns[0].AgentResponse != "Sure, let's look at the watcher code." {
		t.Errorf("unexpected agent response 1: %q", turns[0].AgentResponse)
	}

	if turns[1].UserInput != "Here is the code.\nLine 2" {
		t.Errorf("unexpected user input 2: %q", turns[1].UserInput)
	}
	if turns[1].AgentResponse != "I see the issue." {
		t.Errorf("unexpected agent response 2: %q", turns[1].AgentResponse)
	}
}
