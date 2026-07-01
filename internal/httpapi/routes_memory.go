package httpapi

import (
	"encoding/json"
	"net/http"
)

func (s *Server) registerMemoryRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/projects", s.handleListProjects)
	mux.HandleFunc("GET /api/projects/{repo_id}", s.handleGetProject)
	mux.HandleFunc("GET /api/memory/{repo_id}", s.handleGetRepoMemory)
	mux.HandleFunc("POST /api/memory/{repo_id}/{kind}", s.handleAddMemory)
	mux.HandleFunc("GET /api/memories", s.handleListMemories)
	mux.HandleFunc("POST /api/search", s.handleSearch)
	mux.HandleFunc("POST /api/search/global", s.handleGlobalSearch)
}

func (s *Server) handleListProjects(w http.ResponseWriter, r *http.Request) {
	repos, err := s.memoryService.ListRepos(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	respondJSON(w, repos)
}

func (s *Server) handleGetProject(w http.ResponseWriter, r *http.Request) {
	repoID := r.PathValue("repo_id")
	ctx, err := s.memoryService.GetRepoContext(r.Context(), repoID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	respondJSON(w, ctx)
}

func (s *Server) handleGetRepoMemory(w http.ResponseWriter, r *http.Request) {
	repoID := r.PathValue("repo_id")
	memories, err := s.memoryService.ListMemories(r.Context(), &repoID, nil, 100)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	respondJSON(w, map[string]any{"memories": memories})
}

func (s *Server) handleAddMemory(w http.ResponseWriter, r *http.Request) {
	repoID := r.PathValue("repo_id")
	kind := r.PathValue("kind")
	
	var req struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	memID, isNew, err := s.memoryService.Remember(repoID, kind, req.Content, "http", nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	respondJSON(w, map[string]any{"id": memID, "is_new": isNew})
}

func (s *Server) handleListMemories(w http.ResponseWriter, r *http.Request) {
	repoID := r.URL.Query().Get("repo_id")
	var rID *string
	if repoID != "" {
		rID = &repoID
	}
	
	kind := r.URL.Query().Get("kind")
	var kID *string
	if kind != "" {
		kID = &kind
	}

	memories, err := s.memoryService.ListMemories(r.Context(), rID, kID, 100)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	respondJSON(w, map[string]any{"memories": memories})
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Query  string   `json:"query"`
		RepoID string   `json:"repo_id"`
		Kinds  []string `json:"kinds"`
		Limit  int      `json:"limit"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	var rID *string
	if req.RepoID != "" {
		rID = &req.RepoID
	}
	limit := req.Limit
	if limit <= 0 {
		limit = 10
	}

	mems, err := s.memoryService.Search(r.Context(), req.Query, rID, req.Kinds, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	respondJSON(w, mems)
}

func (s *Server) handleGlobalSearch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Query string   `json:"query"`
		Kinds []string `json:"kinds"`
		Limit int      `json:"limit"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	
	limit := req.Limit
	if limit <= 0 {
		limit = 20
	}

	mems, err := s.memoryService.GlobalSearch(r.Context(), req.Query, req.Kinds, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	respondJSON(w, mems)
}

func respondJSON(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}
