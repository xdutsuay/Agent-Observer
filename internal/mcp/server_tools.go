package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
)

func (s *Server) registerMoreTools() {
	s.mcpServer.AddTool(mcp.NewTool("add_memory",
		mcp.WithDescription("Legacy alias for remember."),
		mcp.WithString("path", mcp.Required(), mcp.Description("Project path")),
		mcp.WithString("kind", mcp.Required(), mcp.Description("Kind of memory")),
		mcp.WithString("text", mcp.Required(), mcp.Description("Content of the memory")),
	), s.handleAddMemory)

	s.mcpServer.AddTool(mcp.NewTool("list_projects",
		mcp.WithDescription("List all tracked projects with metadata (language, framework, health)."),
	), s.handleListProjects)

	s.mcpServer.AddTool(mcp.NewTool("switch_project_context",
		mcp.WithDescription("Get full context for a project by repo_id."),
		mcp.WithString("repo_id", mcp.Required(), mcp.Description("Project repo ID")),
	), s.handleSwitchProjectContext)

	s.mcpServer.AddTool(mcp.NewTool("smart_context",
		mcp.WithDescription("Get the most relevant memories for a specific task."),
		mcp.WithString("path", mcp.Required(), mcp.Description("Project path")),
		mcp.WithString("task", mcp.Required(), mcp.Description("Description of the task")),
		mcp.WithNumber("max_tokens", mcp.Description("Maximum tokens (default 3000)")),
	), s.handleSmartContext)

	s.mcpServer.AddTool(mcp.NewTool("refresh_relevance",
		mcp.WithDescription("Recompute relevance scores and classify noise for a project."),
		mcp.WithString("path", mcp.Description("Project path")),
	), s.handleRefreshRelevance)

	s.mcpServer.AddTool(mcp.NewTool("global_search",
		mcp.WithDescription("Search across all repos globally."),
		mcp.WithString("query", mcp.Required(), mcp.Description("Search query")),
		mcp.WithNumber("limit", mcp.Description("Limit of results (default 10)")),
	), s.handleGlobalSearch)

	s.mcpServer.AddTool(mcp.NewTool("failure_hotspots",
		mcp.WithDescription("Get the repositories with the most unresolved failures."),
		mcp.WithNumber("limit", mcp.Description("Limit of results (default 10)")),
	), s.handleFailureHotspots)

	s.mcpServer.AddTool(mcp.NewTool("memory_feedback",
		mcp.WithDescription("Record feedback on a memory to improve its relevance score."),
		mcp.WithString("memory_id", mcp.Required(), mcp.Description("Memory ID")),
		mcp.WithBoolean("useful", mcp.Required(), mcp.Description("Whether the memory was useful")),
		mcp.WithString("comment", mcp.Description("Optional comment explaining why")),
	), s.handleMemoryFeedback)

	s.mcpServer.AddTool(mcp.NewTool("get_pattern_report",
		mcp.WithDescription("Get a report of failure patterns and health score for a project."),
		mcp.WithString("path", mcp.Description("Optional project path. If omitted, global report.")),
	), s.handleGetPatternReport)

	s.mcpServer.AddTool(mcp.NewTool("get_related_memories",
		mcp.WithDescription("Find other memories that are semantically similar to a given memory."),
		mcp.WithString("memory_id", mcp.Required(), mcp.Description("Memory ID")),
		mcp.WithNumber("limit", mcp.Description("Limit of results (default 5)")),
	), s.handleGetRelatedMemories)

	s.mcpServer.AddTool(mcp.NewTool("find_similar_failures",
		mcp.WithDescription("Find failures in other repositories similar to failures in this one."),
		mcp.WithString("path", mcp.Required(), mcp.Description("Project path")),
		mcp.WithNumber("limit", mcp.Description("Limit of results (default 5)")),
	), s.handleFindSimilarFailures)

	// Session recall tools — search past Claude Code, Codex, and Cursor transcripts
	if s.sessionSvc != nil {
		s.mcpServer.AddTool(mcp.NewTool("recall_sessions",
			mcp.WithDescription("Search past session transcripts from Claude Code, Codex, and Cursor. Find what was discussed or tried in previous coding sessions."),
			mcp.WithString("query", mcp.Required(), mcp.Description("Search query — what you're looking for in past sessions")),
			mcp.WithNumber("limit", mcp.Description("Maximum results (default 10)")),
		), s.handleRecallSessions)

		s.mcpServer.AddTool(mcp.NewTool("list_sessions",
			mcp.WithDescription("List recently indexed session transcripts from Claude Code, Codex, and Cursor."),
			mcp.WithNumber("limit", mcp.Description("Maximum sessions to list (default 20)")),
		), s.handleListSessions)
	}
}

func (s *Server) handleAddMemory(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Alias for remember
	start := time.Now()
	args, _ := request.Params.Arguments.(map[string]interface{})
	var err error
	var result *mcp.CallToolResult

	defer func() {
		respText := ""
		if result != nil && len(result.Content) > 0 {
			if tc, ok := result.Content[0].(mcp.TextContent); ok {
				respText = tc.Text
			}
		}
		s.logUsage(ctx, "add_memory", args, respText, err, start)
	}()

	path, _ := args["path"].(string)
	kind, _ := args["kind"].(string)
	text, _ := args["text"].(string)

	repoID, err := s.memorySvc.ResolveRepo(path)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	memID, isNew, err := s.memorySvc.Remember(repoID, kind, text, "mcp", nil)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	msg := fmt.Sprintf("Memory stored in %s", repoID)
	if isNew {
		msg += fmt.Sprintf(" (id=%s)", memID)
	} else {
		msg += " (deduped)"
	}
	result = mcp.NewToolResultText(msg)
	return result, nil
}

func (s *Server) handleListProjects(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()
	var err error
	var result *mcp.CallToolResult

	defer func() {
		respText := ""
		if result != nil && len(result.Content) > 0 {
			if tc, ok := result.Content[0].(mcp.TextContent); ok {
				respText = tc.Text
			}
		}
		s.logUsage(ctx, "list_projects", nil, respText, err, start)
	}()

	repos, err := s.memorySvc.ListRepos(ctx)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if len(repos) == 0 {
		result = mcp.NewToolResultText("No projects tracked yet.")
		return result, nil
	}

	var text string
	for _, p := range repos {
		text += fmt.Sprintf("- **%s** [%s] — %s | %d errors | %s\n",
			p.ID, p.ID, p.Path, p.ErrorCount, p.Health)
	}

	result = mcp.NewToolResultText(text)
	return result, nil
}

func (s *Server) handleSwitchProjectContext(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()
	args, _ := request.Params.Arguments.(map[string]interface{})
	var err error
	var result *mcp.CallToolResult

	defer func() {
		respText := ""
		if result != nil && len(result.Content) > 0 {
			if tc, ok := result.Content[0].(mcp.TextContent); ok {
				respText = tc.Text
			}
		}
		s.logUsage(ctx, "switch_project_context", args, respText, err, start)
	}()

	repoID, _ := args["repo_id"].(string)
	
	repoCtx, err := s.memorySvc.GetRepoContext(ctx, repoID)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	text := fmt.Sprintf("## Project: %s\n\n## FAILURES\n%s\n\n## DECISIONS\n%s\n\n## ATTEMPTS\n%s\n\n## FACTS\n%s",
		repoID, repoCtx.Failures, repoCtx.Decisions, repoCtx.Attempts, repoCtx.Facts)

	result = mcp.NewToolResultText(text)
	return result, nil
}

func (s *Server) handleSmartContext(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()
	args, _ := request.Params.Arguments.(map[string]interface{})
	var err error
	var result *mcp.CallToolResult

	defer func() {
		respText := ""
		if result != nil && len(result.Content) > 0 {
			if tc, ok := result.Content[0].(mcp.TextContent); ok {
				respText = tc.Text
			}
		}
		s.logUsage(ctx, "smart_context", args, respText, err, start)
	}()

	path, _ := args["path"].(string)
	task, _ := args["task"].(string)
	maxTokens := 3000
	if mt, ok := args["max_tokens"].(float64); ok {
		maxTokens = int(mt)
	}

	repoID, err := s.memorySvc.ResolveRepo(path)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	sc, err := s.memorySvc.SmartContext(ctx, repoID, task, maxTokens)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if len(sc.Memories) == 0 {
		result = mcp.NewToolResultText(fmt.Sprintf("No relevant memories found for this task in %s.", repoID))
		return result, nil
	}

	text := fmt.Sprintf("## Smart Context for: %s\nProject: %s | %d memories | ~%d tokens\n\n",
		task[:minInt(len(task), 100)], repoID, len(sc.Memories), sc.TokenEstimate)
	
	for _, m := range sc.Memories {
		text += fmt.Sprintf("### [%s] (relevance: ?)\n%s\n\n", m.Kind, m.Content)
	}

	result = mcp.NewToolResultText(text)
	return result, nil
}

func (s *Server) handleRefreshRelevance(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()
	args, _ := request.Params.Arguments.(map[string]interface{})
	var err error
	var result *mcp.CallToolResult

	defer func() {
		respText := ""
		if result != nil && len(result.Content) > 0 {
			if tc, ok := result.Content[0].(mcp.TextContent); ok {
				respText = tc.Text
			}
		}
		s.logUsage(ctx, "refresh_relevance", args, respText, err, start)
	}()

	path, _ := args["path"].(string)
	var rid *string
	if path != "" {
		resolved, _ := s.memorySvc.ResolveRepo(path)
		rid = &resolved
	}

	relCount, noiseCount, err := s.memorySvc.RefreshRelevance(ctx, rid)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	result = mcp.NewToolResultText(fmt.Sprintf("Refreshed %d relevance scores. Classified %d memories as noise.", relCount, noiseCount))
	return result, nil
}

func (s *Server) handlePromoteSession(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()
	args, _ := request.Params.Arguments.(map[string]interface{})
	var err error
	var result *mcp.CallToolResult

	defer func() {
		respText := ""
		if result != nil && len(result.Content) > 0 {
			if tc, ok := result.Content[0].(mcp.TextContent); ok {
				respText = tc.Text
			}
		}
		s.logUsage(ctx, "promote_session", args, respText, err, start)
	}()

	repoID, _ := args["repo_id"].(string)
	turnID, _ := args["turn_id"].(string)
	kind, _ := args["kind"].(string)
	content, _ := args["content"].(string)

	meta := map[string]any{
		"promoted_from_turn": turnID,
	}

	id, _, err := s.memorySvc.Remember(repoID, kind, content, "session_promotion", meta)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	result = mcp.NewToolResultText(fmt.Sprintf("Promoted session turn %s into memory %s", turnID, id))
	return result, nil
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (s *Server) handleGlobalSearch(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()
	args, _ := request.Params.Arguments.(map[string]interface{})
	var result *mcp.CallToolResult
	var err error

	defer func() {
		respText := ""
		if result != nil && len(result.Content) > 0 {
			if tc, ok := result.Content[0].(mcp.TextContent); ok {
				respText = tc.Text
			}
		}
		s.logUsage(ctx, "global_search", args, respText, err, start)
	}()

	query, _ := args["query"].(string)
	limit := 10
	if lim, ok := args["limit"].(float64); ok {
		limit = int(lim)
	}

	mems, err := s.memorySvc.GlobalSearch(ctx, query, nil, limit)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if len(mems) == 0 {
		result = mcp.NewToolResultText("No memories found.")
		return result, nil
	}

	var text string
	for i, m := range mems {
		text += fmt.Sprintf("[%0.2f] (%s) [%s] %s\n%s\n", m.Score, m.RepoID, m.Kind, m.CreatedAt, m.Content)
		if i < len(mems)-1 {
			text += "\n---\n"
		}
	}

	result = mcp.NewToolResultText(text)
	return result, nil
}

func (s *Server) handleFailureHotspots(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()
	args, _ := request.Params.Arguments.(map[string]interface{})
	var result *mcp.CallToolResult
	var err error

	defer func() {
		respText := ""
		if result != nil && len(result.Content) > 0 {
			if tc, ok := result.Content[0].(mcp.TextContent); ok {
				respText = tc.Text
			}
		}
		s.logUsage(ctx, "failure_hotspots", args, respText, err, start)
	}()

	limit := 10
	if lim, ok := args["limit"].(float64); ok {
		limit = int(lim)
	}

	hotspots, err := s.memorySvc.FailureHotspots(ctx, limit)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if len(hotspots) == 0 {
		return mcp.NewToolResultText("No failure hotspots found."), nil
	}

	text := "## Failure Hotspots\n\n"
	for _, h := range hotspots {
		text += fmt.Sprintf("- **%s** (%v unresolved failures)\n", h["repo_id"], h["unresolved_failures"])
	}

	result = mcp.NewToolResultText(text)
	return result, nil
}

func (s *Server) handleMemoryFeedback(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()
	args, _ := request.Params.Arguments.(map[string]interface{})
	var result *mcp.CallToolResult
	var err error

	defer func() {
		respText := ""
		if result != nil && len(result.Content) > 0 {
			if tc, ok := result.Content[0].(mcp.TextContent); ok {
				respText = tc.Text
			}
		}
		s.logUsage(ctx, "memory_feedback", args, respText, err, start)
	}()

	memID, _ := args["memory_id"].(string)
	useful, _ := args["useful"].(bool)
	comment, _ := args["comment"].(string)

	err = s.memorySvc.RecordFeedback(ctx, memID, useful, comment)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	result = mcp.NewToolResultText("Feedback recorded successfully.")
	return result, nil
}

func (s *Server) handleGetPatternReport(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()
	args, _ := request.Params.Arguments.(map[string]interface{})
	var result *mcp.CallToolResult
	var err error

	defer func() {
		respText := ""
		if result != nil && len(result.Content) > 0 {
			if tc, ok := result.Content[0].(mcp.TextContent); ok {
				respText = tc.Text
			}
		}
		s.logUsage(ctx, "get_pattern_report", args, respText, err, start)
	}()

	path, hasPath := args["path"].(string)
	var repoID *string
	if hasPath && path != "" {
		rid, e := s.memorySvc.ResolveRepo(path)
		if e != nil {
			return mcp.NewToolResultError(e.Error()), nil
		}
		repoID = &rid
	}

	report, err := s.memorySvc.GetPatternReport(ctx, repoID)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Just format the JSON for MCP
	importJson := func(v any) string {
		b, _ := json.MarshalIndent(v, "", "  ")
		return string(b)
	}

	text := "## Pattern Report\n\n```json\n"
	text += importJson(report)
	text += "\n```"

	result = mcp.NewToolResultText(text)
	return result, nil
}

func (s *Server) handleGetRelatedMemories(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()
	args, _ := request.Params.Arguments.(map[string]interface{})
	var result *mcp.CallToolResult
	var err error

	defer func() {
		respText := ""
		if result != nil && len(result.Content) > 0 {
			if tc, ok := result.Content[0].(mcp.TextContent); ok {
				respText = tc.Text
			}
		}
		s.logUsage(ctx, "get_related_memories", args, respText, err, start)
	}()

	memID, _ := args["memory_id"].(string)
	limit := 5
	if lim, ok := args["limit"].(float64); ok {
		limit = int(lim)
	}

	mems, err := s.memorySvc.GetRelatedMemories(ctx, memID, limit)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if len(mems) == 0 {
		return mcp.NewToolResultText("No related memories found."), nil
	}

	var text string
	for i, m := range mems {
		text += fmt.Sprintf("[%0.2f] (%s) [%s] %s\n%s\n", m.Score, m.RepoID, m.Kind, m.CreatedAt, m.Content)
		if i < len(mems)-1 {
			text += "\n---\n"
		}
	}

	result = mcp.NewToolResultText(text)
	return result, nil
}

func (s *Server) handleRecallSessions(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()
	args, _ := request.Params.Arguments.(map[string]interface{})
	var result *mcp.CallToolResult
	var err error

	defer func() {
		respText := ""
		if result != nil && len(result.Content) > 0 {
			if tc, ok := result.Content[0].(mcp.TextContent); ok {
				respText = tc.Text
			}
		}
		s.logUsage(ctx, "recall_sessions", args, respText, err, start)
	}()

	query, _ := args["query"].(string)
	limit := 10
	if lim, ok := args["limit"].(float64); ok {
		limit = int(lim)
	}

	turns, err := s.sessionSvc.SearchSessions(ctx, query, limit)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if len(turns) == 0 {
		result = mcp.NewToolResultText("No matching session transcripts found.")
		return result, nil
	}

	var text string
	for i, t := range turns {
		text += fmt.Sprintf("### Turn %d (session: %s)\n**User:** %s\n**Agent:** %s\n**Time:** %s\n",
			t.TurnNumber,
			truncateStr(t.SessionID, 60),
			truncateStr(t.UserInput, 200),
			truncateStr(t.AgentResponse, 400),
			t.Timestamp.Format("2006-01-02 15:04"),
		)
		if i < len(turns)-1 {
			text += "\n---\n"
		}
	}

	result = mcp.NewToolResultText(text)
	return result, nil
}

func (s *Server) handleListSessions(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()
	args, _ := request.Params.Arguments.(map[string]interface{})
	var result *mcp.CallToolResult
	var err error

	defer func() {
		respText := ""
		if result != nil && len(result.Content) > 0 {
			if tc, ok := result.Content[0].(mcp.TextContent); ok {
				respText = tc.Text
			}
		}
		s.logUsage(ctx, "list_sessions", args, respText, err, start)
	}()

	limit := 20
	if lim, ok := args["limit"].(float64); ok {
		limit = int(lim)
	}

	// Search with empty query returns recent turns
	turns, err := s.sessionSvc.SearchSessions(ctx, "*", limit)
	if err != nil {
		// FTS might not like bare *, fall back to listing via a broad query
		turns, err = s.sessionSvc.SearchSessions(ctx, "the OR a OR is OR to OR and", limit)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
	}

	if len(turns) == 0 {
		result = mcp.NewToolResultText("No indexed sessions found. Sessions from ~/.claude/, ~/.codex/, and ~/.cursor/ are indexed automatically every 5 minutes.")
		return result, nil
	}

	// Group by session ID to show unique sessions
	seen := map[string]bool{}
	var text string
	for _, t := range turns {
		if seen[t.SessionID] {
			continue
		}
		seen[t.SessionID] = true
		text += fmt.Sprintf("- **%s** (turn %d, %s)\n", truncateStr(t.SessionID, 80), t.TurnNumber, t.Timestamp.Format("2006-01-02 15:04"))
	}

	result = mcp.NewToolResultText(fmt.Sprintf("## Indexed Sessions (%d unique)\n\n%s", len(seen), text))
	return result, nil
}

func truncateStr(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}

func (s *Server) handleFindSimilarFailures(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()
	args, _ := request.Params.Arguments.(map[string]interface{})
	var result *mcp.CallToolResult
	var err error

	defer func() {
		respText := ""
		if result != nil && len(result.Content) > 0 {
			if tc, ok := result.Content[0].(mcp.TextContent); ok {
				respText = tc.Text
			}
		}
		s.logUsage(ctx, "find_similar_failures", args, respText, err, start)
	}()

	path, _ := args["path"].(string)
	limit := 5
	if lim, ok := args["limit"].(float64); ok {
		limit = int(lim)
	}

	repoID, err := s.memorySvc.ResolveRepo(path)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	mems, err := s.memorySvc.FindSimilarFailures(ctx, repoID, limit)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if len(mems) == 0 {
		return mcp.NewToolResultText("No similar failures found."), nil
	}

	var text string
	for i, m := range mems {
		text += fmt.Sprintf("[%0.2f] (%s) [%s] %s\n%s\n", m.Score, m.RepoID, m.Kind, m.CreatedAt, m.Content)
		if i < len(mems)-1 {
			text += "\n---\n"
		}
	}

	result = mcp.NewToolResultText(text)
	return result, nil
}
