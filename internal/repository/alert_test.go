package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/monitoring-engine/monitoring-tool/internal/models"
	"github.com/monitoring-engine/monitoring-tool/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
)

func TestInMemoryAlertRepo_Create(t *testing.T) {
	repo := repository.NewInMemoryAlertRepo()
	ctx := context.Background()

	t.Run("should create alert successfully", func(t *testing.T) {
		alert := &models.Alert{
			ID:          uuid.New(),
			Status:      models.AlertStatusFiring,
			Severity:    "high",
			Message:     "Test alert",
			Source:      "test",
			Labels:      datatypes.JSON([]byte(`{}`)),
			Value:       100.0,
			TriggeredAt: time.Now(),
		}

		err := repo.Create(ctx, alert)
		assert.NoError(t, err)
	})

	t.Run("should create multiple alerts", func(t *testing.T) {
		for i := 0; i < 5; i++ {
			alert := &models.Alert{
				ID:          uuid.New(),
				Status:      models.AlertStatusFiring,
				Severity:    "medium",
				Message:     "Test alert",
				Source:      "test",
				Value:       float64(i),
				TriggeredAt: time.Now(),
			}
			err := repo.Create(ctx, alert)
			assert.NoError(t, err)
		}
	})
}

func TestInMemoryAlertRepo_GetRecent(t *testing.T) {
	repo := repository.NewInMemoryAlertRepo()
	ctx := context.Background()

	t.Run("should return empty when no alerts", func(t *testing.T) {
		alerts, err := repo.GetRecent(ctx, 10)
		assert.NoError(t, err)
		assert.Empty(t, alerts)
	})

	t.Run("should return all alerts when limit is greater than count", func(t *testing.T) {
		repo := repository.NewInMemoryAlertRepo()

		for i := 0; i < 3; i++ {
			alert := &models.Alert{
				ID:          uuid.New(),
				Status:      models.AlertStatusFiring,
				Severity:    "low",
				Message:     "Test alert",
				Source:      "test",
				Value:       float64(i),
				TriggeredAt: time.Now(),
			}
			err := repo.Create(ctx, alert)
			require.NoError(t, err)
		}

		alerts, err := repo.GetRecent(ctx, 10)
		assert.NoError(t, err)
		assert.Len(t, alerts, 3)
	})

	t.Run("should return limited alerts", func(t *testing.T) {
		repo := repository.NewInMemoryAlertRepo()

		for i := 0; i < 10; i++ {
			alert := &models.Alert{
				ID:          uuid.New(),
				Status:      models.AlertStatusFiring,
				Severity:    "critical",
				Message:     "Test alert",
				Source:      "test",
				Value:       float64(i),
				TriggeredAt: time.Now(),
			}
			err := repo.Create(ctx, alert)
			require.NoError(t, err)
		}

		alerts, err := repo.GetRecent(ctx, 5)
		assert.NoError(t, err)
		assert.Len(t, alerts, 5)
	})

	t.Run("should return most recent alerts", func(t *testing.T) {
		repo := repository.NewInMemoryAlertRepo()

		var createdAlerts []*models.Alert
		for i := 0; i < 5; i++ {
			alert := &models.Alert{
				ID:          uuid.New(),
				Status:      models.AlertStatusFiring,
				Severity:    "high",
				Message:     "Test alert",
				Source:      "test",
				Value:       float64(i),
				TriggeredAt: time.Now(),
			}
			err := repo.Create(ctx, alert)
			require.NoError(t, err)
			createdAlerts = append(createdAlerts, alert)
			time.Sleep(time.Millisecond)
		}

		alerts, err := repo.GetRecent(ctx, 3)
		assert.NoError(t, err)
		assert.Len(t, alerts, 3)

		// Should get the last 3 alerts
		assert.Equal(t, createdAlerts[2].Value, alerts[0].Value)
		assert.Equal(t, createdAlerts[3].Value, alerts[1].Value)
		assert.Equal(t, createdAlerts[4].Value, alerts[2].Value)
	})
}

func TestInMemoryAlertRepo_ConcurrentAccess(t *testing.T) {
	repo := repository.NewInMemoryAlertRepo()
	ctx := context.Background()

	t.Run("should handle concurrent writes", func(t *testing.T) {
		done := make(chan bool)

		for i := 0; i < 10; i++ {
			go func(val int) {
				alert := &models.Alert{
					ID:          uuid.New(),
					Status:      models.AlertStatusFiring,
					Severity:    "medium",
					Message:     "Concurrent alert",
					Source:      "test",
					Value:       float64(val),
					TriggeredAt: time.Now(),
				}
				err := repo.Create(ctx, alert)
				assert.NoError(t, err)
				done <- true
			}(i)
		}

		for i := 0; i < 10; i++ {
			<-done
		}

		alerts, err := repo.GetRecent(ctx, 100)
		assert.NoError(t, err)
		assert.Len(t, alerts, 10)
	})

	t.Run("should handle concurrent reads", func(t *testing.T) {
		repo := repository.NewInMemoryAlertRepo()

		for i := 0; i < 5; i++ {
			alert := &models.Alert{
				ID:          uuid.New(),
				Status:      models.AlertStatusFiring,
				Severity:    "low",
				Message:     "Test alert",
				Source:      "test",
				Value:       float64(i),
				TriggeredAt: time.Now(),
			}
			err := repo.Create(ctx, alert)
			require.NoError(t, err)
		}

		done := make(chan bool)
		for i := 0; i < 10; i++ {
			go func() {
				alerts, err := repo.GetRecent(ctx, 3)
				assert.NoError(t, err)
				assert.Len(t, alerts, 3)
				done <- true
			}()
		}

		for i := 0; i < 10; i++ {
			<-done
		}
	})
}

func TestNewInMemoryAlertRepo(t *testing.T) {
	repo := repository.NewInMemoryAlertRepo()
	assert.NotNil(t, repo)

	// Test that it implements the interface
	var _ repository.AlertRepo = repo
}
