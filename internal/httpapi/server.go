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
	fileServer := http.FileServer(http.Dir(distDir))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Don't catch /api/ routes — they should 404 properly
		if len(r.URL.Path) > 4 && r.URL.Path[:5] == "/api/" {
			http.NotFound(w, r)
			return
		}

		path := filepath.Join(distDir, r.URL.Path)

		info, err := os.Stat(path)
		if os.IsNotExist(err) || (err == nil && info.IsDir()) {
			http.ServeFile(w, r, filepath.Join(distDir, "index.html"))
			return
		}

		fileServer.ServeHTTP(w, r)
	})

	// Wrap with CORS
	return corsMiddleware(mux)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
