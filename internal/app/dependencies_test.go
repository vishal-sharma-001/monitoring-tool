package app_test

import (
	"testing"

	"github.com/monitoring-engine/monitoring-tool/internal/app"
	"github.com/monitoring-engine/monitoring-tool/internal/collector"
	"github.com/monitoring-engine/monitoring-tool/internal/processor"
	"github.com/monitoring-engine/monitoring-tool/internal/repository"
	"github.com/monitoring-engine/monitoring-tool/internal/service"
	"github.com/monitoring-engine/monitoring-tool/internal/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	return db
}

func setupMockDependencies(t *testing.T) (*gorm.DB, *collector.K8sClient, service.AlertService, *processor.EventBus, *websocket.Hub) {
	db := setupTestDB(t)

	// Create mock K8s client (can be nil for tests that don't use it)
	var k8sClient *collector.K8sClient

	// Create real alert service with in-memory repository
	repo := repository.NewInMemoryAlertRepo()
	alertService := service.NewAlertService(repo)

	// Create real event bus
	eventBus := processor.NewEventBus()

	// Create real WebSocket hub
	wsHub := websocket.NewHub()

	return db, k8sClient, alertService, eventBus, wsHub
}

func TestNewDependencies(t *testing.T) {
	t.Run("should create dependencies successfully with all required fields", func(t *testing.T) {
		db, k8sClient, alertService, eventBus, wsHub := setupMockDependencies(t)

		// For this test, create a mock K8s client
		k8sClient = &collector.K8sClient{}

		deps, err := app.NewDependencies(db, k8sClient, alertService, eventBus, wsHub)

		assert.NoError(t, err)
		assert.NotNil(t, deps)
		assert.Equal(t, db, deps.DB)
		assert.Equal(t, k8sClient, deps.K8sClient)
		assert.Equal(t, alertService, deps.AlertService)
		assert.Equal(t, eventBus, deps.EventBus)
		assert.Equal(t, wsHub, deps.WSHub)
	})

	t.Run("should return error when database is nil", func(t *testing.T) {
		_, k8sClient, alertService, eventBus, wsHub := setupMockDependencies(t)
		k8sClient = &collector.K8sClient{}

		deps, err := app.NewDependencies(nil, k8sClient, alertService, eventBus, wsHub)

		assert.Error(t, err)
		assert.Nil(t, deps)
		assert.Contains(t, err.Error(), "database is required")
	})

	t.Run("should return error when k8s client is nil", func(t *testing.T) {
		db, _, alertService, eventBus, wsHub := setupMockDependencies(t)

		deps, err := app.NewDependencies(db, nil, alertService, eventBus, wsHub)

		assert.Error(t, err)
		assert.Nil(t, deps)
		assert.Contains(t, err.Error(), "k8s client is required")
	})

	t.Run("should return error when alert service is nil", func(t *testing.T) {
		db, k8sClient, _, eventBus, wsHub := setupMockDependencies(t)
		k8sClient = &collector.K8sClient{}

		deps, err := app.NewDependencies(db, k8sClient, nil, eventBus, wsHub)

		assert.Error(t, err)
		assert.Nil(t, deps)
		assert.Contains(t, err.Error(), "alert service is required")
	})

	t.Run("should return error when event bus is nil", func(t *testing.T) {
		db, k8sClient, alertService, _, wsHub := setupMockDependencies(t)
		k8sClient = &collector.K8sClient{}

		deps, err := app.NewDependencies(db, k8sClient, alertService, nil, wsHub)

		assert.Error(t, err)
		assert.Nil(t, deps)
		assert.Contains(t, err.Error(), "event bus is required")
	})

	t.Run("should return error when websocket hub is nil", func(t *testing.T) {
		db, k8sClient, alertService, eventBus, _ := setupMockDependencies(t)
		k8sClient = &collector.K8sClient{}

		deps, err := app.NewDependencies(db, k8sClient, alertService, eventBus, nil)

		assert.Error(t, err)
		assert.Nil(t, deps)
		assert.Contains(t, err.Error(), "websocket hub is required")
	})
}

func TestDependencies_Fields(t *testing.T) {
	t.Run("should have accessible fields", func(t *testing.T) {
		db, k8sClient, alertService, eventBus, wsHub := setupMockDependencies(t)
		k8sClient = &collector.K8sClient{}

		deps, err := app.NewDependencies(db, k8sClient, alertService, eventBus, wsHub)
		require.NoError(t, err)

		// Verify all fields are accessible
		assert.NotNil(t, deps.DB)
		assert.NotNil(t, deps.K8sClient)
		assert.NotNil(t, deps.AlertService)
		assert.NotNil(t, deps.EventBus)
		assert.NotNil(t, deps.WSHub)
	})
}

func TestDependencies_Validation(t *testing.T) {
	t.Run("should validate all dependencies at creation", func(t *testing.T) {
		// Test that all nil dependencies are caught
		tests := []struct {
			name      string
			db        *gorm.DB
			k8s       *collector.K8sClient
			service   service.AlertService
			eventBus  *processor.EventBus
			hub       *websocket.Hub
			expectErr string
		}{
			{"nil db", nil, &collector.K8sClient{}, service.NewAlertService(repository.NewInMemoryAlertRepo()), processor.NewEventBus(), websocket.NewHub(), "database is required"},
			{"nil k8s", setupTestDB(t), nil, service.NewAlertService(repository.NewInMemoryAlertRepo()), processor.NewEventBus(), websocket.NewHub(), "k8s client is required"},
			{"nil service", setupTestDB(t), &collector.K8sClient{}, nil, processor.NewEventBus(), websocket.NewHub(), "alert service is required"},
			{"nil eventbus", setupTestDB(t), &collector.K8sClient{}, service.NewAlertService(repository.NewInMemoryAlertRepo()), nil, websocket.NewHub(), "event bus is required"},
			{"nil hub", setupTestDB(t), &collector.K8sClient{}, service.NewAlertService(repository.NewInMemoryAlertRepo()), processor.NewEventBus(), nil, "websocket hub is required"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				deps, err := app.NewDependencies(tt.db, tt.k8s, tt.service, tt.eventBus, tt.hub)
				assert.Error(t, err)
				assert.Nil(t, deps)
				assert.Contains(t, err.Error(), tt.expectErr)
			})
		}
	})
}
