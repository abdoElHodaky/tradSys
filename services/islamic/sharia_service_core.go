// Package islamic provides Islamic finance services for TradSys v3
package islamic

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/abdoElHodaky/tradSys/pkg/interfaces"
	"github.com/abdoElHodaky/tradSys/pkg/types"
)

// NewShariaService creates a new Sharia compliance service
func NewShariaService(config *ShariaConfig, db ComplianceDatabase) *ShariaService {
	service := &ShariaService{
		config:          config,
		screeningEngine: NewScreeningEngine(config.ScreeningRules),
		zakatCalculator: NewZakatCalculator(&ZakatConfig{
			StandardRate:   config.ZakatRate,
			NisabThreshold: config.NisabThreshold,
			Currency:       config.Currency,
		}),
		complianceDB: db,
	}

	if config.EnableShariaBoard {
		service.shariaBoard = NewShariaBoard()
	}

	return service
}

// IsShariahCompliant checks if a symbol is Sharia-compliant
func (s *ShariaService) IsShariahCompliant(ctx context.Context, symbol string) (bool, error) {
	result, err := s.GetHalalScreening(ctx, symbol)
	if err != nil {
		return false, err
	}

	return result.IsCompliant, nil
}

// GetHalalScreening performs comprehensive Sharia compliance screening
func (s *ShariaService) GetHalalScreening(ctx context.Context, symbol string) (*interfaces.HalalScreening, error) {
	// Check cache first
	if cached := s.screeningEngine.getCachedResult(symbol); cached != nil {
		return &interfaces.HalalScreening{
			Symbol:          cached.Symbol,
			IsCompliant:     cached.IsCompliant,
			Score:           cached.ComplianceScore,
			Violations:      cached.Violations,
			Recommendations: cached.Recommendations,
			LastUpdated:     cached.LastUpdated,
		}, nil
	}

	// Perform screening
	result, err := s.screeningEngine.PerformScreening(ctx, symbol)
	if err != nil {
		return nil, fmt.Errorf("screening failed: %w", err)
	}

	// Save to database
	if err := s.complianceDB.SaveScreeningResult(ctx, result); err != nil {
		// Log error but don't fail the screening
		fmt.Printf("Failed to save screening result: %v\n", err)
	}

	return &interfaces.HalalScreening{
		Symbol:          result.Symbol,
		IsCompliant:     result.IsCompliant,
		Score:           result.ComplianceScore,
		Violations:      result.Violations,
		Recommendations: result.Recommendations,
		LastUpdated:     result.LastUpdated,
	}, nil
}

// CalculateZakat calculates Zakat for a user's portfolio
func (s *ShariaService) CalculateZakat(ctx context.Context, userID string, portfolio *interfaces.Portfolio) (*interfaces.ZakatCalculation, error) {
	if !s.config.EnableZakat {
		return nil, fmt.Errorf("Zakat calculation is disabled")
	}

	record, err := s.zakatCalculator.CalculateZakat(ctx, userID, portfolio)
	if err != nil {
		return nil, fmt.Errorf("Zakat calculation failed: %w", err)
	}

	// Save to database
	if err := s.complianceDB.SaveZakatRecord(ctx, record); err != nil {
		fmt.Printf("Failed to save Zakat record: %v\n", err)
	}

	return &interfaces.ZakatCalculation{
		UserID:          record.UserID,
		Year:            record.Year,
		TotalWealth:     record.TotalWealth,
		ZakatableAmount: record.ZakatableAmount,
		ZakatDue:        record.ZakatDue,
		Currency:        record.Currency,
		CalculationDate: record.CalculationDate,
		NextDueDate:     record.NextDueDate,
	}, nil
}

// ValidateIslamicOrder validates orders for Islamic assets
func (s *ShariaService) ValidateIslamicOrder(ctx context.Context, order *interfaces.Order) error {
	// Check if asset is Sharia-compliant
	isCompliant, err := s.IsShariahCompliant(ctx, order.Symbol)
	if err != nil {
		return fmt.Errorf("compliance check failed: %w", err)
	}

	if !isCompliant {
		return fmt.Errorf("asset %s is not Sharia-compliant", order.Symbol)
	}

	// Asset-specific validations
	switch order.AssetType {
	case types.SUKUK:
		return s.validateSukukOrder(order)
	case types.ISLAMIC_FUND:
		return s.validateIslamicFundOrder(order)
	case types.TAKAFUL:
		return s.validateTakafulOrder(order)
	case types.SHARIA_STOCK:
		return s.validateShariaStockOrder(order)
	default:
		return fmt.Errorf("unsupported Islamic asset type: %s", order.AssetType)
	}
}

// validateShariaStockOrder validates Sharia-compliant stock orders
func (s *ShariaService) validateShariaStockOrder(order *interfaces.Order) error {
	// Standard stock validations apply
	if order.Quantity <= 0 {
		return fmt.Errorf("order quantity must be positive")
	}

	if order.Price <= 0 && order.Type != interfaces.OrderTypeMarket {
		return fmt.Errorf("price must be positive for non-market orders")
	}

	return nil
}

// validateSukukOrder validates Sukuk-specific requirements
func (s *ShariaService) validateSukukOrder(order *interfaces.Order) error {
	// Sukuk minimum investment requirements
	if order.Quantity < 1000 {
		return fmt.Errorf("minimum Sukuk investment is 1000 units")
	}

	// Check if it's a trading day (no trading on Fridays for some Sukuk)
	if time.Now().Weekday() == time.Friday {
		return fmt.Errorf("Sukuk trading not allowed on Fridays")
	}

	return nil
}

// validateIslamicFundOrder validates Islamic fund requirements
func (s *ShariaService) validateIslamicFundOrder(order *interfaces.Order) error {
	// Islamic fund minimum investment
	if order.Quantity < 100 {
		return fmt.Errorf("minimum Islamic fund investment is 100 units")
	}

	return nil
}

// validateTakafulOrder validates Takaful (Islamic insurance) requirements
func (s *ShariaService) validateTakafulOrder(order *interfaces.Order) error {
	// Takaful specific validations
	if order.Quantity < 1 {
		return fmt.Errorf("minimum Takaful investment is 1 unit")
	}

	return nil
}

// GetShariaBoard returns Sharia board information
func (s *ShariaService) GetShariaBoard(ctx context.Context) (*ShariaBoard, error) {
	if !s.config.EnableShariaBoard || s.shariaBoard == nil {
		return nil, fmt.Errorf("Sharia board information not available")
	}

	return s.shariaBoard, nil
}

// NewScreeningEngine creates a new screening engine
func NewScreeningEngine(rules []ShariaRule) *ScreeningEngine {
	engine := &ScreeningEngine{
		rules:    make(map[string]ShariaRule),
		cache:    make(map[string]*ScreeningResult),
		cacheTTL: 24 * time.Hour, // Cache results for 24 hours
	}

	for _, rule := range rules {
		engine.rules[rule.ID] = rule
	}

	return engine
}

// PerformScreening performs Sharia compliance screening for a symbol
func (e *ScreeningEngine) PerformScreening(ctx context.Context, symbol string) (*ScreeningResult, error) {
	result := &ScreeningResult{
		Symbol:          symbol,
		IsCompliant:     true,
		ComplianceScore: 100.0,
		Violations:      []string{},
		Recommendations: []string{},
		LastUpdated:     time.Now(),
		RulesApplied:    []string{},
	}

	// Apply all active rules
	for _, rule := range e.rules {
		if !rule.IsActive {
			continue
		}

		result.RulesApplied = append(result.RulesApplied, rule.ID)

		// This would typically call external APIs or databases
		// For now, we'll use simplified logic
		if !e.applyRule(rule, symbol) {
			result.IsCompliant = false
			result.ComplianceScore -= 20.0 // Deduct points for violations
			result.Violations = append(result.Violations, rule.Description)
			result.Recommendations = append(result.Recommendations,
				fmt.Sprintf("Address violation: %s", rule.Name))
		}
	}

	// Ensure score doesn't go below 0
	if result.ComplianceScore < 0 {
		result.ComplianceScore = 0
	}

	// Cache the result
	e.cacheResult(symbol, result)

	return result, nil
}

// getCachedResult retrieves cached screening result
func (e *ScreeningEngine) getCachedResult(symbol string) *ScreeningResult {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if result, exists := e.cache[symbol]; exists {
		// Check if cache is still valid
		if time.Since(result.LastUpdated) < e.cacheTTL {
			return result
		}
		// Remove expired cache entry
		delete(e.cache, symbol)
	}

	return nil
}

// cacheResult caches screening result
func (e *ScreeningEngine) cacheResult(symbol string, result *ScreeningResult) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.cache[symbol] = result
}

// applyRule applies a specific Sharia rule to a symbol
func (e *ScreeningEngine) applyRule(rule ShariaRule, symbol string) bool {
	// Simplified rule application logic
	// In a real implementation, this would check against external data sources

	switch rule.ID {
	case "interest_prohibition":
		// Check if company is involved in interest-based activities
		return !e.isInterestBased(symbol)
	case "prohibited_activities":
		// Check if company is involved in prohibited activities
		return !e.hasProhibitedActivities(symbol)
	case "debt_ratio_check":
		// Check debt-to-equity ratio
		return e.checkDebtRatio(symbol)
	default:
		// Unknown rule, assume compliant
		return true
	}
}

// isInterestBased checks if a company is involved in interest-based activities
func (e *ScreeningEngine) isInterestBased(symbol string) bool {
	// Simplified logic - in reality, this would check against financial data
	prohibitedSectors := []string{"BANK", "INSURANCE", "FINANCE"}

	// This would typically query external APIs or databases
	// For demonstration, we'll use simplified logic
	for _, sector := range prohibitedSectors {
		if symbol == sector {
			return true
		}
	}

	return false
}

// hasProhibitedActivities checks if a company has prohibited business activities
func (e *ScreeningEngine) hasProhibitedActivities(symbol string) bool {
	// Simplified logic - check against prohibited activities
	prohibitedActivities := []string{"ALCOHOL", "GAMBLING", "PORK", "TOBACCO"}

	for _, activity := range prohibitedActivities {
		if symbol == activity {
			return true
		}
	}

	return false
}

// checkDebtRatio checks if debt-to-equity ratio is within acceptable limits
func (e *ScreeningEngine) checkDebtRatio(symbol string) bool {
	// Simplified logic - in reality, this would check financial ratios
	// Assume acceptable debt ratio is below 33%

	// This would typically query financial data APIs
	// For demonstration, we'll assume most companies pass this test
	return true
}

// NewZakatCalculator creates a new Zakat calculator
func NewZakatCalculator(config *ZakatConfig) *ZakatCalculator {
	calculator := &ZakatCalculator{
		config:     config,
		rateTable:  make(map[types.AssetType]float64),
		exemptions: make(map[types.AssetType]bool),
	}

	// Set default rates for different asset types
	calculator.rateTable[types.STOCK] = config.StandardRate
	calculator.rateTable[types.SUKUK] = config.StandardRate
	calculator.rateTable[types.ISLAMIC_FUND] = config.StandardRate
	calculator.rateTable[types.SHARIA_STOCK] = config.StandardRate
	calculator.rateTable[types.ISLAMIC_ETF] = config.StandardRate

	// Set exemptions
	calculator.exemptions[types.TAKAFUL] = true // Takaful is typically exempt

	return calculator
}

// CalculateZakat calculates Zakat for a portfolio
func (z *ZakatCalculator) CalculateZakat(ctx context.Context, userID string, portfolio *interfaces.Portfolio) (*ZakatRecord, error) {
	totalWealth := 0.0
	zakatableAmount := 0.0
	exemptAssets := []string{}

	// Calculate total wealth and zakatable amount
	for _, holding := range portfolio.Holdings {
		assetValue := holding.Quantity * holding.CurrentPrice
		totalWealth += assetValue

		// Check if asset is exempt from Zakat
		if exempt, exists := z.exemptions[holding.AssetType]; exists && exempt {
			exemptAssets = append(exemptAssets, holding.Symbol)
			continue
		}

		// Add to zakatable amount
		zakatableAmount += assetValue
	}

	// Check if wealth meets Nisab threshold
	if totalWealth < z.config.NisabThreshold {
		return &ZakatRecord{
			UserID:          userID,
			Year:            time.Now().Year(),
			TotalWealth:     totalWealth,
			ZakatableAmount: 0,
			ZakatRate:       z.config.StandardRate,
			ZakatDue:        0,
			Currency:        z.config.Currency,
			CalculationDate: time.Now(),
			NextDueDate:     time.Now().AddDate(1, 0, 0), // Next year
			ExemptAssets:    exemptAssets,
		}, nil
	}

	// Calculate Zakat due
	zakatDue := zakatableAmount * z.config.StandardRate

	// Round to 2 decimal places
	zakatDue = math.Round(zakatDue*100) / 100

	return &ZakatRecord{
		UserID:          userID,
		Year:            time.Now().Year(),
		TotalWealth:     totalWealth,
		ZakatableAmount: zakatableAmount,
		ZakatRate:       z.config.StandardRate,
		ZakatDue:        zakatDue,
		Currency:        z.config.Currency,
		CalculationDate: time.Now(),
		NextDueDate:     time.Now().AddDate(1, 0, 0), // Next year
		ExemptAssets:    exemptAssets,
	}, nil
}
