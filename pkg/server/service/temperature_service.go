package service

import (
	"context"
	"time"

	"github.com/nickheyer/jacuzzi/pkg/server/models"
	jacuzziv1 "github.com/nickheyer/jacuzzi/proto/gen/go/proto"
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

func (s *TemperatureService) SubmitTemperature(ctx context.Context, req *jacuzziv1.SubmitTemperatureRequest) (*jacuzziv1.SubmitTemperatureResponse, error) {
	if len(req.Readings) == 0 {
		return nil, status.Error(codes.InvalidArgument, "no readings provided")
	}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		for _, reading := range req.Readings {
			// Update or create client
			client := &models.Client{
				ClientID: reading.ClientId,
				LastSeen: time.Now(),
			}
			if err := tx.Where("client_id = ?", reading.ClientId).FirstOrCreate(client).Error; err != nil {
				return err
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

	return &jacuzziv1.SubmitTemperatureResponse{
		Success: true,
		Message: "Temperature readings saved successfully",
	}, nil
}

func (s *TemperatureService) GetTemperatureHistory(ctx context.Context, req *jacuzziv1.GetTemperatureHistoryRequest) (*jacuzziv1.GetTemperatureHistoryResponse, error) {
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

	protoReadings := make([]*jacuzziv1.TemperatureReading, len(readings))
	for i, reading := range readings {
		protoReadings[i] = &jacuzziv1.TemperatureReading{
			SensorId:           reading.SensorID,
			ClientId:           reading.ClientID,
			TemperatureCelsius: reading.TemperatureCelsius,
			Timestamp:          timestamppb.New(reading.CreatedAt),
			SensorType:         reading.SensorType,
			SensorName:         reading.SensorName,
		}
	}

	return &jacuzziv1.GetTemperatureHistoryResponse{
		Readings: protoReadings,
	}, nil
}

func (s *TemperatureService) GetCurrentTemperatures(ctx context.Context, req *jacuzziv1.GetCurrentTemperaturesRequest) (*jacuzziv1.GetCurrentTemperaturesResponse, error) {
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

	protoReadings := make([]*jacuzziv1.TemperatureReading, len(readings))
	for i, reading := range readings {
		protoReadings[i] = &jacuzziv1.TemperatureReading{
			SensorId:           reading.SensorID,
			ClientId:           reading.ClientID,
			TemperatureCelsius: reading.TemperatureCelsius,
			Timestamp:          timestamppb.New(reading.CreatedAt),
			SensorType:         reading.SensorType,
			SensorName:         reading.SensorName,
		}
	}

	return &jacuzziv1.GetCurrentTemperaturesResponse{
		Readings: protoReadings,
	}, nil
}
