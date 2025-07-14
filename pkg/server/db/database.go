package db

import (
	"fmt"
	"log"

	"github.com/nickheyer/jacuzzi/pkg/server/models"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Config struct {
	Type     string // "sqlite" or "postgres"
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
}

func NewDatabase(cfg Config) (*gorm.DB, error) {
	var dialector gorm.Dialector

	switch cfg.Type {
	case "sqlite":
		dialector = sqlite.Open(cfg.DBName)
	case "postgres":
		dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=%s",
			cfg.Host, cfg.User, cfg.Password, cfg.DBName, cfg.Port, cfg.SSLMode)
		dialector = postgres.Open(dsn)
	default:
		return nil, fmt.Errorf("unsupported database type: %s", cfg.Type)
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Run migrations
	if err := RunMigrations(db); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return db, nil
}

func RunMigrations(db *gorm.DB) error {
	log.Println("Running database migrations...")

	// AutoMigrate creates tables, missing columns, and missing indexes
	// It will not delete unused columns to protect data
	err := db.AutoMigrate(
		&models.Client{},
		&models.Sensor{},
		&models.TemperatureReading{},
		&models.AlertRule{},
		&models.AlertAction{},
		&models.Alert{},
		&models.Setting{},
	)
	if err != nil {
		return fmt.Errorf("failed to auto migrate: %w", err)
	}

	log.Println("Database migrations completed successfully")
	return nil
}
