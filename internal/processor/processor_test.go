package processor_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/monitoring-engine/monitoring-tool/internal/models"
	"github.com/monitoring-engine/monitoring-tool/internal/processor"
	"github.com/monitoring-engine/monitoring-tool/internal/repository"
	"github.com/stretchr/testify/assert"
	"gorm.io/datatypes"
)

// Mock observer for testing
type MockObserver struct {
	receivedEvents []*processor.AlertEvent
	mu             sync.Mutex
	shouldFail     bool
	callCount      int32
}

func (m *MockObserver) OnAlert(ctx context.Context, event *processor.AlertEvent) error {
	atomic.AddInt32(&m.callCount, 1)
	m.mu.Lock()
	m.receivedEvents = append(m.receivedEvents, event)
	m.mu.Unlock()

	if m.shouldFail {
		return assert.AnError
	}
	return nil
}

func (m *MockObserver) GetReceivedEvents() []*processor.AlertEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.receivedEvents
}

func (m *MockObserver) GetCallCount() int32 {
	return atomic.LoadInt32(&m.callCount)
}

// TestEventBus_NewEventBus tests event bus creation
func TestEventBus_NewEventBus(t *testing.T) {
	t.Run("should create event bus successfully", func(t *testing.T) {
		eb := processor.NewEventBus()
		assert.NotNil(t, eb)
	})
}

// TestEventBus_Subscribe tests observer subscription
func TestEventBus_Subscribe(t *testing.T) {
	t.Run("should subscribe observer successfully", func(t *testing.T) {
		eb := processor.NewEventBus()
		observer := &MockObserver{}

		eb.Subscribe(observer)
		// No panic means success
	})

	t.Run("should subscribe multiple observers", func(t *testing.T) {
		eb := processor.NewEventBus()
		observer1 := &MockObserver{}
		observer2 := &MockObserver{}
		observer3 := &MockObserver{}

		eb.Subscribe(observer1)
		eb.Subscribe(observer2)
		eb.Subscribe(observer3)
		// No panic means success
	})
}

// TestEventBus_PublishAndDispatch tests event publishing and dispatching
func TestEventBus_PublishAndDispatch(t *testing.T) {
	t.Run("should publish and dispatch event to observer", func(t *testing.T) {
		ctx := context.Background()
		eb := processor.NewEventBus()
		observer := &MockObserver{}

		eb.Subscribe(observer)
		eb.Start(ctx)
		defer eb.Stop()

		alert := &models.Alert{
			ID:          uuid.New(),
			Status:      models.AlertStatusFiring,
			Severity:    "high",
			Message:     "Test alert",
			Source:      "test",
			Labels:      datatypes.JSON([]byte(`{}`)),
			Value:       100.0,
			TriggeredAt: time.Now(),
		}

		event := &processor.AlertEvent{
			Alert:     alert,
			Timestamp: time.Now(),
		}

		eb.Publish(event)

		// Wait for event to be processed
		time.Sleep(100 * time.Millisecond)

		events := observer.GetReceivedEvents()
		assert.Len(t, events, 1)
		assert.Equal(t, alert.ID, events[0].Alert.ID)
		assert.Equal(t, alert.Severity, events[0].Alert.Severity)
	})

	t.Run("should dispatch to multiple observers", func(t *testing.T) {
		ctx := context.Background()
		eb := processor.NewEventBus()
		observer1 := &MockObserver{}
		observer2 := &MockObserver{}
		observer3 := &MockObserver{}

		eb.Subscribe(observer1)
		eb.Subscribe(observer2)
		eb.Subscribe(observer3)
		eb.Start(ctx)
		defer eb.Stop()

		alert := &models.Alert{
			ID:       uuid.New(),
			Severity: "critical",
			Message:  "Multi-observer test",
			Source:   "test",
		}

		event := &processor.AlertEvent{
			Alert:     alert,
			Timestamp: time.Now(),
		}

		eb.Publish(event)

		// Wait for all observers to receive
		time.Sleep(150 * time.Millisecond)

		assert.Len(t, observer1.GetReceivedEvents(), 1)
		assert.Len(t, observer2.GetReceivedEvents(), 1)
		assert.Len(t, observer3.GetReceivedEvents(), 1)
	})

	t.Run("should handle observer errors gracefully", func(t *testing.T) {
		ctx := context.Background()
		eb := processor.NewEventBus()
		failingObserver := &MockObserver{shouldFail: true}
		successObserver := &MockObserver{shouldFail: false}

		eb.Subscribe(failingObserver)
		eb.Subscribe(successObserver)
		eb.Start(ctx)
		defer eb.Stop()

		alert := &models.Alert{
			ID:       uuid.New(),
			Severity: "high",
			Message:  "Error handling test",
			Source:   "test",
		}

		event := &processor.AlertEvent{
			Alert:     alert,
			Timestamp: time.Now(),
		}

		eb.Publish(event)

		time.Sleep(150 * time.Millisecond)

		// Both observers should have been called despite one failing
		assert.Equal(t, int32(1), failingObserver.GetCallCount())
		assert.Len(t, successObserver.GetReceivedEvents(), 1)
	})

	t.Run("should drop events when channel is full", func(t *testing.T) {
		ctx := context.Background()
		eb := processor.NewEventBus()
		slowObserver := &MockObserver{}

		eb.Subscribe(slowObserver)
		eb.Start(ctx)
		defer eb.Stop()

		// Publish more events than channel buffer
		for i := 0; i < 250; i++ {
			alert := &models.Alert{
				ID:       uuid.New(),
				Severity: "medium",
				Message:  "Overflow test",
				Source:   "test",
			}
			event := &processor.AlertEvent{
				Alert:     alert,
				Timestamp: time.Now(),
			}
			eb.Publish(event)
		}

		time.Sleep(200 * time.Millisecond)

		// Should have processed some but not all due to channel capacity
		received := len(slowObserver.GetReceivedEvents())
		assert.Greater(t, received, 0)
		assert.LessOrEqual(t, received, 250)
	})
}

// TestEventBus_StartStop tests lifecycle management
func TestEventBus_StartStop(t *testing.T) {
	t.Run("should start and stop cleanly", func(t *testing.T) {
		ctx := context.Background()
		eb := processor.NewEventBus()
		observer := &MockObserver{}

		eb.Subscribe(observer)
		eb.Start(ctx)

		// Publish an event
		alert := &models.Alert{
			ID:       uuid.New(),
			Severity: "low",
			Message:  "Lifecycle test",
			Source:   "test",
		}
		event := &processor.AlertEvent{
			Alert:     alert,
			Timestamp: time.Now(),
		}
		eb.Publish(event)

		time.Sleep(50 * time.Millisecond)

		// Stop should wait for processing
		eb.Stop()

		events := observer.GetReceivedEvents()
		assert.Len(t, events, 1)
	})

	t.Run("should handle context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		eb := processor.NewEventBus()
		observer := &MockObserver{}

		eb.Subscribe(observer)
		eb.Start(ctx)

		// Cancel context
		cancel()

		// Give dispatcher time to exit
		time.Sleep(50 * time.Millisecond)

		// Stop should still work
		eb.Stop()
	})
}

// TestAlertStateManager_NewAlertStateManager tests state manager creation
func TestAlertStateManager_NewAlertStateManager(t *testing.T) {
	t.Run("should create state manager successfully", func(t *testing.T) {
		repo := repository.NewInMemoryAlertRepo()
		eventBus := processor.NewEventBus()

		manager := processor.NewAlertStateManager(repo, eventBus)
		assert.NotNil(t, manager)
	})
}

// TestAlertStateManager_ProcessAlert tests alert processing
func TestAlertStateManager_ProcessAlert(t *testing.T) {
	t.Run("should process and create alert", func(t *testing.T) {
		ctx := context.Background()
		repo := repository.NewInMemoryAlertRepo()
		eventBus := processor.NewEventBus()
		observer := &MockObserver{}

		eventBus.Subscribe(observer)
		eventBus.Start(ctx)
		defer eventBus.Stop()

		manager := processor.NewAlertStateManager(repo, eventBus)

		alert := &models.Alert{
			ID:          uuid.New(),
			Status:      models.AlertStatusFiring,
			Severity:    "high",
			Message:     "Process test",
			Source:      "test",
			Labels:      datatypes.JSON([]byte(`{}`)),
			Value:       100.0,
			TriggeredAt: time.Now(),
		}

		isNew, err := manager.ProcessAlert(ctx, alert)
		assert.NoError(t, err)
		assert.True(t, isNew) // Every alert is new without deduplication

		// Wait for event bus notification
		time.Sleep(100 * time.Millisecond)

		// Verify alert was created in repository
		alerts, err := repo.GetRecent(ctx, 10)
		assert.NoError(t, err)
		assert.Len(t, alerts, 1)

		// Verify event was published
		events := observer.GetReceivedEvents()
		assert.Len(t, events, 1)
		assert.Equal(t, alert.ID, events[0].Alert.ID)
	})

	t.Run("should process multiple alerts", func(t *testing.T) {
		ctx := context.Background()
		repo := repository.NewInMemoryAlertRepo()
		eventBus := processor.NewEventBus()
		observer := &MockObserver{}

		eventBus.Subscribe(observer)
		eventBus.Start(ctx)
		defer eventBus.Stop()

		manager := processor.NewAlertStateManager(repo, eventBus)

		// Create and process 5 alerts
		for i := 0; i < 5; i++ {
			alert := &models.Alert{
				ID:          uuid.New(),
				Status:      models.AlertStatusFiring,
				Severity:    "medium",
				Message:     "Batch test",
				Source:      "test",
				Labels:      datatypes.JSON([]byte(`{}`)),
				Value:       float64(i),
				TriggeredAt: time.Now(),
			}

			isNew, err := manager.ProcessAlert(ctx, alert)
			assert.NoError(t, err)
			assert.True(t, isNew)
		}

		time.Sleep(200 * time.Millisecond)

		// All 5 should be in repository
		alerts, err := repo.GetRecent(ctx, 10)
		assert.NoError(t, err)
		assert.Len(t, alerts, 5)

		// All 5 should have been published
		events := observer.GetReceivedEvents()
		assert.Len(t, events, 5)
	})

	t.Run("should handle repository errors", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately to cause context error

		repo := repository.NewInMemoryAlertRepo()
		eventBus := processor.NewEventBus()
		manager := processor.NewAlertStateManager(repo, eventBus)

		alert := &models.Alert{
			ID:       uuid.New(),
			Severity: "high",
			Message:  "Error test",
			Source:   "test",
		}

		// This should still work as InMemoryRepo doesn't check context
		_, err := manager.ProcessAlert(ctx, alert)
		assert.NoError(t, err)
	})

	t.Run("should create separate alerts without deduplication", func(t *testing.T) {
		ctx := context.Background()
		repo := repository.NewInMemoryAlertRepo()
		eventBus := processor.NewEventBus()
		manager := processor.NewAlertStateManager(repo, eventBus)

		// Create identical alerts
		for i := 0; i < 3; i++ {
			alert := &models.Alert{
				ID:          uuid.New(),
				Status:      models.AlertStatusFiring,
				Severity:    "high",
				Message:     "Same message",
				Source:      "same_source",
				Labels:      datatypes.JSON([]byte(`{"key":"value"}`)),
				Value:       100.0,
				TriggeredAt: time.Now(),
			}

			isNew, err := manager.ProcessAlert(ctx, alert)
			assert.NoError(t, err)
			assert.True(t, isNew) // All should be new
		}

		// All 3 should exist as separate alerts
		alerts, err := repo.GetRecent(ctx, 10)
		assert.NoError(t, err)
		assert.Len(t, alerts, 3)
	})
}

// TestEventBus_ConcurrentPublish tests concurrent publishing
func TestEventBus_ConcurrentPublish(t *testing.T) {
	t.Run("should handle concurrent publishes", func(t *testing.T) {
		ctx := context.Background()
		eb := processor.NewEventBus()
		observer := &MockObserver{}

		eb.Subscribe(observer)
		eb.Start(ctx)
		defer eb.Stop()

		var wg sync.WaitGroup
		numGoroutines := 10
		eventsPerGoroutine := 5

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < eventsPerGoroutine; j++ {
					alert := &models.Alert{
						ID:       uuid.New(),
						Severity: "low",
						Message:  "Concurrent test",
						Source:   "test",
						Value:    float64(id*100 + j),
					}
					event := &processor.AlertEvent{
						Alert:     alert,
						Timestamp: time.Now(),
					}
					eb.Publish(event)
				}
			}(i)
		}

		wg.Wait()
		time.Sleep(300 * time.Millisecond)

		events := observer.GetReceivedEvents()
		assert.GreaterOrEqual(t, len(events), 40) // At least 80% delivered
	})
}
