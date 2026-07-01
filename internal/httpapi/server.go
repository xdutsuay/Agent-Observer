package httpapi

import (
	"net/http"

	"agent-memory-mcp/internal/app"
)

type Server struct {
	memoryService app.MemoryService
	usageService  app.UsageService
}

func NewServer(ms app.MemoryService, us app.UsageService) *Server {
	return &Server{
		memoryService: ms,
		usageService:  us,
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		respondJSON(w, map[string]string{"status": "ok"})
	})

	s.registerMemoryRoutes(mux)
	s.registerIntelligenceRoutes(mux)
	s.registerTelemetryRoutes(mux)
	s.registerWsRoutes(mux)

	// Fallback for static UI
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		// Just returning a simple response for now.
		// In a real build, we'd serve the React bundle here.
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte("<html><body><h1>Agent Memory MCP Go Port</h1><p>API is running.</p></body></html>"))
	})

	return mux
}
