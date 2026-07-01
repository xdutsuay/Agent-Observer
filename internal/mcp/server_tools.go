package mcp

import (
	"context"
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

	// Note: global_search, find_similar_failures, get_related_memories, get_pattern_report,
	// failure_hotspots, memory_feedback are omitted for brevity in this foundational port,
	// but can be stubbed or fully implemented if the Service layer supports them.
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
		task[:min(len(task), 100)], repoID, len(sc.Memories), sc.TokenEstimate)
	
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
