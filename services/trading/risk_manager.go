// Package trading provides risk management for TradSys v3
package trading

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/pkg/interfaces"
	"github.com/abdoElHodaky/tradSys/pkg/types"
)

// RiskManager provides comprehensive risk management
type RiskManager struct {
	config     *RiskManagerConfig
	riskStore  RiskStore
	calculator *RiskCalculator
	mu         sync.RWMutex
}

// RiskManagerConfig holds risk management configuration
type RiskManagerConfig struct {
	MaxPositionSize     float64
	MaxDailyVolume      float64
	MaxOrderValue       float64
	VolatilityThreshold float64
	ConcentrationLimit  float64
	EnableRealTimeCheck bool
	RiskCheckTimeout    time.Duration
}

// RiskStore interface for risk data persistence
type RiskStore interface {
	GetUserRisk(ctx context.Context, userID string) (*UserRiskProfile, error)
	UpdateUserRisk(ctx context.Context, profile *UserRiskProfile) error
	GetPositions(ctx context.Context, userID string) ([]*Position, error)
	GetDailyVolume(ctx context.Context, userID string, date time.Time) (float64, error)
	SaveRiskCheck(ctx context.Context, check *RiskCheckRecord) error
}

// RiskCalculator performs risk calculations
type RiskCalculator struct {
	volatilityCache map[string]*VolatilityData
	cacheTTL        time.Duration
	mu              sync.RWMutex
}

// UserRiskProfile represents a user's risk profile
type UserRiskProfile struct {
	UserID           string    `json:"user_id"`
	RiskTolerance    string    `json:"risk_tolerance"` // LOW, MEDIUM, HIGH
	MaxPositionSize  float64   `json:"max_position_size"`
	MaxDailyVolume   float64   `json:"max_daily_volume"`
	ConcentrationLimit float64 `json:"concentration_limit"`
	IsActive         bool      `json:"is_active"`
	LastUpdated      time.Time `json:"last_updated"`
}

// Position represents a trading position
type Position struct {
	UserID       string          `json:"user_id"`
	Symbol       string          `json:"symbol"`
	AssetType    types.AssetType `json:"asset_type"`
	Exchange     types.ExchangeType `json:"exchange"`
	Quantity     float64         `json:"quantity"`
	AveragePrice float64         `json:"average_price"`
	MarketValue  float64         `json:"market_value"`
	UnrealizedPL float64         `json:"unrealized_pl"`
	LastUpdated  time.Time       `json:"last_updated"`
}

// RiskCheckResult represents the result of a risk check
type RiskCheckResult struct {
	Approved      bool                   `json:"approved"`
	Reason        string                 `json:"reason"`
	RiskScore     float64                `json:"risk_score"`
	Violations    []RiskViolation        `json:"violations"`
	Recommendations []string             `json:"recommendations"`
	CheckedAt     time.Time              `json:"checked_at"`
	Metadata      map[string]interface{} `json:"metadata"`
}

// RiskViolation represents a risk rule violation
type RiskViolation struct {
	Rule        string  `json:"rule"`
	Description string  `json:"description"`
	Severity    string  `json:"severity"` // LOW, MEDIUM, HIGH, CRITICAL
	Value       float64 `json:"value"`
	Limit       float64 `json:"limit"`
}

// RiskCheckRecord represents a risk check record for audit
type RiskCheckRecord struct {
	ID       string           `json:"id"`
	UserID   string           `json:"user_id"`
	OrderID  string           `json:"order_id"`
	Result   *RiskCheckResult `json:"result"`
	Order    *interfaces.Order `json:"order"`
	CheckedAt time.Time       `json:"checked_at"`
}

// VolatilityData represents volatility information for an asset
type VolatilityData struct {
	Symbol      string    `json:"symbol"`
	Volatility  float64   `json:"volatility"`
	LastUpdated time.Time `json:"last_updated"`
}

// NewRiskManager creates a new risk manager
func NewRiskManager(config *RiskManagerConfig, riskStore RiskStore) *RiskManager {
	if config == nil {
		config = GetDefaultRiskManagerConfig()
	}
	
	return &RiskManager{
		config:     config,
		riskStore:  riskStore,
		calculator: NewRiskCalculator(),
	}
}

// ValidateOrder performs comprehensive risk validation for an order
func (rm *RiskManager) ValidateOrder(ctx context.Context, order *interfaces.Order) (*RiskCheckResult, error) {
	result := &RiskCheckResult{
		Approved:        true,
		RiskScore:       0.0,
		Violations:      []RiskViolation{},
		Recommendations: []string{},
		CheckedAt:       time.Now(),
		Metadata:        make(map[string]interface{}),
	}
	
	// Get user risk profile
	profile, err := rm.riskStore.GetUserRisk(ctx, order.UserID)
	if err != nil {
		return result, fmt.Errorf("failed to get user risk profile: %w", err)
	}
	
	if !profile.IsActive {
		result.Approved = false
		result.Reason = "user risk profile is inactive"
		return result, nil
	}
	
	// 1. Order value check
	orderValue := order.Price * order.Quantity
	if err := rm.checkOrderValue(orderValue, result); err != nil {
		return result, err
	}
	
	// 2. Position size check
	if err := rm.checkPositionSize(ctx, order, profile, result); err != nil {
		return result, err
	}
	
	// 3. Daily volume check
	if err := rm.checkDailyVolume(ctx, order, profile, result); err != nil {
		return result, err
	}
	
	// 4. Concentration check
	if err := rm.checkConcentration(ctx, order, profile, result); err != nil {
		return result, err
	}
	
	// 5. Volatility check
	if err := rm.checkVolatility(ctx, order, result); err != nil {
		return result, err
	}
	
	// Calculate overall risk score
	result.RiskScore = rm.calculateRiskScore(result.Violations)
	
	// Determine approval based on violations
	if len(result.Violations) > 0 {
		for _, violation := range result.Violations {
			if violation.Severity == "CRITICAL" {
				result.Approved = false
				result.Reason = "critical risk violation detected"
				break
			}
		}
	}
	
	// Save risk check record
	record := &RiskCheckRecord{
		ID:        generateRiskCheckID(),
		UserID:    order.UserID,
		OrderID:   order.ID,
		Result:    result,
		Order:     order,
		CheckedAt: time.Now(),
	}
	
	if err := rm.riskStore.SaveRiskCheck(ctx, record); err != nil {
		// Log error but don't fail the check
		fmt.Printf("Failed to save risk check record: %v\n", err)
	}
	
	return result, nil
}

// checkOrderValue validates order value against limits
func (rm *RiskManager) checkOrderValue(orderValue float64, result *RiskCheckResult) error {
	if orderValue > rm.config.MaxOrderValue {
		violation := RiskViolation{
			Rule:        "MAX_ORDER_VALUE",
			Description: "Order value exceeds maximum allowed",
			Severity:    "CRITICAL",
			Value:       orderValue,
			Limit:       rm.config.MaxOrderValue,
		}
		result.Violations = append(result.Violations, violation)
		result.Recommendations = append(result.Recommendations, 
			fmt.Sprintf("Reduce order size to stay within $%.2f limit", rm.config.MaxOrderValue))
	}
	
	return nil
}

// checkPositionSize validates position size limits
func (rm *RiskManager) checkPositionSize(ctx context.Context, order *interfaces.Order, profile *UserRiskProfile, result *RiskCheckResult) error {
	// Get current positions
	positions, err := rm.riskStore.GetPositions(ctx, order.UserID)
	if err != nil {
		return fmt.Errorf("failed to get user positions: %w", err)
	}
	
	// Calculate new position size
	var currentPosition float64
	for _, pos := range positions {
		if pos.Symbol == order.Symbol {
			currentPosition = pos.Quantity
			break
		}
	}
	
	var newPosition float64
	if order.Side == interfaces.OrderSideBuy {
		newPosition = currentPosition + order.Quantity
	} else {
		newPosition = currentPosition - order.Quantity
	}
	
	newPositionValue := newPosition * order.Price
	maxPositionSize := profile.MaxPositionSize
	if maxPositionSize == 0 {
		maxPositionSize = rm.config.MaxPositionSize
	}
	
	if newPositionValue > maxPositionSize {
		violation := RiskViolation{
			Rule:        "MAX_POSITION_SIZE",
			Description: "Position size exceeds maximum allowed",
			Severity:    "HIGH",
			Value:       newPositionValue,
			Limit:       maxPositionSize,
		}
		result.Violations = append(result.Violations, violation)
		result.Recommendations = append(result.Recommendations, 
			fmt.Sprintf("Reduce position size to stay within $%.2f limit", maxPositionSize))
	}
	
	return nil
}

// checkDailyVolume validates daily trading volume limits
func (rm *RiskManager) checkDailyVolume(ctx context.Context, order *interfaces.Order, profile *UserRiskProfile, result *RiskCheckResult) error {
	today := time.Now().Truncate(24 * time.Hour)
	currentVolume, err := rm.riskStore.GetDailyVolume(ctx, order.UserID, today)
	if err != nil {
		return fmt.Errorf("failed to get daily volume: %w", err)
	}
	
	orderValue := order.Price * order.Quantity
	newDailyVolume := currentVolume + orderValue
	
	maxDailyVolume := profile.MaxDailyVolume
	if maxDailyVolume == 0 {
		maxDailyVolume = rm.config.MaxDailyVolume
	}
	
	if newDailyVolume > maxDailyVolume {
		violation := RiskViolation{
			Rule:        "MAX_DAILY_VOLUME",
			Description: "Daily trading volume exceeds maximum allowed",
			Severity:    "HIGH",
			Value:       newDailyVolume,
			Limit:       maxDailyVolume,
		}
		result.Violations = append(result.Violations, violation)
		result.Recommendations = append(result.Recommendations, 
			fmt.Sprintf("Daily volume limit of $%.2f would be exceeded", maxDailyVolume))
	}
	
	return nil
}

// checkConcentration validates portfolio concentration limits
func (rm *RiskManager) checkConcentration(ctx context.Context, order *interfaces.Order, profile *UserRiskProfile, result *RiskCheckResult) error {
	// Get all positions
	positions, err := rm.riskStore.GetPositions(ctx, order.UserID)
	if err != nil {
		return fmt.Errorf("failed to get user positions: %w", err)
	}
	
	// Calculate total portfolio value
	var totalPortfolioValue float64
	var symbolValue float64
	
	for _, pos := range positions {
		totalPortfolioValue += pos.MarketValue
		if pos.Symbol == order.Symbol {
			symbolValue = pos.MarketValue
		}
	}
	
	// Add new order value
	orderValue := order.Price * order.Quantity
	if order.Side == interfaces.OrderSideBuy {
		symbolValue += orderValue
		totalPortfolioValue += orderValue
	}
	
	if totalPortfolioValue > 0 {
		concentration := symbolValue / totalPortfolioValue
		concentrationLimit := profile.ConcentrationLimit
		if concentrationLimit == 0 {
			concentrationLimit = rm.config.ConcentrationLimit
		}
		
		if concentration > concentrationLimit {
			violation := RiskViolation{
				Rule:        "CONCENTRATION_LIMIT",
				Description: "Portfolio concentration exceeds maximum allowed",
				Severity:    "MEDIUM",
				Value:       concentration * 100, // Convert to percentage
				Limit:       concentrationLimit * 100,
			}
			result.Violations = append(result.Violations, violation)
			result.Recommendations = append(result.Recommendations, 
				fmt.Sprintf("Diversify portfolio to stay within %.1f%% concentration limit", concentrationLimit*100))
		}
	}
	
	return nil
}

// checkVolatility validates volatility-based risk limits
func (rm *RiskManager) checkVolatility(ctx context.Context, order *interfaces.Order, result *RiskCheckResult) error {
	volatility, err := rm.calculator.GetVolatility(ctx, order.Symbol)
	if err != nil {
		// If we can't get volatility data, log but don't fail
		fmt.Printf("Failed to get volatility for %s: %v\n", order.Symbol, err)
		return nil
	}
	
	if volatility > rm.config.VolatilityThreshold {
		violation := RiskViolation{
			Rule:        "VOLATILITY_THRESHOLD",
			Description: "Asset volatility exceeds threshold",
			Severity:    "MEDIUM",
			Value:       volatility * 100, // Convert to percentage
			Limit:       rm.config.VolatilityThreshold * 100,
		}
		result.Violations = append(result.Violations, violation)
		result.Recommendations = append(result.Recommendations, 
			fmt.Sprintf("Consider reducing position size due to high volatility (%.1f%%)", volatility*100))
	}
	
	return nil
}

// calculateRiskScore calculates overall risk score based on violations
func (rm *RiskManager) calculateRiskScore(violations []RiskViolation) float64 {
	if len(violations) == 0 {
		return 0.0
	}
	
	var score float64
	for _, violation := range violations {
		switch violation.Severity {
		case "CRITICAL":
			score += 100.0
		case "HIGH":
			score += 50.0
		case "MEDIUM":
			score += 25.0
		case "LOW":
			score += 10.0
		}
	}
	
	// Cap at 100
	if score > 100 {
		score = 100
	}
	
	return score
}

// NewRiskCalculator creates a new risk calculator
func NewRiskCalculator() *RiskCalculator {
	return &RiskCalculator{
		volatilityCache: make(map[string]*VolatilityData),
		cacheTTL:        1 * time.Hour,
	}
}

// GetVolatility retrieves volatility data for a symbol
func (rc *RiskCalculator) GetVolatility(ctx context.Context, symbol string) (float64, error) {
	rc.mu.RLock()
	if cached, exists := rc.volatilityCache[symbol]; exists {
		if time.Since(cached.LastUpdated) < rc.cacheTTL {
			rc.mu.RUnlock()
			return cached.Volatility, nil
		}
	}
	rc.mu.RUnlock()
	
	// In a real implementation, this would fetch from a market data provider
	// For now, return a simulated volatility based on asset type
	volatility := rc.simulateVolatility(symbol)
	
	// Cache the result
	rc.mu.Lock()
	rc.volatilityCache[symbol] = &VolatilityData{
		Symbol:      symbol,
		Volatility:  volatility,
		LastUpdated: time.Now(),
	}
	rc.mu.Unlock()
	
	return volatility, nil
}

// simulateVolatility simulates volatility for demonstration
func (rc *RiskCalculator) simulateVolatility(symbol string) float64 {
	// Simplified volatility simulation based on symbol characteristics
	// In reality, this would come from market data providers
	
	if len(symbol) > 0 {
		switch symbol[0] {
		case 'A', 'B', 'C':
			return 0.15 // 15% volatility
		case 'D', 'E', 'F':
			return 0.25 // 25% volatility
		case 'G', 'H', 'I':
			return 0.35 // 35% volatility
		default:
			return 0.20 // 20% default volatility
		}
	}
	
	return 0.20 // Default 20% volatility
}

// generateRiskCheckID generates a unique risk check ID
func generateRiskCheckID() string {
	return fmt.Sprintf("RISK_%d", time.Now().UnixNano())
}

// GetDefaultRiskManagerConfig returns default risk manager configuration
func GetDefaultRiskManagerConfig() *RiskManagerConfig {
	return &RiskManagerConfig{
		MaxPositionSize:     100000,  // $100K
		MaxDailyVolume:      1000000, // $1M
		MaxOrderValue:       50000,   // $50K
		VolatilityThreshold: 0.30,    // 30%
		ConcentrationLimit:  0.20,    // 20%
		EnableRealTimeCheck: true,
		RiskCheckTimeout:    5 * time.Second,
	}
}

// GetUserRiskProfile retrieves or creates a default risk profile for a user
func (rm *RiskManager) GetUserRiskProfile(ctx context.Context, userID string) (*UserRiskProfile, error) {
	profile, err := rm.riskStore.GetUserRisk(ctx, userID)
	if err != nil {
		// Create default profile if not found
		profile = &UserRiskProfile{
			UserID:             userID,
			RiskTolerance:      "MEDIUM",
			MaxPositionSize:    rm.config.MaxPositionSize,
			MaxDailyVolume:     rm.config.MaxDailyVolume,
			ConcentrationLimit: rm.config.ConcentrationLimit,
			IsActive:           true,
			LastUpdated:        time.Now(),
		}
		
		if err := rm.riskStore.UpdateUserRisk(ctx, profile); err != nil {
			return nil, fmt.Errorf("failed to create default risk profile: %w", err)
		}
	}
	
	return profile, nil
}

// UpdateUserRiskProfile updates a user's risk profile
func (rm *RiskManager) UpdateUserRiskProfile(ctx context.Context, profile *UserRiskProfile) error {
	profile.LastUpdated = time.Now()
	return rm.riskStore.UpdateUserRisk(ctx, profile)
}

// GetRiskSummary returns a risk summary for a user
func (rm *RiskManager) GetRiskSummary(ctx context.Context, userID string) (*RiskSummary, error) {
	profile, err := rm.GetUserRiskProfile(ctx, userID)
	if err != nil {
		return nil, err
	}
	
	positions, err := rm.riskStore.GetPositions(ctx, userID)
	if err != nil {
		return nil, err
	}
	
	today := time.Now().Truncate(24 * time.Hour)
	dailyVolume, err := rm.riskStore.GetDailyVolume(ctx, userID, today)
	if err != nil {
		return nil, err
	}
	
	// Calculate portfolio metrics
	var totalValue float64
	var unrealizedPL float64
	concentrations := make(map[string]float64)
	
	for _, pos := range positions {
		totalValue += pos.MarketValue
		unrealizedPL += pos.UnrealizedPL
		concentrations[pos.Symbol] = pos.MarketValue
	}
	
	// Find highest concentration
	var maxConcentration float64
	var maxConcentrationSymbol string
	for symbol, value := range concentrations {
		if totalValue > 0 {
			concentration := value / totalValue
			if concentration > maxConcentration {
				maxConcentration = concentration
				maxConcentrationSymbol = symbol
			}
		}
	}
	
	return &RiskSummary{
		UserID:                   userID,
		RiskTolerance:           profile.RiskTolerance,
		TotalPortfolioValue:     totalValue,
		UnrealizedPL:            unrealizedPL,
		DailyVolume:             dailyVolume,
		DailyVolumeLimit:        profile.MaxDailyVolume,
		MaxConcentration:        maxConcentration,
		MaxConcentrationSymbol:  maxConcentrationSymbol,
		ConcentrationLimit:      profile.ConcentrationLimit,
		PositionCount:           len(positions),
		IsActive:                profile.IsActive,
		LastUpdated:             time.Now(),
	}, nil
}

// RiskSummary represents a user's risk summary
type RiskSummary struct {
	UserID                  string    `json:"user_id"`
	RiskTolerance          string    `json:"risk_tolerance"`
	TotalPortfolioValue    float64   `json:"total_portfolio_value"`
	UnrealizedPL           float64   `json:"unrealized_pl"`
	DailyVolume            float64   `json:"daily_volume"`
	DailyVolumeLimit       float64   `json:"daily_volume_limit"`
	MaxConcentration       float64   `json:"max_concentration"`
	MaxConcentrationSymbol string    `json:"max_concentration_symbol"`
	ConcentrationLimit     float64   `json:"concentration_limit"`
	PositionCount          int       `json:"position_count"`
	IsActive               bool      `json:"is_active"`
	LastUpdated            time.Time `json:"last_updated"`
}
