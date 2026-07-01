package logs

import (
	"testing"
)

func TestExtractError_Python(t *testing.T) {
	content := `Some log output
Traceback (most recent call last):
  File "main.py", line 10, in <module>
    1 / 0
ZeroDivisionError: division by zero
More logs here`

	err := ExtractError(content)
	if err == nil {
		t.Fatal("expected to extract error, got nil")
	}
	if err.Language != "python" {
		t.Errorf("expected python, got %s", err.Language)
	}
	if err.Message != "ZeroDivisionError: division by zero" {
		t.Errorf("expected ZeroDivisionError, got %s", err.Message)
	}
}

func TestExtractError_Node(t *testing.T) {
	content := `node output
TypeError: Cannot read properties of undefined (reading 'foo')
    at Object.<anonymous> (/app/index.js:2:15)
    at Module._compile (node:internal/modules/cjs/loader:1376:14)`

	err := ExtractError(content)
	if err == nil {
		t.Fatal("expected to extract error, got nil")
	}
	if err.Language != "node" {
		t.Errorf("expected node, got %s", err.Language)
	}
	if err.Message != "TypeError: Cannot read properties of undefined (reading 'foo')" {
		t.Errorf("expected TypeError, got %s", err.Message)
	}
}

func TestExtractError_Go(t *testing.T) {
	content := `panic: runtime error: index out of range [1] with length 1

goroutine 1 [running]:
main.main()
	/app/main.go:5 +0x18
exit status 2`

	err := ExtractError(content)
	if err == nil {
		t.Fatal("expected to extract error, got nil")
	}
	if err.Language != "go" {
		t.Errorf("expected go, got %s", err.Language)
	}
	if err.Message != "panic: runtime error: index out of range [1] with length 1" {
		t.Errorf("expected panic message, got %s", err.Message)
	}
}

func TestExtractError_None(t *testing.T) {
	content := `just a normal log file
everything is ok`
	if ExtractError(content) != nil {
		t.Fatal("expected nil, got error")
	}
}
