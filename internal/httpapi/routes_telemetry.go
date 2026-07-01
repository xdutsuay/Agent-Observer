package httpapi

import (
	"net/http"
	"strconv"
)

func (s *Server) registerTelemetryRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/status", s.handleStatus)
	mux.HandleFunc("GET /api/config", s.handleConfig)
	mux.HandleFunc("GET /api/metrics", s.handleMetrics)
	mux.HandleFunc("GET /api/disk-usage", s.handleDiskUsage)
	mux.HandleFunc("GET /api/usage/summary", s.handleUsageSummary)
	mux.HandleFunc("GET /api/usage/sessions", s.handleUsageSessions)
	mux.HandleFunc("GET /api/usage/interactions", s.handleUsageInteractions)
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, map[string]any{
		"watcher_active": true,
		"version": "0.3.0",
	})
}

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, map[string]any{
		"mcp_enabled": true,
		"http_enabled": true,
	})
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	// Stub for metrics
	respondJSON(w, map[string]any{
		"metrics": map[string]any{
			"total_memories": 0,
			"total_interactions": 0,
		},
	})
}

func (s *Server) handleDiskUsage(w http.ResponseWriter, r *http.Request) {
	// Stub for disk usage
	respondJSON(w, map[string]any{
		"total_bytes": 0,
		"memory_db_bytes": 0,
		"usage_db_bytes": 0,
	})
}

func (s *Server) handleUsageSummary(w http.ResponseWriter, r *http.Request) {
	summary, err := s.usageService.Summary(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	respondJSON(w, summary)
}

func (s *Server) handleUsageSessions(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := 100
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}
	
	sessions, err := s.usageService.ListSessions(r.Context(), limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	respondJSON(w, map[string]any{"sessions": sessions})
}

func (s *Server) handleUsageInteractions(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := 100
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	hostIDE := r.URL.Query().Get("host_ide")
	var hostIDEPtr *string
	if hostIDE != "" {
		hostIDEPtr = &hostIDE
	}
	
	interactions, err := s.usageService.ListInteractions(r.Context(), limit, hostIDEPtr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	respondJSON(w, map[string]any{"interactions": interactions})
}
