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
	repos, _ := MemoryServiceFromContext(r.Context()).ListRepos(r.Context())
	repoCount := 0
	if repos != nil {
		repoCount = len(repos)
	}
	respondJSON(w, map[string]any{
		"running":            true,
		"watcher_active":     true,
		"version":            "0.4.0-go",
		"repos_count":        repoCount,
		"data_root":          "",
		"llm_provider":       "none",
		"nvidia_configured":  false,
		"watch_paths":        []string{},
	})
}

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, map[string]any{
		"mcp_enabled": true,
		"http_enabled": true,
	})
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, map[string]any{
		"total_memories":          0,
		"total_interactions":      0,
		"agent_processes_detected": 0,
		"activity_score":          0.0,
	})
}

func (s *Server) handleDiskUsage(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, map[string]any{
		"data_root": "",
		"overall": map[string]any{
			"data_root_bytes":                0,
			"data_root_bytes_human":          "0 B",
			"memory_db_bytes":                0,
			"usage_db_bytes":                 0,
			"legacy_markdown_bytes":          0,
			"total_memory_attributed_bytes":  0,
			"total_memory_attributed_human":  "0 B",
			"total_workspace_bytes":          0,
			"total_workspace_human":          "0 B",
			"project_count":                  0,
		},
		"breakdown":       map[string]int{},
		"breakdown_human": map[string]string{},
		"projects":        []any{},
	})
}

func (s *Server) handleUsageSummary(w http.ResponseWriter, r *http.Request) {
	summary, err := UsageServiceFromContext(r.Context()).Summary(r.Context())
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
	
	sessions, err := UsageServiceFromContext(r.Context()).ListSessions(r.Context(), limit)
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
	
	interactions, err := UsageServiceFromContext(r.Context()).ListInteractions(r.Context(), limit, hostIDEPtr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	respondJSON(w, map[string]any{"interactions": interactions})
}
