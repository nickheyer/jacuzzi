package service

import (
	"context"
	"encoding/json"
	"time"

	jacuzziv1 "github.com/nickheyer/jacuzzi/pkg/gen/go/jacuzzi/v1"
	clientv1 "github.com/nickheyer/jacuzzi/pkg/gen/go/jacuzzi/v1/client/v1"
	"github.com/nickheyer/jacuzzi/pkg/server/models"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

type ClientService struct {
	jacuzziv1.UnimplementedClientServiceServer
	db *gorm.DB
}

func NewClientService(db *gorm.DB) *ClientService {
	return &ClientService{db: db}
}

func (s *ClientService) ListClients(ctx context.Context, req *clientv1.ListClientsRequest) (*clientv1.ListClientsResponse, error) {
	query := s.db.Model(&models.Client{})
	
	if req.OnlineOnly {
		query = query.Where("is_online = ?", true)
	}
	
	// Get total count
	var totalCount int64
	if err := query.Count(&totalCount).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "failed to count clients: %v", err)
	}
	
	// Apply pagination
	limit := int(req.Limit)
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	offset := int(req.Offset)
	
	var clients []models.Client
	if err := query.Order("client_id").Limit(limit).Offset(offset).Find(&clients).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list clients: %v", err)
	}
	
	// Convert to proto
	protoClients := make([]*clientv1.Client, len(clients))
	for i, client := range clients {
		protoClient, err := s.modelToProtoClient(&client)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to convert client: %v", err)
		}
		protoClients[i] = protoClient
	}
	
	return &clientv1.ListClientsResponse{
		Clients:    protoClients,
		TotalCount: int32(totalCount),
	}, nil
}

func (s *ClientService) GetClient(ctx context.Context, req *clientv1.GetClientRequest) (*clientv1.GetClientResponse, error) {
	if req.ClientId == "" {
		return nil, status.Error(codes.InvalidArgument, "client_id is required")
	}
	
	var client models.Client
	if err := s.db.Where("client_id = ?", req.ClientId).First(&client).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, status.Error(codes.NotFound, "client not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get client: %v", err)
	}
	
	protoClient, err := s.modelToProtoClient(&client)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert client: %v", err)
	}
	
	// Get sensors for this client
	var sensors []models.Sensor
	if err := s.db.Where("client_id = ?", req.ClientId).Find(&sensors).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get sensors: %v", err)
	}
	
	// Get latest temperature for each sensor
	sensorInfos := make([]*clientv1.SensorInfo, len(sensors))
	for i, sensor := range sensors {
		var latestReading models.TemperatureReading
		err := s.db.Where("sensor_id = ?", sensor.SensorID).
			Order("created_at DESC").
			First(&latestReading).Error
		
		sensorInfo := &clientv1.SensorInfo{
			SensorId:   sensor.SensorID,
			SensorType: sensor.SensorType,
			SensorName: sensor.SensorName,
		}
		
		if err == nil {
			sensorInfo.CurrentTemperature = latestReading.TemperatureCelsius
			sensorInfo.LastReading = timestamppb.New(latestReading.CreatedAt)
		}
		
		sensorInfos[i] = sensorInfo
	}
	
	return &clientv1.GetClientResponse{
		Client:  protoClient,
		Sensors: sensorInfos,
	}, nil
}

func (s *ClientService) UpdateClient(ctx context.Context, req *clientv1.UpdateClientRequest) (*clientv1.UpdateClientResponse, error) {
	if req.ClientId == "" {
		return nil, status.Error(codes.InvalidArgument, "client_id is required")
	}
	
	var client models.Client
	if err := s.db.Where("client_id = ?", req.ClientId).First(&client).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, status.Error(codes.NotFound, "client not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get client: %v", err)
	}
	
	// Update metadata if provided
	if len(req.Metadata) > 0 {
		metadataJSON, err := json.Marshal(req.Metadata)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to marshal metadata: %v", err)
		}
		client.Metadata = string(metadataJSON)
		
		if err := s.db.Save(&client).Error; err != nil {
			return nil, status.Errorf(codes.Internal, "failed to update client: %v", err)
		}
	}
	
	return &clientv1.UpdateClientResponse{
		Success: true,
		Message: "Client updated successfully",
	}, nil
}

// Helper function to convert model to proto
func (s *ClientService) modelToProtoClient(client *models.Client) (*clientv1.Client, error) {
	// Parse metadata
	metadata := make(map[string]string)
	if client.Metadata != "" {
		if err := json.Unmarshal([]byte(client.Metadata), &metadata); err != nil {
			// Log error but don't fail the request
			metadata = make(map[string]string)
		}
	}
	
	// Check if client is online (last seen within 5 minutes)
	isOnline := time.Since(client.LastSeen) < 5*time.Minute
	
	return &clientv1.Client{
		Id:        client.ClientID,
		Hostname:  client.Hostname,
		IpAddress: client.IPAddress,
		Os:        client.OS,
		Arch:      client.Arch,
		FirstSeen: timestamppb.New(client.FirstSeen),
		LastSeen:  timestamppb.New(client.LastSeen),
		IsOnline:  isOnline,
		Metadata:  metadata,
	}, nil
}