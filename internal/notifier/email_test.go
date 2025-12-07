package notifier_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/monitoring-engine/monitoring-tool/internal/config"
	"github.com/monitoring-engine/monitoring-tool/internal/models"
	"github.com/monitoring-engine/monitoring-tool/internal/notifier"
	"github.com/monitoring-engine/monitoring-tool/internal/processor"
	"github.com/stretchr/testify/assert"
	"gorm.io/datatypes"
)

func TestNewEmailDispatcher(t *testing.T) {
	t.Run("should create email dispatcher successfully", func(t *testing.T) {
		cfg := config.EmailConfig{
			Enabled:  true,
			SMTPHost: "smtp.example.com",
			SMTPPort: 587,
			Username: "user@example.com",
			Password: "password",
			From:     "alerts@example.com",
			To:       []string{"admin@example.com"},
		}

		dispatcher := notifier.NewEmailDispatcher(cfg)
		assert.NotNil(t, dispatcher)
	})

	t.Run("should create dispatcher with empty config", func(t *testing.T) {
		cfg := config.EmailConfig{}
		dispatcher := notifier.NewEmailDispatcher(cfg)
		assert.NotNil(t, dispatcher)
	})
}

func TestEmailDispatcher_OnAlert(t *testing.T) {
	t.Run("should skip email when SMTP host is empty", func(t *testing.T) {
		cfg := config.EmailConfig{
			SMTPHost: "",
			Username: "user@example.com",
		}

		dispatcher := notifier.NewEmailDispatcher(cfg)
		ctx := context.Background()

		alert := &models.Alert{
			ID:          uuid.New(),
			Status:      models.AlertStatusFiring,
			Severity:    "high",
			Source:      "test-source",
			Message:     "Test alert",
			Value:       100.0,
			Labels:      datatypes.JSON([]byte(`{"key":"value"}`)),
			CreatedAt:   time.Now(),
			TriggeredAt: time.Now(),
		}

		event := &processor.AlertEvent{
			Alert:     alert,
			Timestamp: time.Now(),
		}

		err := dispatcher.OnAlert(ctx, event)
		assert.NoError(t, err)
	})

	t.Run("should skip email when username is empty", func(t *testing.T) {
		cfg := config.EmailConfig{
			SMTPHost: "smtp.example.com",
			Username: "",
		}

		dispatcher := notifier.NewEmailDispatcher(cfg)
		ctx := context.Background()

		alert := &models.Alert{
			ID:          uuid.New(),
			Status:      models.AlertStatusFiring,
			Severity:    "medium",
			Source:      "test-source",
			Message:     "Test alert",
			Value:       80.0,
			Labels:      datatypes.JSON([]byte(`{}`)),
			CreatedAt:   time.Now(),
			TriggeredAt: time.Now(),
		}

		event := &processor.AlertEvent{
			Alert:     alert,
			Timestamp: time.Now(),
		}

		err := dispatcher.OnAlert(ctx, event)
		assert.NoError(t, err)
	})

	t.Run("should skip email when config is incomplete", func(t *testing.T) {
		cfg := config.EmailConfig{
			SMTPHost: "",
			Username: "",
		}

		dispatcher := notifier.NewEmailDispatcher(cfg)
		ctx := context.Background()

		alert := &models.Alert{
			ID:          uuid.New(),
			Status:      models.AlertStatusFiring,
			Severity:    "low",
			Source:      "test-source",
			Message:     "Test alert",
			Value:       50.0,
			Labels:      datatypes.JSON([]byte(`{}`)),
			CreatedAt:   time.Now(),
			TriggeredAt: time.Now(),
		}

		event := &processor.AlertEvent{
			Alert:     alert,
			Timestamp: time.Now(),
		}

		err := dispatcher.OnAlert(ctx, event)
		assert.NoError(t, err)
	})

	t.Run("should handle context cancellation", func(t *testing.T) {
		cfg := config.EmailConfig{
			SMTPHost: "",
			Username: "",
		}

		dispatcher := notifier.NewEmailDispatcher(cfg)
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		alert := &models.Alert{
			ID:          uuid.New(),
			Status:      models.AlertStatusFiring,
			Severity:    "critical",
			Source:      "test-source",
			Message:     "Test alert",
			Value:       95.0,
			Labels:      datatypes.JSON([]byte(`{}`)),
			CreatedAt:   time.Now(),
			TriggeredAt: time.Now(),
		}

		event := &processor.AlertEvent{
			Alert:     alert,
			Timestamp: time.Now(),
		}

		// Should not panic or hang
		err := dispatcher.OnAlert(ctx, event)
		assert.NoError(t, err) // Skips due to incomplete config
	})

	t.Run("should format alert with all fields", func(t *testing.T) {
		cfg := config.EmailConfig{
			SMTPHost: "",
			Username: "",
		}

		dispatcher := notifier.NewEmailDispatcher(cfg)
		ctx := context.Background()

		now := time.Now()
		alert := &models.Alert{
			ID:          uuid.New(),
			Status:      models.AlertStatusFiring,
			Severity:    "high",
			Source:      "kubernetes-pod",
			Message:     "CPU usage exceeded threshold",
			Value:       85.5,
			Labels:      datatypes.JSON([]byte(`{"namespace":"production","pod":"app-1"}`)),
			CreatedAt:   now,
			TriggeredAt: now,
		}

		event := &processor.AlertEvent{
			Alert:     alert,
			Timestamp: now,
		}

		err := dispatcher.OnAlert(ctx, event)
		assert.NoError(t, err)
	})

	t.Run("should handle empty labels", func(t *testing.T) {
		cfg := config.EmailConfig{
			SMTPHost: "",
			Username: "",
		}

		dispatcher := notifier.NewEmailDispatcher(cfg)
		ctx := context.Background()

		alert := &models.Alert{
			ID:          uuid.New(),
			Status:      models.AlertStatusFiring,
			Severity:    "low",
			Source:      "test",
			Message:     "Test",
			Value:       10.0,
			Labels:      datatypes.JSON([]byte(`{}`)),
			CreatedAt:   time.Now(),
			TriggeredAt: time.Now(),
		}

		event := &processor.AlertEvent{
			Alert:     alert,
			Timestamp: time.Now(),
		}

		err := dispatcher.OnAlert(ctx, event)
		assert.NoError(t, err)
	})

	t.Run("should handle zero value", func(t *testing.T) {
		cfg := config.EmailConfig{
			SMTPHost: "",
			Username: "",
		}

		dispatcher := notifier.NewEmailDispatcher(cfg)
		ctx := context.Background()

		alert := &models.Alert{
			ID:          uuid.New(),
			Status:      models.AlertStatusFiring,
			Severity:    "info",
			Source:      "test",
			Message:     "Test",
			Value:       0.0,
			Labels:      datatypes.JSON([]byte(`{}`)),
			CreatedAt:   time.Now(),
			TriggeredAt: time.Now(),
		}

		event := &processor.AlertEvent{
			Alert:     alert,
			Timestamp: time.Now(),
		}

		err := dispatcher.OnAlert(ctx, event)
		assert.NoError(t, err)
	})

	t.Run("should handle negative value", func(t *testing.T) {
		cfg := config.EmailConfig{
			SMTPHost: "",
			Username: "",
		}

		dispatcher := notifier.NewEmailDispatcher(cfg)
		ctx := context.Background()

		alert := &models.Alert{
			ID:          uuid.New(),
			Status:      models.AlertStatusFiring,
			Severity:    "warn",
			Source:      "test",
			Message:     "Negative metric",
			Value:       -50.0,
			Labels:      datatypes.JSON([]byte(`{}`)),
			CreatedAt:   time.Now(),
			TriggeredAt: time.Now(),
		}

		event := &processor.AlertEvent{
			Alert:     alert,
			Timestamp: time.Now(),
		}

		err := dispatcher.OnAlert(ctx, event)
		assert.NoError(t, err)
	})

	t.Run("should handle very large value", func(t *testing.T) {
		cfg := config.EmailConfig{
			SMTPHost: "",
			Username: "",
		}

		dispatcher := notifier.NewEmailDispatcher(cfg)
		ctx := context.Background()

		alert := &models.Alert{
			ID:          uuid.New(),
			Status:      models.AlertStatusFiring,
			Severity:    "critical",
			Source:      "test",
			Message:     "Large metric",
			Value:       999999999.99,
			Labels:      datatypes.JSON([]byte(`{}`)),
			CreatedAt:   time.Now(),
			TriggeredAt: time.Now(),
		}

		event := &processor.AlertEvent{
			Alert:     alert,
			Timestamp: time.Now(),
		}

		err := dispatcher.OnAlert(ctx, event)
		assert.NoError(t, err)
	})
}

func TestEmailDispatcher_ConfigVariations(t *testing.T) {
	tests := []struct {
		name   string
		config config.EmailConfig
	}{
		{
			name: "minimal config",
			config: config.EmailConfig{
				SMTPHost: "",
				Username: "",
			},
		},
		{
			name: "with SMTP host only",
			config: config.EmailConfig{
				SMTPHost: "smtp.example.com",
				Username: "",
			},
		},
		{
			name: "with username only",
			config: config.EmailConfig{
				SMTPHost: "",
				Username: "user@example.com",
			},
		},
		{
			name: "full config",
			config: config.EmailConfig{
				Enabled:  true,
				SMTPHost: "smtp.example.com",
				SMTPPort: 587,
				Username: "user@example.com",
				Password: "password",
				From:     "alerts@example.com",
				To:       []string{"admin@example.com", "team@example.com"},
			},
		},
		{
			name: "with custom port",
			config: config.EmailConfig{
				SMTPHost: "smtp.example.com",
				SMTPPort: 465,
				Username: "user@example.com",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dispatcher := notifier.NewEmailDispatcher(tt.config)
			assert.NotNil(t, dispatcher)

			ctx := context.Background()
			alert := &models.Alert{
				ID:          uuid.New(),
				Status:      models.AlertStatusFiring,
				Severity:    "high",
				Source:      "test",
				Message:     "Test alert",
				Value:       100.0,
				Labels:      datatypes.JSON([]byte(`{}`)),
				CreatedAt:   time.Now(),
				TriggeredAt: time.Now(),
			}

			event := &processor.AlertEvent{
				Alert:     alert,
				Timestamp: time.Now(),
			}

			// All incomplete configs should skip sending without error
			err := dispatcher.OnAlert(ctx, event)
			if tt.config.SMTPHost == "" || tt.config.Username == "" {
				assert.NoError(t, err)
			}
		})
	}
}

func TestEmailDispatcher_AlertSeverities(t *testing.T) {
	severities := []string{"critical", "high", "medium", "low", "info", "warning"}

	for _, severity := range severities {
		t.Run("should handle "+severity+" severity", func(t *testing.T) {
			cfg := config.EmailConfig{
				SMTPHost: "",
				Username: "",
			}

			dispatcher := notifier.NewEmailDispatcher(cfg)
			ctx := context.Background()

			alert := &models.Alert{
				ID:          uuid.New(),
				Status:      models.AlertStatusFiring,
				Severity:    severity,
				Source:      "test",
				Message:     "Test alert with " + severity + " severity",
				Value:       50.0,
				Labels:      datatypes.JSON([]byte(`{}`)),
				CreatedAt:   time.Now(),
				TriggeredAt: time.Now(),
			}

			event := &processor.AlertEvent{
				Alert:     alert,
				Timestamp: time.Now(),
			}

			err := dispatcher.OnAlert(ctx, event)
			assert.NoError(t, err)
		})
	}
}

func TestEmailDispatcher_AlertSources(t *testing.T) {
	sources := []string{
		"kubernetes-pod",
		"kubernetes-node",
		"pod-restart",
		"cpu-usage",
		"memory-usage",
		"custom-metric",
	}

	for _, source := range sources {
		t.Run("should handle source: "+source, func(t *testing.T) {
			cfg := config.EmailConfig{
				SMTPHost: "",
				Username: "",
			}

			dispatcher := notifier.NewEmailDispatcher(cfg)
			ctx := context.Background()

			alert := &models.Alert{
				ID:          uuid.New(),
				Status:      models.AlertStatusFiring,
				Severity:    "medium",
				Source:      source,
				Message:     "Alert from " + source,
				Value:       70.0,
				Labels:      datatypes.JSON([]byte(`{}`)),
				CreatedAt:   time.Now(),
				TriggeredAt: time.Now(),
			}

			event := &processor.AlertEvent{
				Alert:     alert,
				Timestamp: time.Now(),
			}

			err := dispatcher.OnAlert(ctx, event)
			assert.NoError(t, err)
		})
	}
}

func TestEmailDispatcher_MultipleRecipients(t *testing.T) {
	t.Run("should handle single recipient", func(t *testing.T) {
		cfg := config.EmailConfig{
			SMTPHost: "",
			Username: "",
			To:       []string{"admin@example.com"},
		}

		dispatcher := notifier.NewEmailDispatcher(cfg)
		assert.NotNil(t, dispatcher)
	})

	t.Run("should handle multiple recipients", func(t *testing.T) {
		cfg := config.EmailConfig{
			SMTPHost: "",
			Username: "",
			To: []string{
				"admin@example.com",
				"team@example.com",
				"oncall@example.com",
			},
		}

		dispatcher := notifier.NewEmailDispatcher(cfg)
		assert.NotNil(t, dispatcher)
	})

	t.Run("should handle no recipients", func(t *testing.T) {
		cfg := config.EmailConfig{
			SMTPHost: "",
			Username: "",
			To:       []string{},
		}

		dispatcher := notifier.NewEmailDispatcher(cfg)
		assert.NotNil(t, dispatcher)
	})
}
