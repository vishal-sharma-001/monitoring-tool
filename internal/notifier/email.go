package notifier

import (
	"context"
	"fmt"
	"net/smtp"
	"time"

	"github.com/monitoring-engine/monitoring-tool/internal/config"
	"github.com/monitoring-engine/monitoring-tool/internal/processor"
	"github.com/monitoring-engine/monitoring-tool/internal/logger"
)

// EmailDispatcher sends alerts via email
type EmailDispatcher struct {
	config config.EmailConfig
}

func NewEmailDispatcher(cfg config.EmailConfig) *EmailDispatcher {
	return &EmailDispatcher{
		config: cfg,
	}
}

// OnAlert implements AlertObserver interface
func (ed *EmailDispatcher) OnAlert(ctx context.Context, event *processor.AlertEvent) error {
	if ed.config.SMTPHost == "" || ed.config.Username == "" {
		logger.Warn().Msg("Email configuration incomplete, skipping email dispatch")
		return nil
	}

	// Format email
	subject := fmt.Sprintf("Alert: %s - %s", event.Alert.Severity, event.Alert.Source)
	body := fmt.Sprintf(`
Monitoring Alert

Severity: %s
Source: %s
Message: %s
Value: %.2f
Timestamp: %s

Labels:
%v

--
Monitoring Engine
`, event.Alert.Severity, event.Alert.Source, event.Alert.Message,
		event.Alert.Value, event.Alert.CreatedAt.Format(time.RFC3339), event.Alert.Labels)

	message := []byte(fmt.Sprintf("Subject: %s\r\n\r\n%s", subject, body))

	// Setup authentication
	auth := smtp.PlainAuth("", ed.config.Username, ed.config.Password, ed.config.SMTPHost)

	// Send email with retry
	addr := fmt.Sprintf("%s:%d", ed.config.SMTPHost, ed.config.SMTPPort)

	var err error
	for attempt := 0; attempt < 2; attempt++ {
		err = smtp.SendMail(addr, auth, ed.config.From, ed.config.To, message)
		if err == nil {
			logger.Info().
				Strs("to", ed.config.To).
				Str("severity", string(event.Alert.Severity)).
				Msg("Alert email sent")
			return nil
		}
		time.Sleep(1 * time.Second)
	}

	return fmt.Errorf("email dispatch failed after retries: %w", err)
}
