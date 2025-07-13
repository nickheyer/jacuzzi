package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
}

type ServerConfig struct {
	Port int    `mapstructure:"port"`
	Host string `mapstructure:"host"`
}

type DatabaseConfig struct {
	Type     string `mapstructure:"type"`
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Name     string `mapstructure:"name"`
	SSLMode  string `mapstructure:"sslmode"`
}

func Load() (*Config, error) {
	viper.SetConfigName("server")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")
	viper.AddConfigPath("/etc/jacuzzi")
	viper.AddConfigPath("$HOME/.jacuzzi")

	// Set defaults
	viper.SetDefault("server.port", 50051)
	viper.SetDefault("server.host", "")
	viper.SetDefault("database.type", "sqlite")
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.user", "jacuzzi")
	viper.SetDefault("database.password", "")
	viper.SetDefault("database.name", "data/db/jacuzzi.db")
	viper.SetDefault("database.sslmode", "disable")

	// Environment variables
	viper.SetEnvPrefix("JACUZZI")
	viper.AutomaticEnv()

	// Bind specific environment variables
	viper.BindEnv("server.port", "JACUZZI_SERVER_PORT")
	viper.BindEnv("server.host", "JACUZZI_SERVER_HOST")
	viper.BindEnv("database.type", "JACUZZI_DB_TYPE")
	viper.BindEnv("database.host", "JACUZZI_DB_HOST")
	viper.BindEnv("database.port", "JACUZZI_DB_PORT")
	viper.BindEnv("database.user", "JACUZZI_DB_USER")
	viper.BindEnv("database.password", "JACUZZI_DB_PASSWORD")
	viper.BindEnv("database.name", "JACUZZI_DB_NAME")
	viper.BindEnv("database.sslmode", "JACUZZI_DB_SSLMODE")

	// Try to read config file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
		// Config file not found; use defaults and environment
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Ensure data directory exists for SQLite
	if config.Database.Type == "sqlite" {
		dbDir := filepath.Dir(config.Database.Name)
		if err := os.MkdirAll(dbDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create database directory: %w", err)
		}
	}

	return &config, nil
}

func (c *Config) GetServerAddress() string {
	if c.Server.Host == "" {
		return fmt.Sprintf(":%d", c.Server.Port)
	}
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
}
