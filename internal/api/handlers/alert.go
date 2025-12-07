package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/monitoring-engine/monitoring-tool/internal/service"
)

// AlertHandler handles alert HTTP requests
type AlertHandler struct {
	service service.AlertService
}

// NewAlertHandler creates a new alert handler
func NewAlertHandler(service service.AlertService) *AlertHandler {
	return &AlertHandler{
		service: service,
	}
}

// GetRecentAlerts handles GET /api/alerts/recent
func (h *AlertHandler) GetRecentAlerts(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "50")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 50
	}

	alerts, err := h.service.GetRecentAlerts(c.Request.Context(), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"alerts": alerts,
		"count":  len(alerts),
	})
}

// GetAlertsCount handles GET /api/alerts/count
func (h *AlertHandler) GetAlertsCount(c *gin.Context) {
	count, err := h.service.GetTotalAlertsCount(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"count": count,
	})
}

// GetFiringAlertsCount handles GET /api/alerts/active/count
func (h *AlertHandler) GetFiringAlertsCount(c *gin.Context) {
	count, err := h.service.GetFiringAlertsCount(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"count": count,
	})
}

// GetSeverityCounts handles GET /api/alerts/severity/counts
func (h *AlertHandler) GetSeverityCounts(c *gin.Context) {
	counts, err := h.service.GetSeverityCounts(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, counts)
}
