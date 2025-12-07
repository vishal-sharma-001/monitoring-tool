package processor

import (
	"context"
	"time"

	"github.com/monitoring-engine/monitoring-tool/internal/models"
	"github.com/monitoring-engine/monitoring-tool/internal/repository"
	"github.com/monitoring-engine/monitoring-tool/internal/logger"
)

// AlertStateManager manages alert lifecycle (no deduplication - every alert is created as new)
type AlertStateManager struct {
	alertRepo repository.AlertRepo
	eventBus  *EventBus
}

// NewAlertStateManager creates a new alert state manager
func NewAlertStateManager(alertRepo repository.AlertRepo, eventBus *EventBus) *AlertStateManager {
	return &AlertStateManager{
		alertRepo: alertRepo,
		eventBus:  eventBus,
	}
}

// ProcessAlert handles alert without deduplication - creates every alert as new
// Returns true always (every alert is new)
func (asm *AlertStateManager) ProcessAlert(ctx context.Context, alert *models.Alert) (bool, error) {
	// Create every alert as new - no deduplication
	if err := asm.alertRepo.Create(ctx, alert); err != nil {
		return false, err
	}

	// Publish to event bus for real-time notifications
	asm.eventBus.Publish(&AlertEvent{
		Alert:     alert,
		Timestamp: time.Now(),
	})

	logger.Info().
		Str("severity", alert.Severity).
		Str("source", alert.Source).
		Str("message", alert.Message).
		Msg("Alert created and published")

	return true, nil
}
