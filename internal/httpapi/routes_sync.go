package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"

	"agent-memory-mcp/internal/core"
)

func (s *Server) registerSyncRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/export", s.handleExport)
	mux.HandleFunc("POST /api/v1/import", s.handleImport)
}

func (s *Server) handleExport(w http.ResponseWriter, r *http.Request) {
	repoID := r.URL.Query().Get("repo_id")
	var repoPtr *string
	if repoID != "" {
		repoPtr = &repoID
	}

	memories, err := MemoryServiceFromContext(r.Context()).Export(r.Context(), repoPtr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Output as JSONL to a buffer first to avoid partial writes on error
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	for _, m := range memories {
		if err := encoder.Encode(m); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/jsonlines+json")
	w.Write(buf.Bytes())
}

func (s *Server) handleImport(w http.ResponseWriter, r *http.Request) {
	// Parse JSONL from body
	var memories []core.Memory
	decoder := json.NewDecoder(r.Body)
	for decoder.More() {
		var m core.Memory
		if err := decoder.Decode(&m); err != nil {
			http.Error(w, "Invalid JSON in body", http.StatusBadRequest)
			return
		}
		memories = append(memories, m)
	}

	count, err := MemoryServiceFromContext(r.Context()).Import(r.Context(), memories)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, map[string]any{
		"imported": count,
		"total":    len(memories),
	})
}
