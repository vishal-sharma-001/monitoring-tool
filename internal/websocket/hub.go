package websocket

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/monitoring-engine/monitoring-tool/internal/processor"
	"github.com/monitoring-engine/monitoring-tool/internal/logger"
)

// Message sent over WebSocket
type Message struct {
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
	Timestamp time.Time       `json:"timestamp"`
}

// Client represents a WebSocket client with write synchronization
type Client struct {
	conn    *websocket.Conn
	writeMu sync.Mutex
	hub     *Hub
}

// WriteJSON safely writes JSON to the WebSocket connection
func (c *Client) WriteJSON(v interface{}) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	return c.conn.WriteJSON(v)
}

// WriteControl safely writes control message to the WebSocket connection
func (c *Client) WriteControl(messageType int, data []byte, deadline time.Time) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	return c.conn.WriteControl(messageType, data, deadline)
}

// Hub manages WebSocket connections
type Hub struct {
	clients    map[*Client]bool
	broadcast  chan *Message
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan *Message, 500),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// Run starts the hub goroutine
func (h *Hub) Run(ctx context.Context) {
	logger.Info().Msg("Starting WebSocket Hub")

	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			logger.Info().Msg("WebSocket client registered")

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				client.conn.Close()
			}
			h.mu.Unlock()
			logger.Info().Msg("WebSocket client unregistered")

		case message := <-h.broadcast:
			h.mu.RLock()
			// Make a copy of clients to avoid holding lock during writes
			clients := make([]*Client, 0, len(h.clients))
			for client := range h.clients {
				clients = append(clients, client)
			}
			h.mu.RUnlock()

			// Send to each client (write mutex per client ensures no concurrent writes)
			for _, client := range clients {
				if err := client.WriteJSON(message); err != nil {
					logger.Error().Err(err).Msg("WebSocket write failed")
					h.unregister <- client
				}
			}

		case <-ctx.Done():
			return
		}
	}
}

// Register adds a client
func (h *Hub) Register(client *Client) {
	h.register <- client
}

// Unregister removes a client
func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

// Broadcast sends a message to all clients
func (h *Hub) Broadcast(msg *Message) {
	select {
	case h.broadcast <- msg:
	default:
		logger.Warn().Msg("Broadcast channel full")
	}
}

// OnAlert implements AlertObserver interface
func (h *Hub) OnAlert(ctx context.Context, event *processor.AlertEvent) error {
	payload, err := json.Marshal(event.Alert)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to marshal alert for WebSocket broadcast")
		return err
	}
	msg := &Message{
		Type:      "alert",
		Payload:   payload,
		Timestamp: time.Now(),
	}
	h.Broadcast(msg)
	return nil
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// For local development, allow localhost origins
		// In production, this should be configured via environment variable
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true // Allow same-origin requests
		}
		// Allow localhost for development
		return origin == "http://localhost:8080" ||
			origin == "http://localhost:3000" ||
			origin == "http://127.0.0.1:8080"
	},
}

// ServeWS handles WebSocket connections (goroutine per connection)
func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to upgrade WebSocket connection")
		return
	}

	// Create client wrapper with write mutex
	client := &Client{
		conn: conn,
		hub:  h,
	}

	// Set pong handler to reset read deadline
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// Register client
	h.Register(client)
	logger.Info().Msg("New WebSocket client connected")

	// Handle disconnection in a goroutine
	go func() {
		defer func() {
			h.Unregister(client)
		}()

		// Read messages (keep connection alive and handle ping/pong)
		for {
			var msg map[string]interface{}
			err := conn.ReadJSON(&msg)
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					logger.Error().Err(err).Msg("WebSocket unexpected close")
				}
				break
			}

			// Handle ping messages from client
			if msgType, ok := msg["type"].(string); ok && msgType == "ping" {
				pongMsg := &Message{
					Type:      "pong",
					Payload:   json.RawMessage(`{}`),
					Timestamp: time.Now(),
				}
				if err := client.WriteJSON(pongMsg); err != nil {
					logger.Error().Err(err).Msg("Failed to send pong")
					break
				}
			}
		}
	}()

	// Start a ticker to send periodic pings from server to client
	ticker := time.NewTicker(45 * time.Second)
	go func() {
		defer ticker.Stop()
		for range ticker.C {
			h.mu.RLock()
			_, exists := h.clients[client]
			h.mu.RUnlock()

			if !exists {
				return
			}

			if err := client.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(10*time.Second)); err != nil {
				logger.Error().Err(err).Msg("Failed to send ping")
				return
			}
		}
	}()
}
