package parsers

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"agent-memory-mcp/internal/core"
)

// Maximum lengths for truncating tool content to keep turns manageable.
const (
	toolResultMax = 600
	toolInputMax  = 160
)

// genericMsg handles both flat and nested JSONL formats.
type genericMsg struct {
	// Flat format (Cursor-style)
	Type   string `json:"type"`
	Source string `json:"source"`
	Role   string `json:"role"`

	// Flat content (string)
	ContentStr string `json:"-"`

	// Claude Code / Codex nested format
	Message *nestedMessage `json:"message,omitempty"`
}

type nestedMessage struct {
	Role      string          `json:"role"`
	Content   json.RawMessage `json:"content"` // string or []contentBlock
	CreatedAt string          `json:"created_at,omitempty"`
}

type contentBlock struct {
	Type  string `json:"type"`
	Text  string `json:"text,omitempty"`
	Name  string `json:"name,omitempty"`  // tool_use
	Input any    `json:"input,omitempty"` // tool_use
	// tool_result
	Content    json.RawMessage `json:"content,omitempty"`
	ToolUseID  string          `json:"tool_use_id,omitempty"`
	IsError    bool            `json:"is_error,omitempty"`
}

// UnmarshalJSON handles the "content" field being either a string or missing.
func (m *genericMsg) UnmarshalJSON(data []byte) error {
	// First try with a raw content field
	type Alias struct {
		Type    string          `json:"type"`
		Source  string          `json:"source"`
		Role    string          `json:"role"`
		Content json.RawMessage `json:"content"`
		Message *nestedMessage  `json:"message,omitempty"`
	}
	var a Alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	m.Type = a.Type
	m.Source = a.Source
	m.Role = a.Role
	m.Message = a.Message

	// Try to parse top-level content as a string (flat format)
	if len(a.Content) > 0 {
		var s string
		if err := json.Unmarshal(a.Content, &s); err == nil {
			m.ContentStr = s
		}
	}
	return nil
}

// blocksToText normalizes content blocks into a single text string.
func blocksToText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}

	// Try as plain string first
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}

	// Try as array of content blocks
	var blocks []contentBlock
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return ""
	}

	var parts []string
	for _, b := range blocks {
		switch b.Type {
		case "text":
			if t := strings.TrimSpace(b.Text); t != "" {
				parts = append(parts, t)
			}
		case "thinking":
			if t := strings.TrimSpace(b.Text); t != "" {
				parts = append(parts, "[thinking] "+truncate(t, toolResultMax))
			}
		case "tool_use":
			inputStr := ""
			if b.Input != nil {
				ij, _ := json.Marshal(b.Input)
				inputStr = truncate(string(ij), toolInputMax)
			}
			parts = append(parts, fmt.Sprintf("tool_use: %s(%s)", b.Name, inputStr))
		case "tool_result":
			text := extractToolResultText(b.Content)
			if b.IsError {
				parts = append(parts, "tool_error: "+truncate(text, toolResultMax))
			} else if text != "" {
				parts = append(parts, "tool_result: "+truncate(text, toolResultMax))
			}
		}
	}
	return strings.Join(parts, "\n")
}

func extractToolResultText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	// Try as string
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	// Try as array of blocks
	var blocks []contentBlock
	if json.Unmarshal(raw, &blocks) == nil {
		var parts []string
		for _, b := range blocks {
			if b.Text != "" {
				parts = append(parts, b.Text)
			}
		}
		return strings.Join(parts, "\n")
	}
	return ""
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}

func ParseJSONL(sessionID, content string, mtime time.Time) []core.SessionTurn {
	lines := strings.Split(content, "\n")
	var turns []core.SessionTurn

	var pendingInput string

	for _, line := range lines {
		trim := strings.TrimSpace(line)
		if trim == "" {
			continue
		}

		var msg genericMsg
		if err := json.Unmarshal([]byte(trim), &msg); err != nil {
			continue // Skip malformed lines
		}

		var role, text string
		var ts time.Time

		// === Claude Code / Codex nested format ===
		if msg.Message != nil {
			role = strings.ToLower(msg.Message.Role)
			text = blocksToText(msg.Message.Content)
			if msg.Message.CreatedAt != "" {
				ts, _ = time.Parse(time.RFC3339Nano, msg.Message.CreatedAt)
			}
			// Claude Code uses type field: "user", "assistant", "summary"
			if role == "" {
				switch strings.ToLower(msg.Type) {
				case "user":
					role = "user"
				case "assistant", "summary":
					role = "agent"
				}
			}
		} else {
			// === Flat format (Cursor / generic) ===
			role = strings.ToLower(msg.Role)
			text = strings.TrimSpace(msg.ContentStr)
			if role == "" {
				if msg.Type == "USER_INPUT" || msg.Source == "USER_EXPLICIT" {
					role = "user"
				} else if msg.Type == "PLANNER_RESPONSE" || msg.Type == "MODEL" || msg.Source == "MODEL" {
					role = "agent"
				}
			}
		}

		// Map "assistant" to "agent" for consistency
		if role == "assistant" {
			role = "agent"
		}

		if text == "" {
			continue
		}

		if ts.IsZero() {
			ts = mtime
		}

		if role == "user" {
			pendingInput = text
		} else if role == "agent" {
			turns = append(turns, core.SessionTurn{
				ID:            sessionID + "_turn_" + strconv.Itoa(len(turns)),
				SessionID:     sessionID,
				TurnNumber:    len(turns) + 1,
				UserInput:     pendingInput,
				AgentResponse: text,
				Timestamp:     ts,
			})
			pendingInput = "" // reset
		}
	}

	return turns
}
