package api

import (
	"github.com/gin-gonic/gin"
	"github.com/monitoring-engine/monitoring-tool/internal/api/handlers"
	"github.com/monitoring-engine/monitoring-tool/internal/app"
	"github.com/monitoring-engine/monitoring-tool/internal/health"
)

// RegisterRoutes registers all application routes using dependencies container
// This follows the central router registration pattern from portal-backend-v3
func RegisterRoutes(deps *app.Dependencies, router *gin.Engine) {
	// Serve static files for the web UI
	router.Static("/static", "./web/static")

	// Serve dashboard UI at root
	router.GET("/", func(c *gin.Context) {
		c.File("./web/static/index.html")
	})

	// Health routes (no authentication required)
	health.RegisterHealthRoutes(router, deps.DB)

	// Alert API routes (versioned)
	alertHandler := handlers.NewAlertHandler(deps.AlertService)
	apiV1 := router.Group("/api")
	{
		alertGroup := apiV1.Group("/alerts")
		{
			alertGroup.GET("/recent", alertHandler.GetRecentAlerts)
			alertGroup.GET("/count", alertHandler.GetAlertsCount)
			alertGroup.GET("/active/count", alertHandler.GetFiringAlertsCount)
			alertGroup.GET("/severity/counts", alertHandler.GetSeverityCounts)
		}
	}

	// WebSocket route
	handlers.RegisterWebSocketRoutes(router, deps.WSHub)
}
