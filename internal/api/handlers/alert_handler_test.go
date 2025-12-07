package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/monitoring-engine/monitoring-tool/internal/models"
	"github.com/monitoring-engine/monitoring-tool/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/datatypes"
)

// MockAlertService is a mock implementation of AlertService
type MockAlertService struct {
	mock.Mock
}

func (m *MockAlertService) CreateAlert(ctx context.Context, alert *models.Alert) error {
	args := m.Called(ctx, alert)
	return args.Error(0)
}

func (m *MockAlertService) GetRecentAlerts(ctx context.Context, limit int) ([]*models.Alert, error) {
	args := m.Called(ctx, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Alert), args.Error(1)
}

func (m *MockAlertService) GetTotalAlertsCount(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockAlertService) GetFiringAlertsCount(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockAlertService) GetSeverityCounts(ctx context.Context) (*service.SeverityCounts, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.SeverityCounts), args.Error(1)
}

func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.Default()
}

func TestAlertHandler_GetRecentAlerts_Success(t *testing.T) {
	mockService := new(MockAlertService)
	now := time.Now()
	mockAlerts := []*models.Alert{
		{
			ID:          uuid.New(),
			Status:      models.AlertStatusFiring,
			Severity:    "high",
			Message:     "Test alert 1",
			Source:      "k8s_pod",
			Labels:      datatypes.JSON([]byte(`{"pod":"test"}`)),
			Value:       95.0,
			TriggeredAt: now,
		},
		{
			ID:          uuid.New(),
			Status:      models.AlertStatusFiring,
			Severity:    "low",
			Message:     "Test alert 2",
			Source:      "rule",
			Labels:      datatypes.JSON([]byte(`{}`)),
			Value:       50.0,
			TriggeredAt: now,
		},
	}

	mockService.On("GetRecentAlerts", mock.Anything, 50).Return(mockAlerts, nil)

	handler := NewAlertHandler(mockService)
	router := setupRouter()
	router.GET("/alerts/recent", handler.GetRecentAlerts)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/alerts/recent", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, float64(2), response["count"])
	assert.NotNil(t, response["alerts"])

	mockService.AssertExpectations(t)
}

func TestAlertHandler_GetRecentAlerts_WithLimit(t *testing.T) {
	mockService := new(MockAlertService)
	mockAlerts := []*models.Alert{}

	mockService.On("GetRecentAlerts", mock.Anything, 10).Return(mockAlerts, nil)

	handler := NewAlertHandler(mockService)
	router := setupRouter()
	router.GET("/alerts/recent", handler.GetRecentAlerts)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/alerts/recent?limit=10", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestAlertHandler_GetRecentAlerts_InvalidLimit(t *testing.T) {
	mockService := new(MockAlertService)
	mockAlerts := []*models.Alert{}

	mockService.On("GetRecentAlerts", mock.Anything, 50).Return(mockAlerts, nil)

	handler := NewAlertHandler(mockService)
	router := setupRouter()
	router.GET("/alerts/recent", handler.GetRecentAlerts)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/alerts/recent?limit=invalid", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestAlertHandler_GetRecentAlerts_ServiceError(t *testing.T) {
	mockService := new(MockAlertService)
	mockService.On("GetRecentAlerts", mock.Anything, 50).Return(nil, errors.New("service error"))

	handler := NewAlertHandler(mockService)
	router := setupRouter()
	router.GET("/alerts/recent", handler.GetRecentAlerts)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/alerts/recent", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["error"], "service error")

	mockService.AssertExpectations(t)
}

func TestAlertHandler_GetAlertsCount_Success(t *testing.T) {
	mockService := new(MockAlertService)

	mockService.On("GetTotalAlertsCount", mock.Anything).Return(int64(42), nil)

	handler := NewAlertHandler(mockService)
	router := setupRouter()
	router.GET("/alerts/count", handler.GetAlertsCount)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/alerts/count", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, float64(42), response["count"])

	mockService.AssertExpectations(t)
}

func TestAlertHandler_GetAlertsCount_ServiceError(t *testing.T) {
	mockService := new(MockAlertService)
	mockService.On("GetTotalAlertsCount", mock.Anything).Return(int64(0), errors.New("database error"))

	handler := NewAlertHandler(mockService)
	router := setupRouter()
	router.GET("/alerts/count", handler.GetAlertsCount)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/alerts/count", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

func TestNewAlertHandler(t *testing.T) {
	mockService := new(MockAlertService)
	handler := NewAlertHandler(mockService)

	assert.NotNil(t, handler)
	assert.NotNil(t, handler.service)
}
