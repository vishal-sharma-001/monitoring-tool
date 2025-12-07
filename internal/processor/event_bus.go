package processor

import (
	"context"
	"sync"
	"time"

	"github.com/monitoring-engine/monitoring-tool/internal/models"
	"github.com/monitoring-engine/monitoring-tool/internal/logger"
)

// AlertEvent represents an alert event
type AlertEvent struct {
	Alert     *models.Alert
	Timestamp time.Time
}

// AlertObserver interface (Observer Pattern)
type AlertObserver interface {
	OnAlert(ctx context.Context, event *AlertEvent) error
}

// EventBus distributes alert events to observers (Pub/Sub pattern)
type EventBus struct {
	observers []AlertObserver
	eventChan chan *AlertEvent
	stopCh    chan struct{}
	wg        sync.WaitGroup
}

func NewEventBus() *EventBus {
	return &EventBus{
		observers: make([]AlertObserver, 0),
		eventChan: make(chan *AlertEvent, 200),
		stopCh:    make(chan struct{}),
	}
}

// Subscribe adds an observer
func (eb *EventBus) Subscribe(observer AlertObserver) {
	eb.observers = append(eb.observers, observer)
	logger.Info().Msg("Observer subscribed to event bus")
}

// Publish sends an event to all observers
func (eb *EventBus) Publish(event *AlertEvent) {
	select {
	case eb.eventChan <- event:
	default:
		logger.Warn().Msg("Event bus channel full, dropping event")
	}
}

// Start begins processing events
func (eb *EventBus) Start(ctx context.Context) {
	logger.Info().Msg("Starting Alert Event Bus")

	eb.wg.Add(1)
	go eb.dispatcher(ctx)
}

// dispatcher goroutine distributes events to observers
func (eb *EventBus) dispatcher(ctx context.Context) {
	defer eb.wg.Done()

	for {
		select {
		case event := <-eb.eventChan:
			eb.notifyObservers(ctx, event)

		case <-eb.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

// notifyObservers sends event to all observers in parallel
func (eb *EventBus) notifyObservers(ctx context.Context, event *AlertEvent) {
	for _, observer := range eb.observers {
		// Notify each observer in a goroutine (concurrent)
		go func(obs AlertObserver) {
			ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()

			if err := obs.OnAlert(ctx, event); err != nil {
				logger.Error().Err(err).Msg("Observer notification failed")
			}
		}(observer)
	}
}

func (eb *EventBus) Stop() {
	close(eb.stopCh)
	eb.wg.Wait()
}
