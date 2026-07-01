package logs

import (
	"regexp"
	"strings"
)

var (
	// Minimal set of regexes for python/node/go stack traces
	pyTrace   = regexp.MustCompile(`(?m)^Traceback \(most recent call last\):[\s\S]+?^\w+Error:.*$`)
	nodeTrace = regexp.MustCompile(`(?m)^\w+Error:.*[\r\n]+(\s+at .*[\r\n]+)+`)
	goTrace   = regexp.MustCompile(`(?m)^panic: .*[\s\S]+?goroutine \d+ \[.*\]:[\s\S]+?\.go:\d+.*$`)
)

// ExtractedError represents a parsed error from a log or file.
type ExtractedError struct {
	Language string
	Message  string
	Full     string
}

// ExtractError searches the text for known stack trace formats.
func ExtractError(content string) *ExtractedError {
	if match := pyTrace.FindString(content); match != "" {
		lines := strings.Split(match, "\n")
		msg := lines[len(lines)-1]
		return &ExtractedError{
			Language: "python",
			Message:  msg,
			Full:     match,
		}
	}

	if match := nodeTrace.FindString(content); match != "" {
		lines := strings.Split(match, "\n")
		return &ExtractedError{
			Language: "node",
			Message:  lines[0],
			Full:     match,
		}
	}

	if match := goTrace.FindString(content); match != "" {
		lines := strings.Split(match, "\n")
		return &ExtractedError{
			Language: "go",
			Message:  lines[0],
			Full:     match,
		}
	}

	return nil
}
