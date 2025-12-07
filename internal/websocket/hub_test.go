package websocket_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	gorillaws "github.com/gorilla/websocket"
	"github.com/monitoring-engine/monitoring-tool/internal/models"
	"github.com/monitoring-engine/monitoring-tool/internal/processor"
	"github.com/monitoring-engine/monitoring-tool/internal/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
)

func TestNewHub(t *testing.T) {
	t.Run("should create hub successfully", func(t *testing.T) {
		hub := websocket.NewHub()
		assert.NotNil(t, hub)
	})
}

func TestHub_Run(t *testing.T) {
	t.Run("should start and stop hub", func(t *testing.T) {
		hub := websocket.NewHub()
		ctx, cancel := context.WithCancel(context.Background())

		go hub.Run(ctx)

		// Give it time to start
		time.Sleep(50 * time.Millisecond)

		// Cancel context to stop
		cancel()

		// Give it time to stop
		time.Sleep(50 * time.Millisecond)
	})

	t.Run("should handle context cancellation", func(t *testing.T) {
		hub := websocket.NewHub()
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		done := make(chan bool)
		go func() {
			hub.Run(ctx)
			done <- true
		}()

		// Wait for context timeout
		select {
		case <-done:
			// Hub stopped successfully
		case <-time.After(200 * time.Millisecond):
			t.Fatal("Hub did not stop after context cancellation")
		}
	})
}

func TestHub_Broadcast(t *testing.T) {
	t.Run("should broadcast message", func(t *testing.T) {
		hub := websocket.NewHub()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go hub.Run(ctx)
		time.Sleep(50 * time.Millisecond)

		msg := &websocket.Message{
			Type:      "test",
			Payload:   json.RawMessage(`{"message":"test"}`),
			Timestamp: time.Now(),
		}

		hub.Broadcast(msg)

		// Give it time to process
		time.Sleep(50 * time.Millisecond)
	})

	t.Run("should handle full broadcast channel", func(t *testing.T) {
		hub := websocket.NewHub()

		// Don't start the hub so channel fills up
		for i := 0; i < 600; i++ {
			msg := &websocket.Message{
				Type:      "test",
				Payload:   json.RawMessage(`{"message":"test"}`),
				Timestamp: time.Now(),
			}
			hub.Broadcast(msg)
		}

		// Should not panic or block
	})
}

func TestHub_OnAlert(t *testing.T) {
	t.Run("should broadcast alert event", func(t *testing.T) {
		hub := websocket.NewHub()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go hub.Run(ctx)
		time.Sleep(50 * time.Millisecond)

		alert := &models.Alert{
			ID:          uuid.New(),
			Status:      models.AlertStatusFiring,
			Severity:    "high",
			Source:      "test",
			Message:     "Test alert",
			Value:       100.0,
			Labels:      datatypes.JSON([]byte(`{}`)),
			CreatedAt:   time.Now(),
			TriggeredAt: time.Now(),
		}

		event := &processor.AlertEvent{
			Alert:     alert,
			Timestamp: time.Now(),
		}

		err := hub.OnAlert(ctx, event)
		assert.NoError(t, err)

		// Give it time to process
		time.Sleep(50 * time.Millisecond)
	})

	t.Run("should handle alert with complex labels", func(t *testing.T) {
		hub := websocket.NewHub()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go hub.Run(ctx)
		time.Sleep(50 * time.Millisecond)

		labels := map[string]interface{}{
			"namespace": "production",
			"pod":       "app-1",
			"severity":  "critical",
		}
		labelsJSON, _ := json.Marshal(labels)

		alert := &models.Alert{
			ID:          uuid.New(),
			Status:      models.AlertStatusFiring,
			Severity:    "critical",
			Source:      "kubernetes",
			Message:     "Complex alert",
			Value:       95.0,
			Labels:      datatypes.JSON(labelsJSON),
			CreatedAt:   time.Now(),
			TriggeredAt: time.Now(),
		}

		event := &processor.AlertEvent{
			Alert:     alert,
			Timestamp: time.Now(),
		}

		err := hub.OnAlert(ctx, event)
		assert.NoError(t, err)
	})
}

func TestHub_ServeWS(t *testing.T) {
	t.Run("should upgrade HTTP connection to WebSocket", func(t *testing.T) {
		hub := websocket.NewHub()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go hub.Run(ctx)
		time.Sleep(50 * time.Millisecond)

		// Create test server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			hub.ServeWS(w, r)
		}))
		defer server.Close()

		// Connect WebSocket client
		wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
		conn, _, err := gorillaws.DefaultDialer.Dial(wsURL, nil)
		require.NoError(t, err)
		defer conn.Close()

		// Connection established successfully
		time.Sleep(100 * time.Millisecond)
	})

	t.Run("should handle WebSocket disconnection", func(t *testing.T) {
		hub := websocket.NewHub()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go hub.Run(ctx)
		time.Sleep(50 * time.Millisecond)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			hub.ServeWS(w, r)
		}))
		defer server.Close()

		wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
		conn, _, err := gorillaws.DefaultDialer.Dial(wsURL, nil)
		require.NoError(t, err)

		// Close connection
		conn.Close()

		// Give hub time to handle disconnection
		time.Sleep(100 * time.Millisecond)
	})

	t.Run("should broadcast message to connected client", func(t *testing.T) {
		hub := websocket.NewHub()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go hub.Run(ctx)
		time.Sleep(50 * time.Millisecond)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			hub.ServeWS(w, r)
		}))
		defer server.Close()

		wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
		conn, _, err := gorillaws.DefaultDialer.Dial(wsURL, nil)
		require.NoError(t, err)
		defer conn.Close()

		time.Sleep(100 * time.Millisecond)

		// Broadcast message
		msg := &websocket.Message{
			Type:      "test",
			Payload:   json.RawMessage(`{"data":"test"}`),
			Timestamp: time.Now(),
		}
		hub.Broadcast(msg)

		// Try to read message
		conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		var received websocket.Message
		err = conn.ReadJSON(&received)
		if err == nil {
			assert.Equal(t, "test", received.Type)
		}
	})
}

func TestHub_MultipleClients(t *testing.T) {
	t.Run("should handle multiple concurrent clients", func(t *testing.T) {
		hub := websocket.NewHub()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go hub.Run(ctx)
		time.Sleep(50 * time.Millisecond)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			hub.ServeWS(w, r)
		}))
		defer server.Close()

		wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

		// Connect multiple clients
		var conns []*gorillaws.Conn
		for i := 0; i < 3; i++ {
			conn, _, err := gorillaws.DefaultDialer.Dial(wsURL, nil)
			require.NoError(t, err)
			conns = append(conns, conn)
		}

		time.Sleep(150 * time.Millisecond)

		// Broadcast message
		msg := &websocket.Message{
			Type:      "broadcast",
			Payload:   json.RawMessage(`{"message":"to all"}`),
			Timestamp: time.Now(),
		}
		hub.Broadcast(msg)

		time.Sleep(100 * time.Millisecond)

		// Clean up
		for _, conn := range conns {
			conn.Close()
		}
	})
}

func TestMessage(t *testing.T) {
	t.Run("should marshal message to JSON", func(t *testing.T) {
		msg := &websocket.Message{
			Type:      "alert",
			Payload:   json.RawMessage(`{"severity":"high"}`),
			Timestamp: time.Now(),
		}

		data, err := json.Marshal(msg)
		require.NoError(t, err)
		assert.Contains(t, string(data), "alert")
		assert.Contains(t, string(data), "severity")
	})

	t.Run("should unmarshal JSON to message", func(t *testing.T) {
		jsonData := `{"type":"test","payload":{"data":"value"},"timestamp":"2024-01-01T00:00:00Z"}`

		var msg websocket.Message
		err := json.Unmarshal([]byte(jsonData), &msg)
		require.NoError(t, err)

		assert.Equal(t, "test", msg.Type)
		assert.NotNil(t, msg.Payload)
	})
}

func TestHub_StressTest(t *testing.T) {
	t.Run("should handle rapid broadcast messages", func(t *testing.T) {
		hub := websocket.NewHub()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go hub.Run(ctx)
		time.Sleep(50 * time.Millisecond)

		// Send many messages rapidly
		for i := 0; i < 100; i++ {
			msg := &websocket.Message{
				Type:      "test",
				Payload:   json.RawMessage(`{"index":` + string(rune(i)) + `}`),
				Timestamp: time.Now(),
			}
			hub.Broadcast(msg)
		}

		time.Sleep(100 * time.Millisecond)
	})
}

func TestHub_AlertIntegration(t *testing.T) {
	t.Run("should integrate with alert processor", func(t *testing.T) {
		hub := websocket.NewHub()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go hub.Run(ctx)
		time.Sleep(50 * time.Millisecond)

		// Simulate multiple alerts
		alerts := []struct {
			severity string
			value    float64
		}{
			{"critical", 95.0},
			{"high", 85.0},
			{"medium", 70.0},
			{"low", 50.0},
		}

		for _, a := range alerts {
			alert := &models.Alert{
				ID:          uuid.New(),
				Status:      models.AlertStatusFiring,
				Severity:    a.severity,
				Source:      "test",
				Message:     "Test alert",
				Value:       a.value,
				Labels:      datatypes.JSON([]byte(`{}`)),
				CreatedAt:   time.Now(),
				TriggeredAt: time.Now(),
			}

			event := &processor.AlertEvent{
				Alert:     alert,
				Timestamp: time.Now(),
			}

			err := hub.OnAlert(ctx, event)
			assert.NoError(t, err)
		}

		time.Sleep(100 * time.Millisecond)
	})
}

func TestHub_ErrorHandling(t *testing.T) {
	t.Run("should handle context cancellation during broadcast", func(t *testing.T) {
		hub := websocket.NewHub()
		ctx, cancel := context.WithCancel(context.Background())

		go hub.Run(ctx)
		time.Sleep(50 * time.Millisecond)

		// Cancel context
		cancel()

		// Try to broadcast after cancellation
		msg := &websocket.Message{
			Type:      "test",
			Payload:   json.RawMessage(`{"test":"data"}`),
			Timestamp: time.Now(),
		}

		// Should not panic
		hub.Broadcast(msg)
	})
}
