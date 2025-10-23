package services

import (
	"context"
	"fmt"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/db/models"
	"github.com/abdoElHodaky/tradSys/internal/trading/types"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// REITService provides REIT-specific operations
type REITService struct {
	db           *gorm.DB
	assetService *AssetService
	logger       *zap.Logger
}

// NewREITService creates a new REIT service
func NewREITService(db *gorm.DB, assetService *AssetService, logger *zap.Logger) *REITService {
	return &REITService{
		db:           db,
		assetService: assetService,
		logger:       logger,
	}
}

// REITType represents different types of REITs
type REITType string

const (
	REITTypeEquity   REITType = "equity"
	REITTypeMortgage REITType = "mortgage"
	REITTypeHybrid   REITType = "hybrid"
)

// PropertySector represents different property sectors for REITs
type PropertySector string

const (
	PropertySectorResidential  PropertySector = "residential"
	PropertySectorCommercial   PropertySector = "commercial"
	PropertySectorIndustrial   PropertySector = "industrial"
	PropertySectorRetail       PropertySector = "retail"
	PropertySectorOffice       PropertySector = "office"
	PropertySectorHealthcare   PropertySector = "healthcare"
	PropertySectorHospitality  PropertySector = "hospitality"
	PropertySectorDataCenter   PropertySector = "data_center"
	PropertySectorSelfStorage  PropertySector = "self_storage"
	PropertySectorTimberland   PropertySector = "timberland"
)

// REITMetrics represents key REIT performance metrics
type REITMetrics struct {
	Symbol              string    `json:"symbol"`
	FFO                 float64   `json:"ffo"`                   // Funds From Operations
	AFFO                float64   `json:"affo"`                  // Adjusted Funds From Operations
	NAVPerShare         float64   `json:"nav_per_share"`         // Net Asset Value per share
	DividendYield       float64   `json:"dividend_yield"`        // Current dividend yield
	PayoutRatio         float64   `json:"payout_ratio"`          // Dividend payout ratio
	DebtToEquity        float64   `json:"debt_to_equity"`        // Debt-to-equity ratio
	OccupancyRate       float64   `json:"occupancy_rate"`        // Property occupancy rate
	PriceToFFO          float64   `json:"price_to_ffo"`          // Price-to-FFO ratio
	PriceToNAV          float64   `json:"price_to_nav"`          // Price-to-NAV ratio
	TotalReturn         float64   `json:"total_return"`          // Total return including dividends
	CalculationDate     time.Time `json:"calculation_date"`
}

// REITPortfolioInfo represents REIT portfolio composition
type REITPortfolioInfo struct {
	Symbol              string                    `json:"symbol"`
	TotalProperties     int                       `json:"total_properties"`
	TotalSquareFeet     float64                   `json:"total_square_feet"`
	GeographicDiversity map[string]float64        `json:"geographic_diversity"` // State/Region -> % allocation
	PropertyTypes       map[PropertySector]float64 `json:"property_types"`       // Sector -> % allocation
	TopTenants          []TenantInfo              `json:"top_tenants"`
	AverageLeaseLength  float64                   `json:"average_lease_length"` // in years
	UpdatedAt           time.Time                 `json:"updated_at"`
}

// TenantInfo represents information about major tenants
type TenantInfo struct {
	Name            string  `json:"name"`
	PercentOfRent   float64 `json:"percent_of_rent"`
	LeaseExpiration string  `json:"lease_expiration"`
	CreditRating    string  `json:"credit_rating,omitempty"`
}

// CreateREITMetadata creates REIT-specific metadata
func (s *REITService) CreateREITMetadata(ctx context.Context, symbol string, reitType REITType, sector PropertySector, attributes map[string]interface{}) error {
	// Ensure it's a REIT asset type
	metadata := &models.AssetMetadata{
		Symbol:    symbol,
		AssetType: types.AssetTypeREIT,
		Sector:    string(sector),
		Attributes: models.AssetAttributes{
			types.AttrREITType:       string(reitType),
			types.AttrPropertySector: string(sector),
		},
		IsActive: true,
	}

	// Add additional attributes
	for key, value := range attributes {
		metadata.Attributes[key] = value
	}

	return s.assetService.CreateAssetMetadata(ctx, metadata)
}

// UpdateREITMetrics updates REIT performance metrics
func (s *REITService) UpdateREITMetrics(ctx context.Context, metrics *REITMetrics) error {
	// Validate that this is a REIT
	assetMetadata, err := s.assetService.GetAssetMetadata(ctx, metrics.Symbol)
	if err != nil {
		return fmt.Errorf("failed to get asset metadata: %w", err)
	}

	if assetMetadata.AssetType != types.AssetTypeREIT {
		return fmt.Errorf("symbol %s is not a REIT", metrics.Symbol)
	}

	// Update the asset metadata with new metrics
	if assetMetadata.Attributes == nil {
		assetMetadata.Attributes = make(models.AssetAttributes)
	}

	assetMetadata.Attributes[types.AttrFFO] = metrics.FFO
	assetMetadata.Attributes[types.AttrNAVPerShare] = metrics.NAVPerShare
	assetMetadata.Attributes[types.AttrDividendYield] = metrics.DividendYield
	assetMetadata.Attributes["affo"] = metrics.AFFO
	assetMetadata.Attributes["payout_ratio"] = metrics.PayoutRatio
	assetMetadata.Attributes["debt_to_equity"] = metrics.DebtToEquity
	assetMetadata.Attributes["occupancy_rate"] = metrics.OccupancyRate
	assetMetadata.Attributes["price_to_ffo"] = metrics.PriceToFFO
	assetMetadata.Attributes["price_to_nav"] = metrics.PriceToNAV
	assetMetadata.Attributes["total_return"] = metrics.TotalReturn
	assetMetadata.Attributes["metrics_updated_at"] = metrics.CalculationDate.Format(time.RFC3339)

	err = s.assetService.UpdateAssetMetadata(ctx, metrics.Symbol, assetMetadata)
	if err != nil {
		return fmt.Errorf("failed to update REIT metrics: %w", err)
	}

	s.logger.Info("Updated REIT metrics",
		zap.String("symbol", metrics.Symbol),
		zap.Float64("ffo", metrics.FFO),
		zap.Float64("dividend_yield", metrics.DividendYield))

	return nil
}

// GetREITMetrics retrieves REIT performance metrics
func (s *REITService) GetREITMetrics(ctx context.Context, symbol string) (*REITMetrics, error) {
	assetMetadata, err := s.assetService.GetAssetMetadata(ctx, symbol)
	if err != nil {
		return nil, fmt.Errorf("failed to get asset metadata: %w", err)
	}

	if assetMetadata.AssetType != types.AssetTypeREIT {
		return nil, fmt.Errorf("symbol %s is not a REIT", symbol)
	}

	metrics := &REITMetrics{
		Symbol: symbol,
	}

	// Extract metrics from attributes
	if ffo, ok := assetMetadata.Attributes.GetFloatAttribute(types.AttrFFO); ok {
		metrics.FFO = ffo
	}
	if nav, ok := assetMetadata.Attributes.GetFloatAttribute(types.AttrNAVPerShare); ok {
		metrics.NAVPerShare = nav
	}
	if yield, ok := assetMetadata.Attributes.GetFloatAttribute(types.AttrDividendYield); ok {
		metrics.DividendYield = yield
	}
	if affo, ok := assetMetadata.Attributes.GetFloatAttribute("affo"); ok {
		metrics.AFFO = affo
	}
	if payout, ok := assetMetadata.Attributes.GetFloatAttribute("payout_ratio"); ok {
		metrics.PayoutRatio = payout
	}
	if debt, ok := assetMetadata.Attributes.GetFloatAttribute("debt_to_equity"); ok {
		metrics.DebtToEquity = debt
	}
	if occupancy, ok := assetMetadata.Attributes.GetFloatAttribute("occupancy_rate"); ok {
		metrics.OccupancyRate = occupancy
	}
	if priceFFO, ok := assetMetadata.Attributes.GetFloatAttribute("price_to_ffo"); ok {
		metrics.PriceToFFO = priceFFO
	}
	if priceNAV, ok := assetMetadata.Attributes.GetFloatAttribute("price_to_nav"); ok {
		metrics.PriceToNAV = priceNAV
	}
	if totalReturn, ok := assetMetadata.Attributes.GetFloatAttribute("total_return"); ok {
		metrics.TotalReturn = totalReturn
	}
	if dateStr, ok := assetMetadata.Attributes.GetStringAttribute("metrics_updated_at"); ok {
		if date, err := time.Parse(time.RFC3339, dateStr); err == nil {
			metrics.CalculationDate = date
		}
	}

	return metrics, nil
}

// CalculateDividendYield calculates the current dividend yield for a REIT
func (s *REITService) CalculateDividendYield(ctx context.Context, symbol string) (float64, error) {
	// Get current price
	pricing, err := s.assetService.GetAssetPricing(ctx, symbol)
	if err != nil {
		return 0, fmt.Errorf("failed to get pricing: %w", err)
	}

	// Get recent dividends (last 4 quarters)
	dividends, err := s.assetService.GetAssetDividends(ctx, symbol, 4)
	if err != nil {
		return 0, fmt.Errorf("failed to get dividends: %w", err)
	}

	if len(dividends) == 0 {
		return 0, nil // No dividends
	}

	// Calculate annual dividend (sum of last 4 quarters)
	var annualDividend float64
	for _, dividend := range dividends {
		annualDividend += dividend.Amount
	}

	// Calculate yield
	if pricing.Price > 0 {
		return (annualDividend / pricing.Price) * 100, nil
	}

	return 0, nil
}

// GetREITsByPropertySector retrieves REITs by property sector
func (s *REITService) GetREITsByPropertySector(ctx context.Context, sector PropertySector) ([]*models.AssetMetadata, error) {
	var reits []*models.AssetMetadata
	
	err := s.db.WithContext(ctx).
		Where("asset_type = ? AND sector = ? AND is_active = ?", types.AssetTypeREIT, string(sector), true).
		Find(&reits).Error
	
	if err != nil {
		return nil, fmt.Errorf("failed to get REITs by sector: %w", err)
	}

	return reits, nil
}

// ValidateREITOrder validates a REIT order with specific rules
func (s *REITService) ValidateREITOrder(ctx context.Context, symbol string, quantity, price float64) error {
	// First, validate with general asset rules
	if err := s.assetService.ValidateOrderForAsset(ctx, symbol, types.AssetTypeREIT, quantity, price); err != nil {
		return err
	}

	// REIT-specific validations
	assetMetadata, err := s.assetService.GetAssetMetadata(ctx, symbol)
	if err != nil {
		return fmt.Errorf("failed to get REIT metadata: %w", err)
	}

	// Check if REIT is actively trading (not suspended)
	if suspended, ok := assetMetadata.Attributes.GetStringAttribute("trading_status"); ok && suspended == "suspended" {
		return fmt.Errorf("REIT %s is currently suspended from trading", symbol)
	}

	// Check minimum investment requirements for certain REITs
	if minInvestment, ok := assetMetadata.Attributes.GetFloatAttribute("min_investment"); ok {
		orderValue := quantity * price
		if orderValue < minInvestment {
			return fmt.Errorf("order value %.2f is below minimum investment %.2f for REIT %s", 
				orderValue, minInvestment, symbol)
		}
	}

	return nil
}

// GetREITDividendSchedule returns the dividend payment schedule for a REIT
func (s *REITService) GetREITDividendSchedule(ctx context.Context, symbol string) ([]models.AssetDividend, error) {
	// Get upcoming dividends (next 12 months)
	var dividends []models.AssetDividend
	
	oneYearFromNow := time.Now().AddDate(1, 0, 0)
	
	err := s.db.WithContext(ctx).
		Where("symbol = ? AND asset_type = ? AND ex_date > ? AND ex_date <= ?", 
			symbol, types.AssetTypeREIT, time.Now(), oneYearFromNow).
		Order("ex_date ASC").
		Find(&dividends).Error
	
	if err != nil {
		return nil, fmt.Errorf("failed to get REIT dividend schedule: %w", err)
	}

	return dividends, nil
}

// AnalyzeREITPerformance provides comprehensive REIT performance analysis
func (s *REITService) AnalyzeREITPerformance(ctx context.Context, symbol string) (map[string]interface{}, error) {
	// Get REIT metrics
	metrics, err := s.GetREITMetrics(ctx, symbol)
	if err != nil {
		return nil, fmt.Errorf("failed to get REIT metrics: %w", err)
	}

	// Get current pricing
	pricing, err := s.assetService.GetAssetPricing(ctx, symbol)
	if err != nil {
		return nil, fmt.Errorf("failed to get pricing: %w", err)
	}

	// Calculate dividend yield
	dividendYield, err := s.CalculateDividendYield(ctx, symbol)
	if err != nil {
		s.logger.Warn("Failed to calculate dividend yield", zap.Error(err))
		dividendYield = 0
	}

	analysis := map[string]interface{}{
		"symbol":           symbol,
		"current_price":    pricing.Price,
		"nav_per_share":    metrics.NAVPerShare,
		"ffo":              metrics.FFO,
		"affo":             metrics.AFFO,
		"dividend_yield":   dividendYield,
		"payout_ratio":     metrics.PayoutRatio,
		"debt_to_equity":   metrics.DebtToEquity,
		"occupancy_rate":   metrics.OccupancyRate,
		"price_to_ffo":     metrics.PriceToFFO,
		"price_to_nav":     metrics.PriceToNAV,
		"total_return":     metrics.TotalReturn,
		"analysis_date":    time.Now(),
	}

	// Add performance ratings
	analysis["performance_rating"] = s.calculatePerformanceRating(metrics, dividendYield)
	analysis["risk_rating"] = s.calculateRiskRating(metrics)

	return analysis, nil
}

// calculatePerformanceRating calculates a performance rating for the REIT
func (s *REITService) calculatePerformanceRating(metrics *REITMetrics, dividendYield float64) string {
	score := 0

	// FFO growth (assuming positive is good)
	if metrics.FFO > 0 {
		score++
	}

	// Dividend yield (4-8% is typically good for REITs)
	if dividendYield >= 4.0 && dividendYield <= 8.0 {
		score++
	}

	// Occupancy rate (>90% is good)
	if metrics.OccupancyRate > 90.0 {
		score++
	}

	// Debt-to-equity ratio (<1.0 is generally good)
	if metrics.DebtToEquity < 1.0 {
		score++
	}

	// Payout ratio (60-80% is sustainable)
	if metrics.PayoutRatio >= 60.0 && metrics.PayoutRatio <= 80.0 {
		score++
	}

	switch score {
	case 5:
		return "Excellent"
	case 4:
		return "Good"
	case 3:
		return "Average"
	case 2:
		return "Below Average"
	default:
		return "Poor"
	}
}

// calculateRiskRating calculates a risk rating for the REIT
func (s *REITService) calculateRiskRating(metrics *REITMetrics) string {
	riskScore := 0

	// High debt-to-equity increases risk
	if metrics.DebtToEquity > 1.5 {
		riskScore += 2
	} else if metrics.DebtToEquity > 1.0 {
		riskScore += 1
	}

	// Low occupancy increases risk
	if metrics.OccupancyRate < 85.0 {
		riskScore += 2
	} else if metrics.OccupancyRate < 90.0 {
		riskScore += 1
	}

	// High payout ratio increases risk
	if metrics.PayoutRatio > 90.0 {
		riskScore += 2
	} else if metrics.PayoutRatio > 80.0 {
		riskScore += 1
	}

	switch riskScore {
	case 0, 1:
		return "Low"
	case 2, 3:
		return "Medium"
	case 4, 5:
		return "High"
	default:
		return "Very High"
	}
}
