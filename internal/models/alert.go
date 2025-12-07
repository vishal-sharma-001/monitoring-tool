package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// AlertStatus represents the lifecycle state of an alert
type AlertStatus string

const (
	AlertStatusFiring   AlertStatus = "firing"
	AlertStatusResolved AlertStatus = "resolved"
)

// Alert represents a triggered alert
type Alert struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Status      AlertStatus    `gorm:"type:varchar(20);not null;default:'firing';index" json:"status"`
	Severity    string         `gorm:"type:varchar(50);not null;index" json:"severity"`    // critical, high, medium, low
	Message     string         `gorm:"type:text;not null" json:"message"`
	Source      string         `gorm:"type:varchar(100);not null;index" json:"source"`     // k8s_pod, k8s_node, k8s_metrics
	Labels      datatypes.JSON `gorm:"type:jsonb;default:'{}'" json:"labels"`
	Value       float64        `gorm:"type:double precision" json:"value"`
	TriggeredAt time.Time      `gorm:"not null;index:,sort:desc" json:"triggered_at"`
	ResolvedAt  *time.Time     `gorm:"type:timestamp with time zone" json:"resolved_at,omitempty"`
	CreatedAt   time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName specifies the table name for GORM
func (Alert) TableName() string {
	return "alerts"
}

// NewAlert creates a new alert
func NewAlert(severity, message, source string, value float64, labels map[string]string) *Alert {
	now := time.Now()
	labelsJSON, err := datatypes.NewJSONType(labels).MarshalJSON()
	if err != nil {
		// Fallback to empty JSON if marshaling fails
		labelsJSON = datatypes.JSON([]byte("{}"))
	}

	return &Alert{
		ID:          uuid.New(),
		Status:      AlertStatusFiring,
		Severity:    severity,
		Message:     message,
		Source:      source,
		Labels:      labelsJSON,
		Value:       value,
		TriggeredAt: now,
		ResolvedAt:  nil,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// Resolve marks an alert as resolved
func (a *Alert) Resolve() {
	now := time.Now()
	a.Status = AlertStatusResolved
	a.ResolvedAt = &now
	a.UpdatedAt = now
}

// IsFiring returns true if the alert is currently firing
func (a *Alert) IsFiring() bool {
	return a.Status == AlertStatusFiring
}

// GetLabelsMap returns labels as a map
func (a *Alert) GetLabelsMap() map[string]string {
	var labels map[string]string
	if err := json.Unmarshal(a.Labels, &labels); err != nil {
		return map[string]string{}
	}
	return labels
}
