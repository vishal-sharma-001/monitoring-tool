package app

import (
	"fmt"

	"github.com/monitoring-engine/monitoring-tool/internal/collector"
	"github.com/monitoring-engine/monitoring-tool/internal/processor"
	"github.com/monitoring-engine/monitoring-tool/internal/service"
	"github.com/monitoring-engine/monitoring-tool/internal/websocket"
	"gorm.io/gorm"
)

// Dependencies holds all application-wide dependencies
type Dependencies struct {
	DB           *gorm.DB
	K8sClient    *collector.K8sClient
	AlertService service.AlertService
	EventBus     *processor.EventBus
	WSHub        *websocket.Hub
}

// NewDependencies creates a new dependencies container with validation
func NewDependencies(
	db *gorm.DB,
	k8sClient *collector.K8sClient,
	alertService service.AlertService,
	eventBus *processor.EventBus,
	wsHub *websocket.Hub,
) (*Dependencies, error) {
	// Validate required dependencies
	if db == nil {
		return nil, fmt.Errorf("database is required")
	}
	if k8sClient == nil {
		return nil, fmt.Errorf("k8s client is required")
	}
	if alertService == nil {
		return nil, fmt.Errorf("alert service is required")
	}
	if eventBus == nil {
		return nil, fmt.Errorf("event bus is required")
	}
	if wsHub == nil {
		return nil, fmt.Errorf("websocket hub is required")
	}

	return &Dependencies{
		DB:           db,
		K8sClient:    k8sClient,
		AlertService: alertService,
		EventBus:     eventBus,
		WSHub:        wsHub,
	}, nil
}
