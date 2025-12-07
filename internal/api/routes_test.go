package api_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/monitoring-engine/monitoring-tool/internal/api"
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

func setupTestDependencies(t *testing.T) *app.Dependencies {
	db := setupTestDB(t)
	k8sClient := &collector.K8sClient{}
	repo := repository.NewInMemoryAlertRepo()
	alertService := service.NewAlertService(repo)
	eventBus := processor.NewEventBus()
	wsHub := websocket.NewHub()

	deps, err := app.NewDependencies(db, k8sClient, alertService, eventBus, wsHub)
	require.NoError(t, err)

	// Start the WebSocket hub
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go wsHub.Run(ctx)

	return deps
}

func TestRegisterRoutes(t *testing.T) {
	t.Run("should register all routes successfully", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		router := gin.New()
		deps := setupTestDependencies(t)

		// Should not panic
		api.RegisterRoutes(deps, router)

		// Verify router is not nil
		assert.NotNil(t, router)
	})

	t.Run("should register health routes", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		router := gin.New()
		deps := setupTestDependencies(t)

		api.RegisterRoutes(deps, router)

		// Test health endpoint exists
		req, _ := http.NewRequest("GET", "/health", nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
	})

	t.Run("should register API info route", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		router := gin.New()
		deps := setupTestDependencies(t)

		api.RegisterRoutes(deps, router)

		// Test API info endpoint exists
		req, _ := http.NewRequest("GET", "/api/info", nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
	})

	t.Run("should register alert API routes", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		router := gin.New()
		deps := setupTestDependencies(t)

		api.RegisterRoutes(deps, router)

		// Test alerts recent endpoint exists
		req, _ := http.NewRequest("GET", "/api/alerts/recent", nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		// Should return 200 with empty array
		assert.Equal(t, http.StatusOK, resp.Code)
	})

	t.Run("should register alerts count endpoint", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		router := gin.New()
		deps := setupTestDependencies(t)

		api.RegisterRoutes(deps, router)

		// Test alerts count endpoint exists
		req, _ := http.NewRequest("GET", "/api/alerts/count", nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
	})

	t.Run("should register WebSocket route", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		router := gin.New()
		deps := setupTestDependencies(t)

		api.RegisterRoutes(deps, router)

		// Test WebSocket endpoint exists (will return 400 for non-WebSocket request)
		req, _ := http.NewRequest("GET", "/ws", nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		// WebSocket upgrade should fail with bad request
		assert.Equal(t, http.StatusBadRequest, resp.Code)
	})

	t.Run("should handle 404 for non-existent routes", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		router := gin.New()
		deps := setupTestDependencies(t)

		api.RegisterRoutes(deps, router)

		// Test non-existent endpoint
		req, _ := http.NewRequest("GET", "/api/nonexistent", nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusNotFound, resp.Code)
	})
}

func TestRegisterRoutes_WithDifferentMethods(t *testing.T) {
	t.Run("should only accept GET for health endpoint", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		router := gin.New()
		deps := setupTestDependencies(t)

		api.RegisterRoutes(deps, router)

		// POST should return 404 (method not allowed)
		req, _ := http.NewRequest("POST", "/health", nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusNotFound, resp.Code)
	})

	t.Run("should only accept GET for alerts endpoints", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		router := gin.New()
		deps := setupTestDependencies(t)

		api.RegisterRoutes(deps, router)

		// POST to recent endpoint should fail
		req, _ := http.NewRequest("POST", "/api/alerts/recent", nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusNotFound, resp.Code)
	})
}

func TestRegisterRoutes_Integration(t *testing.T) {
	t.Run("should handle multiple concurrent requests", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		router := gin.New()
		deps := setupTestDependencies(t)

		api.RegisterRoutes(deps, router)

		done := make(chan bool, 10)

		// Send concurrent requests
		for i := 0; i < 10; i++ {
			go func() {
				req, _ := http.NewRequest("GET", "/health", nil)
				resp := httptest.NewRecorder()
				router.ServeHTTP(resp, req)
				assert.Equal(t, http.StatusOK, resp.Code)
				done <- true
			}()
		}

		// Wait for all requests to complete
		for i := 0; i < 10; i++ {
			<-done
		}
	})

	t.Run("should serve all endpoints with valid dependencies", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		router := gin.New()
		deps := setupTestDependencies(t)

		api.RegisterRoutes(deps, router)

		// Test multiple endpoints
		endpoints := []string{
			"/health",
			"/api/info",
			"/api/alerts/recent",
			"/api/alerts/count",
		}

		for _, endpoint := range endpoints {
			req, _ := http.NewRequest("GET", endpoint, nil)
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			assert.Equal(t, http.StatusOK, resp.Code, "endpoint %s should return 200", endpoint)
		}
	})
}
