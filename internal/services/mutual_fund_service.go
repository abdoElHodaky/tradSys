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

// MutualFundService provides mutual fund-specific operations
type MutualFundService struct {
	db           *gorm.DB
	assetService *AssetService
	logger       *zap.Logger
}

// NewMutualFundService creates a new mutual fund service
func NewMutualFundService(db *gorm.DB, assetService *AssetService, logger *zap.Logger) *MutualFundService {
	return &MutualFundService{
		db:           db,
		assetService: assetService,
		logger:       logger,
	}
}

// FundCategory represents different categories of mutual funds
type FundCategory string

const (
	FundCategoryEquity        FundCategory = "equity"
	FundCategoryBond          FundCategory = "bond"
	FundCategoryMoney         FundCategory = "money_market"
	FundCategoryBalanced      FundCategory = "balanced"
	FundCategoryIndex         FundCategory = "index"
	FundCategoryTarget        FundCategory = "target_date"
	FundCategorySector        FundCategory = "sector"
	FundCategoryInternational FundCategory = "international"
)

// InvestmentStyle represents the investment style of the fund
type InvestmentStyle string

const (
	InvestmentStyleGrowth InvestmentStyle = "growth"
	InvestmentStyleValue  InvestmentStyle = "value"
	InvestmentStyleBlend  InvestmentStyle = "blend"
)

// MutualFundMetrics represents key mutual fund performance metrics
type MutualFundMetrics struct {
	Symbol            string    `json:"symbol"`
	NAV               float64   `json:"nav"`                // Net Asset Value
	ExpenseRatio      float64   `json:"expense_ratio"`      // Annual expense ratio
	YTDReturn         float64   `json:"ytd_return"`         // Year-to-date return
	OneYearReturn     float64   `json:"one_year_return"`    // 1-year return
	ThreeYearReturn   float64   `json:"three_year_return"`  // 3-year annualized return
	FiveYearReturn    float64   `json:"five_year_return"`   // 5-year annualized return
	TenYearReturn     float64   `json:"ten_year_return"`    // 10-year annualized return
	Alpha             float64   `json:"alpha"`              // Alpha vs benchmark
	Beta              float64   `json:"beta"`               // Beta vs benchmark
	Sharpe            float64   `json:"sharpe_ratio"`       // Sharpe ratio
	StandardDeviation float64   `json:"standard_deviation"` // Standard deviation
	TurnoverRatio     float64   `json:"turnover_ratio"`     // Portfolio turnover ratio
	TotalAssets       float64   `json:"total_assets"`       // Total fund assets (AUM)
	CalculationDate   time.Time `json:"calculation_date"`
}

// FundHolding represents a holding in the mutual fund portfolio
type FundHolding struct {
	Symbol      string  `json:"symbol"`
	Name        string  `json:"name"`
	Percentage  float64 `json:"percentage"`
	Shares      float64 `json:"shares"`
	MarketValue float64 `json:"market_value"`
	Sector      string  `json:"sector,omitempty"`
}

// MutualFundPortfolio represents the portfolio composition of a mutual fund
type MutualFundPortfolio struct {
	Symbol               string             `json:"symbol"`
	TopHoldings          []FundHolding      `json:"top_holdings"`
	SectorAllocation     map[string]float64 `json:"sector_allocation"`
	AssetAllocation      map[string]float64 `json:"asset_allocation"`
	GeographicAllocation map[string]float64 `json:"geographic_allocation"`
	MarketCapAllocation  map[string]float64 `json:"market_cap_allocation"`
	TotalHoldings        int                `json:"total_holdings"`
	UpdatedAt            time.Time          `json:"updated_at"`
}

// CreateMutualFundMetadata creates mutual fund-specific metadata
func (s *MutualFundService) CreateMutualFundMetadata(ctx context.Context, symbol, fundFamily string, category FundCategory, style InvestmentStyle, attributes map[string]interface{}) error {
	metadata := &models.AssetMetadata{
		Symbol:    symbol,
		AssetType: types.AssetTypeMutualFund,
		Sector:    string(category),
		Attributes: models.AssetAttributes{
			types.AttrFundFamily: fundFamily,
			types.AttrAssetClass: string(style),
			"fund_category":      string(category),
		},
		IsActive: true,
	}

	// Add additional attributes
	for key, value := range attributes {
		metadata.Attributes[key] = value
	}

	return s.assetService.CreateAssetMetadata(ctx, metadata)
}

// UpdateMutualFundMetrics updates mutual fund performance metrics
func (s *MutualFundService) UpdateMutualFundMetrics(ctx context.Context, metrics *MutualFundMetrics) error {
	// Validate that this is a mutual fund
	assetMetadata, err := s.assetService.GetAssetMetadata(ctx, metrics.Symbol)
	if err != nil {
		return fmt.Errorf("failed to get asset metadata: %w", err)
	}

	if assetMetadata.AssetType != types.AssetTypeMutualFund {
		return fmt.Errorf("symbol %s is not a mutual fund", metrics.Symbol)
	}

	// Update the asset metadata with new metrics
	if assetMetadata.Attributes == nil {
		assetMetadata.Attributes = make(models.AssetAttributes)
	}

	assetMetadata.Attributes["nav"] = metrics.NAV
	assetMetadata.Attributes[types.AttrExpenseRatio] = metrics.ExpenseRatio
	assetMetadata.Attributes["ytd_return"] = metrics.YTDReturn
	assetMetadata.Attributes["one_year_return"] = metrics.OneYearReturn
	assetMetadata.Attributes["three_year_return"] = metrics.ThreeYearReturn
	assetMetadata.Attributes["five_year_return"] = metrics.FiveYearReturn
	assetMetadata.Attributes["ten_year_return"] = metrics.TenYearReturn
	assetMetadata.Attributes["alpha"] = metrics.Alpha
	assetMetadata.Attributes["beta"] = metrics.Beta
	assetMetadata.Attributes["sharpe_ratio"] = metrics.Sharpe
	assetMetadata.Attributes["standard_deviation"] = metrics.StandardDeviation
	assetMetadata.Attributes["turnover_ratio"] = metrics.TurnoverRatio
	assetMetadata.Attributes["total_assets"] = metrics.TotalAssets
	assetMetadata.Attributes["metrics_updated_at"] = metrics.CalculationDate.Format(time.RFC3339)

	err = s.assetService.UpdateAssetMetadata(ctx, metrics.Symbol, assetMetadata)
	if err != nil {
		return fmt.Errorf("failed to update mutual fund metrics: %w", err)
	}

	// Also update pricing with NAV
	pricing := &models.AssetPricing{
		Symbol:    metrics.Symbol,
		AssetType: types.AssetTypeMutualFund,
		Price:     metrics.NAV,
		Source:    "NAV",
	}

	if err := s.assetService.UpdateAssetPricing(ctx, pricing); err != nil {
		s.logger.Warn("Failed to update NAV pricing", zap.Error(err))
	}

	s.logger.Info("Updated mutual fund metrics",
		zap.String("symbol", metrics.Symbol),
		zap.Float64("nav", metrics.NAV),
		zap.Float64("expense_ratio", metrics.ExpenseRatio))

	return nil
}

// GetMutualFundMetrics retrieves mutual fund performance metrics
func (s *MutualFundService) GetMutualFundMetrics(ctx context.Context, symbol string) (*MutualFundMetrics, error) {
	assetMetadata, err := s.assetService.GetAssetMetadata(ctx, symbol)
	if err != nil {
		return nil, fmt.Errorf("failed to get asset metadata: %w", err)
	}

	if assetMetadata.AssetType != types.AssetTypeMutualFund {
		return nil, fmt.Errorf("symbol %s is not a mutual fund", symbol)
	}

	metrics := &MutualFundMetrics{
		Symbol: symbol,
	}

	// Extract metrics from attributes
	if nav, ok := assetMetadata.Attributes.GetFloatAttribute("nav"); ok {
		metrics.NAV = nav
	}
	if expense, ok := assetMetadata.Attributes.GetFloatAttribute(types.AttrExpenseRatio); ok {
		metrics.ExpenseRatio = expense
	}
	if ytd, ok := assetMetadata.Attributes.GetFloatAttribute("ytd_return"); ok {
		metrics.YTDReturn = ytd
	}
	if oneYear, ok := assetMetadata.Attributes.GetFloatAttribute("one_year_return"); ok {
		metrics.OneYearReturn = oneYear
	}
	if threeYear, ok := assetMetadata.Attributes.GetFloatAttribute("three_year_return"); ok {
		metrics.ThreeYearReturn = threeYear
	}
	if fiveYear, ok := assetMetadata.Attributes.GetFloatAttribute("five_year_return"); ok {
		metrics.FiveYearReturn = fiveYear
	}
	if tenYear, ok := assetMetadata.Attributes.GetFloatAttribute("ten_year_return"); ok {
		metrics.TenYearReturn = tenYear
	}
	if alpha, ok := assetMetadata.Attributes.GetFloatAttribute("alpha"); ok {
		metrics.Alpha = alpha
	}
	if beta, ok := assetMetadata.Attributes.GetFloatAttribute("beta"); ok {
		metrics.Beta = beta
	}
	if sharpe, ok := assetMetadata.Attributes.GetFloatAttribute("sharpe_ratio"); ok {
		metrics.Sharpe = sharpe
	}
	if stdDev, ok := assetMetadata.Attributes.GetFloatAttribute("standard_deviation"); ok {
		metrics.StandardDeviation = stdDev
	}
	if turnover, ok := assetMetadata.Attributes.GetFloatAttribute("turnover_ratio"); ok {
		metrics.TurnoverRatio = turnover
	}
	if assets, ok := assetMetadata.Attributes.GetFloatAttribute("total_assets"); ok {
		metrics.TotalAssets = assets
	}
	if dateStr, ok := assetMetadata.Attributes.GetStringAttribute("metrics_updated_at"); ok {
		if date, err := time.Parse(time.RFC3339, dateStr); err == nil {
			metrics.CalculationDate = date
		}
	}

	return metrics, nil
}

// GetMutualFundsByFamily retrieves all mutual funds from a specific fund family
func (s *MutualFundService) GetMutualFundsByFamily(ctx context.Context, fundFamily string) ([]*models.AssetMetadata, error) {
	var funds []*models.AssetMetadata

	err := s.db.WithContext(ctx).
		Where("asset_type = ? AND is_active = ? AND JSON_EXTRACT(attributes, '$.fund_family') = ?",
			types.AssetTypeMutualFund, true, fundFamily).
		Find(&funds).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get mutual funds by family: %w", err)
	}

	return funds, nil
}

// GetMutualFundsByCategory retrieves mutual funds by category
func (s *MutualFundService) GetMutualFundsByCategory(ctx context.Context, category FundCategory) ([]*models.AssetMetadata, error) {
	var funds []*models.AssetMetadata

	err := s.db.WithContext(ctx).
		Where("asset_type = ? AND sector = ? AND is_active = ?",
			types.AssetTypeMutualFund, string(category), true).
		Find(&funds).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get mutual funds by category: %w", err)
	}

	return funds, nil
}

// ValidateMutualFundOrder validates a mutual fund order with specific rules
func (s *MutualFundService) ValidateMutualFundOrder(ctx context.Context, symbol string, quantity, price float64) error {
	// First, validate with general asset rules
	if err := s.assetService.ValidateOrderForAsset(ctx, symbol, types.AssetTypeMutualFund, quantity, price); err != nil {
		return err
	}

	// Mutual fund-specific validations
	assetMetadata, err := s.assetService.GetAssetMetadata(ctx, symbol)
	if err != nil {
		return fmt.Errorf("failed to get mutual fund metadata: %w", err)
	}

	// Check minimum investment requirement
	if minInvestment, ok := assetMetadata.Attributes.GetFloatAttribute(types.AttrMinInvestment); ok {
		orderValue := quantity * price
		if orderValue < minInvestment {
			return fmt.Errorf("order value %.2f is below minimum investment %.2f for fund %s",
				orderValue, minInvestment, symbol)
		}
	}

	// Check if fund is closed to new investors
	if closed, ok := assetMetadata.Attributes.GetStringAttribute("fund_status"); ok && closed == "closed" {
		return fmt.Errorf("mutual fund %s is closed to new investors", symbol)
	}

	// Mutual funds typically trade at NAV at end of day
	currentTime := time.Now()
	if currentTime.Hour() < 16 { // Before 4 PM EST
		s.logger.Info("Mutual fund order will be executed at next NAV calculation",
			zap.String("symbol", symbol))
	}

	return nil
}

// CalculateExpenseImpact calculates the impact of expense ratio on returns
func (s *MutualFundService) CalculateExpenseImpact(ctx context.Context, symbol string, investmentAmount float64, years int) (map[string]float64, error) {
	metrics, err := s.GetMutualFundMetrics(ctx, symbol)
	if err != nil {
		return nil, fmt.Errorf("failed to get mutual fund metrics: %w", err)
	}

	if metrics.ExpenseRatio == 0 {
		return nil, fmt.Errorf("expense ratio not available for fund %s", symbol)
	}

	// Calculate compound impact of expense ratio
	annualExpense := investmentAmount * (metrics.ExpenseRatio / 100)
	totalExpenseOverTime := annualExpense * float64(years)

	// Assuming 7% average annual return for calculation
	assumedReturn := 0.07
	futureValueWithoutExpenses := investmentAmount * (1 + assumedReturn)
	for i := 1; i < years; i++ {
		futureValueWithoutExpenses *= (1 + assumedReturn)
	}

	futureValueWithExpenses := investmentAmount * (1 + assumedReturn - metrics.ExpenseRatio/100)
	for i := 1; i < years; i++ {
		futureValueWithExpenses *= (1 + assumedReturn - metrics.ExpenseRatio/100)
	}

	impact := map[string]float64{
		"initial_investment":        investmentAmount,
		"annual_expense_ratio":      metrics.ExpenseRatio,
		"annual_expense_amount":     annualExpense,
		"total_expenses_over_time":  totalExpenseOverTime,
		"future_value_without_fees": futureValueWithoutExpenses,
		"future_value_with_fees":    futureValueWithExpenses,
		"total_fee_impact":          futureValueWithoutExpenses - futureValueWithExpenses,
		"years":                     float64(years),
	}

	return impact, nil
}

// CompareMutualFunds compares multiple mutual funds across key metrics
func (s *MutualFundService) CompareMutualFunds(ctx context.Context, symbols []string) (map[string]interface{}, error) {
	if len(symbols) == 0 {
		return nil, fmt.Errorf("no symbols provided for comparison")
	}

	comparison := map[string]interface{}{
		"funds":           make(map[string]*MutualFundMetrics),
		"comparison_date": time.Now(),
	}

	funds := make(map[string]*MutualFundMetrics)

	for _, symbol := range symbols {
		metrics, err := s.GetMutualFundMetrics(ctx, symbol)
		if err != nil {
			s.logger.Warn("Failed to get metrics for fund", zap.String("symbol", symbol), zap.Error(err))
			continue
		}
		funds[symbol] = metrics
	}

	comparison["funds"] = funds

	// Add comparative analysis
	if len(funds) > 1 {
		comparison["analysis"] = s.generateComparativeAnalysis(funds)
	}

	return comparison, nil
}

// generateComparativeAnalysis generates comparative analysis between funds
func (s *MutualFundService) generateComparativeAnalysis(funds map[string]*MutualFundMetrics) map[string]interface{} {
	analysis := make(map[string]interface{})

	// Find best performers in different categories
	var lowestExpense, highestOneYear, highestFiveYear, highestSharpe string
	var lowestExpenseRatio, highestOneYearReturn, highestFiveYearReturn, highestSharpeRatio float64

	first := true
	for symbol, metrics := range funds {
		if first {
			lowestExpense = symbol
			lowestExpenseRatio = metrics.ExpenseRatio
			highestOneYear = symbol
			highestOneYearReturn = metrics.OneYearReturn
			highestFiveYear = symbol
			highestFiveYearReturn = metrics.FiveYearReturn
			highestSharpe = symbol
			highestSharpeRatio = metrics.Sharpe
			first = false
			continue
		}

		if metrics.ExpenseRatio < lowestExpenseRatio && metrics.ExpenseRatio > 0 {
			lowestExpense = symbol
			lowestExpenseRatio = metrics.ExpenseRatio
		}
		if metrics.OneYearReturn > highestOneYearReturn {
			highestOneYear = symbol
			highestOneYearReturn = metrics.OneYearReturn
		}
		if metrics.FiveYearReturn > highestFiveYearReturn {
			highestFiveYear = symbol
			highestFiveYearReturn = metrics.FiveYearReturn
		}
		if metrics.Sharpe > highestSharpeRatio {
			highestSharpe = symbol
			highestSharpeRatio = metrics.Sharpe
		}
	}

	analysis["lowest_expense_ratio"] = map[string]interface{}{
		"symbol": lowestExpense,
		"value":  lowestExpenseRatio,
	}
	analysis["highest_one_year_return"] = map[string]interface{}{
		"symbol": highestOneYear,
		"value":  highestOneYearReturn,
	}
	analysis["highest_five_year_return"] = map[string]interface{}{
		"symbol": highestFiveYear,
		"value":  highestFiveYearReturn,
	}
	analysis["highest_sharpe_ratio"] = map[string]interface{}{
		"symbol": highestSharpe,
		"value":  highestSharpeRatio,
	}

	return analysis
}

// GetFundPerformanceRating calculates a performance rating for a mutual fund
func (s *MutualFundService) GetFundPerformanceRating(ctx context.Context, symbol string) (string, error) {
	metrics, err := s.GetMutualFundMetrics(ctx, symbol)
	if err != nil {
		return "", fmt.Errorf("failed to get fund metrics: %w", err)
	}

	score := 0

	// Low expense ratio (< 1% is good)
	if metrics.ExpenseRatio < 1.0 && metrics.ExpenseRatio > 0 {
		score++
	}

	// Positive 1-year return
	if metrics.OneYearReturn > 0 {
		score++
	}

	// Positive 3-year return
	if metrics.ThreeYearReturn > 0 {
		score++
	}

	// Good Sharpe ratio (> 1.0 is good)
	if metrics.Sharpe > 1.0 {
		score++
	}

	// Reasonable turnover ratio (< 100% is generally good)
	if metrics.TurnoverRatio < 100.0 && metrics.TurnoverRatio > 0 {
		score++
	}

	switch score {
	case 5:
		return "Excellent", nil
	case 4:
		return "Good", nil
	case 3:
		return "Average", nil
	case 2:
		return "Below Average", nil
	default:
		return "Poor", nil
	}
}
