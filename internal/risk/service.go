package risk

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/db"
	"github.com/abdoElHodaky/tradSys/internal/db/repositories"
	"github.com/abdoElHodaky/tradSys/proto/risk"
	"github.com/google/uuid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// ServiceParams contains the parameters for creating a risk service
type ServiceParams struct {
	fx.In

	Logger     *zap.Logger
	Repository *repositories.RiskRepository
}

// Service provides risk management operations
type Service struct {
	logger     *zap.Logger
	repository *repositories.RiskRepository
}

// NewService creates a new risk service with fx dependency injection
func NewService(p ServiceParams) *Service {
	return &Service{
		logger:     p.Logger,
		repository: p.Repository,
	}
}

// mapLimitTypeToString maps a limit type enum to a string
func mapLimitTypeToString(limitType risk.LimitType) string {
	switch limitType {
	case risk.LimitType_MAX_POSITION_SIZE:
		return "max_position_size"
	case risk.LimitType_MAX_ORDER_SIZE:
		return "max_order_size"
	case risk.LimitType_MAX_DAILY_VOLUME:
		return "max_daily_volume"
	case risk.LimitType_MAX_DAILY_LOSS:
		return "max_daily_loss"
	case risk.LimitType_MAX_LEVERAGE:
		return "max_leverage"
	case risk.LimitType_MAX_CONCENTRATION:
		return "max_concentration"
	default:
		return "unknown"
	}
}

// mapStringToLimitType maps a string to a limit type enum
func mapStringToLimitType(limitType string) risk.LimitType {
	switch limitType {
	case "max_position_size":
		return risk.LimitType_MAX_POSITION_SIZE
	case "max_order_size":
		return risk.LimitType_MAX_ORDER_SIZE
	case "max_daily_volume":
		return risk.LimitType_MAX_DAILY_VOLUME
	case "max_daily_loss":
		return risk.LimitType_MAX_DAILY_LOSS
	case "max_leverage":
		return risk.LimitType_MAX_LEVERAGE
	case "max_concentration":
		return risk.LimitType_MAX_CONCENTRATION
	default:
		return risk.LimitType_UNKNOWN_LIMIT
	}
}

// dbPositionToProto converts a database position to a proto position
func dbPositionToProto(dbPosition *db.Position) *risk.PositionResponse {
	return &risk.PositionResponse{
		Symbol:            dbPosition.Symbol,
		Quantity:          dbPosition.Quantity,
		AverageEntryPrice: dbPosition.AverageEntryPrice,
		UnrealizedPnl:     dbPosition.UnrealizedPnL,
		RealizedPnl:       dbPosition.RealizedPnL,
		LastUpdated:       dbPosition.LastUpdated.UnixMilli(),
	}
}

// dbRiskLimitToProto converts a database risk limit to a proto risk limit
func dbRiskLimitToProto(dbLimit *db.RiskLimit) *risk.RiskLimitResponse {
	return &risk.RiskLimitResponse{
		Id:        dbLimit.ID,
		UserId:    dbLimit.UserID,
		Symbol:    dbLimit.Symbol,
		Type:      mapStringToLimitType(dbLimit.Type),
		Value:     dbLimit.Value,
		Enabled:   dbLimit.Enabled,
		CreatedAt: dbLimit.CreatedAt.UnixMilli(),
		UpdatedAt: dbLimit.UpdatedAt.UnixMilli(),
	}
}

// GetPositions gets positions for a user
func (s *Service) GetPositions(ctx context.Context, userID, symbol string) (*risk.GetPositionsResponse, error) {
	s.logger.Info("Getting positions",
		zap.String("user_id", userID),
		zap.String("symbol", symbol))

	// Validate inputs
	if userID == "" {
		return nil, errors.New("user ID is required")
	}

	var positions []*db.Position
	var err error

	// Get positions from database
	if symbol != "" {
		// Get specific position
		position, err := s.repository.GetPositionByUserAndSymbol(ctx, userID, symbol)
		if err != nil {
			s.logger.Error("Failed to get position",
				zap.Error(err),
				zap.String("user_id", userID),
				zap.String("symbol", symbol))
			return nil, fmt.Errorf("failed to get position: %w", err)
		}

		if position != nil {
			positions = append(positions, position)
		}
	} else {
		// Get all positions
		positions, err = s.repository.GetPositionsByUserID(ctx, userID)
		if err != nil {
			s.logger.Error("Failed to get positions",
				zap.Error(err),
				zap.String("user_id", userID))
			return nil, fmt.Errorf("failed to get positions: %w", err)
		}
	}

	// Convert to proto response
	var protoPositions []*risk.PositionResponse
	for _, position := range positions {
		protoPositions = append(protoPositions, dbPositionToProto(position))
	}

	return &risk.GetPositionsResponse{
		Positions: protoPositions,
	}, nil
}

// GetLimits gets risk limits for a user
func (s *Service) GetLimits(ctx context.Context, userID, symbol string, limitType risk.LimitType) (*risk.GetLimitsResponse, error) {
	s.logger.Info("Getting risk limits",
		zap.String("user_id", userID),
		zap.String("symbol", symbol),
		zap.String("limit_type", limitType.String()))

	// Validate inputs
	if userID == "" {
		return nil, errors.New("user ID is required")
	}

	var limits []*db.RiskLimit
	var err error

	// Get limits from database
	if limitType != risk.LimitType_UNKNOWN_LIMIT && symbol != "" {
		// Get specific limit
		limit, err := s.repository.GetRiskLimitByUserAndType(ctx, userID, mapLimitTypeToString(limitType), symbol)
		if err != nil {
			s.logger.Error("Failed to get risk limit",
				zap.Error(err),
				zap.String("user_id", userID),
				zap.String("limit_type", limitType.String()),
				zap.String("symbol", symbol))
			return nil, fmt.Errorf("failed to get risk limit: %w", err)
		}

		if limit != nil {
			limits = append(limits, limit)
		}
	} else {
		// Get all limits
		limits, err = s.repository.GetRiskLimitsByUserID(ctx, userID)
		if err != nil {
			s.logger.Error("Failed to get risk limits",
				zap.Error(err),
				zap.String("user_id", userID))
			return nil, fmt.Errorf("failed to get risk limits: %w", err)
		}

		// Filter by symbol if provided
		if symbol != "" {
			var filteredLimits []*db.RiskLimit
			for _, limit := range limits {
				if limit.Symbol == symbol || limit.Symbol == "" {
					filteredLimits = append(filteredLimits, limit)
				}
			}
			limits = filteredLimits
		}

		// Filter by type if provided
		if limitType != risk.LimitType_UNKNOWN_LIMIT {
			var filteredLimits []*db.RiskLimit
			for _, limit := range limits {
				if mapStringToLimitType(limit.Type) == limitType {
					filteredLimits = append(filteredLimits, limit)
				}
			}
			limits = filteredLimits
		}
	}

	// Convert to proto response
	var protoLimits []*risk.RiskLimitResponse
	for _, limit := range limits {
		protoLimits = append(protoLimits, dbRiskLimitToProto(limit))
	}

	return &risk.GetLimitsResponse{
		Limits: protoLimits,
	}, nil
}

// SetLimit sets a risk limit for a user
func (s *Service) SetLimit(ctx context.Context, userID, symbol string, limitType risk.LimitType, value float64, enabled bool) (*risk.RiskLimitResponse, error) {
	s.logger.Info("Setting risk limit",
		zap.String("user_id", userID),
		zap.String("symbol", symbol),
		zap.String("limit_type", limitType.String()),
		zap.Float64("value", value),
		zap.Bool("enabled", enabled))

	// Validate inputs
	if userID == "" {
		return nil, errors.New("user ID is required")
	}

	if limitType == risk.LimitType_UNKNOWN_LIMIT {
		return nil, errors.New("valid limit type is required")
	}

	if value <= 0 {
		return nil, errors.New("value must be greater than 0")
	}

	// Check if limit already exists
	existingLimit, err := s.repository.GetRiskLimitByUserAndType(ctx, userID, mapLimitTypeToString(limitType), symbol)
	if err != nil {
		s.logger.Error("Failed to check if risk limit exists",
			zap.Error(err),
			zap.String("user_id", userID),
			zap.String("limit_type", limitType.String()),
			zap.String("symbol", symbol))
		return nil, fmt.Errorf("failed to check if risk limit exists: %w", err)
	}

	var limit *db.RiskLimit

	if existingLimit != nil {
		// Update existing limit
		existingLimit.Value = value
		existingLimit.Enabled = enabled
		limit = existingLimit

		if err := s.repository.UpdateRiskLimit(ctx, limit); err != nil {
			s.logger.Error("Failed to update risk limit",
				zap.Error(err),
				zap.String("limit_id", limit.ID))
			return nil, fmt.Errorf("failed to update risk limit: %w", err)
		}
	} else {
		// Create new limit
		limit = &db.RiskLimit{
			ID:      uuid.New().String(),
			UserID:  userID,
			Symbol:  symbol,
			Type:    mapLimitTypeToString(limitType),
			Value:   value,
			Enabled: enabled,
		}

		if err := s.repository.CreateRiskLimit(ctx, limit); err != nil {
			s.logger.Error("Failed to create risk limit",
				zap.Error(err),
				zap.String("user_id", userID),
				zap.String("limit_type", limitType.String()))
			return nil, fmt.Errorf("failed to create risk limit: %w", err)
		}
	}

	// Convert to proto response
	return dbRiskLimitToProto(limit), nil
}

// DeleteLimit deletes a risk limit
func (s *Service) DeleteLimit(ctx context.Context, limitID, userID string) (*risk.DeleteLimitResponse, error) {
	s.logger.Info("Deleting risk limit",
		zap.String("limit_id", limitID),
		zap.String("user_id", userID))

	// Validate inputs
	if limitID == "" {
		return nil, errors.New("limit ID is required")
	}

	if userID == "" {
		return nil, errors.New("user ID is required")
	}

	// Delete limit
	if err := s.repository.DeleteRiskLimit(ctx, limitID); err != nil {
		s.logger.Error("Failed to delete risk limit",
			zap.Error(err),
			zap.String("limit_id", limitID))
		return &risk.DeleteLimitResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to delete risk limit: %v", err),
		}, nil
	}

	return &risk.DeleteLimitResponse{
		Success: true,
	}, nil
}

// ValidateOrder validates an order against risk limits
func (s *Service) ValidateOrder(ctx context.Context, userID, symbol, side, orderType string, quantity, price float64) (*risk.ValidateOrderResponse, error) {
	s.logger.Info("Validating order",
		zap.String("user_id", userID),
		zap.String("symbol", symbol),
		zap.String("side", side),
		zap.String("type", orderType),
		zap.Float64("quantity", quantity),
		zap.Float64("price", price))

	// Validate inputs
	if userID == "" {
		return nil, errors.New("user ID is required")
	}

	if symbol == "" {
		return nil, errors.New("symbol is required")
	}

	if quantity <= 0 {
		return nil, errors.New("quantity must be greater than 0")
	}

	// Get all limits for the user
	limits, err := s.repository.GetRiskLimitsByUserID(ctx, userID)
	if err != nil {
		s.logger.Error("Failed to get risk limits",
			zap.Error(err),
			zap.String("user_id", userID))
		return nil, fmt.Errorf("failed to get risk limits: %w", err)
	}

	// Filter limits by symbol or global
	var applicableLimits []*db.RiskLimit
	for _, limit := range limits {
		if limit.Symbol == symbol || limit.Symbol == "" {
			applicableLimits = append(applicableLimits, limit)
		}
	}

	// Get current position
	position, err := s.repository.GetPositionByUserAndSymbol(ctx, userID, symbol)
	if err != nil {
		s.logger.Error("Failed to get position",
			zap.Error(err),
			zap.String("user_id", userID),
			zap.String("symbol", symbol))
		return nil, fmt.Errorf("failed to get position: %w", err)
	}

	// Initialize response
	response := &risk.ValidateOrderResponse{
		Valid: true,
	}

	// Calculate order value
	orderValue := quantity * price

	// Check each limit
	for _, limit := range applicableLimits {
		if !limit.Enabled {
			continue
		}

		switch limit.Type {
		case "max_order_size":
			if orderValue > limit.Value {
				response.Valid = false
				response.RejectionReasons = append(response.RejectionReasons,
					fmt.Sprintf("Order value %.2f exceeds max order size limit %.2f", orderValue, limit.Value))
			}

		case "max_position_size":
			var newPositionSize float64
			if position != nil {
				if side == "buy" {
					newPositionSize = position.Quantity + quantity
				} else {
					newPositionSize = position.Quantity - quantity
				}
				if newPositionSize < 0 {
					newPositionSize = -newPositionSize // Absolute value for short positions
				}
			} else {
				newPositionSize = quantity
			}

			if newPositionSize*price > limit.Value {
				response.Valid = false
				response.RejectionReasons = append(response.RejectionReasons,
					fmt.Sprintf("New position value %.2f exceeds max position size limit %.2f", newPositionSize*price, limit.Value))
			}

		case "max_daily_volume":
			// In a real implementation, we would check the user's daily trading volume
			// For now, we'll just add a warning
			response.Warnings = append(response.Warnings,
				"Daily volume limit check not implemented")

		case "max_daily_loss":
			// In a real implementation, we would check the user's daily PnL
			// For now, we'll just add a warning
			response.Warnings = append(response.Warnings,
				"Daily loss limit check not implemented")

		case "max_leverage":
			// In a real implementation, we would check the user's leverage
			// For now, we'll just add a warning
			response.Warnings = append(response.Warnings,
				"Leverage limit check not implemented")

		case "max_concentration":
			// In a real implementation, we would check the user's portfolio concentration
			// For now, we'll just add a warning
			response.Warnings = append(response.Warnings,
				"Concentration limit check not implemented")
		}
	}

	return response, nil
}

// ServiceModule provides the risk service module for fx
var ServiceModule = fx.Options(
	fx.Provide(NewService),
)

