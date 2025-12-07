package repository

import (
	"context"
	"sync"

	"github.com/monitoring-engine/monitoring-tool/internal/models"
	"gorm.io/gorm"
)

// AlertRepo interface for alert storage
type AlertRepo interface {
	Create(ctx context.Context, alert *models.Alert) error
	GetRecent(ctx context.Context, limit int) ([]*models.Alert, error)
	Count(ctx context.Context) (int64, error)
	CountByStatus(ctx context.Context, status models.AlertStatus) (int64, error)
	CountBySeverity(ctx context.Context, severity string) (int64, error)
}

// InMemoryAlertRepo stores alerts in memory
type InMemoryAlertRepo struct {
	alerts []*models.Alert
	mu     sync.RWMutex
}

func NewInMemoryAlertRepo() AlertRepo {
	return &InMemoryAlertRepo{
		alerts: make([]*models.Alert, 0, 1000),
	}
}

func (r *InMemoryAlertRepo) Create(ctx context.Context, alert *models.Alert) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.alerts = append(r.alerts, alert)
	return nil
}

func (r *InMemoryAlertRepo) GetRecent(ctx context.Context, limit int) ([]*models.Alert, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	start := len(r.alerts) - limit
	if start < 0 {
		start = 0
	}
	return r.alerts[start:], nil
}

func (r *InMemoryAlertRepo) Count(ctx context.Context) (int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return int64(len(r.alerts)), nil
}

func (r *InMemoryAlertRepo) CountByStatus(ctx context.Context, status models.AlertStatus) (int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	count := int64(0)
	for _, alert := range r.alerts {
		if alert.Status == status {
			count++
		}
	}
	return count, nil
}

func (r *InMemoryAlertRepo) CountBySeverity(ctx context.Context, severity string) (int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	count := int64(0)
	for _, alert := range r.alerts {
		if alert.Severity == severity && alert.Status == models.AlertStatusFiring {
			count++
		}
	}
	return count, nil
}

// PostgresAlertRepo stores alerts in PostgreSQL
type PostgresAlertRepo struct {
	db *gorm.DB
}

func NewPostgresAlertRepo(db *gorm.DB) AlertRepo {
	return &PostgresAlertRepo{db: db}
}

func (r *PostgresAlertRepo) Create(ctx context.Context, alert *models.Alert) error {
	return r.db.WithContext(ctx).Create(alert).Error
}

func (r *PostgresAlertRepo) GetRecent(ctx context.Context, limit int) ([]*models.Alert, error) {
	var alerts []*models.Alert
	err := r.db.WithContext(ctx).
		Order("triggered_at DESC").
		Limit(limit).
		Find(&alerts).Error
	return alerts, err
}

func (r *PostgresAlertRepo) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.Alert{}).
		Count(&count).Error
	return count, err
}

func (r *PostgresAlertRepo) CountByStatus(ctx context.Context, status models.AlertStatus) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.Alert{}).
		Where("status = ?", status).
		Count(&count).Error
	return count, err
}

func (r *PostgresAlertRepo) CountBySeverity(ctx context.Context, severity string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.Alert{}).
		Where("severity = ? AND status = ?", severity, models.AlertStatusFiring).
		Count(&count).Error
	return count, err
}
