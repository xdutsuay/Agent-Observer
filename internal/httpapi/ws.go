package httpapi

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

type Hub struct {
	mu      sync.Mutex
	clients map[string]map[*websocket.Conn]bool // tenantID -> set of connections
}

func NewHub() *Hub {
	return &Hub{
		clients: make(map[string]map[*websocket.Conn]bool),
	}
}

func (h *Hub) AddClient(tenantID string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.clients[tenantID] == nil {
		h.clients[tenantID] = make(map[*websocket.Conn]bool)
	}
	h.clients[tenantID][conn] = true
}

func (h *Hub) RemoveClient(tenantID string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if tenants, ok := h.clients[tenantID]; ok {
		delete(tenants, conn)
		if len(tenants) == 0 {
			delete(h.clients, tenantID)
		}
	}
}

func (h *Hub) Broadcast(tenantID string, eventType string, payload any) {
	h.mu.Lock()
	defer h.mu.Unlock()

	tenants, ok := h.clients[tenantID]
	if !ok || len(tenants) == 0 {
		return
	}

	msg := map[string]any{
		"type":    eventType,
		"payload": payload,
	}
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("failed to marshal ws event: %v", err)
		return
	}

	for conn := range tenants {
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			log.Printf("failed to write ws message to tenant %s: %v", tenantID, err)
			conn.Close()
			delete(tenants, conn)
		}
	}
}
