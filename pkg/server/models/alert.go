package models

import (
	"time"
)

type AlertRule struct {
	ID          uint      `gorm:"primaryKey"`
	RuleID      string    `gorm:"uniqueIndex;not null"`
	Name        string    `gorm:"not null"`
	Description string
	ClientID    string    `gorm:"index"` // Apply to specific client or empty for all
	SensorID    string    `gorm:"index"` // Apply to specific sensor or empty for all
	SensorType  string    `gorm:"index"` // Apply to sensor type (CPU, GPU, etc) or empty for all
	
	// Condition fields
	Operator         string  `gorm:"not null"` // OPERATOR_GREATER_THAN, OPERATOR_LESS_THAN, etc.
	Threshold        float64 `gorm:"not null"`
	DurationSeconds  int32   `gorm:"not null"` // How long condition must be true
	
	Enabled   bool      `gorm:"default:true"`
	CreatedAt time.Time
	UpdatedAt time.Time
	
	// Relations
	Actions []AlertAction `gorm:"foreignKey:RuleID;references:RuleID"`
	Alerts  []Alert       `gorm:"foreignKey:RuleID;references:RuleID"`
}

func (AlertRule) TableName() string {
	return "alert_rules"
}

type AlertAction struct {
	ID       uint   `gorm:"primaryKey"`
	RuleID   string `gorm:"index;not null"`
	Type     string `gorm:"not null"` // ACTION_TYPE_EMAIL, ACTION_TYPE_WEBHOOK, etc.
	Config   string `gorm:"type:text"` // JSON string for action-specific configuration
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (AlertAction) TableName() string {
	return "alert_actions"
}

type Alert struct {
	ID         uint      `gorm:"primaryKey"`
	AlertID    string    `gorm:"uniqueIndex;not null"`
	RuleID     string    `gorm:"index;not null"`
	ClientID   string    `gorm:"index;not null"`
	SensorID   string    `gorm:"index;not null"`
	Value      float64   `gorm:"not null"`
	TriggeredAt time.Time `gorm:"index;not null"`
	ResolvedAt  *time.Time `gorm:"index"`
	IsActive    bool      `gorm:"default:true;index"`
	Message     string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (Alert) TableName() string {
	return "alerts"
}