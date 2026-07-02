package httpapi

import (
	"context"
	"net/http"
	"strings"

	"agent-memory-mcp/internal/app"
)

type contextKey string

const (
	tenantKey         contextKey = "tenant_id"
	memoryServiceKey  contextKey = "memory_service"
	usageServiceKey   contextKey = "usage_service"
	watcherServiceKey contextKey = "watcher_service"
	sessionServiceKey contextKey = "session_service"
)

// TenantFromContext extracts the tenant ID from the request context.
func TenantFromContext(ctx context.Context) string {
	if tenant, ok := ctx.Value(tenantKey).(string); ok && tenant != "" {
		return tenant
	}
	return "local"
}

func MemoryServiceFromContext(ctx context.Context) app.MemoryService {
	if ms, ok := ctx.Value(memoryServiceKey).(app.MemoryService); ok {
		return ms
	}
	return nil
}

func UsageServiceFromContext(ctx context.Context) app.UsageService {
	if us, ok := ctx.Value(usageServiceKey).(app.UsageService); ok {
		return us
	}
	return nil
}

func WatcherServiceFromContext(ctx context.Context) app.WatcherService {
	if ws, ok := ctx.Value(watcherServiceKey).(app.WatcherService); ok {
		return ws
	}
	return nil
}

func SessionServiceFromContext(ctx context.Context) app.SessionService {
	if ss, ok := ctx.Value(sessionServiceKey).(app.SessionService); ok {
		return ss
	}
	return nil
}

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID := "local"

		if len(s.config.ApiKeys) > 0 {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Unauthorized: Missing Authorization header", http.StatusUnauthorized)
				return
			}
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				http.Error(w, "Unauthorized: Invalid Authorization header format", http.StatusUnauthorized)
				return
			}
			token := parts[1]
			tid, ok := s.config.ApiKeys[token]
			if !ok {
				http.Error(w, "Unauthorized: Invalid API key", http.StatusUnauthorized)
				return
			}
			tenantID = tid
		}

		ts, err := s.tenantManager.Get(tenantID)
		if err != nil {
			http.Error(w, "Internal Server Error: failed to load tenant services", http.StatusInternalServerError)
			return
		}

		ctx := context.WithValue(r.Context(), tenantKey, tenantID)
		ctx = context.WithValue(ctx, memoryServiceKey, ts.MemoryService)
		ctx = context.WithValue(ctx, usageServiceKey, ts.UsageService)
		ctx = context.WithValue(ctx, watcherServiceKey, ts.WatcherService)
		// Store implements SessionService
		ctx = context.WithValue(ctx, sessionServiceKey, app.SessionService(ts.Store))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// requireServiceMiddleware ensures that the MemoryService is present in the context.
func requireServiceMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if MemoryServiceFromContext(r.Context()) == nil {
			http.Error(w, "Internal Server Error: MemoryService not found in context", http.StatusInternalServerError)
			return
		}
		next.ServeHTTP(w, r)
	})
}
