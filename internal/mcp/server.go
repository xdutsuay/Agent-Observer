package mcp

import (
	"context"
	"fmt"
	"time"

	"agent-memory-mcp/internal/app"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type Server struct {
	mcpServer *server.MCPServer
	memorySvc app.MemoryService
	usageSvc  app.UsageService
}

func NewServer(ms app.MemoryService, us app.UsageService) *Server {
	s := server.NewMCPServer("agent-memory", "1.0.0")

	srv := &Server{
		mcpServer: s,
		memorySvc: ms,
		usageSvc:  us,
	}

	srv.registerTools()
	srv.registerMoreTools()
	srv.registerResources()
	srv.registerPrompts()

	return srv
}

func (s *Server) ServeStdio() error {
	return server.ServeStdio(s.mcpServer)
}

func (s *Server) logUsage(ctx context.Context, toolName string, arguments map[string]interface{}, response string, err error, start time.Time) {
	ok := err == nil
	if err != nil {
		response = err.Error()
	}
	durationMS := float64(time.Since(start).Milliseconds())
	
	// Ideally we get client_name and version from ctx, but mcp-go might not expose initialization params directly per-request.
	// We'll use defaults for now.
	s.usageSvc.Record(ctx, "mcp", toolName, arguments, response, "unknown", "", "Unknown", durationMS, ok)
}

func (s *Server) registerTools() {
	s.mcpServer.AddTool(mcp.NewTool("remember",
		mcp.WithDescription("Store a memory entry for a project. Use 'decision', 'failure', 'fact', 'preference', 'attempt'."),
		mcp.WithString("path", mcp.Required(), mcp.Description("Project path or file path")),
		mcp.WithString("kind", mcp.Required(), mcp.Description("Kind of memory (failure, decision, attempt, fact, preference)")),
		mcp.WithString("content", mcp.Required(), mcp.Description("Content of the memory")),
	), s.handleRemember)

	s.mcpServer.AddTool(mcp.NewTool("search_memory",
		mcp.WithDescription("Search repo-scoped memories by keyword and semantic similarity."),
		mcp.WithString("query", mcp.Required(), mcp.Description("Search query")),
		mcp.WithString("path", mcp.Description("Optional project path to scope search")),
		mcp.WithNumber("limit", mcp.Description("Limit of results (default 10)")),
	), s.handleSearchMemory)

	s.mcpServer.AddTool(mcp.NewTool("promote_session",
		mcp.WithDescription("Promotes a session turn into a permanent memory (decision, fact, preference)."),
		mcp.WithString("repo_id", mcp.Required(), mcp.Description("Target repository ID.")),
		mcp.WithString("turn_id", mcp.Required(), mcp.Description("ID of the session turn to promote.")),
		mcp.WithString("kind", mcp.Required(), mcp.Description("Kind of memory (decision, fact, preference).")),
		mcp.WithString("content", mcp.Required(), mcp.Description("The exact content to store.")),
	), s.handlePromoteSession)

	s.mcpServer.AddTool(mcp.NewTool("get_repo_context",
		mcp.WithDescription("Get failures, decisions, facts, and recent attempts for a project."),
		mcp.WithString("path", mcp.Required(), mcp.Description("Project path")),
	), s.handleGetRepoContext)

	s.mcpServer.AddTool(mcp.NewTool("mark_failure_resolved",
		mcp.WithDescription("Mark a recurring failure signature as resolved."),
		mcp.WithString("path", mcp.Required(), mcp.Description("Project path")),
		mcp.WithString("signature", mcp.Required(), mcp.Description("Failure signature")),
	), s.handleMarkFailureResolved)

	s.mcpServer.AddTool(mcp.NewTool("forget",
		mcp.WithDescription("Soft-delete a memory by id or failure signature."),
		mcp.WithString("path", mcp.Description("Project path")),
		mcp.WithString("memory_id", mcp.Description("Memory ID to delete")),
		mcp.WithString("signature", mcp.Description("Failure signature to delete")),
	), s.handleForget)
}

func (s *Server) handleRemember(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()
	args, _ := request.Params.Arguments.(map[string]interface{})

	path, _ := args["path"].(string)
	kind, _ := args["kind"].(string)
	content, _ := args["content"].(string)

	var result *mcp.CallToolResult
	var err error

	defer func() {
		respText := ""
		if result != nil && len(result.Content) > 0 {
			if tc, ok := result.Content[0].(mcp.TextContent); ok {
				respText = tc.Text
			}
		}
		s.logUsage(ctx, "remember", args, respText, err, start)
	}()

	repoID, err := s.memorySvc.ResolveRepo(path)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	memID, isNew, err := s.memorySvc.Remember(repoID, kind, content, "mcp", nil)
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

func (s *Server) handleSearchMemory(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
		s.logUsage(ctx, "search_memory", args, respText, err, start)
	}()

	query, _ := args["query"].(string)
	path, hasPath := args["path"].(string)
	
	limit := 10
	if lim, ok := args["limit"].(float64); ok {
		limit = int(lim)
	}

	var repoID *string
	if hasPath && path != "" {
		rid, e := s.memorySvc.ResolveRepo(path)
		if e != nil {
			return mcp.NewToolResultError(e.Error()), nil
		}
		repoID = &rid
	}

	mems, err := s.memorySvc.Search(ctx, query, repoID, nil, limit)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if len(mems) == 0 {
		result = mcp.NewToolResultText("No memories found.")
		return result, nil
	}

	var text string
	for i, m := range mems {
		score := 0.0
		// the store search returns core.Memory which doesn't directly expose search score currently, 
		// but we'll format it with what we have.
		text += fmt.Sprintf("[%0.2f] (%s) %s\n%s\n", score, m.Kind, m.CreatedAt, m.Content)
		if i < len(mems)-1 {
			text += "\n---\n"
		}
	}

	result = mcp.NewToolResultText(text)
	return result, nil
}

func (s *Server) handleGetRepoContext(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
		s.logUsage(ctx, "get_repo_context", args, respText, err, start)
	}()

	path, _ := args["path"].(string)
	rid, err := s.memorySvc.ResolveRepo(path)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	repoCtx, err := s.memorySvc.GetRepoContext(ctx, rid)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	text := fmt.Sprintf("## FAILURES\n%s\n\n## DECISIONS\n%s\n\n## ATTEMPTS\n%s\n\n## FACTS\n%s",
		repoCtx.Failures, repoCtx.Decisions, repoCtx.Attempts, repoCtx.Facts)
	
	result = mcp.NewToolResultText(text)
	return result, nil
}

func (s *Server) handleMarkFailureResolved(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
		s.logUsage(ctx, "mark_failure_resolved", args, respText, err, start)
	}()

	path, _ := args["path"].(string)
	sig, _ := args["signature"].(string)

	rid, err := s.memorySvc.ResolveRepo(path)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	ok, err := s.memorySvc.MarkFailureResolved(ctx, rid, sig)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if ok {
		result = mcp.NewToolResultText("Resolved.")
	} else {
		result = mcp.NewToolResultText("Signature not found.")
	}
	return result, nil
}

func (s *Server) handleForget(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
		s.logUsage(ctx, "forget", args, respText, err, start)
	}()

	path, _ := args["path"].(string)
	memID, _ := args["memory_id"].(string)
	sig, _ := args["signature"].(string)

	rid := ""
	if path != "" {
		rid, _ = s.memorySvc.ResolveRepo(path)
	}

	n, err := s.memorySvc.Forget(ctx, memID, sig, rid)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	result = mcp.NewToolResultText(fmt.Sprintf("Forgot %d record(s).", n))
	return result, nil
}

func (s *Server) registerResources() {
	// The mcp-go library currently has s.mcpServer.AddResource
	// We might need to handle dynamic templates if the library supports it, or just use Prompts.
	// Memory resources were using `memory://{rid}/{kind}`.
	// mcp-go might have specific ways to register templates vs resources. Let's register a resource template.
	
	// Assuming mcp-go supports resource templates:
	// s.mcpServer.AddResourceTemplate(mcp.NewResourceTemplate(...))
	// If it doesn't, we can skip resources or implement a workaround. Let's implement prompts first.
}

func (s *Server) registerPrompts() {
	s.mcpServer.AddPrompt(mcp.NewPrompt("inject_memory_context",
		mcp.WithPromptDescription("Inject repo failure/decision context before starting a task."),
		mcp.WithArgument("path", mcp.ArgumentDescription("Project path"), mcp.RequiredArgument()),
	), s.handleInjectMemoryContext)
}

func (s *Server) handleInjectMemoryContext(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	start := time.Now()
	args := request.Params.Arguments
	var err error
	var responseText string

	defer func() {
		argsIf := make(map[string]interface{})
		for k, v := range args {
			argsIf[k] = v
		}
		s.logUsage(ctx, "inject_memory_context", argsIf, responseText, err, start)
	}()

	path, ok := args["path"]
	if !ok || path == "" {
		path = "."
	}

	rid, err := s.memorySvc.ResolveRepo(path)
	if err != nil {
		return nil, err
	}

	repoCtx, err := s.memorySvc.GetRepoContext(ctx, rid)
	if err != nil {
		return nil, err
	}

	body := fmt.Sprintf("You are working on repo `%s`. Relevant memory:\n\n### Failures\n%s\n\n### Decisions\n%s\n\n### Recent attempts\n%s\n\n### Facts\n%s\n",
		rid, repoCtx.Failures, repoCtx.Decisions, repoCtx.Attempts, repoCtx.Facts)

	responseText = body

	return &mcp.GetPromptResult{
		Description: fmt.Sprintf("Memory context for %s", rid),
		Messages: []mcp.PromptMessage{
			{
				Role: mcp.RoleUser,
				Content: mcp.TextContent{
					Type: "text",
					Text: body,
				},
			},
		},
	}, nil
}
