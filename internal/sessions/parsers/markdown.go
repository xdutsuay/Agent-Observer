package parsers

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"agent-memory-mcp/internal/core"
)

var (
	// Matches common headers like "## User", "**User:**", "User:"
	userRegex = regexp.MustCompile(`(?i)^(?:#+\s+|\*\*)?(User|Human|You)(?:[:\*]*)\s*$`)
	// Matches "## Assistant", "**Assistant:**", "Claude:"
	agentRegex = regexp.MustCompile(`(?i)^(?:#+\s+|\*\*)?(Assistant|Agent|Claude|Cursor)(?:[:\*]*)\s*$`)
)

// ParseMarkdown generically parses a markdown transcript into SessionTurns.
func ParseMarkdown(sessionID, content string, mtime time.Time) []core.SessionTurn {
	lines := strings.Split(content, "\n")
	
	var turns []core.SessionTurn
	
	var currentRole string
	var currentBlock []string
	
	var pendingInput string
	
	flushBlock := func() {
		if len(currentBlock) == 0 {
			return
		}
		
		text := strings.TrimSpace(strings.Join(currentBlock, "\n"))
		if text == "" {
			return
		}
		
		if currentRole == "user" {
			pendingInput = text
		} else if currentRole == "agent" {
			turns = append(turns, core.SessionTurn{
				// Generate a deterministic pseudo-ID based on session and turn
				ID:            sessionID + "_turn_" + strconv.Itoa(len(turns)),
				SessionID:     sessionID,
				TurnNumber:    len(turns) + 1,
				UserInput:     pendingInput,
				AgentResponse: text,
				Timestamp:     mtime,
			})
			pendingInput = "" // reset
		}
		
		currentBlock = nil
	}

	for _, line := range lines {
		trim := strings.TrimSpace(line)
		
		if userRegex.MatchString(trim) {
			flushBlock()
			currentRole = "user"
			continue
		} else if agentRegex.MatchString(trim) {
			flushBlock()
			currentRole = "agent"
			continue
		}
		
		if currentRole != "" {
			currentBlock = append(currentBlock, line)
		}
	}
	flushBlock()

	return turns
}
