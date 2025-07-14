package service

import (
	"context"
	"encoding/json"

	jacuzziv1 "github.com/nickheyer/jacuzzi/pkg/gen/go/jacuzzi/v1"
	settingsv1 "github.com/nickheyer/jacuzzi/pkg/gen/go/jacuzzi/v1/settings/v1"
	"github.com/nickheyer/jacuzzi/pkg/server/models"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

type SettingsService struct {
	jacuzziv1.UnimplementedSettingsServiceServer
	db *gorm.DB
}

func NewSettingsService(db *gorm.DB) *SettingsService {
	// Initialize default settings on service creation
	service := &SettingsService{db: db}
	service.initializeDefaultSettings()
	return service
}

func (s *SettingsService) GetSettings(ctx context.Context, req *settingsv1.GetSettingsRequest) (*settingsv1.GetSettingsResponse, error) {
	settings, err := s.loadSettings()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to load settings: %v", err)
	}
	
	return &settingsv1.GetSettingsResponse{
		Settings: settings,
	}, nil
}

func (s *SettingsService) UpdateSettings(ctx context.Context, req *settingsv1.UpdateSettingsRequest) (*settingsv1.UpdateSettingsResponse, error) {
	if req.Settings == nil {
		return nil, status.Error(codes.InvalidArgument, "settings are required")
	}
	
	err := s.saveSettings(req.Settings)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update settings: %v", err)
	}
	
	return &settingsv1.UpdateSettingsResponse{
		Success: true,
		Message: "Settings updated successfully",
	}, nil
}

// Helper function to load settings from database
func (s *SettingsService) loadSettings() (*settingsv1.Settings, error) {
	settingsMap := make(map[string]string)
	
	var dbSettings []models.Setting
	if err := s.db.Find(&dbSettings).Error; err != nil {
		return nil, err
	}
	
	for _, setting := range dbSettings {
		settingsMap[setting.Key] = setting.Value
	}
	
	// Build settings object
	settings := &settingsv1.Settings{
		SiteName:                    s.getStringSetting(settingsMap, "general.site_name", "Jacuzzi"),
		Timezone:                    s.getStringSetting(settingsMap, "general.timezone", "UTC"),
		RetentionDays:               int32(s.getIntSetting(settingsMap, "data.retention_days", 30)),
		AggregationIntervalSeconds:  int32(s.getIntSetting(settingsMap, "data.aggregation_interval_seconds", 60)),
		TemperatureUnit:             s.getStringSetting(settingsMap, "display.temperature_unit", "celsius"),
		Theme:                       s.getStringSetting(settingsMap, "display.theme", "system"),
		AlertsEnabled:               s.getBoolSetting(settingsMap, "alerts.enabled", true),
		AlertCheckIntervalSeconds:   int32(s.getIntSetting(settingsMap, "alerts.check_interval_seconds", 60)),
		MaxConcurrentClients:        int32(s.getIntSetting(settingsMap, "performance.max_concurrent_clients", 100)),
		ApiRateLimit:                int32(s.getIntSetting(settingsMap, "performance.api_rate_limit", 1000)),
	}
	
	// Load email settings
	emailSettings := &settingsv1.EmailSettings{
		SmtpHost:     s.getStringSetting(settingsMap, models.SettingEmailSMTPHost, ""),
		SmtpPort:     int32(s.getIntSetting(settingsMap, models.SettingEmailSMTPPort, 587)),
		SmtpUsername: s.getStringSetting(settingsMap, models.SettingEmailUsername, ""),
		SmtpPassword: s.getStringSetting(settingsMap, models.SettingEmailPassword, ""),
		UseTls:       s.getBoolSetting(settingsMap, models.SettingEmailUseTLS, true),
		FromAddress:  s.getStringSetting(settingsMap, models.SettingEmailFrom, ""),
	}
	
	// Load admin emails
	adminEmailsJSON := s.getStringSetting(settingsMap, "email.admin_emails", "[]")
	var adminEmails []string
	json.Unmarshal([]byte(adminEmailsJSON), &adminEmails)
	emailSettings.AdminEmails = adminEmails
	
	settings.EmailSettings = emailSettings
	
	return settings, nil
}

// Helper function to save settings to database
func (s *SettingsService) saveSettings(settings *settingsv1.Settings) error {
	settingsToSave := []models.Setting{
		{Key: "general.site_name", Value: settings.SiteName, ValueType: "string", Category: "general"},
		{Key: "general.timezone", Value: settings.Timezone, ValueType: "string", Category: "general"},
		{Key: "data.retention_days", Value: s.intToString(int(settings.RetentionDays)), ValueType: "int", Category: "data"},
		{Key: "data.aggregation_interval_seconds", Value: s.intToString(int(settings.AggregationIntervalSeconds)), ValueType: "int", Category: "data"},
		{Key: "display.temperature_unit", Value: settings.TemperatureUnit, ValueType: "string", Category: "display"},
		{Key: "display.theme", Value: settings.Theme, ValueType: "string", Category: "display"},
		{Key: "alerts.enabled", Value: s.boolToString(settings.AlertsEnabled), ValueType: "bool", Category: "alerts"},
		{Key: "alerts.check_interval_seconds", Value: s.intToString(int(settings.AlertCheckIntervalSeconds)), ValueType: "int", Category: "alerts"},
		{Key: "performance.max_concurrent_clients", Value: s.intToString(int(settings.MaxConcurrentClients)), ValueType: "int", Category: "performance"},
		{Key: "performance.api_rate_limit", Value: s.intToString(int(settings.ApiRateLimit)), ValueType: "int", Category: "performance"},
	}
	
	// Add email settings if provided
	if settings.EmailSettings != nil {
		emailSettings := settings.EmailSettings
		settingsToSave = append(settingsToSave,
			models.Setting{Key: models.SettingEmailSMTPHost, Value: emailSettings.SmtpHost, ValueType: "string", Category: "email"},
			models.Setting{Key: models.SettingEmailSMTPPort, Value: s.intToString(int(emailSettings.SmtpPort)), ValueType: "int", Category: "email"},
			models.Setting{Key: models.SettingEmailUsername, Value: emailSettings.SmtpUsername, ValueType: "string", Category: "email"},
			models.Setting{Key: models.SettingEmailPassword, Value: emailSettings.SmtpPassword, ValueType: "string", Category: "email"},
			models.Setting{Key: models.SettingEmailUseTLS, Value: s.boolToString(emailSettings.UseTls), ValueType: "bool", Category: "email"},
			models.Setting{Key: models.SettingEmailFrom, Value: emailSettings.FromAddress, ValueType: "string", Category: "email"},
		)
		
		// Save admin emails as JSON
		adminEmailsJSON, _ := json.Marshal(emailSettings.AdminEmails)
		settingsToSave = append(settingsToSave,
			models.Setting{Key: "email.admin_emails", Value: string(adminEmailsJSON), ValueType: "json", Category: "email"},
		)
	}
	
	return s.db.Transaction(func(tx *gorm.DB) error {
		for _, setting := range settingsToSave {
			if err := tx.Where("key = ?", setting.Key).
				Assign(setting).
				FirstOrCreate(&setting).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// Initialize default settings if they don't exist
func (s *SettingsService) initializeDefaultSettings() error {
	defaultSettings := []models.Setting{
		{Key: "general.site_name", Value: "Jacuzzi", ValueType: "string", Category: "general", Description: "Site name"},
		{Key: "general.timezone", Value: "UTC", ValueType: "string", Category: "general", Description: "System timezone"},
		{Key: "data.retention_days", Value: "30", ValueType: "int", Category: "data", Description: "Days to retain temperature data"},
		{Key: "data.aggregation_interval_seconds", Value: "60", ValueType: "int", Category: "data", Description: "Data aggregation interval"},
		{Key: "display.temperature_unit", Value: "celsius", ValueType: "string", Category: "display", Description: "Temperature display unit"},
		{Key: "display.theme", Value: "system", ValueType: "string", Category: "display", Description: "UI theme"},
		{Key: "alerts.enabled", Value: "true", ValueType: "bool", Category: "alerts", Description: "Enable alerts"},
		{Key: "alerts.check_interval_seconds", Value: "60", ValueType: "int", Category: "alerts", Description: "Alert check interval"},
		{Key: "performance.max_concurrent_clients", Value: "100", ValueType: "int", Category: "performance", Description: "Max concurrent clients"},
		{Key: "performance.api_rate_limit", Value: "1000", ValueType: "int", Category: "performance", Description: "API rate limit per minute"},
		{Key: models.SettingEmailSMTPHost, Value: "", ValueType: "string", Category: "email", Description: "SMTP host"},
		{Key: models.SettingEmailSMTPPort, Value: "587", ValueType: "int", Category: "email", Description: "SMTP port"},
		{Key: models.SettingEmailUsername, Value: "", ValueType: "string", Category: "email", Description: "SMTP username"},
		{Key: models.SettingEmailPassword, Value: "", ValueType: "string", Category: "email", Description: "SMTP password"},
		{Key: models.SettingEmailUseTLS, Value: "true", ValueType: "bool", Category: "email", Description: "Use TLS"},
		{Key: models.SettingEmailFrom, Value: "", ValueType: "string", Category: "email", Description: "From address"},
		{Key: "email.admin_emails", Value: "[]", ValueType: "json", Category: "email", Description: "Admin email addresses"},
	}
	
	return s.db.Transaction(func(tx *gorm.DB) error {
		for _, setting := range defaultSettings {
			var existing models.Setting
			if err := tx.Where("key = ?", setting.Key).First(&existing).Error; err == gorm.ErrRecordNotFound {
				if err := tx.Create(&setting).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}

// Utility functions
func (s *SettingsService) getStringSetting(settings map[string]string, key string, defaultValue string) string {
	if val, ok := settings[key]; ok {
		return val
	}
	return defaultValue
}

func (s *SettingsService) getIntSetting(settings map[string]string, key string, defaultValue int) int {
	if val, ok := settings[key]; ok {
		var intVal int
		json.Unmarshal([]byte(val), &intVal)
		if intVal != 0 {
			return intVal
		}
	}
	return defaultValue
}

func (s *SettingsService) getBoolSetting(settings map[string]string, key string, defaultValue bool) bool {
	if val, ok := settings[key]; ok {
		var boolVal bool
		json.Unmarshal([]byte(val), &boolVal)
		return boolVal
	}
	return defaultValue
}

func (s *SettingsService) intToString(val int) string {
	data, _ := json.Marshal(val)
	return string(data)
}

func (s *SettingsService) boolToString(val bool) string {
	if val {
		return "true"
	}
	return "false"
}