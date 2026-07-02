package httpapi

import (
	"net/http"
	"os"
	"path/filepath"

	"agent-memory-mcp/internal/app"
	"agent-memory-mcp/internal/config"
	"agent-memory-mcp/internal/tenant"
)

type Server struct {
	tenantManager tenant.Provider
	config        config.Config
	hub           *Hub
}

func NewServer(tm tenant.Provider, cfg config.Config) *Server {
	return &Server{
		tenantManager: tm,
		config:        cfg,
		hub:           NewHub(),
	}
}

// getServices extracts the tenant ID from the context and returns the appropriate services
func (s *Server) getServices(r *http.Request) (app.MemoryService, app.UsageService, error) {
	tenantID := TenantFromContext(r.Context())
	ts, err := s.tenantManager.Get(tenantID)
	if err != nil {
		return nil, nil, err
	}
	return ts.MemoryService, ts.UsageService, nil
}

func (s *Server) Handler() http.Handler {
	apiMux := http.NewServeMux()
	s.registerMemoryRoutes(apiMux)
	s.registerIntelligenceRoutes(apiMux)
	s.registerTelemetryRoutes(apiMux)
	s.registerWsRoutes(apiMux)
	s.registerSyncRoutes(apiMux)

	mainMux := http.NewServeMux()
	mainMux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		respondJSON(w, map[string]string{"status": "ok"})
	})

	// Wrap API mux with auth middleware and service requirement
	authAPI := s.authMiddleware(requireServiceMiddleware(apiMux))

	// Fallback for static UI
	distDir := "ui/dist"
	fileServer := http.FileServer(http.Dir(distDir))
	
	mainMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Route /api/ and /ws/ to the authenticated apiMux
		if (len(r.URL.Path) > 4 && r.URL.Path[:5] == "/api/") || (len(r.URL.Path) > 3 && r.URL.Path[:4] == "/ws/") {
			authAPI.ServeHTTP(w, r)
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
	return corsMiddleware(mainMux)
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
