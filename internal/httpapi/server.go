package httpapi

import (
	"net/http"
	"os"
	"path/filepath"

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
	distDir := "ui/dist"
	fs := http.FileServer(http.Dir(distDir))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := filepath.Join(distDir, r.URL.Path)
		
		// Ensure we don't serve directories directly (FileServer does this, but for SPA we want index.html)
		info, err := os.Stat(path)
		if os.IsNotExist(err) || (err == nil && info.IsDir()) {
			// Not found or is directory -> fallback to SPA index
			http.ServeFile(w, r, filepath.Join(distDir, "index.html"))
			return
		}
		
		// Otherwise serve the static asset
		fs.ServeHTTP(w, r)
	})

	return mux
}
