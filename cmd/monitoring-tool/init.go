package main

import (
	"context"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/monitoring-engine/monitoring-tool/internal/app"
	k8sclient "github.com/monitoring-engine/monitoring-tool/internal/collector"
	"github.com/monitoring-engine/monitoring-tool/internal/config"
	"github.com/monitoring-engine/monitoring-tool/internal/logger"
	"github.com/monitoring-engine/monitoring-tool/internal/notifier"
	"github.com/monitoring-engine/monitoring-tool/internal/processor"
	alertservice "github.com/monitoring-engine/monitoring-tool/internal/service"
	"github.com/monitoring-engine/monitoring-tool/internal/storage"
	alertrepo "github.com/monitoring-engine/monitoring-tool/internal/repository"
	"github.com/monitoring-engine/monitoring-tool/internal/websocket"
	"gorm.io/gorm"
)

// initDatabase initializes the PostgreSQL connection
func initDatabase(cfg config.PostgresConfig) (*gorm.DB, error) {
	postgresDB, err := storage.GetPostgresInstance(cfg)
	if err != nil {
		return nil, err
	}
	logger.Info().Msg("PostgreSQL initialized")

	// Run migrations if auto_migrate is enabled
	if cfg.AutoMigrate {
		if err := runMigrations(cfg); err != nil {
			logger.Warn().Err(err).Msg("Failed to run migrations automatically")
		}
	}

	return postgresDB, nil
}

// runMigrations runs database migrations using golang-migrate
func runMigrations(cfg config.PostgresConfig) error {
	databaseURL := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database)

	m, err := migrate.New(
		"file://migrations",
		databaseURL,
	)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	if err == migrate.ErrNoChange {
		logger.Info().Msg("No new migrations to apply")
	} else {
		logger.Info().Msg("Database migrations applied successfully")
	}

	return nil
}

// initAlertService initializes the alert service with all its dependencies
func initAlertService(postgresDB *gorm.DB) alertservice.AlertService {
	alertRepo := alertrepo.NewPostgresAlertRepo(postgresDB)
	logger.Info().Msg("Alert repository initialized (PostgreSQL)")

	alertService := alertservice.NewAlertService(alertRepo)
	logger.Info().Msg("Alert service initialized")

	return alertService
}

// initK8sClient initializes the Kubernetes client
func initK8sClient(ctx context.Context) (*k8sclient.K8sClient, error) {
	k8sClient, err := k8sclient.NewK8sClient()
	if err != nil {
		return nil, err
	}
	k8sClient.Start(ctx)
	logger.Info().Msg("K8s client initialized")
	return k8sClient, nil
}

// initEventBus initializes the alert event bus
func initEventBus(ctx context.Context) *processor.EventBus {
	eventBus := processor.NewEventBus()
	eventBus.Start(ctx)
	logger.Info().Msg("Alert event bus started")
	return eventBus
}

// initWebSocketHub initializes the WebSocket hub for real-time alerts
func initWebSocketHub(ctx context.Context, eventBus *processor.EventBus) *websocket.Hub {
	wsHub := websocket.NewHub()
	eventBus.Subscribe(wsHub)
	go wsHub.Run(ctx)
	logger.Info().Msg("WebSocket hub started (real-time alert streaming)")
	return wsHub
}

// initEmailDispatcher initializes the email notification dispatcher if configured
func initEmailDispatcher(cfg config.EmailConfig, eventBus *processor.EventBus) {
	if !cfg.Enabled {
		logger.Info().Msg("Email notifications disabled in configuration")
		return
	}

	if cfg.SMTPHost != "" && cfg.Username != "" {
		emailDispatcher := notifier.NewEmailDispatcher(cfg)
		eventBus.Subscribe(emailDispatcher)
		logger.Info().
			Str("smtp_host", cfg.SMTPHost).
			Strs("to", cfg.To).
			Msg("Email dispatcher enabled")
	} else {
		logger.Warn().Msg("Email configuration incomplete - notifications disabled")
	}
}

// initAlertEngine initializes the alert evaluator engine
func initAlertEngine(ctx context.Context, alertRepo alertrepo.AlertRepo, eventBus *processor.EventBus) *processor.EvaluatorEngine {
	alertEngine := processor.NewEvaluatorEngine(alertRepo, eventBus)
	alertEngine.Start(ctx)
	logger.Info().Msg("Alert evaluator engine started")
	return alertEngine
}

// initK8sWatchers initializes the Kubernetes pod, node, and metrics watchers
func initK8sWatchers(
	ctx context.Context,
	k8sClient *k8sclient.K8sClient,
	alertEngine *processor.EvaluatorEngine,
) (*k8sclient.PodWatcher, *k8sclient.NodeWatcher, *k8sclient.MetricsWatcher) {
	// Get the state manager and worker pool from alert engine
	stateManager := alertEngine.GetStateManager()
	workerPool := alertEngine.GetWorkerPool()

	// Pod watcher
	podWatcher := k8sclient.NewPodWatcher(k8sClient, stateManager, workerPool)
	podWatcher.Start(ctx)
	logger.Info().Msg("Pod watcher started with worker pool")

	// Node watcher
	nodeWatcher := k8sclient.NewNodeWatcher(k8sClient, stateManager, workerPool)
	nodeWatcher.Start(ctx)
	logger.Info().Msg("Node watcher started with worker pool")

	// Metrics watcher
	metricsWatcher := k8sclient.NewMetricsWatcher(k8sClient, stateManager, workerPool)
	metricsWatcher.Start(ctx)
	logger.Info().Msg("Metrics watcher started for CPU/memory monitoring")

	return podWatcher, nodeWatcher, metricsWatcher
}

// initDependencies creates and validates the dependencies container
func initDependencies(
	postgresDB *gorm.DB,
	k8sClient *k8sclient.K8sClient,
	alertService alertservice.AlertService,
	eventBus *processor.EventBus,
	wsHub *websocket.Hub,
) (*app.Dependencies, error) {
	deps, err := app.NewDependencies(postgresDB, k8sClient, alertService, eventBus, wsHub)
	if err != nil {
		return nil, err
	}
	logger.Info().Msg("Dependencies container initialized")
	return deps, nil
}

// closeDatabase closes the database connection
func closeDatabase(postgresDB *gorm.DB) {
	storage.Close(postgresDB)
	logger.Info().Msg("Database connection closed")
}
