package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/monitoring-engine/monitoring-tool/internal/api"
	"github.com/monitoring-engine/monitoring-tool/internal/config"
	"github.com/monitoring-engine/monitoring-tool/internal/app"
	"github.com/monitoring-engine/monitoring-tool/internal/collector"
	"github.com/monitoring-engine/monitoring-tool/internal/logger"
	"github.com/monitoring-engine/monitoring-tool/internal/processor"
	"github.com/monitoring-engine/monitoring-tool/internal/service"
	"github.com/monitoring-engine/monitoring-tool/internal/repository"
	"github.com/monitoring-engine/monitoring-tool/internal/websocket"
	"gorm.io/gorm"
)

// Package-level variables for application components
var (
	cfg          *config.Config
	appCtx       context.Context
	appCancel    context.CancelFunc
	postgresDB   *gorm.DB
	alertService service.AlertService
	k8sClient    *collector.K8sClient
	eventBus     *processor.EventBus
	wsHub        *websocket.Hub
	alertEngine    *processor.EvaluatorEngine
	podWatcher     *collector.PodWatcher
	nodeWatcher    *collector.NodeWatcher
	metricsWatcher *collector.MetricsWatcher
	deps           *app.Dependencies
)

func init() {
	// 1. Load configuration
	var err error
	cfg, err = config.Load("configs/config.yaml")
	config.SetGlobalConfig(cfg)
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// 2. Initialize logger
	logger.InitLogger(cfg.Logging.Level, cfg.Logging.Format)
	logger.Info().Msg("Starting Monitoring Engine...")

	// 3. Create application context for graceful shutdown
	appCtx, appCancel = context.WithCancel(context.Background())

	// 4. Initialize infrastructure (Database)
	postgresDB, err = initDatabase(cfg.Postgres)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to initialize PostgreSQL")
	}

	// 5. Initialize alert service (handler → service → repo architecture)
	alertService = initAlertService(postgresDB)

	// 6. Initialize K8s Monitoring & Alerting
	logger.Info().Msg("Initializing K8s monitoring components...")

	k8sClient, err = initK8sClient(appCtx)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to initialize K8s client")
	}

	eventBus = initEventBus(appCtx)
	wsHub = initWebSocketHub(appCtx, eventBus)
	initEmailDispatcher(cfg.Email, eventBus)

	alertRepo := repository.NewPostgresAlertRepo(postgresDB)
	alertEngine = initAlertEngine(appCtx, alertRepo, eventBus)
	podWatcher, nodeWatcher, metricsWatcher = initK8sWatchers(appCtx, k8sClient, alertEngine)

	logger.Info().Msg("Monitoring system initialized: K8s observers + Metrics → Alerts → WebSocket + Email")

	// 7. Create dependencies container
	deps, err = initDependencies(postgresDB, k8sClient, alertService, eventBus, wsHub)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to create dependencies container")
	}
}

func main() {
	defer appCancel()

	// Setup HTTP server
	srv := setupHTTPServer()

	// Start HTTP server in background
	startServer(srv)

	logger.Info().Msg("Monitoring Engine MVP is running")

	// Wait for shutdown signal
	waitForShutdown()

	// Graceful cleanup
	shutdown(srv)
}

func setupHTTPServer() *http.Server {
	gin.SetMode(gin.ReleaseMode)
	appEngine := gin.New()
	appEngine.Use(gin.Recovery())

	api.RegisterRoutes(deps, appEngine)

	return &http.Server{
		Addr:           fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:        appEngine,
		ReadTimeout:    time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout:   time.Duration(cfg.Server.WriteTimeout) * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
}

func startServer(srv *http.Server) {
	go func() {
		logger.Info().Int("port", cfg.Server.Port).Msg("HTTP server starting")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal().Err(err).Msg("HTTP server failed")
		}
	}()
}

func waitForShutdown() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info().Msg("Shutting down server...")
}

func shutdown(srv *http.Server) {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Shutdown HTTP server
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error().Err(err).Msg("Server forced to shutdown")
	}

	// Cancel application context to stop all goroutines
	appCancel()

	// Stop all monitoring components in reverse order
	logger.Info().Msg("Stopping monitoring components...")
	metricsWatcher.Stop()
	podWatcher.Stop()
	nodeWatcher.Stop()
	alertEngine.Stop()
	eventBus.Stop()
	k8sClient.Stop()
	logger.Info().Msg("All monitoring components stopped")

	// Close database connection
	closeDatabase(postgresDB)

	logger.Info().Msg("Server exited successfully")
}
