package health

import (
	"github.com/gin-gonic/gin"
	"github.com/monitoring-engine/monitoring-tool/internal/api/handlers"
	"gorm.io/gorm"
)

// RegisterHealthRoutes registers health check and info endpoints
// This follows the module-based router pattern from portal-backend-v3
func RegisterHealthRoutes(router *gin.Engine, db *gorm.DB) {
	// Create handler with dependencies
	healthHandler := handlers.NewHealthHandler(db)

	// Register routes (no authentication required)
	router.GET("/health", healthHandler.GetHealth)
	router.GET("/api/info", healthHandler.GetAPIInfo)
}
