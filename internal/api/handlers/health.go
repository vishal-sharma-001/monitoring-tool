package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/monitoring-engine/monitoring-tool/internal/storage"
	"gorm.io/gorm"
)

// HealthHandler handles health check requests
type HealthHandler struct {
	db *gorm.DB
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(db *gorm.DB) *HealthHandler {
	return &HealthHandler{
		db: db,
	}
}

// GetHealth handles GET /health
func (h *HealthHandler) GetHealth(c *gin.Context) {
	pgHealth := storage.HealthCheck(h.db)

	status := "healthy"
	code := http.StatusOK

	if pgHealth != nil {
		status = "degraded"
		code = http.StatusServiceUnavailable
	}

	c.JSON(code, gin.H{
		"status":    status,
		"timestamp": time.Now().Format(time.RFC3339),
		"database": gin.H{
			"postgres": pgHealth == nil,
		},
	})
}

// GetAPIInfo handles GET /api/info
func (h *HealthHandler) GetAPIInfo(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"name":    "Monitoring Engine MVP",
		"version": "1.0.0",
		"status":  "running",
	})
}
