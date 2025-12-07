package service

import (
	"context"

	"github.com/monitoring-engine/monitoring-tool/internal/models"
	"github.com/monitoring-engine/monitoring-tool/internal/repository"
)

// SeverityCounts holds counts for each severity level
type SeverityCounts struct {
	Critical int64 `json:"critical"`
	High     int64 `json:"high"`
	Medium   int64 `json:"medium"`
	Low      int64 `json:"low"`
}

// AlertService handles alert business logic
type AlertService interface {
	CreateAlert(ctx context.Context, alert *models.Alert) error
	GetRecentAlerts(ctx context.Context, limit int) ([]*models.Alert, error)
	GetTotalAlertsCount(ctx context.Context) (int64, error)
	GetFiringAlertsCount(ctx context.Context) (int64, error)
	GetSeverityCounts(ctx context.Context) (*SeverityCounts, error)
}

type alertService struct {
	repo repository.AlertRepo
}

// NewAlertService creates a new alert service
func NewAlertService(repo repository.AlertRepo) AlertService {
	return &alertService{
		repo: repo,
	}
}

func (s *alertService) CreateAlert(ctx context.Context, alert *models.Alert) error {
	return s.repo.Create(ctx, alert)
}

func (s *alertService) GetRecentAlerts(ctx context.Context, limit int) ([]*models.Alert, error) {
	if limit <= 0 || limit > 1000 {
		limit = 100 // default limit
	}
	return s.repo.GetRecent(ctx, limit)
}

func (s *alertService) GetTotalAlertsCount(ctx context.Context) (int64, error) {
	return s.repo.Count(ctx)
}

func (s *alertService) GetFiringAlertsCount(ctx context.Context) (int64, error) {
	return s.repo.CountByStatus(ctx, models.AlertStatusFiring)
}

func (s *alertService) GetSeverityCounts(ctx context.Context) (*SeverityCounts, error) {
	critical, err := s.repo.CountBySeverity(ctx, "critical")
	if err != nil {
		return nil, err
	}
	high, err := s.repo.CountBySeverity(ctx, "high")
	if err != nil {
		return nil, err
	}
	medium, err := s.repo.CountBySeverity(ctx, "medium")
	if err != nil {
		return nil, err
	}
	low, err := s.repo.CountBySeverity(ctx, "low")
	if err != nil {
		return nil, err
	}
	return &SeverityCounts{
		Critical: critical,
		High:     high,
		Medium:   medium,
		Low:      low,
	}, nil
}
