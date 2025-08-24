package examples

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/orders"
	"github.com/abdoElHodaky/tradSys/internal/risk/validator/plugin"
	"go.uber.org/zap"
)

// PluginInfo contains information about the plugin
var VolatilityRiskPluginInfo = &plugin.PluginInfo{
	Name:          "Volatility Risk Validator",
	Version:       "1.0.0",
	Author:        "TradSys Team",
	Description:   "A validator that checks if an order is too risky based on market volatility",
	ValidatorType: "volatility_risk",
}

// VolatilityRiskValidator implements a volatility-based risk validator
type VolatilityRiskValidator struct {
	config      plugin.ValidatorConfig
	logger      *zap.Logger
	name        string
	description string
	enabled     bool
	priority    int
	
	// Volatility parameters
	maxVolatility     float64
	volatilityWindow  time.Duration
	riskMultiplier    float64
	
	// Market data (in a real implementation, this would be fetched from a market data service)
	volatilityData map[string]float64
	lastUpdated    time.Time
}

// CreateVolatilityValidator creates a new volatility risk validator
func CreateVolatilityValidator(config plugin.ValidatorConfig, logger *zap.Logger) (plugin.RiskValidator, error) {
	// Get validator-specific configuration
	maxVolatility, ok := config.Params["max_volatility"].(float64)
	if !ok {
		maxVolatility = 0.05 // Default value (5%)
	}
	
	volatilityWindowSec, ok := config.Params["volatility_window_sec"].(float64)
	if !ok {
		volatilityWindowSec = 3600.0 // Default value (1 hour)
	}
	
	riskMultiplier, ok := config.Params["risk_multiplier"].(float64)
	if !ok {
		riskMultiplier = 1.5 // Default value
	}
	
	return &VolatilityRiskValidator{
		config:           config,
		logger:           logger,
		name:             config.Name,
		description:      "Validates that orders are not too risky based on market volatility",
		enabled:          config.Enabled,
		priority:         config.Priority,
		maxVolatility:    maxVolatility,
		volatilityWindow: time.Duration(volatilityWindowSec) * time.Second,
		riskMultiplier:   riskMultiplier,
		volatilityData:   make(map[string]float64),
		lastUpdated:      time.Now(),
	}, nil
}

// Validate validates an order against volatility risk
func (v *VolatilityRiskValidator) Validate(ctx context.Context, order *orders.Order) (bool, string, error) {
	if !v.enabled {
		return true, "", nil
	}
	
	v.logger.Debug("Validating order against volatility risk",
		zap.String("order_id", order.OrderID),
		zap.String("symbol", order.Symbol),
		zap.String("side", order.Side),
		zap.Float64("quantity", order.Quantity),
	)
	
	// Check if volatility data is stale
	if time.Since(v.lastUpdated) > v.volatilityWindow {
		// In a real implementation, we would fetch fresh data
		// For this example, we'll use a placeholder
		v.updateVolatilityData()
	}
	
	// Get volatility for the symbol
	volatility, ok := v.volatilityData[order.Symbol]
	if !ok {
		// If we don't have data for this symbol, use a default
		volatility = 0.02 // 2%
	}
	
	// Calculate risk score based on volatility and order size
	riskScore := volatility * order.Quantity * order.Price * v.riskMultiplier
	
	// Calculate maximum allowed risk
	maxRisk := v.maxVolatility * order.Price * order.Quantity
	
	// Check if risk is too high
	if riskScore > maxRisk {
		reason := fmt.Sprintf("Order risk too high based on market volatility (%.2f > %.2f)",
			riskScore, maxRisk)
		v.logger.Warn("Volatility risk validation failed",
			zap.String("order_id", order.OrderID),
			zap.String("reason", reason),
			zap.Float64("volatility", volatility),
			zap.Float64("risk_score", riskScore),
			zap.Float64("max_risk", maxRisk),
		)
		return false, reason, nil
	}
	
	v.logger.Debug("Volatility risk validation passed",
		zap.String("order_id", order.OrderID),
		zap.Float64("volatility", volatility),
		zap.Float64("risk_score", riskScore),
		zap.Float64("max_risk", maxRisk),
	)
	
	return true, "", nil
}

// GetName returns the name of the validator
func (v *VolatilityRiskValidator) GetName() string {
	return v.name
}

// GetDescription returns the description of the validator
func (v *VolatilityRiskValidator) GetDescription() string {
	return v.description
}

// GetType returns the type of validator
func (v *VolatilityRiskValidator) GetType() string {
	return "volatility_risk"
}

// GetPriority returns the priority of the validator
func (v *VolatilityRiskValidator) GetPriority() int {
	return v.priority
}

// IsEnabled returns whether the validator is enabled
func (v *VolatilityRiskValidator) IsEnabled() bool {
	return v.enabled
}

// SetEnabled sets whether the validator is enabled
func (v *VolatilityRiskValidator) SetEnabled(enabled bool) {
	v.enabled = enabled
}

// updateVolatilityData updates the volatility data
// In a real implementation, this would fetch data from a market data service
func (v *VolatilityRiskValidator) updateVolatilityData() {
	// Placeholder implementation
	// In a real system, this would fetch actual market data
	
	// Generate some random volatility data for common symbols
	symbols := []string{"BTC-USD", "ETH-USD", "XRP-USD", "LTC-USD", "BCH-USD"}
	
	for _, symbol := range symbols {
		// Generate a random volatility between 1% and 10%
		volatility := 0.01 + 0.09*math.Abs(math.Sin(float64(time.Now().UnixNano())))
		v.volatilityData[symbol] = volatility
	}
	
	v.lastUpdated = time.Now()
	
	v.logger.Debug("Updated volatility data",
		zap.Time("timestamp", v.lastUpdated),
		zap.Int("symbols", len(v.volatilityData)),
	)
}

