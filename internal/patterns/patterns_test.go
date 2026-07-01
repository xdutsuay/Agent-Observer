package patterns

import (
	"testing"
)

func TestCategorizeError(t *testing.T) {
	tests := []struct {
		content  string
		expected string
	}{
		{"Connection refused on port 8080", "Connection Error"},
		{"TimeoutError after 30 seconds", "Timeout"},
		{"ImportError: no module named foo", "Import Error"},
		{"SyntaxError on line 42", "Syntax Error"},
		{"TypeError: cannot add int and str", "Type Error"},
		{"Permission denied for /etc/shadow", "Permission Error"},
		{"404 page not found", "Not Found"},
		{"out of memory", "Memory Error"},
		{"null pointer dereference", "Null Reference"},
		{"assertion failed: expected true", "Assertion Error"},
		{"test failure in TestFoo", "Test Failure"},
		{"build failed: compile error", "Build Error"},
		{"something completely unrelated", "Other"},
	}

	for _, tt := range tests {
		got := categorizeError(tt.content)
		if got != tt.expected {
			t.Errorf("categorizeError(%q) = %q, want %q", tt.content, got, tt.expected)
		}
	}
}

func TestExtractFileRefs(t *testing.T) {
	text := "The error is in src/main.go and lib/utils/helper.py, see also config.yaml"
	refs := extractFileRefs(text)

	if len(refs) == 0 {
		t.Error("expected at least one file reference extracted")
	}

	found := false
	for _, r := range refs {
		if r == "src/main.go" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected to find 'src/main.go' in refs: %v", refs)
	}
}
