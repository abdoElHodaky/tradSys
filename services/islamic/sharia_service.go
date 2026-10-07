// Package islamic provides Islamic finance services for TradSys v3
package islamic

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/pkg/interfaces"
	"github.com/abdoElHodaky/tradSys/pkg/types"
)

// ShariaService provides Sharia compliance and Islamic finance services
type ShariaService struct {
	config          *ShariaConfig
	screeningEngine *ScreeningEngine
	zakatCalculator *ZakatCalculator
	complianceDB    ComplianceDatabase
	shariaBoard     *ShariaBoard
	mu              sync.RWMutex
}

// ShariaConfig holds configuration for Islamic finance services
type ShariaConfig struct {
	EnableScreening     bool
	EnableZakat         bool
	EnableShariaBoard   bool
	ScreeningRules      []ShariaRule
	ZakatRate           float64
	NisabThreshold      float64
	Currency            string
	ShariaStandard      string
	ComplianceLevel     ComplianceLevel
}

// ShariaRule represents an Islamic finance compliance rule
type ShariaRule struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	Description     string                 `json:"description"`
	Category        string                 `json:"category"`
	AssetTypes      []types.AssetType      `json:"asset_types"`
	Validator       func(interface{}) bool `json:"-"`
	ComplianceLevel ComplianceLevel        `json:"compliance_level"`
	IsActive        bool                   `json:"is_active"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

// ComplianceLevel represents the level of Sharia compliance
type ComplianceLevel string

const (
	STRICT     ComplianceLevel = "STRICT"
	MODERATE   ComplianceLevel = "MODERATE"
	FLEXIBLE   ComplianceLevel = "FLEXIBLE"
)

// ScreeningEngine performs Sharia compliance screening
type ScreeningEngine struct {
	rules       map[string]ShariaRule
	cache       map[string]*ScreeningResult
	cacheTTL    time.Duration
	mu          sync.RWMutex
}

// ZakatCalculator calculates Zakat for Islamic portfolios
type ZakatCalculator struct {
	config      *ZakatConfig
	rateTable   map[types.AssetType]float64
	exemptions  map[types.AssetType]bool
	mu          sync.RWMutex
}

// ZakatConfig holds Zakat calculation configuration
type ZakatConfig struct {
	StandardRate    float64 // 2.5% standard rate
	NisabThreshold  float64 // Minimum wealth threshold
	Currency        string
	CalculationDate time.Time
	HijriYear       int
}

// ComplianceDatabase interface for storing compliance data
type ComplianceDatabase interface {
	GetScreeningResult(ctx context.Context, symbol string) (*ScreeningResult, error)
	SaveScreeningResult(ctx context.Context, result *ScreeningResult) error
	GetZakatRecord(ctx context.Context, userID string, year int) (*ZakatRecord, error)
	SaveZakatRecord(ctx context.Context, record *ZakatRecord) error
}

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

// CalculateZakat calculates Zakat for an Islamic portfolio
func (s *ShariaService) CalculateZakat(ctx context.Context, portfolio *IslamicPortfolio) (*ZakatCalculation, error) {
	if !s.config.EnableZakat {
		return nil, fmt.Errorf("Zakat calculation not enabled")
	}
	
	return s.zakatCalculator.Calculate(ctx, portfolio)
}

// ValidateOrder validates an order for Islamic finance compliance
func (s *ShariaService) ValidateOrder(ctx context.Context, order *interfaces.Order) error {
	if !order.AssetType.IsIslamic() {
		return nil // No validation needed for non-Islamic assets
	}
	
	// Check if asset is Sharia-compliant
	isCompliant, err := s.IsShariahCompliant(ctx, order.Symbol)
	if err != nil {
		return fmt.Errorf("compliance check failed: %w", err)
	}
	
	if !isCompliant {
		return fmt.Errorf("asset %s is not Sharia-compliant", order.Symbol)
	}
	
	// Additional Islamic finance validations
	return s.validateIslamicOrder(order)
}

// validateIslamicOrder performs Islamic-specific order validations
func (s *ShariaService) validateIslamicOrder(order *interfaces.Order) error {
	switch order.AssetType {
	case types.SUKUK:
		return s.validateSukukOrder(order)
	case types.ISLAMIC_FUND:
		return s.validateIslamicFundOrder(order)
	case types.TAKAFUL:
		return s.validateTakafulOrder(order)
	default:
		return nil
	}
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

// applyRule applies a specific Sharia rule (simplified implementation)
func (e *ScreeningEngine) applyRule(rule ShariaRule, symbol string) bool {
	// This is a simplified implementation
	// In reality, this would involve complex business logic and external data
	
	switch rule.Category {
	case "interest_based":
		// Check if company is involved in interest-based activities
		return !e.isInterestBased(symbol)
	case "prohibited_activities":
		// Check for prohibited activities (alcohol, gambling, etc.)
		return !e.hasProhibitedActivities(symbol)
	case "debt_ratio":
		// Check debt-to-equity ratio
		return e.checkDebtRatio(symbol)
	default:
		return true
	}
}

// Simplified screening methods (would be more complex in reality)
func (e *ScreeningEngine) isInterestBased(symbol string) bool {
	// Simplified: assume banks and financial institutions are interest-based
	return false // This would check actual business activities
}

func (e *ScreeningEngine) hasProhibitedActivities(symbol string) bool {
	// Check for prohibited business activities
	return false // This would check actual business activities
}

func (e *ScreeningEngine) checkDebtRatio(symbol string) bool {
	// Check if debt-to-equity ratio is within acceptable limits
	return true // This would check actual financial ratios
}

// getCachedResult retrieves cached screening result
func (e *ScreeningEngine) getCachedResult(symbol string) *ScreeningResult {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	if result, exists := e.cache[symbol]; exists {
		if time.Since(result.LastUpdated) < e.cacheTTL {
			return result
		}
		// Remove expired cache entry
		delete(e.cache, symbol)
	}
	
	return nil
}

// cacheResult caches a screening result
func (e *ScreeningEngine) cacheResult(symbol string, result *ScreeningResult) {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	e.cache[symbol] = result
}

// NewZakatCalculator creates a new Zakat calculator
func NewZakatCalculator(config *ZakatConfig) *ZakatCalculator {
	calculator := &ZakatCalculator{
		config:     config,
		rateTable:  make(map[types.AssetType]float64),
		exemptions: make(map[types.AssetType]bool),
	}
	
	// Set standard rates for different asset types
	calculator.rateTable[types.STOCK] = 0.025        // 2.5%
	calculator.rateTable[types.SUKUK] = 0.025        // 2.5%
	calculator.rateTable[types.ISLAMIC_FUND] = 0.025 // 2.5%
	calculator.rateTable[types.SHARIA_STOCK] = 0.025 // 2.5%
	
	// Set exemptions
	calculator.exemptions[types.TAKAFUL] = true // Takaful is typically exempt
	
	return calculator
}

// Calculate calculates Zakat for an Islamic portfolio
func (z *ZakatCalculator) Calculate(ctx context.Context, portfolio *IslamicPortfolio) (*ZakatCalculation, error) {
	if portfolio.TotalValue < z.config.NisabThreshold {
		return &ZakatCalculation{
			PortfolioValue:  portfolio.TotalValue,
			ZakatableAmount: 0,
			ZakatRate:       0,
			ZakatDue:        0,
			Currency:        portfolio.Currency,
			CalculationDate: time.Now(),
			NextDueDate:     z.getNextZakatDueDate(),
			ExemptAssets:    z.getExemptAssets(portfolio),
		}, nil
	}
	
	zakatableAmount := z.calculateZakatableAmount(portfolio)
	zakatDue := zakatableAmount * z.config.StandardRate
	
	return &ZakatCalculation{
		PortfolioValue:  portfolio.TotalValue,
		ZakatableAmount: zakatableAmount,
		ZakatRate:       z.config.StandardRate,
		ZakatDue:        zakatDue,
		Currency:        portfolio.Currency,
		CalculationDate: time.Now(),
		NextDueDate:     z.getNextZakatDueDate(),
		ExemptAssets:    z.getExemptAssets(portfolio),
	}, nil
}

// calculateZakatableAmount calculates the amount subject to Zakat
func (z *ZakatCalculator) calculateZakatableAmount(portfolio *IslamicPortfolio) float64 {
	zakatableAmount := 0.0
	
	for _, asset := range portfolio.Assets {
		if z.exemptions[asset.AssetType] {
			continue // Skip exempt assets
		}
		
		if asset.IsHalal {
			zakatableAmount += asset.MarketValue
		}
	}
	
	return zakatableAmount
}

// getNextZakatDueDate calculates the next Zakat due date
func (z *ZakatCalculator) getNextZakatDueDate() time.Time {
	now := time.Now()
	// Zakat is typically due annually
	return now.AddDate(1, 0, 0)
}

// getExemptAssets returns list of exempt assets
func (z *ZakatCalculator) getExemptAssets(portfolio *IslamicPortfolio) []string {
	var exemptAssets []string
	
	for symbol, asset := range portfolio.Assets {
		if z.exemptions[asset.AssetType] {
			exemptAssets = append(exemptAssets, symbol)
		}
	}
	
	return exemptAssets
}

// Supporting types and structures

// ScreeningResult represents the result of Sharia compliance screening
type ScreeningResult struct {
	Symbol          string    `json:"symbol"`
	IsCompliant     bool      `json:"is_compliant"`
	ComplianceScore float64   `json:"compliance_score"`
	Violations      []string  `json:"violations"`
	Recommendations []string  `json:"recommendations"`
	RulesApplied    []string  `json:"rules_applied"`
	LastUpdated     time.Time `json:"last_updated"`
}

// ZakatRecord represents a Zakat calculation record
type ZakatRecord struct {
	ID              string    `json:"id"`
	UserID          string    `json:"user_id"`
	Year            int       `json:"year"`
	PortfolioValue  float64   `json:"portfolio_value"`
	ZakatableAmount float64   `json:"zakatable_amount"`
	ZakatDue        float64   `json:"zakat_due"`
	Currency        string    `json:"currency"`
	CalculatedAt    time.Time `json:"calculated_at"`
	PaidAt          *time.Time `json:"paid_at,omitempty"`
	Status          string    `json:"status"`
}

// IslamicPortfolio represents a portfolio of Islamic assets
type IslamicPortfolio struct {
	UserID     string                    `json:"user_id"`
	Assets     map[string]IslamicAsset   `json:"assets"`
	TotalValue float64                   `json:"total_value"`
	Currency   string                    `json:"currency"`
	AsOfDate   time.Time                 `json:"as_of_date"`
}

// IslamicAsset represents an Islamic financial asset
type IslamicAsset struct {
	Symbol       string          `json:"symbol"`
	AssetType    types.AssetType `json:"asset_type"`
	Quantity     float64         `json:"quantity"`
	CurrentPrice float64         `json:"current_price"`
	MarketValue  float64         `json:"market_value"`
	IsHalal      bool            `json:"is_halal"`
}

// ZakatCalculation represents Zakat calculation results
type ZakatCalculation struct {
	PortfolioValue    float64   `json:"portfolio_value"`
	ZakatableAmount   float64   `json:"zakatable_amount"`
	ZakatRate         float64   `json:"zakat_rate"`
	ZakatDue          float64   `json:"zakat_due"`
	Currency          string    `json:"currency"`
	CalculationDate   time.Time `json:"calculation_date"`
	NextDueDate       time.Time `json:"next_due_date"`
	ExemptAssets      []string  `json:"exempt_assets"`
}

// ShariaBoard represents Sharia board information
type ShariaBoard struct {
	Name           string              `json:"name"`
	Members        []ShariaBoardMember `json:"members"`
	Established    time.Time           `json:"established"`
	Certifications []string            `json:"certifications"`
	ContactInfo    ContactInfo         `json:"contact_info"`
}

// ShariaBoardMember represents a Sharia board member
type ShariaBoardMember struct {
	Name           string   `json:"name"`
	Title          string   `json:"title"`
	Qualifications []string `json:"qualifications"`
	Experience     int      `json:"experience_years"`
}

// ContactInfo represents contact information
type ContactInfo struct {
	Email   string `json:"email"`
	Phone   string `json:"phone"`
	Address string `json:"address"`
	Website string `json:"website"`
}

// NewShariaBoard creates a default Sharia board
func NewShariaBoard() *ShariaBoard {
	return &ShariaBoard{
		Name:        "TradSys Islamic Finance Advisory Board",
		Established: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		Members: []ShariaBoardMember{
			{
				Name:           "Dr. Ahmed Al-Rashid",
				Title:          "Chairman",
				Qualifications: []string{"PhD Islamic Finance", "Certified Sharia Advisor"},
				Experience:     15,
			},
			{
				Name:           "Sheikh Mohammed Al-Qasimi",
				Title:          "Senior Advisor",
				Qualifications: []string{"Islamic Jurisprudence", "Financial Markets Expert"},
				Experience:     20,
			},
		},
		Certifications: []string{"AAOIFI Certified", "IFSB Compliant"},
		ContactInfo: ContactInfo{
			Email:   "sharia@tradsys.com",
			Phone:   "+971-4-123-4567",
			Address: "Dubai International Financial Centre, UAE",
			Website: "https://tradsys.com/islamic-finance",
		},
	}
}

// GetDefaultShariaConfig returns default Sharia configuration
func GetDefaultShariaConfig() *ShariaConfig {
	return &ShariaConfig{
		EnableScreening:   true,
		EnableZakat:       true,
		EnableShariaBoard: true,
		ZakatRate:         0.025, // 2.5%
		NisabThreshold:    10000, // $10,000 minimum threshold
		Currency:          "USD",
		ShariaStandard:    "AAOIFI",
		ComplianceLevel:   MODERATE,
		ScreeningRules: []ShariaRule{
			{
				ID:              "interest_prohibition",
				Name:            "Interest Prohibition",
				Description:     "Prohibits investment in interest-based activities",
				Category:        "interest_based",
				AssetTypes:      []types.AssetType{types.STOCK, types.BOND, types.ETF},
				ComplianceLevel: STRICT,
				IsActive:        true,
				CreatedAt:       time.Now(),
				UpdatedAt:       time.Now(),
			},
			{
				ID:              "prohibited_activities",
				Name:            "Prohibited Business Activities",
				Description:     "Prohibits investment in alcohol, gambling, pork, etc.",
				Category:        "prohibited_activities",
				AssetTypes:      []types.AssetType{types.STOCK, types.ETF, types.REIT},
				ComplianceLevel: STRICT,
				IsActive:        true,
				CreatedAt:       time.Now(),
				UpdatedAt:       time.Now(),
			},
			{
				ID:              "debt_ratio_check",
				Name:            "Debt Ratio Check",
				Description:     "Ensures debt-to-equity ratio is within acceptable limits",
				Category:        "debt_ratio",
				AssetTypes:      []types.AssetType{types.STOCK, types.REIT},
				ComplianceLevel: MODERATE,
				IsActive:        true,
				CreatedAt:       time.Now(),
				UpdatedAt:       time.Now(),
			},
		},
	}
}
