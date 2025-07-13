package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server     ServerConfig     `mapstructure:"server"`
	Client     ClientConfig     `mapstructure:"client"`
	Monitoring MonitoringConfig `mapstructure:"monitoring"`
}

type ServerConfig struct {
	Address string        `mapstructure:"address"`
	Timeout time.Duration `mapstructure:"timeout"`
}

type ClientConfig struct {
	ID       string        `mapstructure:"id"`
	Interval time.Duration `mapstructure:"interval"`
}

type MonitoringConfig struct {
	CPU  bool `mapstructure:"cpu"`
	GPU  bool `mapstructure:"gpu"`
	Disk bool `mapstructure:"disk"`
}

func Load() (*Config, error) {
	viper.SetConfigName("client")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")
	viper.AddConfigPath("/etc/jacuzzi")
	viper.AddConfigPath("$HOME/.jacuzzi")

	// Set defaults
	viper.SetDefault("server.address", "localhost:50051")
	viper.SetDefault("server.timeout", 10*time.Second)
	viper.SetDefault("client.id", "")
	viper.SetDefault("client.interval", 30*time.Second)
	viper.SetDefault("monitoring.cpu", true)
	viper.SetDefault("monitoring.gpu", true)
	viper.SetDefault("monitoring.disk", true)

	// Environment variables
	viper.SetEnvPrefix("JACUZZI_CLIENT")
	viper.AutomaticEnv()

	// Bind specific environment variables
	viper.BindEnv("server.address", "JACUZZI_CLIENT_SERVER_ADDRESS")
	viper.BindEnv("server.timeout", "JACUZZI_CLIENT_SERVER_TIMEOUT")
	viper.BindEnv("client.id", "JACUZZI_CLIENT_ID")
	viper.BindEnv("client.interval", "JACUZZI_CLIENT_INTERVAL")
	viper.BindEnv("monitoring.cpu", "JACUZZI_CLIENT_MONITORING_CPU")
	viper.BindEnv("monitoring.gpu", "JACUZZI_CLIENT_MONITORING_GPU")
	viper.BindEnv("monitoring.disk", "JACUZZI_CLIENT_MONITORING_DISK")

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

	return &config, nil
}
