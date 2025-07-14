package service

import (
	"context"
	"time"

	jacuzziv1 "github.com/nickheyer/jacuzzi/pkg/gen/go/jacuzzi/v1"
	temperaturev1 "github.com/nickheyer/jacuzzi/pkg/gen/go/jacuzzi/v1/temperature/v1"
	"github.com/nickheyer/jacuzzi/pkg/server/models"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

type TemperatureService struct {
	jacuzziv1.UnimplementedTemperatureServiceServer
	db *gorm.DB
}

func NewTemperatureService(db *gorm.DB) *TemperatureService {
	return &TemperatureService{db: db}
}

func (s *TemperatureService) SubmitTemperature(ctx context.Context, req *temperaturev1.SubmitTemperatureRequest) (*temperaturev1.SubmitTemperatureResponse, error) {
	if len(req.Readings) == 0 {
		return nil, status.Error(codes.InvalidArgument, "no readings provided")
	}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		for _, reading := range req.Readings {
			// Update or create client
			now := time.Now()
			client := &models.Client{
				ClientID: reading.ClientId,
				LastSeen: now,
				IsOnline: true,
			}
			if err := tx.Where("client_id = ?", reading.ClientId).FirstOrCreate(client).Error; err != nil {
				return err
			}
			// Update LastSeen and IsOnline for existing clients
			if client.ID != 0 {
				tx.Model(client).Updates(map[string]interface{}{
					"last_seen": now,
					"is_online": true,
				})
			} else {
				// Set FirstSeen for new clients
				client.FirstSeen = now
			}

			// Update or create sensor
			sensor := &models.Sensor{
				SensorID:   reading.SensorId,
				ClientID:   reading.ClientId,
				SensorType: reading.SensorType,
				SensorName: reading.SensorName,
			}
			if err := tx.Where("sensor_id = ?", reading.SensorId).FirstOrCreate(sensor).Error; err != nil {
				return err
			}

			// Insert temperature reading
			tempReading := &models.TemperatureReading{
				SensorID:           reading.SensorId,
				ClientID:           reading.ClientId,
				TemperatureCelsius: reading.TemperatureCelsius,
				SensorType:         reading.SensorType,
				SensorName:         reading.SensorName,
				CreatedAt:          reading.Timestamp.AsTime(),
			}
			if err := tx.Create(tempReading).Error; err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to save readings: %v", err)
	}

	return &temperaturev1.SubmitTemperatureResponse{
		Success: true,
		Message: "Temperature readings saved successfully",
	}, nil
}

func (s *TemperatureService) GetTemperatureHistory(ctx context.Context, req *temperaturev1.GetTemperatureHistoryRequest) (*temperaturev1.GetTemperatureHistoryResponse, error) {
	query := s.db.Model(&models.TemperatureReading{})

	if req.ClientId != "" {
		query = query.Where("client_id = ?", req.ClientId)
	}
	if req.SensorId != "" {
		query = query.Where("sensor_id = ?", req.SensorId)
	}
	if req.StartTime != nil {
		query = query.Where("created_at >= ?", req.StartTime.AsTime())
	}
	if req.EndTime != nil {
		query = query.Where("created_at <= ?", req.EndTime.AsTime())
	}

	limit := int(req.Limit)
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	query = query.Order("created_at DESC").Limit(limit)

	var readings []models.TemperatureReading
	if err := query.Find(&readings).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "failed to query temperature history: %v", err)
	}

	protoReadings := make([]*temperaturev1.TemperatureReading, len(readings))
	for i, reading := range readings {
		protoReadings[i] = &temperaturev1.TemperatureReading{
			SensorId:           reading.SensorID,
			ClientId:           reading.ClientID,
			TemperatureCelsius: reading.TemperatureCelsius,
			Timestamp:          timestamppb.New(reading.CreatedAt),
			SensorType:         reading.SensorType,
			SensorName:         reading.SensorName,
		}
	}

	return &temperaturev1.GetTemperatureHistoryResponse{
		Readings: protoReadings,
	}, nil
}

func (s *TemperatureService) GetCurrentTemperatures(ctx context.Context, req *temperaturev1.GetCurrentTemperaturesRequest) (*temperaturev1.GetCurrentTemperaturesResponse, error) {
	if req.ClientId == "" {
		return nil, status.Error(codes.InvalidArgument, "client_id is required")
	}

	// Get the latest reading for each sensor of the client
	var readings []models.TemperatureReading
	subQuery := s.db.Model(&models.TemperatureReading{}).
		Select("sensor_id, MAX(created_at) as max_created_at").
		Where("client_id = ?", req.ClientId).
		Group("sensor_id")

	err := s.db.Model(&models.TemperatureReading{}).
		Joins("INNER JOIN (?) as latest ON temperature_readings.sensor_id = latest.sensor_id AND temperature_readings.created_at = latest.max_created_at", subQuery).
		Where("temperature_readings.client_id = ?", req.ClientId).
		Find(&readings).Error

	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to query current temperatures: %v", err)
	}

	protoReadings := make([]*temperaturev1.TemperatureReading, len(readings))
	for i, reading := range readings {
		protoReadings[i] = &temperaturev1.TemperatureReading{
			SensorId:           reading.SensorID,
			ClientId:           reading.ClientID,
			TemperatureCelsius: reading.TemperatureCelsius,
			Timestamp:          timestamppb.New(reading.CreatedAt),
			SensorType:         reading.SensorType,
			SensorName:         reading.SensorName,
		}
	}

	return &temperaturev1.GetCurrentTemperaturesResponse{
		Readings: protoReadings,
	}, nil
}

func (s *TemperatureService) GetDistinctClients(ctx context.Context) ([]string, error) {
	var clients []string
	err := s.db.Model(&models.Client{}).
		Distinct("client_id").
		Order("client_id").
		Pluck("client_id", &clients).Error

	if err != nil {
		return nil, err
	}

	return clients, nil
}

func (s *TemperatureService) GetTemperatureStats(ctx context.Context, req *temperaturev1.GetTemperatureStatsRequest) (*temperaturev1.GetTemperatureStatsResponse, error) {
	baseQuery := s.db.Model(&models.TemperatureReading{})

	if req.ClientId != "" {
		baseQuery = baseQuery.Where("client_id = ?", req.ClientId)
	}
	if req.StartTime != nil {
		baseQuery = baseQuery.Where("created_at >= ?", req.StartTime.AsTime())
	}
	if req.EndTime != nil {
		baseQuery = baseQuery.Where("created_at <= ?", req.EndTime.AsTime())
	}

	// If specific sensor_id is requested, only get stats for that sensor
	var sensorIds []string
	if req.SensorId != "" {
		sensorIds = []string{req.SensorId}
	} else {
		// Get all sensor IDs that match the criteria
		if err := baseQuery.Distinct("sensor_id").Pluck("sensor_id", &sensorIds).Error; err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get sensor IDs: %v", err)
		}
	}

	sensorStats := make(map[string]*temperaturev1.TemperatureStats)
	
	for _, sensorId := range sensorIds {
		query := baseQuery.Where("sensor_id = ?", sensorId)
		
		var stats struct {
			AvgTemp float64
			MinTemp float64
			MaxTemp float64
			Count   int32
		}

		err := query.Select(`
			AVG(temperature_celsius) as avg_temp,
			MIN(temperature_celsius) as min_temp,
			MAX(temperature_celsius) as max_temp,
			COUNT(*) as count
		`).Scan(&stats).Error

		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to calculate temperature stats for sensor %s: %v", sensorId, err)
		}

		sensorStats[sensorId] = &temperaturev1.TemperatureStats{
			MinTemperature: stats.MinTemp,
			MaxTemperature: stats.MaxTemp,
			AvgTemperature: stats.AvgTemp,
			ReadingCount:   stats.Count,
			PeriodStart:    req.StartTime,
			PeriodEnd:      req.EndTime,
		}
	}

	return &temperaturev1.GetTemperatureStatsResponse{
		SensorStats: sensorStats,
	}, nil
}
