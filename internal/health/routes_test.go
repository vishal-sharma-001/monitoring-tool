package health_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/monitoring-engine/monitoring-tool/internal/health"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestRouter(db *gorm.DB) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	health.RegisterHealthRoutes(router, db)
	return router
}

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	return db
}

func TestRegisterHealthRoutes(t *testing.T) {
	t.Run("should register routes successfully", func(t *testing.T) {
		db := setupTestDB(t)
		router := setupTestRouter(db)

		// Verify routes are registered
		routes := router.Routes()
		assert.NotEmpty(t, routes)

		// Check for health route
		hasHealthRoute := false
		hasInfoRoute := false
		for _, route := range routes {
			if route.Path == "/health" && route.Method == "GET" {
				hasHealthRoute = true
			}
			if route.Path == "/api/info" && route.Method == "GET" {
				hasInfoRoute = true
			}
		}

		assert.True(t, hasHealthRoute, "Expected /health route to be registered")
		assert.True(t, hasInfoRoute, "Expected /api/info route to be registered")
	})
}

func TestGetHealth(t *testing.T) {
	t.Run("should return healthy status with working database", func(t *testing.T) {
		db := setupTestDB(t)
		router := setupTestRouter(db)

		req, _ := http.NewRequest("GET", "/health", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)

		var response map[string]interface{}
		err := json.Unmarshal(resp.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "healthy", response["status"])
		assert.NotEmpty(t, response["timestamp"])

		// Verify timestamp is valid RFC3339
		_, err = time.Parse(time.RFC3339, response["timestamp"].(string))
		assert.NoError(t, err)

		// Check database health
		database, ok := response["database"].(map[string]interface{})
		require.True(t, ok)
		assert.True(t, database["postgres"].(bool))
	})

	t.Run("should return degraded status with nil database", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		router := gin.New()
		health.RegisterHealthRoutes(router, nil)

		req, _ := http.NewRequest("GET", "/health", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusServiceUnavailable, resp.Code)

		var response map[string]interface{}
		err := json.Unmarshal(resp.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "degraded", response["status"])
		assert.NotEmpty(t, response["timestamp"])

		// Check database health
		database, ok := response["database"].(map[string]interface{})
		require.True(t, ok)
		assert.False(t, database["postgres"].(bool))
	})

	t.Run("should return degraded status with closed database", func(t *testing.T) {
		db := setupTestDB(t)
		sqlDB, err := db.DB()
		require.NoError(t, err)

		// Close the database to simulate failure
		sqlDB.Close()

		router := setupTestRouter(db)

		req, _ := http.NewRequest("GET", "/health", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusServiceUnavailable, resp.Code)

		var response map[string]interface{}
		err = json.Unmarshal(resp.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "degraded", response["status"])
	})

	t.Run("should include timestamp in response", func(t *testing.T) {
		db := setupTestDB(t)
		router := setupTestRouter(db)

		before := time.Now()
		req, _ := http.NewRequest("GET", "/health", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)
		after := time.Now()

		var response map[string]interface{}
		err := json.Unmarshal(resp.Body.Bytes(), &response)
		require.NoError(t, err)

		timestamp, err := time.Parse(time.RFC3339, response["timestamp"].(string))
		require.NoError(t, err)

		// Verify timestamp is within reasonable range
		assert.True(t, timestamp.After(before.Add(-1*time.Second)))
		assert.True(t, timestamp.Before(after.Add(1*time.Second)))
	})
}

func TestGetAPIInfo(t *testing.T) {
	t.Run("should return API info successfully", func(t *testing.T) {
		db := setupTestDB(t)
		router := setupTestRouter(db)

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
		gin.SetMode(gin.TestMode)
		router := gin.New()
		health.RegisterHealthRoutes(router, nil)

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

	t.Run("should have correct content type", func(t *testing.T) {
		db := setupTestDB(t)
		router := setupTestRouter(db)

		req, _ := http.NewRequest("GET", "/api/info", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Contains(t, resp.Header().Get("Content-Type"), "application/json")
	})
}

func TestHealthEndpoint_Multiple_Calls(t *testing.T) {
	t.Run("should handle multiple concurrent health checks", func(t *testing.T) {
		db := setupTestDB(t)
		router := setupTestRouter(db)

		// Make multiple concurrent requests
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

		// Wait for all requests to complete
		for i := 0; i < 10; i++ {
			<-done
		}
	})
}

func TestAPIInfoEndpoint_Multiple_Calls(t *testing.T) {
	t.Run("should handle multiple concurrent info requests", func(t *testing.T) {
		db := setupTestDB(t)
		router := setupTestRouter(db)

		// Make multiple concurrent requests
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

		// Wait for all requests to complete
		for i := 0; i < 10; i++ {
			<-done
		}
	})
}

func TestHealthEndpoint_ResponseStructure(t *testing.T) {
	t.Run("should return correct JSON structure", func(t *testing.T) {
		db := setupTestDB(t)
		router := setupTestRouter(db)

		req, _ := http.NewRequest("GET", "/health", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		var response map[string]interface{}
		err := json.Unmarshal(resp.Body.Bytes(), &response)
		require.NoError(t, err)

		// Verify all expected fields exist
		assert.Contains(t, response, "status")
		assert.Contains(t, response, "timestamp")
		assert.Contains(t, response, "database")

		// Verify database object structure
		database, ok := response["database"].(map[string]interface{})
		require.True(t, ok)
		assert.Contains(t, database, "postgres")
	})
}

func TestAPIInfoEndpoint_ResponseStructure(t *testing.T) {
	t.Run("should return correct JSON structure", func(t *testing.T) {
		db := setupTestDB(t)
		router := setupTestRouter(db)

		req, _ := http.NewRequest("GET", "/api/info", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		var response map[string]interface{}
		err := json.Unmarshal(resp.Body.Bytes(), &response)
		require.NoError(t, err)

		// Verify all expected fields exist
		assert.Contains(t, response, "name")
		assert.Contains(t, response, "version")
		assert.Contains(t, response, "status")

		// Verify field types
		_, ok := response["name"].(string)
		assert.True(t, ok)
		_, ok = response["version"].(string)
		assert.True(t, ok)
		_, ok = response["status"].(string)
		assert.True(t, ok)
	})
}
