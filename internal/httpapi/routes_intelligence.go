package httpapi

import (
	"encoding/json"
	"net/http"
)

func (s *Server) registerIntelligenceRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/patterns", s.handleGetPatterns)
	mux.HandleFunc("GET /api/patterns/{repo_id}", s.handleGetPatternsRepo)
	mux.HandleFunc("GET /api/hotspots", s.handleGetHotspots)
	mux.HandleFunc("GET /api/related/{memory_id}", s.handleGetRelated)
	mux.HandleFunc("POST /api/v1/context/smart", s.handleSmartContext)
	mux.HandleFunc("POST /api/v1/context/generate", s.handleContextGenerate)
	mux.HandleFunc("POST /api/v1/feedback", s.handleFeedback)
	mux.HandleFunc("POST /api/v1/relevance/refresh", s.handleRelevanceRefresh)
	// Session recall endpoints
	mux.HandleFunc("POST /api/sessions/search", s.handleSessionSearch)
	mux.HandleFunc("POST /api/sessions/ingest", s.handleSessionIngest)
}

func (s *Server) handleGetPatterns(w http.ResponseWriter, r *http.Request) {
	report, err := MemoryServiceFromContext(r.Context()).GetPatternReport(r.Context(), nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	respondJSON(w, report)
}

func (s *Server) handleGetPatternsRepo(w http.ResponseWriter, r *http.Request) {
	repoID := r.PathValue("repo_id")
	report, err := MemoryServiceFromContext(r.Context()).GetPatternReport(r.Context(), &repoID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	respondJSON(w, report)
}

func (s *Server) handleGetHotspots(w http.ResponseWriter, r *http.Request) {
	hotspots, err := MemoryServiceFromContext(r.Context()).FailureHotspots(r.Context(), 10)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	respondJSON(w, map[string]any{"hotspots": hotspots})
}

func (s *Server) handleGetRelated(w http.ResponseWriter, r *http.Request) {
	memID := r.PathValue("memory_id")
	mems, err := MemoryServiceFromContext(r.Context()).GetRelatedMemories(r.Context(), memID, 5)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	respondJSON(w, map[string]any{"memories": mems})
}

func (s *Server) handleSmartContext(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RepoID    string `json:"repo_id"`
		Task      string `json:"task"`
		MaxTokens int    `json:"max_tokens"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4000
	}

	ctx, err := MemoryServiceFromContext(r.Context()).SmartContext(r.Context(), req.RepoID, req.Task, maxTokens)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	respondJSON(w, ctx)
}

func (s *Server) handleContextGenerate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RepoID      string `json:"repo_id"`
		ProjectPath string `json:"project_path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	err := MemoryServiceFromContext(r.Context()).GenerateContextFile(r.Context(), req.RepoID, req.ProjectPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	respondJSON(w, map[string]string{"status": "ok"})
}

func (s *Server) handleFeedback(w http.ResponseWriter, r *http.Request) {
	var req struct {
		MemoryID string `json:"memory_id"`
		Useful   bool   `json:"useful"`
		Context  string `json:"context"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	err := MemoryServiceFromContext(r.Context()).RecordFeedback(r.Context(), req.MemoryID, req.Useful, req.Context)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	respondJSON(w, map[string]string{"status": "ok"})
}

func (s *Server) handleRelevanceRefresh(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RepoID *string `json:"repo_id,omitempty"`
	}
	// Decode is optional since body might be empty
	json.NewDecoder(r.Body).Decode(&req)

	updated, demoted, err := MemoryServiceFromContext(r.Context()).RefreshRelevance(r.Context(), req.RepoID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	respondJSON(w, map[string]any{"updated": updated, "demoted": demoted})
}

func (s *Server) handleSessionSearch(w http.ResponseWriter, r *http.Request) {
	ss := SessionServiceFromContext(r.Context())
	if ss == nil {
		http.Error(w, "Session service not available", http.StatusServiceUnavailable)
		return
	}

	var req struct {
		Query string `json:"query"`
		Limit int    `json:"limit"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if req.Limit == 0 {
		req.Limit = 20
	}

	turns, err := ss.SearchSessions(r.Context(), req.Query, req.Limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	respondJSON(w, map[string]any{"results": turns})
}

func (s *Server) handleSessionIngest(w http.ResponseWriter, r *http.Request) {
	ms := MemoryServiceFromContext(r.Context())
	if ms == nil {
		http.Error(w, "Memory service not available", http.StatusServiceUnavailable)
		return
	}

	var req struct {
		RepoID  string `json:"repo_id"`
		TurnID  string `json:"turn_id"`
		Kind    string `json:"kind"`
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	meta := map[string]any{"promoted_from_turn": req.TurnID}
	id, _, err := ms.Remember(req.RepoID, req.Kind, req.Content, "session_promotion", meta)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	respondJSON(w, map[string]any{"id": id, "status": "promoted"})
}
