package service

import (
	"context"
	"encoding/json"
	"fmt"

	jacuzziv1 "github.com/nickheyer/jacuzzi/pkg/gen/go/jacuzzi/v1"
	alertv1 "github.com/nickheyer/jacuzzi/pkg/gen/go/jacuzzi/v1/alert/v1"
	"github.com/nickheyer/jacuzzi/pkg/server/models"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
	"github.com/google/uuid"
)

type AlertService struct {
	jacuzziv1.UnimplementedAlertServiceServer
	db *gorm.DB
}

func NewAlertService(db *gorm.DB) *AlertService {
	return &AlertService{db: db}
}

func (s *AlertService) CreateAlertRule(ctx context.Context, req *alertv1.CreateAlertRuleRequest) (*alertv1.CreateAlertRuleResponse, error) {
	if req.Rule == nil {
		return nil, status.Error(codes.InvalidArgument, "rule is required")
	}
	
	rule := req.Rule
	if rule.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "rule name is required")
	}
	if rule.Condition == nil {
		return nil, status.Error(codes.InvalidArgument, "rule condition is required")
	}
	
	// Generate a new rule ID
	ruleID := uuid.New().String()
	
	// Create the alert rule
	alertRule := &models.AlertRule{
		RuleID:          ruleID,
		Name:            rule.Name,
		Description:     rule.Description,
		ClientID:        rule.ClientId,
		SensorID:        rule.SensorId,
		SensorType:      rule.SensorType,
		Operator:        rule.Condition.Operator.String(),
		Threshold:       rule.Condition.Threshold,
		DurationSeconds: rule.Condition.DurationSeconds,
		Enabled:         rule.Enabled,
	}
	
	// Start transaction
	err := s.db.Transaction(func(tx *gorm.DB) error {
		// Create the rule
		if err := tx.Create(alertRule).Error; err != nil {
			return fmt.Errorf("failed to create alert rule: %w", err)
		}
		
		// Create actions
		for _, action := range rule.Actions {
			configJSON, err := json.Marshal(action.Config)
			if err != nil {
				return fmt.Errorf("failed to marshal action config: %w", err)
			}
			
			alertAction := &models.AlertAction{
				RuleID: ruleID,
				Type:   action.Type.String(),
				Config: string(configJSON),
			}
			
			if err := tx.Create(alertAction).Error; err != nil {
				return fmt.Errorf("failed to create alert action: %w", err)
			}
		}
		
		return nil
	})
	
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create alert rule: %v", err)
	}
	
	return &alertv1.CreateAlertRuleResponse{
		RuleId:  ruleID,
		Success: true,
		Message: "Alert rule created successfully",
	}, nil
}

func (s *AlertService) ListAlertRules(ctx context.Context, req *alertv1.ListAlertRulesRequest) (*alertv1.ListAlertRulesResponse, error) {
	query := s.db.Model(&models.AlertRule{}).Preload("Actions")
	
	if req.ClientId != "" {
		query = query.Where("client_id = ?", req.ClientId)
	}
	if req.EnabledOnly {
		query = query.Where("enabled = ?", true)
	}
	
	// Get total count
	var totalCount int64
	if err := query.Count(&totalCount).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "failed to count alert rules: %v", err)
	}
	
	// Apply pagination
	limit := int(req.Limit)
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	offset := int(req.Offset)
	
	var rules []models.AlertRule
	if err := query.Order("created_at DESC").Limit(limit).Offset(offset).Find(&rules).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list alert rules: %v", err)
	}
	
	// Convert to proto
	protoRules := make([]*alertv1.AlertRule, len(rules))
	for i, rule := range rules {
		protoRule, err := s.modelToProtoAlertRule(&rule)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to convert alert rule: %v", err)
		}
		protoRules[i] = protoRule
	}
	
	return &alertv1.ListAlertRulesResponse{
		Rules:      protoRules,
		TotalCount: int32(totalCount),
	}, nil
}

func (s *AlertService) DeleteAlertRule(ctx context.Context, req *alertv1.DeleteAlertRuleRequest) (*alertv1.DeleteAlertRuleResponse, error) {
	if req.RuleId == "" {
		return nil, status.Error(codes.InvalidArgument, "rule_id is required")
	}
	
	// Delete the rule and its actions (cascade delete)
	result := s.db.Where("rule_id = ?", req.RuleId).Delete(&models.AlertRule{})
	if result.Error != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete alert rule: %v", result.Error)
	}
	
	if result.RowsAffected == 0 {
		return nil, status.Error(codes.NotFound, "alert rule not found")
	}
	
	// Also delete associated actions
	s.db.Where("rule_id = ?", req.RuleId).Delete(&models.AlertAction{})
	
	return &alertv1.DeleteAlertRuleResponse{
		Success: true,
		Message: "Alert rule deleted successfully",
	}, nil
}

func (s *AlertService) GetAlertHistory(ctx context.Context, req *alertv1.GetAlertHistoryRequest) (*alertv1.GetAlertHistoryResponse, error) {
	query := s.db.Model(&models.Alert{})
	
	if req.RuleId != "" {
		query = query.Where("rule_id = ?", req.RuleId)
	}
	if req.ClientId != "" {
		query = query.Where("client_id = ?", req.ClientId)
	}
	if req.ActiveOnly {
		query = query.Where("is_active = ?", true)
	}
	if req.StartTime != nil {
		query = query.Where("triggered_at >= ?", req.StartTime.AsTime())
	}
	if req.EndTime != nil {
		query = query.Where("triggered_at <= ?", req.EndTime.AsTime())
	}
	
	limit := int(req.Limit)
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	
	var alerts []models.Alert
	if err := query.Order("triggered_at DESC").Limit(limit).Find(&alerts).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get alert history: %v", err)
	}
	
	// Convert to proto
	protoAlerts := make([]*alertv1.Alert, len(alerts))
	for i, alert := range alerts {
		protoAlert := &alertv1.Alert{
			Id:          alert.AlertID,
			RuleId:      alert.RuleID,
			ClientId:    alert.ClientID,
			SensorId:    alert.SensorID,
			Value:       alert.Value,
			TriggeredAt: timestamppb.New(alert.TriggeredAt),
			IsActive:    alert.IsActive,
			Message:     alert.Message,
		}
		if alert.ResolvedAt != nil {
			protoAlert.ResolvedAt = timestamppb.New(*alert.ResolvedAt)
		}
		protoAlerts[i] = protoAlert
	}
	
	return &alertv1.GetAlertHistoryResponse{
		Alerts: protoAlerts,
	}, nil
}

// Helper function to convert model to proto
func (s *AlertService) modelToProtoAlertRule(rule *models.AlertRule) (*alertv1.AlertRule, error) {
	// Parse operator
	operator := alertv1.AlertCondition_OPERATOR_UNSPECIFIED
	switch rule.Operator {
	case "OPERATOR_GREATER_THAN":
		operator = alertv1.AlertCondition_OPERATOR_GREATER_THAN
	case "OPERATOR_LESS_THAN":
		operator = alertv1.AlertCondition_OPERATOR_LESS_THAN
	case "OPERATOR_EQUAL":
		operator = alertv1.AlertCondition_OPERATOR_EQUAL
	case "OPERATOR_NOT_EQUAL":
		operator = alertv1.AlertCondition_OPERATOR_NOT_EQUAL
	}
	
	// Convert actions
	protoActions := make([]*alertv1.AlertAction, len(rule.Actions))
	for i, action := range rule.Actions {
		actionType := alertv1.AlertAction_ACTION_TYPE_UNSPECIFIED
		switch action.Type {
		case "ACTION_TYPE_EMAIL":
			actionType = alertv1.AlertAction_ACTION_TYPE_EMAIL
		case "ACTION_TYPE_WEBHOOK":
			actionType = alertv1.AlertAction_ACTION_TYPE_WEBHOOK
		case "ACTION_TYPE_LOG":
			actionType = alertv1.AlertAction_ACTION_TYPE_LOG
		}
		
		config := make(map[string]string)
		if action.Config != "" {
			if err := json.Unmarshal([]byte(action.Config), &config); err != nil {
				// Log error but don't fail
				config = make(map[string]string)
			}
		}
		
		protoActions[i] = &alertv1.AlertAction{
			Type:   actionType,
			Config: config,
		}
	}
	
	return &alertv1.AlertRule{
		Id:          rule.RuleID,
		Name:        rule.Name,
		Description: rule.Description,
		ClientId:    rule.ClientID,
		SensorId:    rule.SensorID,
		SensorType:  rule.SensorType,
		Condition: &alertv1.AlertCondition{
			Operator:        operator,
			Threshold:       rule.Threshold,
			DurationSeconds: rule.DurationSeconds,
		},
		Actions:   protoActions,
		Enabled:   rule.Enabled,
		CreatedAt: timestamppb.New(rule.CreatedAt),
		UpdatedAt: timestamppb.New(rule.UpdatedAt),
	}, nil
}