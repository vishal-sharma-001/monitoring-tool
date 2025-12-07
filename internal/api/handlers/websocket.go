package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/monitoring-engine/monitoring-tool/internal/websocket"
)

// WebSocketHandler handles WebSocket requests
type WebSocketHandler struct {
	hub *websocket.Hub
}

// NewWebSocketHandler creates a new WebSocket handler
func NewWebSocketHandler(hub *websocket.Hub) *WebSocketHandler {
	return &WebSocketHandler{
		hub: hub,
	}
}

// HandleWebSocket handles GET /ws
func (h *WebSocketHandler) HandleWebSocket(c *gin.Context) {
	h.hub.ServeWS(c.Writer, c.Request)
}

// RegisterWebSocketRoutes registers WebSocket endpoint
func RegisterWebSocketRoutes(router *gin.Engine, wsHub *websocket.Hub) {
	// Create handler with WebSocket hub dependency
	wsHandler := NewWebSocketHandler(wsHub)

	// Register WebSocket endpoint
	router.GET("/ws", wsHandler.HandleWebSocket)
}
