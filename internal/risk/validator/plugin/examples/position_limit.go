package examples

import (
	"context"
	"fmt"

	"github.com/abdoElHodaky/tradSys/internal/orders"
	"github.com/abdoElHodaky/tradSys/internal/risk/validator/plugin"
	"go.uber.org/zap"
)

// PluginInfo contains information about the plugin
var PluginInfo = &plugin.PluginInfo{
	Name:          "Position Limit Validator",
	Version:       "1.0.0",
	Author:        "TradSys Team",
	Description:   "A validator that checks if an order would exceed position limits",
	ValidatorType: "position_limit",
}

// PositionLimitValidator implements a position limit validator
type PositionLimitValidator struct {
	config      plugin.ValidatorConfig
	logger      *zap.Logger
	name        string
	description string
	enabled     bool
	priority    int
	
	// Position limit parameters
	maxLongPosition  float64
	maxShortPosition float64
	maxTotalPosition float64
	
	// Current positions (in a real implementation, this would be fetched from a position service)
	currentPositions map[string]float64
}

// CreateValidator creates a new position limit validator
func CreateValidator(config plugin.ValidatorConfig, logger *zap.Logger) (plugin.RiskValidator, error) {
	// Get validator-specific configuration
	maxLongPosition, ok := config.Params["max_long_position"].(float64)
	if !ok {
		maxLongPosition = 100.0 // Default value
	}
	
	maxShortPosition, ok := config.Params["max_short_position"].(float64)
	if !ok {
		maxShortPosition = 100.0 // Default value
	}
	
	maxTotalPosition, ok := config.Params["max_total_position"].(float64)
	if !ok {
		maxTotalPosition = 200.0 // Default value
	}
	
	return &PositionLimitValidator{
		config:           config,
		logger:           logger,
		name:             config.Name,
		description:      "Validates that orders do not exceed position limits",
		enabled:          config.Enabled,
		priority:         config.Priority,
		maxLongPosition:  maxLongPosition,
		maxShortPosition: maxShortPosition,
		maxTotalPosition: maxTotalPosition,
		currentPositions: make(map[string]float64),
	}, nil
}

// Validate validates an order against position limits
func (v *PositionLimitValidator) Validate(ctx context.Context, order *orders.Order) (bool, string, error) {
	if !v.enabled {
		return true, "", nil
	}
	
	v.logger.Debug("Validating order against position limits",
		zap.String("order_id", order.OrderID),
		zap.String("symbol", order.Symbol),
		zap.String("side", order.Side),
		zap.Float64("quantity", order.Quantity),
	)
	
	// Get current position for the symbol
	currentPosition, ok := v.currentPositions[order.Symbol]
	if !ok {
		// In a real implementation, we would fetch this from a position service
		// For this example, we'll assume no position
		currentPosition = 0.0
	}
	
	// Calculate new position after the order
	newPosition := currentPosition
	if order.Side == "BUY" {
		newPosition += order.Quantity
	} else if order.Side == "SELL" {
		newPosition -= order.Quantity
	}
	
	// Check against limits
	if newPosition > v.maxLongPosition {
		reason := fmt.Sprintf("Order would exceed maximum long position limit (%.2f > %.2f)",
			newPosition, v.maxLongPosition)
		v.logger.Warn("Position limit validation failed",
			zap.String("order_id", order.OrderID),
			zap.String("reason", reason),
		)
		return false, reason, nil
	}
	
	if newPosition < -v.maxShortPosition {
		reason := fmt.Sprintf("Order would exceed maximum short position limit (%.2f < %.2f)",
			newPosition, -v.maxShortPosition)
		v.logger.Warn("Position limit validation failed",
			zap.String("order_id", order.OrderID),
			zap.String("reason", reason),
		)
		return false, reason, nil
	}
	
	// Check total position across all symbols
	totalPosition := 0.0
	for _, pos := range v.currentPositions {
		totalPosition += pos
	}
	totalPosition = totalPosition - currentPosition + newPosition
	
	if totalPosition > v.maxTotalPosition {
		reason := fmt.Sprintf("Order would exceed maximum total position limit (%.2f > %.2f)",
			totalPosition, v.maxTotalPosition)
		v.logger.Warn("Position limit validation failed",
			zap.String("order_id", order.OrderID),
			zap.String("reason", reason),
		)
		return false, reason, nil
	}
	
	v.logger.Debug("Position limit validation passed",
		zap.String("order_id", order.OrderID),
		zap.Float64("current_position", currentPosition),
		zap.Float64("new_position", newPosition),
	)
	
	return true, "", nil
}

// GetName returns the name of the validator
func (v *PositionLimitValidator) GetName() string {
	return v.name
}

// GetDescription returns the description of the validator
func (v *PositionLimitValidator) GetDescription() string {
	return v.description
}

// GetType returns the type of validator
func (v *PositionLimitValidator) GetType() string {
	return "position_limit"
}

// GetPriority returns the priority of the validator
func (v *PositionLimitValidator) GetPriority() int {
	return v.priority
}

// IsEnabled returns whether the validator is enabled
func (v *PositionLimitValidator) IsEnabled() bool {
	return v.enabled
}

// SetEnabled sets whether the validator is enabled
func (v *PositionLimitValidator) SetEnabled(enabled bool) {
	v.enabled = enabled
}

// UpdatePositions updates the current positions
// In a real implementation, this would be called by a position service
func (v *PositionLimitValidator) UpdatePositions(positions map[string]float64) {
	v.currentPositions = positions
}

