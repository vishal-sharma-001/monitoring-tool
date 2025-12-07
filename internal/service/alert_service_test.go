package service_test

import (
	"context"
	"errors"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/google/uuid"
	"github.com/monitoring-engine/monitoring-tool/internal/models"
	"github.com/monitoring-engine/monitoring-tool/internal/service"
	"gorm.io/datatypes"
)

// MockAlertRepo is a mock implementation of AlertRepo for testing
type MockAlertRepo struct {
	CreateFunc          func(ctx context.Context, alert *models.Alert) error
	GetRecentFunc       func(ctx context.Context, limit int) ([]*models.Alert, error)
	CountFunc           func(ctx context.Context) (int64, error)
	CountByStatusFunc   func(ctx context.Context, status models.AlertStatus) (int64, error)
	CountBySeverityFunc func(ctx context.Context, severity string) (int64, error)
}

func (m *MockAlertRepo) Create(ctx context.Context, alert *models.Alert) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, alert)
	}
	return nil
}

func (m *MockAlertRepo) GetRecent(ctx context.Context, limit int) ([]*models.Alert, error) {
	if m.GetRecentFunc != nil {
		return m.GetRecentFunc(ctx, limit)
	}
	return []*models.Alert{}, nil
}

func (m *MockAlertRepo) Count(ctx context.Context) (int64, error) {
	if m.CountFunc != nil {
		return m.CountFunc(ctx)
	}
	return 0, nil
}

func (m *MockAlertRepo) CountByStatus(ctx context.Context, status models.AlertStatus) (int64, error) {
	if m.CountByStatusFunc != nil {
		return m.CountByStatusFunc(ctx, status)
	}
	return 0, nil
}

func (m *MockAlertRepo) CountBySeverity(ctx context.Context, severity string) (int64, error) {
	if m.CountBySeverityFunc != nil {
		return m.CountBySeverityFunc(ctx, severity)
	}
	return 0, nil
}

var _ = Describe("AlertService", func() {
	var (
		mockRepo     *MockAlertRepo
		alertService service.AlertService
		ctx          context.Context
	)

	BeforeEach(func() {
		mockRepo = &MockAlertRepo{}
		alertService = service.NewAlertService(mockRepo)
		ctx = context.Background()
	})

	Describe("NewAlertService", func() {
		It("should create a new alert service", func() {
			Expect(alertService).NotTo(BeNil())
		})
	})

	Describe("CreateAlert", func() {
		Context("when alert creation succeeds", func() {
			It("should create an alert successfully", func() {
				now := time.Now()
				testAlert := &models.Alert{
					ID:          uuid.New(),
					Status:      models.AlertStatusFiring,
					Severity:    "high",
					Message:     "Test alert",
					Source:      "test",
					Labels:      datatypes.JSON([]byte(`{}`)),
					Value:       100.0,
					TriggeredAt: now,
				}

				mockRepo.CreateFunc = func(ctx context.Context, alert *models.Alert) error {
					Expect(alert).To(Equal(testAlert))
					return nil
				}

				err := alertService.CreateAlert(ctx, testAlert)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when alert creation fails", func() {
			It("should return an error", func() {
				now := time.Now()
				testAlert := &models.Alert{
					ID:          uuid.New(),
					Status:      models.AlertStatusFiring,
					Severity:    "high",
					Message:     "Test alert",
					Source:      "test",
					Labels:      datatypes.JSON([]byte(`{}`)),
					Value:       100.0,
					TriggeredAt: now,
				}

				expectedErr := errors.New("database error")
				mockRepo.CreateFunc = func(ctx context.Context, alert *models.Alert) error {
					return expectedErr
				}

				err := alertService.CreateAlert(ctx, testAlert)
				Expect(err).To(MatchError(expectedErr))
			})
		})
	})

	Describe("GetRecentAlerts", func() {
		Context("with valid limit", func() {
			It("should return alerts successfully", func() {
				mockAlerts := []*models.Alert{
					{ID: uuid.New(), Severity: "high", Message: "Alert 1"},
					{ID: uuid.New(), Severity: "low", Message: "Alert 2"},
				}

				mockRepo.GetRecentFunc = func(ctx context.Context, limit int) ([]*models.Alert, error) {
					Expect(limit).To(Equal(10))
					return mockAlerts, nil
				}

				alerts, err := alertService.GetRecentAlerts(ctx, 10)
				Expect(err).NotTo(HaveOccurred())
				Expect(alerts).To(Equal(mockAlerts))
			})
		})

		Context("with limit too low", func() {
			It("should default to 100", func() {
				mockRepo.GetRecentFunc = func(ctx context.Context, limit int) ([]*models.Alert, error) {
					Expect(limit).To(Equal(100))
					return []*models.Alert{}, nil
				}

				_, err := alertService.GetRecentAlerts(ctx, 0)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("with limit too high", func() {
			It("should default to 100", func() {
				mockRepo.GetRecentFunc = func(ctx context.Context, limit int) ([]*models.Alert, error) {
					Expect(limit).To(Equal(100))
					return []*models.Alert{}, nil
				}

				_, err := alertService.GetRecentAlerts(ctx, 2000)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("with negative limit", func() {
			It("should default to 100", func() {
				mockRepo.GetRecentFunc = func(ctx context.Context, limit int) ([]*models.Alert, error) {
					Expect(limit).To(Equal(100))
					return []*models.Alert{}, nil
				}

				_, err := alertService.GetRecentAlerts(ctx, -5)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when repository returns an error", func() {
			It("should return the error", func() {
				expectedErr := errors.New("database error")
				mockRepo.GetRecentFunc = func(ctx context.Context, limit int) ([]*models.Alert, error) {
					return []*models.Alert{}, expectedErr
				}

				_, err := alertService.GetRecentAlerts(ctx, 10)
				Expect(err).To(MatchError(expectedErr))
			})
		})
	})
})
