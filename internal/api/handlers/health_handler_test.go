package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/monitoring-engine/monitoring-tool/internal/api/handlers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupHealthTestRouter(db *gorm.DB) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	handler := handlers.NewHealthHandler(db)
	router.GET("/health", handler.GetHealth)
	router.GET("/api/info", handler.GetAPIInfo)

	return router
}

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	return db
}

func TestNewHealthHandler(t *testing.T) {
	t.Run("should create health handler successfully", func(t *testing.T) {
		db := setupTestDB(t)
		handler := handlers.NewHealthHandler(db)
		assert.NotNil(t, handler)
	})

	t.Run("should create handler with nil db", func(t *testing.T) {
		handler := handlers.NewHealthHandler(nil)
		assert.NotNil(t, handler)
	})
}

func TestHealthHandler_GetHealth(t *testing.T) {
	t.Run("should return healthy status with working database", func(t *testing.T) {
		db := setupTestDB(t)
		router := setupHealthTestRouter(db)

		req, _ := http.NewRequest("GET", "/health", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)

		var response map[string]interface{}
		err := json.Unmarshal(resp.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "healthy", response["status"])
		assert.NotEmpty(t, response["timestamp"])

		database, ok := response["database"].(map[string]interface{})
		require.True(t, ok)
		assert.True(t, database["postgres"].(bool))
	})

	t.Run("should return degraded status with nil database", func(t *testing.T) {
		router := setupHealthTestRouter(nil)

		req, _ := http.NewRequest("GET", "/health", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusServiceUnavailable, resp.Code)

		var response map[string]interface{}
		err := json.Unmarshal(resp.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "degraded", response["status"])
		assert.NotEmpty(t, response["timestamp"])

		database, ok := response["database"].(map[string]interface{})
		require.True(t, ok)
		assert.False(t, database["postgres"].(bool))
	})

	t.Run("should return degraded status with closed database", func(t *testing.T) {
		db := setupTestDB(t)
		sqlDB, err := db.DB()
		require.NoError(t, err)
		sqlDB.Close()

		router := setupHealthTestRouter(db)

		req, _ := http.NewRequest("GET", "/health", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusServiceUnavailable, resp.Code)

		var response map[string]interface{}
		err = json.Unmarshal(resp.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "degraded", response["status"])
	})

	t.Run("should include correct JSON structure", func(t *testing.T) {
		db := setupTestDB(t)
		router := setupHealthTestRouter(db)

		req, _ := http.NewRequest("GET", "/health", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		var response map[string]interface{}
		err := json.Unmarshal(resp.Body.Bytes(), &response)
		require.NoError(t, err)

		// Verify all required fields
		assert.Contains(t, response, "status")
		assert.Contains(t, response, "timestamp")
		assert.Contains(t, response, "database")
	})

	t.Run("should have correct content type", func(t *testing.T) {
		db := setupTestDB(t)
		router := setupHealthTestRouter(db)

		req, _ := http.NewRequest("GET", "/health", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Contains(t, resp.Header().Get("Content-Type"), "application/json")
	})
}

func TestHealthHandler_GetAPIInfo(t *testing.T) {
	t.Run("should return API info successfully", func(t *testing.T) {
		db := setupTestDB(t)
		router := setupHealthTestRouter(db)

		req, _ := http.NewRequest("GET", "/api/info", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)

		var response map[string]interface{}
		err := json.Unmarshal(resp.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "Monitoring Engine MVP", response["name"])
		assert.Equal(t, "1.0.0", response["version"])
		assert.Equal(t, "running", response["status"])
	})

	t.Run("should return API info regardless of database status", func(t *testing.T) {
		router := setupHealthTestRouter(nil)

		req, _ := http.NewRequest("GET", "/api/info", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)

		var response map[string]interface{}
		err := json.Unmarshal(resp.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "Monitoring Engine MVP", response["name"])
		assert.Equal(t, "1.0.0", response["version"])
		assert.Equal(t, "running", response["status"])
	})

	t.Run("should include all required fields", func(t *testing.T) {
		db := setupTestDB(t)
		router := setupHealthTestRouter(db)

		req, _ := http.NewRequest("GET", "/api/info", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		var response map[string]interface{}
		err := json.Unmarshal(resp.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Contains(t, response, "name")
		assert.Contains(t, response, "version")
		assert.Contains(t, response, "status")
	})

	t.Run("should have correct content type", func(t *testing.T) {
		db := setupTestDB(t)
		router := setupHealthTestRouter(db)

		req, _ := http.NewRequest("GET", "/api/info", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Contains(t, resp.Header().Get("Content-Type"), "application/json")
	})
}

func TestHealthHandler_ConcurrentRequests(t *testing.T) {
	t.Run("should handle concurrent health checks", func(t *testing.T) {
		db := setupTestDB(t)
		router := setupHealthTestRouter(db)

		done := make(chan bool, 10)
		for i := 0; i < 10; i++ {
			go func() {
				req, _ := http.NewRequest("GET", "/health", nil)
				resp := httptest.NewRecorder()
				router.ServeHTTP(resp, req)
				assert.Equal(t, http.StatusOK, resp.Code)
				done <- true
			}()
		}

		for i := 0; i < 10; i++ {
			<-done
		}
	})

	t.Run("should handle concurrent API info requests", func(t *testing.T) {
		db := setupTestDB(t)
		router := setupHealthTestRouter(db)

		done := make(chan bool, 10)
		for i := 0; i < 10; i++ {
			go func() {
				req, _ := http.NewRequest("GET", "/api/info", nil)
				resp := httptest.NewRecorder()
				router.ServeHTTP(resp, req)
				assert.Equal(t, http.StatusOK, resp.Code)
				done <- true
			}()
		}

		for i := 0; i < 10; i++ {
			<-done
		}
	})
}
