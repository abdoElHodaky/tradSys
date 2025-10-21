package risk

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"go.uber.org/zap"
)



// OrderRiskCheck represents risk parameters for an order
type OrderRiskCheck struct {
	UserID       string  `json:"user_id"`
	Symbol       string  `json:"symbol"`
	Side         string  `json:"side"`
	Quantity     float64 `json:"quantity"`
	Price        float64 `json:"price"`
	OrderType    string  `json:"order_type"`
	Value        float64 `json:"value"`
	CurrentPrice float64 `json:"current_price"`
}

// RiskLimits represents risk limits for a user or symbol
type RiskLimits struct {
	UserID              string  `json:"user_id"`
	Symbol              string  `json:"symbol,omitempty"`
	MaxPositionSize     float64 `json:"max_position_size"`
	MaxOrderSize        float64 `json:"max_order_size"`
	MaxDailyVolume      float64 `json:"max_daily_volume"`
	MaxDrawdown         float64 `json:"max_drawdown"`
	MaxLeverage         float64 `json:"max_leverage"`
	VaRLimit            float64 `json:"var_limit"`
	ConcentrationLimit  float64 `json:"concentration_limit"`
	LastUpdated         time.Time `json:"last_updated"`
}

// PositionRisk represents position risk metrics
type PositionRisk struct {
	UserID           string  `json:"user_id"`
	Symbol           string  `json:"symbol"`
	Quantity         float64 `json:"quantity"`
	MarketValue      float64 `json:"market_value"`
	UnrealizedPL     float64 `json:"unrealized_pl"`
	VaR              float64 `json:"var"`
	Beta             float64 `json:"beta"`
	Volatility       float64 `json:"volatility"`
	ConcentrationPct float64 `json:"concentration_pct"`
}

// RiskEngine handles real-time risk management
type RiskEngine struct {
	limits          map[string]*RiskLimits // userID -> limits
	symbolLimits    map[string]*RiskLimits // symbol -> limits
	positions       map[string]map[string]*PositionRisk // userID -> symbol -> risk
	dailyVolumes    map[string]float64 // userID -> daily volume
	logger          *zap.Logger
	mu              sync.RWMutex
	
	// Performance metrics
	checkCount       int64
	avgCheckTime     time.Duration
	violationCount   int64
	
	// Market data for risk calculations
	marketPrices     map[string]float64 // symbol -> price
	volatilities     map[string]float64 // symbol -> volatility
	correlations     map[string]map[string]float64 // symbol -> symbol -> correlation
	pricesMu         sync.RWMutex
	
	// Configuration
	varConfidence    float64 // VaR confidence level (e.g., 0.95)
	varHorizon       int     // VaR time horizon in days
	maxCheckTime     time.Duration
}

// NewRiskEngine creates a new risk engine
func NewRiskEngine(logger *zap.Logger) *RiskEngine {
	return &RiskEngine{
		limits:        make(map[string]*RiskLimits),
		symbolLimits:  make(map[string]*RiskLimits),
		positions:     make(map[string]map[string]*PositionRisk),
		dailyVolumes:  make(map[string]float64),
		logger:        logger,
		marketPrices:  make(map[string]float64),
		volatilities:  make(map[string]float64),
		correlations:  make(map[string]map[string]float64),
		varConfidence: 0.95,
		varHorizon:    1,
		maxCheckTime:  10 * time.Microsecond, // Ultra-fast for HFT
	}
}

// CheckOrderRisk performs pre-trade risk checks
func (re *RiskEngine) CheckOrderRisk(ctx context.Context, order *OrderRiskCheck) (*RiskCheckResult, error) {
	start := time.Now()
	defer func() {
		re.avgCheckTime = (re.avgCheckTime + time.Since(start)) / 2
		re.checkCount++
	}()

	// Create context with timeout
	checkCtx, cancel := context.WithTimeout(ctx, re.maxCheckTime)
	defer cancel()

	result := &RiskCheckResult{
		Passed:         true,
		RiskLevel:      RiskLevelLow,
		Violations:     make([]string, 0),
		Warnings:       make([]string, 0),
		CheckedAt:      time.Now(),
	}

	// Check if context is cancelled
	select {
	case <-checkCtx.Done():
		return nil, fmt.Errorf("risk check timeout")
	default:
	}

	// Get user limits
	re.mu.RLock()
	userLimits, hasUserLimits := re.limits[order.UserID]
	symbolLimits, hasSymbolLimits := re.symbolLimits[order.Symbol]
	userPositions, hasPositions := re.positions[order.UserID]
	dailyVolume := re.dailyVolumes[order.UserID]
	re.mu.RUnlock()

	// Check order size limits
	if hasUserLimits && order.Quantity > userLimits.MaxOrderSize {
		result.Violations = append(result.Violations, 
			fmt.Sprintf("Order size %.2f exceeds limit %.2f", order.Quantity, userLimits.MaxOrderSize))
		result.Passed = false
		result.RiskLevel = RiskLevelHigh
	}

	if hasSymbolLimits && order.Quantity > symbolLimits.MaxOrderSize {
		result.Violations = append(result.Violations, 
			fmt.Sprintf("Order size %.2f exceeds symbol limit %.2f", order.Quantity, symbolLimits.MaxOrderSize))
		result.Passed = false
		result.RiskLevel = RiskLevelHigh
	}

	// Check daily volume limits
	newDailyVolume := dailyVolume + order.Value
	if hasUserLimits && newDailyVolume > userLimits.MaxDailyVolume {
		result.Violations = append(result.Violations, 
			fmt.Sprintf("Daily volume %.2f would exceed limit %.2f", newDailyVolume, userLimits.MaxDailyVolume))
		result.Passed = false
		result.RiskLevel = RiskLevelHigh
	}

	// Check position size limits
	if hasPositions {
		if position, exists := userPositions[order.Symbol]; exists {
			newQuantity := position.Quantity
			if order.Side == "buy" {
				newQuantity += order.Quantity
			} else {
				newQuantity -= order.Quantity
			}

			if hasUserLimits && math.Abs(newQuantity) > userLimits.MaxPositionSize {
				result.Violations = append(result.Violations, 
					fmt.Sprintf("Position size %.2f would exceed limit %.2f", math.Abs(newQuantity), userLimits.MaxPositionSize))
				result.Passed = false
				result.RiskLevel = RiskLevelHigh
			}
		}
	}

	// Check concentration limits
	if err := re.checkConcentrationRisk(order, result); err != nil {
		re.logger.Error("Concentration risk check failed", zap.Error(err))
	}

	// Check VaR limits
	if err := re.checkVaRRisk(order, result); err != nil {
		re.logger.Error("VaR risk check failed", zap.Error(err))
	}

	// Check leverage limits
	if err := re.checkLeverageRisk(order, result); err != nil {
		re.logger.Error("Leverage risk check failed", zap.Error(err))
	}

	// Update metrics
	if !result.Passed {
		re.violationCount++
	}

	processingTime := time.Since(start)

	re.logger.Debug("Risk check completed",
		zap.String("user_id", order.UserID),
		zap.String("symbol", order.Symbol),
		zap.Bool("passed", result.Passed),
		zap.String("risk_level", string(result.RiskLevel)),
		zap.Duration("processing_time", processingTime),
	)

	return result, nil
}

// checkConcentrationRisk checks concentration risk
func (re *RiskEngine) checkConcentrationRisk(order *OrderRiskCheck, result *RiskCheckResult) error {
	re.mu.RLock()
	userLimits, hasLimits := re.limits[order.UserID]
	userPositions, hasPositions := re.positions[order.UserID]
	re.mu.RUnlock()

	if !hasLimits || !hasPositions {
		return nil
	}

	// Calculate total portfolio value
	totalValue := 0.0
	for _, position := range userPositions {
		totalValue += math.Abs(position.MarketValue)
	}

	// Calculate concentration after this order
	currentSymbolValue := 0.0
	if position, exists := userPositions[order.Symbol]; exists {
		currentSymbolValue = math.Abs(position.MarketValue)
	}

	newSymbolValue := currentSymbolValue + order.Value
	concentration := newSymbolValue / (totalValue + order.Value)

	if concentration > userLimits.ConcentrationLimit {
		result.Violations = append(result.Violations, 
			fmt.Sprintf("Concentration %.2f%% would exceed limit %.2f%%", 
				concentration*100, userLimits.ConcentrationLimit*100))
		result.Passed = false
		if result.RiskLevel < RiskLevelMedium {
			result.RiskLevel = RiskLevelMedium
		}
	}

	return nil
}

// checkVaRRisk checks Value at Risk limits
func (re *RiskEngine) checkVaRRisk(order *OrderRiskCheck, result *RiskCheckResult) error {
	re.mu.RLock()
	userLimits, hasLimits := re.limits[order.UserID]
	re.mu.RUnlock()

	if !hasLimits {
		return nil
	}

	// Calculate VaR for the order
	var orderVaR float64
	re.pricesMu.RLock()
	if volatility, exists := re.volatilities[order.Symbol]; exists {
		// Simple VaR calculation: VaR = Value * Volatility * Z-score
		zScore := re.getZScore(re.varConfidence)
		orderVaR = order.Value * volatility * zScore * math.Sqrt(float64(re.varHorizon))
	}
	re.pricesMu.RUnlock()

	if orderVaR > userLimits.VaRLimit {
		result.Violations = append(result.Violations, 
			fmt.Sprintf("Order VaR %.2f exceeds limit %.2f", orderVaR, userLimits.VaRLimit))
		result.Passed = false
		if result.RiskLevel < RiskLevelMedium {
			result.RiskLevel = RiskLevelMedium
		}
	}

	return nil
}

// checkLeverageRisk checks leverage limits
func (re *RiskEngine) checkLeverageRisk(order *OrderRiskCheck, result *RiskCheckResult) error {
	re.mu.RLock()
	userLimits, hasLimits := re.limits[order.UserID]
	userPositions, hasPositions := re.positions[order.UserID]
	re.mu.RUnlock()

	if !hasLimits || !hasPositions {
		return nil
	}

	// Calculate current leverage
	totalExposure := 0.0
	totalEquity := 0.0

	for _, position := range userPositions {
		totalExposure += math.Abs(position.MarketValue)
		totalEquity += position.MarketValue + position.UnrealizedPL
	}

	// Add this order's exposure
	totalExposure += order.Value
	
	leverage := totalExposure / totalEquity
	if leverage > userLimits.MaxLeverage {
		result.Violations = append(result.Violations, 
			fmt.Sprintf("Leverage %.2fx would exceed limit %.2fx", leverage, userLimits.MaxLeverage))
		result.Passed = false
		if result.RiskLevel < RiskLevelHigh {
			result.RiskLevel = RiskLevelHigh
		}
	}

	return nil
}

// getZScore returns the Z-score for a given confidence level
func (re *RiskEngine) getZScore(confidence float64) float64 {
	// Simplified Z-score mapping
	switch {
	case confidence >= 0.99:
		return 2.33
	case confidence >= 0.95:
		return 1.65
	case confidence >= 0.90:
		return 1.28
	default:
		return 1.0
	}
}

// SetUserLimits sets risk limits for a user
func (re *RiskEngine) SetUserLimits(userID string, limits *RiskLimits) {
	re.mu.Lock()
	limits.UserID = userID
	limits.LastUpdated = time.Now()
	re.limits[userID] = limits
	re.mu.Unlock()

	re.logger.Info("User risk limits updated",
		zap.String("user_id", userID),
		zap.Float64("max_position_size", limits.MaxPositionSize),
		zap.Float64("max_order_size", limits.MaxOrderSize),
	)
}

// SetSymbolLimits sets risk limits for a symbol
func (re *RiskEngine) SetSymbolLimits(symbol string, limits *RiskLimits) {
	re.mu.Lock()
	limits.Symbol = symbol
	limits.LastUpdated = time.Now()
	re.symbolLimits[symbol] = limits
	re.mu.Unlock()

	re.logger.Info("Symbol risk limits updated",
		zap.String("symbol", symbol),
		zap.Float64("max_position_size", limits.MaxPositionSize),
		zap.Float64("max_order_size", limits.MaxOrderSize),
	)
}

// UpdatePosition updates position risk metrics
func (re *RiskEngine) UpdatePosition(userID, symbol string, position *PositionRisk) {
	re.mu.Lock()
	if re.positions[userID] == nil {
		re.positions[userID] = make(map[string]*PositionRisk)
	}
	re.positions[userID][symbol] = position
	re.mu.Unlock()
}

// UpdateMarketData updates market data for risk calculations
func (re *RiskEngine) UpdateMarketData(symbol string, price, volatility float64) {
	re.pricesMu.Lock()
	re.marketPrices[symbol] = price
	re.volatilities[symbol] = volatility
	re.pricesMu.Unlock()
}

// UpdateDailyVolume updates daily trading volume for a user
func (re *RiskEngine) UpdateDailyVolume(userID string, volume float64) {
	re.mu.Lock()
	re.dailyVolumes[userID] += volume
	re.mu.Unlock()
}

// ResetDailyVolumes resets daily volumes (called at market open)
func (re *RiskEngine) ResetDailyVolumes() {
	re.mu.Lock()
	re.dailyVolumes = make(map[string]float64)
	re.mu.Unlock()

	re.logger.Info("Daily volumes reset")
}

// GetUserLimits returns risk limits for a user
func (re *RiskEngine) GetUserLimits(userID string) (*RiskLimits, bool) {
	re.mu.RLock()
	defer re.mu.RUnlock()
	
	limits, exists := re.limits[userID]
	return limits, exists
}

// GetPerformanceMetrics returns risk engine performance metrics
func (re *RiskEngine) GetPerformanceMetrics() map[string]interface{} {
	re.mu.RLock()
	defer re.mu.RUnlock()

	violationRate := 0.0
	if re.checkCount > 0 {
		violationRate = float64(re.violationCount) / float64(re.checkCount)
	}

	return map[string]interface{}{
		"total_checks":        re.checkCount,
		"total_violations":    re.violationCount,
		"violation_rate":      violationRate,
		"avg_check_time_ns":   re.avgCheckTime.Nanoseconds(),
		"users_tracked":       len(re.limits),
		"symbols_tracked":     len(re.symbolLimits),
		"var_confidence":      re.varConfidence,
		"var_horizon_days":    re.varHorizon,
	}
}

// GetRiskSummary returns a risk summary for a user
func (re *RiskEngine) GetRiskSummary(userID string) map[string]interface{} {
	re.mu.RLock()
	defer re.mu.RUnlock()

	summary := map[string]interface{}{
		"user_id": userID,
		"daily_volume": re.dailyVolumes[userID],
	}

	if limits, exists := re.limits[userID]; exists {
		summary["limits"] = limits
	}

	if positions, exists := re.positions[userID]; exists {
		summary["positions"] = positions
		
		// Calculate aggregate risk metrics
		totalValue := 0.0
		totalVaR := 0.0
		for _, position := range positions {
			totalValue += math.Abs(position.MarketValue)
			totalVaR += position.VaR
		}
		summary["total_value"] = totalValue
		summary["total_var"] = totalVaR
	}

	return summary
}
