package models

import (
	"time"
)

type TemperatureReading struct {
	ID               uint      `gorm:"primaryKey"`
	SensorID         string    `gorm:"index;not null"`
	ClientID         string    `gorm:"index;not null"`
	TemperatureCelsius float64 `gorm:"not null"`
	SensorType       string    `gorm:"index"`
	SensorName       string
	CreatedAt        time.Time `gorm:"index"`
	UpdatedAt        time.Time
}

func (TemperatureReading) TableName() string {
	return "temperature_readings"
}

type Client struct {
	ID        uint   `gorm:"primaryKey"`
	ClientID  string `gorm:"uniqueIndex;not null"`
	Hostname  string
	IPAddress string
	LastSeen  time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (Client) TableName() string {
	return "clients"
}

type Sensor struct {
	ID         uint   `gorm:"primaryKey"`
	SensorID   string `gorm:"uniqueIndex;not null"`
	ClientID   string `gorm:"index;not null"`
	SensorType string
	SensorName string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (Sensor) TableName() string {
	return "sensors"
}