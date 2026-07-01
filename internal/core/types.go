package core

import "time"

type Memory struct {
	ID             string                 `json:"id"`
	RepoID         string                 `json:"repo_id"`
	Kind           string                 `json:"kind"`
	Content        string                 `json:"content"`
	Source         string                 `json:"source"`
	Metadata       map[string]any         `json:"metadata"`
	SessionID      string                 `json:"session_id,omitempty"`
	Summary        string                 `json:"summary,omitempty"`
	CreatedAt      string                 `json:"created_at"`
	AccessCount    int                    `json:"access_count,omitempty"`
	LastAccessed   string                 `json:"last_accessed,omitempty"`
	RelevanceScore float64                `json:"relevance_score,omitempty"`
	QualityTier    string                 `json:"quality_tier,omitempty"`
	Score          float64                `json:"score,omitempty"`
	Match          string                 `json:"match,omitempty"`
	Extra          map[string]interface{} `json:"extra,omitempty"`
}

type Repo struct {
	ID        string `json:"id"`
	Path      string `json:"path"`
	CreatedAt string `json:"created_at"`
}

type RepoListing struct {
	ID           string `json:"id"`
	Path         string `json:"path"`
	Health       string `json:"health"`
	ErrorCount   int    `json:"error_count"`
	LastModified string `json:"last_modified,omitempty"`
}

type RepoContext struct {
	Failures          string `json:"failures"`
	Decisions         string `json:"decisions"`
	Attempts          string `json:"attempts"`
	Facts             string `json:"facts"`
	FailureSignatures string `json:"failure_signatures"`
}

type SmartContext struct {
	Memories             []Memory `json:"memories"`
	TokenEstimate        int      `json:"token_estimate"`
	SystemPromptFragment string   `json:"system_prompt_fragment"`
}

type UsageInteraction struct {
	ID              string  `json:"id"`
	SessionID       string  `json:"session_id"`
	Transport       string  `json:"transport"`
	Method          string  `json:"method"`
	ClientName      string  `json:"client_name"`
	ClientVersion   string  `json:"client_version"`
	HostIDE         string  `json:"host_ide"`
	QuerySummary    string  `json:"query_summary"`
	QueryJSON       string  `json:"query_json"`
	ResponsePreview string  `json:"response_preview"`
	DurationMS      float64 `json:"duration_ms"`
	OK              bool    `json:"ok"`
	CreatedAt       string  `json:"created_at"`
}

type UsageSession struct {
	ID            string `json:"id"`
	ClientName    string `json:"client_name"`
	ClientVersion string `json:"client_version"`
	HostIDE       string `json:"host_ide"`
	Transport     string `json:"transport"`
	ConnectedAt   string `json:"connected_at"`
	LastSeenAt    string `json:"last_seen_at"`
	CallCount     int    `json:"call_count"`
	LastCall      string `json:"last_call,omitempty"`
}

type CountByName struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type RunningIDE struct {
	Label        string `json:"label"`
	ProcessCount int    `json:"process_count"`
}

type UsageSummary struct {
	TotalInteractions int          `json:"total_interactions"`
	Last24H           int          `json:"last_24h"`
	Reads             int          `json:"reads"`
	Writes            int          `json:"writes"`
	ByMethod          []CountByName `json:"by_method"`
	ByHostIDE         []CountByName `json:"by_host_ide"`
	ByTransport       []CountByName `json:"by_transport"`
	RunningIDEs       []RunningIDE `json:"running_ides"`
}

// SessionTurn represents a single back-and-forth interaction between the user and an agent.
type SessionTurn struct {
	ID            string    `json:"id"`
	SessionID     string    `json:"session_id"`
	TurnNumber    int       `json:"turn_number"`
	UserInput     string    `json:"user_input"`
	AgentResponse string    `json:"agent_response"`
	Timestamp     time.Time `json:"timestamp"`
}

// IndexState tracks the last modified time of a transcript file.
type IndexState struct {
	FilePath     string    `json:"file_path"`
	LastModified time.Time `json:"last_modified"`
}
