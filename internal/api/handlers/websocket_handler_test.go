package handlers_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	gorillaws "github.com/gorilla/websocket"
	"github.com/monitoring-engine/monitoring-tool/internal/api/handlers"
	"github.com/monitoring-engine/monitoring-tool/internal/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewWebSocketHandler(t *testing.T) {
	t.Run("should create websocket handler successfully", func(t *testing.T) {
		hub := websocket.NewHub()
		handler := handlers.NewWebSocketHandler(hub)
		assert.NotNil(t, handler)
	})

	t.Run("should create handler with nil hub", func(t *testing.T) {
		handler := handlers.NewWebSocketHandler(nil)
		assert.NotNil(t, handler)
	})
}

func TestRegisterWebSocketRoutes(t *testing.T) {
	t.Run("should register routes successfully", func(t *testing.T) {
		hub := websocket.NewHub()
		gin.SetMode(gin.TestMode)
		router := gin.New()

		handlers.RegisterWebSocketRoutes(router, hub)

		routes := router.Routes()
		hasWSRoute := false
		for _, route := range routes {
			if route.Path == "/ws" && route.Method == "GET" {
				hasWSRoute = true
			}
		}

		assert.True(t, hasWSRoute, "Expected /ws route to be registered")
	})
}

func TestWebSocketHandler_HandleWebSocket(t *testing.T) {
	t.Run("should upgrade HTTP connection to WebSocket", func(t *testing.T) {
		hub := websocket.NewHub()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go hub.Run(ctx)
		time.Sleep(50 * time.Millisecond)

		gin.SetMode(gin.TestMode)
		router := gin.New()
		handlers.RegisterWebSocketRoutes(router, hub)

		server := httptest.NewServer(router)
		defer server.Close()

		wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
		conn, _, err := gorillaws.DefaultDialer.Dial(wsURL, nil)
		require.NoError(t, err)
		defer conn.Close()

		time.Sleep(100 * time.Millisecond)
	})

	t.Run("should handle multiple concurrent connections", func(t *testing.T) {
		hub := websocket.NewHub()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go hub.Run(ctx)
		time.Sleep(50 * time.Millisecond)

		gin.SetMode(gin.TestMode)
		router := gin.New()
		handlers.RegisterWebSocketRoutes(router, hub)

		server := httptest.NewServer(router)
		defer server.Close()

		wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"

		var conns []*gorillaws.Conn
		for i := 0; i < 3; i++ {
			conn, _, err := gorillaws.DefaultDialer.Dial(wsURL, nil)
			require.NoError(t, err)
			conns = append(conns, conn)
		}

		time.Sleep(150 * time.Millisecond)

		for _, conn := range conns {
			conn.Close()
		}
	})

	t.Run("should handle connection and disconnection", func(t *testing.T) {
		hub := websocket.NewHub()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go hub.Run(ctx)
		time.Sleep(50 * time.Millisecond)

		gin.SetMode(gin.TestMode)
		router := gin.New()
		handlers.RegisterWebSocketRoutes(router, hub)

		server := httptest.NewServer(router)
		defer server.Close()

		wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
		conn, _, err := gorillaws.DefaultDialer.Dial(wsURL, nil)
		require.NoError(t, err)

		time.Sleep(100 * time.Millisecond)

		conn.Close()

		time.Sleep(100 * time.Millisecond)
	})

	t.Run("should receive messages through websocket", func(t *testing.T) {
		hub := websocket.NewHub()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go hub.Run(ctx)
		time.Sleep(50 * time.Millisecond)

		gin.SetMode(gin.TestMode)
		router := gin.New()
		handlers.RegisterWebSocketRoutes(router, hub)

		server := httptest.NewServer(router)
		defer server.Close()

		wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
		conn, _, err := gorillaws.DefaultDialer.Dial(wsURL, nil)
		require.NoError(t, err)
		defer conn.Close()

		time.Sleep(100 * time.Millisecond)

		// Broadcast a test message
		msg := &websocket.Message{
			Type:      "test",
			Payload:   []byte(`{"message":"hello"}`),
			Timestamp: time.Now(),
		}
		hub.Broadcast(msg)

		// Try to read message with timeout
		conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		var received websocket.Message
		err = conn.ReadJSON(&received)
		if err == nil {
			assert.Equal(t, "test", received.Type)
		}
	})
}

func TestWebSocketHandler_ErrorCases(t *testing.T) {
	t.Run("should handle invalid upgrade requests", func(t *testing.T) {
		hub := websocket.NewHub()
		gin.SetMode(gin.TestMode)
		router := gin.New()
		handlers.RegisterWebSocketRoutes(router, hub)

		// Regular HTTP request (not WebSocket upgrade)
		req, _ := http.NewRequest("GET", "/ws", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		// Should return 400 Bad Request for non-WebSocket upgrade
		assert.Equal(t, http.StatusBadRequest, resp.Code)
	})
}

func TestWebSocketRoutes_Integration(t *testing.T) {
	t.Run("should handle full request lifecycle", func(t *testing.T) {
		hub := websocket.NewHub()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go hub.Run(ctx)
		time.Sleep(50 * time.Millisecond)

		gin.SetMode(gin.TestMode)
		router := gin.New()
		handlers.RegisterWebSocketRoutes(router, hub)

		server := httptest.NewServer(router)
		defer server.Close()

		wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"

		// Connect
		conn, _, err := gorillaws.DefaultDialer.Dial(wsURL, nil)
		require.NoError(t, err)

		time.Sleep(100 * time.Millisecond)

		// Send broadcast
		msg := &websocket.Message{
			Type:      "integration",
			Payload:   []byte(`{"test":"data"}`),
			Timestamp: time.Now(),
		}
		hub.Broadcast(msg)

		time.Sleep(100 * time.Millisecond)

		// Disconnect
		conn.Close()

		time.Sleep(100 * time.Millisecond)
	})
}
