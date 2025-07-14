package models

import (
	"time"
)

type Setting struct {
	ID        uint      `gorm:"primaryKey"`
	Key       string    `gorm:"uniqueIndex;not null"`
	Value     string    `gorm:"type:text"`
	ValueType string    `gorm:"not null;default:'string'"` // string, int, float, bool, json
	Category  string    `gorm:"index"`                      // email, notifications, general, etc.
	Description string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (Setting) TableName() string {
	return "settings"
}

// Common setting keys
const (
	// Email settings
	SettingEmailEnabled     = "email.enabled"
	SettingEmailSMTPHost    = "email.smtp_host"
	SettingEmailSMTPPort    = "email.smtp_port"
	SettingEmailUsername    = "email.username"
	SettingEmailPassword    = "email.password"
	SettingEmailFrom        = "email.from_address"
	SettingEmailUseTLS      = "email.use_tls"
	
	// Temperature thresholds
	SettingTempWarningThreshold   = "temperature.warning_threshold"
	SettingTempCriticalThreshold  = "temperature.critical_threshold"
	
	// Data retention
	SettingDataRetentionDays = "data.retention_days"
	
	// General settings  
	SettingSystemName = "general.system_name"
	SettingTimezone   = "general.timezone"
)