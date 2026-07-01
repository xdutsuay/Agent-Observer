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
}

func (s *Server) handleGetPatterns(w http.ResponseWriter, r *http.Request) {
	report, err := s.memoryService.GetPatternReport(r.Context(), nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	respondJSON(w, report)
}

func (s *Server) handleGetPatternsRepo(w http.ResponseWriter, r *http.Request) {
	repoID := r.PathValue("repo_id")
	report, err := s.memoryService.GetPatternReport(r.Context(), &repoID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	respondJSON(w, report)
}

func (s *Server) handleGetHotspots(w http.ResponseWriter, r *http.Request) {
	hotspots, err := s.memoryService.FailureHotspots(r.Context(), 10)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	respondJSON(w, map[string]any{"hotspots": hotspots})
}

func (s *Server) handleGetRelated(w http.ResponseWriter, r *http.Request) {
	memID := r.PathValue("memory_id")
	mems, err := s.memoryService.GetRelatedMemories(r.Context(), memID, 5)
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

	ctx, err := s.memoryService.SmartContext(r.Context(), req.RepoID, req.Task, maxTokens)
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

	err := s.memoryService.GenerateContextFile(r.Context(), req.RepoID, req.ProjectPath)
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

	err := s.memoryService.RecordFeedback(r.Context(), req.MemoryID, req.Useful, req.Context)
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

	updated, demoted, err := s.memoryService.RefreshRelevance(r.Context(), req.RepoID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	respondJSON(w, map[string]any{"updated": updated, "demoted": demoted})
}
