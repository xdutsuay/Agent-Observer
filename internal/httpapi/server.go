package httpapi

import (
	"encoding/json"
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

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	mux.HandleFunc("/api/memories", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		
		repoID := r.URL.Query().Get("repo_id")
		var rID *string
		if repoID != "" {
			rID = &repoID
		}

		mems, err := s.memoryService.ListMemories(r.Context(), rID, nil, 50)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		
		json.NewEncoder(w).Encode(map[string]any{"memories": mems})
	})

	return mux
}
