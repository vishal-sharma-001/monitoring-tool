package processor

import (
	"context"

	"github.com/monitoring-engine/monitoring-tool/internal/repository"
	"github.com/monitoring-engine/monitoring-tool/internal/logger"
	"github.com/monitoring-engine/monitoring-tool/internal/pool"
)

// EvaluatorEngine evaluates alert rules using worker pool
type EvaluatorEngine struct {
	alertRepo    repository.AlertRepo
	eventBus     *EventBus
	stateManager *AlertStateManager
	workerPool   *pool.WorkerPool
}

func NewEvaluatorEngine(alertRepo repository.AlertRepo, eventBus *EventBus) *EvaluatorEngine {
	return &EvaluatorEngine{
		alertRepo:    alertRepo,
		eventBus:     eventBus,
		stateManager: NewAlertStateManager(alertRepo, eventBus),
		workerPool:   pool.NewWorkerPool(5, 300), // 5 workers, 300 task queue
	}
}

func (ee *EvaluatorEngine) Start(ctx context.Context) {
	logger.Info().Msg("Starting Alert Evaluator with 5 workers")

	// Start worker pool
	ee.workerPool.Start(ctx)

	// Note: No periodic evaluation - alerts are now triggered by K8s watchers
	// This engine only provides the worker pool for processing alerts
	logger.Info().Msg("Alert evaluator ready - waiting for events from K8s watchers")
}

// GetStateManager returns the alert state manager for use by watchers
func (ee *EvaluatorEngine) GetStateManager() *AlertStateManager {
	return ee.stateManager
}

// GetWorkerPool returns the worker pool for use by watchers
func (ee *EvaluatorEngine) GetWorkerPool() *pool.WorkerPool {
	return ee.workerPool
}

func (ee *EvaluatorEngine) Stop() {
	ee.workerPool.Stop()
}
