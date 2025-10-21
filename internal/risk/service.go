package risk

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/db/repositories"
	"github.com/abdoElHodaky/tradSys/proto/risk"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// AccountRisk represents risk metrics for an account
type AccountRisk struct {
	AccountID           string
	AvailableMargin     float64
	UsedMargin          float64
	TotalEquity         float64
	MarginLevel         float64
	MaxPositionSize     float64
	MaxNotionalValue    float64
	DailyLossLimit      float64
	CurrentDailyLoss    float64
	LastUpdated         time.Time
}

// ServiceParams contains the parameters for creating a risk service
type ServiceParams struct {
	fx.In

	Logger     *zap.Logger
	Repository *repositories.RiskRepository `optional:"true"`
}

// Service provides risk management operations with real-time risk calculations
type Service struct {
	logger     *zap.Logger
	repository *repositories.RiskRepository
	
	// In-memory account risk tracking for high-performance access
	accountRisks map[string]*AccountRisk
	riskMux      sync.RWMutex
}

// NewService creates a new risk service with fx dependency injection
func NewService(p ServiceParams) *Service {
	service := &Service{
		logger:       p.Logger,
		repository:   p.Repository,
		accountRisks: make(map[string]*AccountRisk),
	}
	
	// Initialize default account risks for demo purposes
	service.initializeDefaultAccounts()
	
	return service
}

// initializeDefaultAccounts creates default account risk profiles for testing
func (s *Service) initializeDefaultAccounts() {
	defaultAccounts := []string{"DEMO-001", "DEMO-002", "DEMO-003"}
	
	for _, accountID := range defaultAccounts {
		s.accountRisks[accountID] = &AccountRisk{
			AccountID:           accountID,
			AvailableMargin:     100000.0,  // $100k available margin
			UsedMargin:          0.0,
			TotalEquity:         100000.0,
			MarginLevel:         100.0,     // 100% margin level
			MaxPositionSize:     10.0,      // Max 10 units per position
			MaxNotionalValue:    500000.0,  // Max $500k notional
			DailyLossLimit:      5000.0,    // Max $5k daily loss
			CurrentDailyLoss:    0.0,
			LastUpdated:         time.Now(),
		}
	}
}

// ValidateOrder validates an order against real-time risk parameters with <10μs target
func (s *Service) ValidateOrder(ctx context.Context, symbol string, side risk.OrderSide, orderType risk.OrderType, quantity, price float64, accountID string) (*risk.ValidateOrderResponse, error) {
	startTime := time.Now()
	
	s.logger.Info("Validating order",
		zap.String("symbol", symbol),
		zap.String("side", side.String()),
		zap.Float64("quantity", quantity),
		zap.Float64("price", price),
		zap.String("account_id", accountID))

	// Get account risk profile (optimized for speed)
	s.riskMux.RLock()
	accountRisk, exists := s.accountRisks[accountID]
	s.riskMux.RUnlock()

	if !exists {
		// Create default account risk if not found
		accountRisk = &AccountRisk{
			AccountID:           accountID,
			AvailableMargin:     50000.0,
			UsedMargin:          0.0,
			TotalEquity:         50000.0,
			MarginLevel:         100.0,
			MaxPositionSize:     5.0,
			MaxNotionalValue:    250000.0,
			DailyLossLimit:      2500.0,
			CurrentDailyLoss:    0.0,
			LastUpdated:         time.Now(),
		}
		
		s.riskMux.Lock()
		s.accountRisks[accountID] = accountRisk
		s.riskMux.Unlock()
	}

	// Calculate order metrics
	notionalValue := quantity * price
	marginRequirement := s.calculateMarginRequirement(symbol, quantity, price, orderType)
	
	// Initialize response
	response := &risk.ValidateOrderResponse{
		IsValid: true,
		RiskMetrics: &risk.OrderRiskResponse{
			AccountId:            accountID,
			Symbol:               symbol,
			Side:                 side,
			Type:                 orderType,
			Quantity:             quantity,
			Price:                price,
			RequiredMargin:       marginRequirement,
			AvailableMarginAfter: accountRisk.AvailableMargin - marginRequirement,
			MarginLevelAfter:     s.calculateMarginLevelAfter(accountRisk, marginRequirement),
			RiskLevel:            risk.RiskLevel_LOW,
			IsAllowed:            true,
		},
	}

	// Perform risk checks (optimized for speed)
	if err := s.performRiskChecks(accountRisk, quantity, notionalValue, marginRequirement, response); err != nil {
		response.IsValid = false
		response.RejectionReason = err.Error()
		response.RiskMetrics.IsAllowed = false
		response.RiskMetrics.RejectionReason = err.Error()
	}

	// Calculate risk level based on multiple factors
	response.RiskMetrics.RiskLevel = s.calculateRiskLevel(accountRisk, notionalValue, marginRequirement)

	// Log performance metrics
	duration := time.Since(startTime)
	s.logger.Info("Order validation completed",
		zap.String("account_id", accountID),
		zap.Bool("is_valid", response.IsValid),
		zap.Duration("duration", duration),
		zap.Int64("duration_ns", duration.Nanoseconds()))

	return response, nil
}

// calculateMarginRequirement calculates margin requirement based on instrument and order type
func (s *Service) calculateMarginRequirement(symbol string, quantity, price float64, orderType risk.OrderType) float64 {
	// Base margin rate (varies by instrument)
	marginRate := 0.2 // 20% for most instruments
	
	// Adjust margin rate based on symbol (simplified)
	switch {
	case symbol == "BTC-USD" || symbol == "ETH-USD":
		marginRate = 0.5 // 50% for crypto
	case symbol == "EUR-USD" || symbol == "GBP-USD":
		marginRate = 0.02 // 2% for major FX pairs
	}
	
	// Market orders may require higher margin
	if orderType == risk.OrderType_MARKET {
		marginRate *= 1.1 // 10% buffer for market orders
	}
	
	return quantity * price * marginRate
}

// calculateMarginLevelAfter calculates margin level after the order
func (s *Service) calculateMarginLevelAfter(accountRisk *AccountRisk, marginRequirement float64) float64 {
	newUsedMargin := accountRisk.UsedMargin + marginRequirement
	if newUsedMargin == 0 {
		return 100.0
	}
	return (accountRisk.TotalEquity / newUsedMargin) * 100
}

// performRiskChecks performs all risk validations (optimized for speed)
func (s *Service) performRiskChecks(accountRisk *AccountRisk, quantity, notionalValue, marginRequirement float64, response *risk.ValidateOrderResponse) error {
	// Check position size limit
	if quantity > accountRisk.MaxPositionSize {
		return fmt.Errorf("order quantity %.2f exceeds maximum allowed %.2f", quantity, accountRisk.MaxPositionSize)
	}
	
	// Check notional value limit
	if notionalValue > accountRisk.MaxNotionalValue {
		return fmt.Errorf("order notional value %.2f exceeds maximum allowed %.2f", notionalValue, accountRisk.MaxNotionalValue)
	}
	
	// Check available margin
	if marginRequirement > accountRisk.AvailableMargin {
		return fmt.Errorf("insufficient margin: required %.2f, available %.2f", marginRequirement, accountRisk.AvailableMargin)
	}
	
	// Check margin level after order
	marginLevelAfter := s.calculateMarginLevelAfter(accountRisk, marginRequirement)
	if marginLevelAfter < 50.0 { // Minimum 50% margin level
		return fmt.Errorf("margin level after order would be %.2f%%, minimum required 50%%", marginLevelAfter)
	}
	
	// Check daily loss limit
	if accountRisk.CurrentDailyLoss >= accountRisk.DailyLossLimit {
		return fmt.Errorf("daily loss limit reached: %.2f/%.2f", accountRisk.CurrentDailyLoss, accountRisk.DailyLossLimit)
	}
	
	return nil
}

// calculateRiskLevel determines risk level based on multiple factors
func (s *Service) calculateRiskLevel(accountRisk *AccountRisk, notionalValue, marginRequirement float64) risk.RiskLevel {
	// Calculate risk score based on multiple factors
	riskScore := 0.0
	
	// Factor 1: Margin utilization
	marginUtilization := (accountRisk.UsedMargin + marginRequirement) / accountRisk.TotalEquity
	riskScore += marginUtilization * 40 // 40% weight
	
	// Factor 2: Position size relative to max
	positionRatio := notionalValue / accountRisk.MaxNotionalValue
	riskScore += positionRatio * 30 // 30% weight
	
	// Factor 3: Daily loss ratio
	lossRatio := accountRisk.CurrentDailyLoss / accountRisk.DailyLossLimit
	riskScore += lossRatio * 30 // 30% weight
	
	// Determine risk level
	if riskScore < 0.3 {
		return risk.RiskLevel_LOW
	} else if riskScore < 0.7 {
		return risk.RiskLevel_MEDIUM
	} else {
		return risk.RiskLevel_HIGH
	}
}

// GetPositions returns current positions
func (s *Service) GetPositions(ctx context.Context, accountID, symbol string) ([]*risk.Position, error) {
	s.logger.Info("Getting positions",
		zap.String("account_id", accountID),
		zap.String("symbol", symbol))

	// Implementation would go here
	// For now, just return placeholder positions
	positions := []*risk.Position{
		{
			Symbol:          "BTC-USD",
			Size:            1.5,
			EntryPrice:      48000.0,
			CurrentPrice:    50000.0,
			LiquidationPrice: 40000.0,
			UnrealizedPnl:   3000.0,
			RealizedPnl:     1000.0,
		},
	}

	// If symbol is specified, filter the positions
	if symbol != "" {
		var filteredPositions []*risk.Position
		for _, pos := range positions {
			if pos.Symbol == symbol {
				filteredPositions = append(filteredPositions, pos)
			}
		}
		positions = filteredPositions
	}

	return positions, nil
}

// GetRiskLimits returns risk limits for a symbol
func (s *Service) GetRiskLimits(ctx context.Context, symbol, accountID string) (*risk.RiskLimits, error) {
	s.logger.Info("Getting risk limits",
		zap.String("symbol", symbol),
		zap.String("account_id", accountID))

	// Implementation would go here
	// For now, just return placeholder risk limits
	limits := &risk.RiskLimits{
		MaxPositionSize:   10.0,
		MaxOrderSize:      5.0,
		MaxLeverage:       5.0,
		MaxDailyLoss:      1000.0,
		MaxTotalLoss:      5000.0,
		MinMarginLevel:    120.0,
		MarginCallLevel:   120.0,
		LiquidationLevel:  100.0,
	}

	return limits, nil
}

// UpdateRiskLimits updates risk limits for a symbol
func (s *Service) UpdateRiskLimits(ctx context.Context, symbol, accountID string, limits *risk.RiskLimits) (*risk.RiskLimits, error) {
	s.logger.Info("Updating risk limits",
		zap.String("symbol", symbol),
		zap.String("account_id", accountID),
		zap.Float64("max_position_size", limits.MaxPositionSize),
		zap.Float64("max_order_size", limits.MaxOrderSize))

	// Implementation would go here
	// For now, just return the updated limits
	return limits, nil
}

// ServiceModule is defined in module.go to avoid duplication
