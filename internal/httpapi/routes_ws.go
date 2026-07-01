package httpapi

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for local dashboard
	},
}

func (s *Server) registerWsRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /ws/events", s.handleWsEvents)
	// Some event history route the python version had
	mux.HandleFunc("GET /api/events/history", func(w http.ResponseWriter, r *http.Request) {
		respondJSON(w, map[string]any{"events": []any{}})
	})
	mux.HandleFunc("GET /api/events", func(w http.ResponseWriter, r *http.Request) {
		respondJSON(w, map[string]any{"events": []any{}})
	})
}

func (s *Server) handleWsEvents(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("failed to upgrade to websocket: %v", err)
		return
	}
	defer conn.Close()

	// Minimal mock loop to keep connection alive
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}
