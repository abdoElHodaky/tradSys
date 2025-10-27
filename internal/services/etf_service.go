package services

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/db/models"
	"github.com/abdoElHodaky/tradSys/internal/trading/types"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ETFService handles ETF-specific operations and calculations
type ETFService struct {
	db           *gorm.DB
	assetService *AssetService
	logger       *zap.Logger
}

// NewETFService creates a new ETF service instance
func NewETFService(db *gorm.DB, assetService *AssetService, logger *zap.Logger) *ETFService {
	return &ETFService{
		db:           db,
		assetService: assetService,
		logger:       logger,
	}
}

// ETFMetrics represents comprehensive ETF performance and operational metrics
type ETFMetrics struct {
	Symbol             string               `json:"symbol"`
	NAV                float64              `json:"nav"`
	MarketPrice        float64              `json:"market_price"`
	Premium            float64              `json:"premium"`
	TrackingError      float64              `json:"tracking_error"`
	ExpenseRatio       float64              `json:"expense_ratio"`
	AUM                float64              `json:"aum"`
	DividendYield      float64              `json:"dividend_yield"`
	BenchmarkIndex     string               `json:"benchmark_index"`
	CreationUnitSize   int                  `json:"creation_unit_size"`
	LastCreationDate   time.Time            `json:"last_creation_date"`
	LastRedemptionDate time.Time            `json:"last_redemption_date"`
	Liquidity          LiquidityMetrics     `json:"liquidity"`
	Holdings           []ETFHolding         `json:"holdings"`
	PerformanceMetrics ETFPerformance       `json:"performance_metrics"`
	RiskMetrics        ETFRiskMetrics       `json:"risk_metrics"`
	TaxEfficiency      TaxEfficiencyMetrics `json:"tax_efficiency"`
}

// LiquidityMetrics represents ETF liquidity characteristics
type LiquidityMetrics struct {
	BidAskSpread        float64 `json:"bid_ask_spread"`
	AverageVolume       int64   `json:"average_volume"`
	MedianVolume        int64   `json:"median_volume"`
	VolumeWeightedPrice float64 `json:"volume_weighted_price"`
	LiquidityScore      float64 `json:"liquidity_score"`
	MarketImpact        float64 `json:"market_impact"`
}

// ETFHolding represents individual holdings within an ETF
type ETFHolding struct {
	Symbol      string  `json:"symbol"`
	Name        string  `json:"name"`
	Weight      float64 `json:"weight"`
	Shares      int64   `json:"shares"`
	MarketValue float64 `json:"market_value"`
	Sector      string  `json:"sector"`
	Country     string  `json:"country"`
}

// ETFPerformance represents ETF performance metrics
type ETFPerformance struct {
	OneDay         float64 `json:"one_day"`
	OneWeek        float64 `json:"one_week"`
	OneMonth       float64 `json:"one_month"`
	ThreeMonth     float64 `json:"three_month"`
	SixMonth       float64 `json:"six_month"`
	YTD            float64 `json:"ytd"`
	OneYear        float64 `json:"one_year"`
	ThreeYear      float64 `json:"three_year"`
	FiveYear       float64 `json:"five_year"`
	TenYear        float64 `json:"ten_year"`
	SinceInception float64 `json:"since_inception"`
}

// ETFRiskMetrics represents ETF risk characteristics
type ETFRiskMetrics struct {
	Beta               float64 `json:"beta"`
	Alpha              float64 `json:"alpha"`
	Volatility         float64 `json:"volatility"`
	SharpeRatio        float64 `json:"sharpe_ratio"`
	MaxDrawdown        float64 `json:"max_drawdown"`
	VaR95              float64 `json:"var_95"`
	VaR99              float64 `json:"var_99"`
	CorrelationToIndex float64 `json:"correlation_to_index"`
}

// TaxEfficiencyMetrics represents ETF tax characteristics
type TaxEfficiencyMetrics struct {
	TaxEfficiencyRatio       float64   `json:"tax_efficiency_ratio"`
	CapitalGainsDistribution float64   `json:"capital_gains_distribution"`
	DividendDistribution     float64   `json:"dividend_distribution"`
	LastDistributionDate     time.Time `json:"last_distribution_date"`
	TurnoverRatio            float64   `json:"turnover_ratio"`
}

// CreationRedemptionOperation represents ETF creation/redemption activity
type CreationRedemptionOperation struct {
	ID                    string    `json:"id"`
	Symbol                string    `json:"symbol"`
	OperationType         string    `json:"operation_type"` // "creation" or "redemption"
	Units                 int       `json:"units"`
	SharesPerUnit         int       `json:"shares_per_unit"`
	TotalShares           int       `json:"total_shares"`
	NAVPerShare           float64   `json:"nav_per_share"`
	TotalValue            float64   `json:"total_value"`
	AuthorizedParticipant string    `json:"authorized_participant"`
	Timestamp             time.Time `json:"timestamp"`
	Status                string    `json:"status"`
}

// CreateETF creates a new ETF with initial metadata
func (s *ETFService) CreateETF(symbol, benchmarkIndex string, creationUnitSize int, expenseRatio float64) error {
	s.logger.Info("Creating new ETF", zap.String("symbol", symbol))

	// Create asset metadata
	assetAttributes := models.AssetAttributes{
		"benchmark_index":    benchmarkIndex,
		"creation_unit_size": creationUnitSize,
		"expense_ratio":      expenseRatio,
		"etf_type":           "equity", // default
		"inception_date":     time.Now(),
		"fund_family":        "",
		"investment_style":   "",
		"geographic_focus":   "",
	}

	asset := &models.AssetMetadata{
		Symbol:     symbol,
		AssetType:  types.AssetTypeETF,
		Sector:     "financial",
		Attributes: assetAttributes,
		IsActive:   true,
	}

	if err := s.db.Create(asset).Error; err != nil {
		return fmt.Errorf("failed to create ETF asset: %w", err)
	}

	// Create default configuration
	config := &models.AssetConfiguration{
		AssetType:      types.AssetTypeETF,
		TradingEnabled: true,
		MinOrderSize:   1.0,
		MaxOrderSize:   1000000.0,
		RiskMultiplier: 1.0,
		SettlementDays: 2,
		TradingHours:   "09:30-16:00",
	}

	if err := s.db.Create(config).Error; err != nil {
		s.logger.Warn("Failed to create ETF configuration", zap.Error(err))
	}

	s.logger.Info("ETF created successfully", zap.String("symbol", symbol))
	return nil
}

// GetETFMetrics retrieves comprehensive ETF metrics
func (s *ETFService) GetETFMetrics(symbol string) (*ETFMetrics, error) {
	s.logger.Debug("Retrieving ETF metrics", zap.String("symbol", symbol))

	// Get asset metadata
	asset, err := s.assetService.GetAssetBySymbol(context.Background(), symbol)
	if err != nil {
		return nil, fmt.Errorf("failed to get ETF asset: %w", err)
	}

	if asset.AssetType != types.AssetTypeETF {
		return nil, fmt.Errorf("asset %s is not an ETF", symbol)
	}

	// Parse attributes
	attributes := asset.Attributes

	// Get current pricing
	pricing, err := s.assetService.GetCurrentPricing(context.Background(), symbol)
	if err != nil {
		s.logger.Warn("Failed to get current pricing", zap.String("symbol", symbol), zap.Error(err))
		pricing = &models.AssetPricing{Price: 0.0}
	}

	// Calculate metrics
	metrics := &ETFMetrics{
		Symbol:           symbol,
		MarketPrice:      pricing.Price,
		BenchmarkIndex:   s.getStringAttribute(attributes, "benchmark_index"),
		CreationUnitSize: s.getIntAttribute(attributes, "creation_unit_size"),
		ExpenseRatio:     s.getFloatAttribute(attributes, "expense_ratio"),
	}

	// Calculate derived metrics
	s.calculateETFMetrics(metrics, attributes)
	s.calculateLiquidityMetrics(metrics, symbol)
	s.calculatePerformanceMetrics(metrics, symbol)
	s.calculateRiskMetrics(metrics, symbol)
	s.calculateTaxEfficiency(metrics, symbol)

	return metrics, nil
}

// UpdateETFMetrics updates ETF-specific metrics
func (s *ETFService) UpdateETFMetrics(symbol string, nav, trackingError, aum, dividendYield float64) error {
	s.logger.Info("Updating ETF metrics", zap.String("symbol", symbol))

	asset, err := s.assetService.GetAssetBySymbol(context.Background(), symbol)
	if err != nil {
		return fmt.Errorf("failed to get ETF asset: %w", err)
	}

	// Parse existing attributes
	attributes := map[string]interface{}(asset.Attributes)

	// Update metrics
	attributes["nav"] = nav
	attributes["tracking_error"] = trackingError
	attributes["aum"] = aum
	attributes["dividend_yield"] = dividendYield
	attributes["last_updated"] = time.Now()

	// Calculate premium/discount
	pricing, err := s.assetService.GetCurrentPricing(context.Background(), symbol)
	if err == nil && pricing.Price > 0 && nav > 0 {
		premium := ((pricing.Price - nav) / nav) * 100
		attributes["premium"] = premium
	}

	// Update asset
	asset.Attributes = attributes
	asset.UpdatedAt = time.Now()

	if err := s.db.Save(asset).Error; err != nil {
		return fmt.Errorf("failed to update ETF metrics: %w", err)
	}

	s.logger.Info("ETF metrics updated successfully", zap.String("symbol", symbol))
	return nil
}

// GetTrackingError calculates and returns the tracking error for an ETF
func (s *ETFService) GetTrackingError(symbol string, days int) (float64, error) {
	s.logger.Debug("Calculating tracking error", zap.String("symbol", symbol), zap.Int("days", days))

	// Get ETF pricing history
	etfPrices, err := s.getETFPriceHistory(symbol, days)
	if err != nil {
		return 0, fmt.Errorf("failed to get ETF price history: %w", err)
	}

	// Get benchmark pricing history
	asset, err := s.assetService.GetAssetBySymbol(context.Background(), symbol)
	if err != nil {
		return 0, fmt.Errorf("failed to get ETF asset: %w", err)
	}

	attributes := map[string]interface{}(asset.Attributes)

	benchmarkIndex := s.getStringAttribute(attributes, "benchmark_index")
	if benchmarkIndex == "" {
		return 0, fmt.Errorf("no benchmark index specified for ETF %s", symbol)
	}

	benchmarkPrices, err := s.getBenchmarkPriceHistory(benchmarkIndex, days)
	if err != nil {
		return 0, fmt.Errorf("failed to get benchmark price history: %w", err)
	}

	// Calculate tracking error
	trackingError := s.calculateTrackingError(etfPrices, benchmarkPrices)

	s.logger.Debug("Tracking error calculated",
		zap.String("symbol", symbol),
		zap.Float64("tracking_error", trackingError))

	return trackingError, nil
}

// ProcessCreationRedemption handles ETF creation/redemption operations
func (s *ETFService) ProcessCreationRedemption(operation *CreationRedemptionOperation) error {
	s.logger.Info("Processing creation/redemption operation",
		zap.String("symbol", operation.Symbol),
		zap.String("type", operation.OperationType),
		zap.Int("units", operation.Units))

	// Validate operation
	if err := s.validateCreationRedemption(operation); err != nil {
		return fmt.Errorf("invalid creation/redemption operation: %w", err)
	}

	// Get ETF metadata
	asset, err := s.assetService.GetAssetBySymbol(context.Background(), operation.Symbol)
	if err != nil {
		return fmt.Errorf("failed to get ETF asset: %w", err)
	}

	attributes := map[string]interface{}(asset.Attributes)

	// Update shares outstanding
	currentShares := s.getInt64Attribute(attributes, "shares_outstanding")

	if operation.OperationType == "creation" {
		attributes["shares_outstanding"] = currentShares + int64(operation.TotalShares)
		attributes["last_creation_date"] = operation.Timestamp
	} else {
		attributes["shares_outstanding"] = currentShares - int64(operation.TotalShares)
		attributes["last_redemption_date"] = operation.Timestamp
	}

	// Update AUM
	currentAUM := s.getFloatAttribute(attributes, "aum")
	if operation.OperationType == "creation" {
		attributes["aum"] = currentAUM + operation.TotalValue
	} else {
		attributes["aum"] = currentAUM - operation.TotalValue
	}

	// Save updated attributes
	asset.Attributes = attributes
	asset.UpdatedAt = time.Now()

	if err := s.db.Save(asset).Error; err != nil {
		return fmt.Errorf("failed to update ETF after creation/redemption: %w", err)
	}

	// Log operation
	s.logger.Info("Creation/redemption operation processed successfully",
		zap.String("symbol", operation.Symbol),
		zap.String("type", operation.OperationType),
		zap.Int("units", operation.Units),
		zap.Float64("total_value", operation.TotalValue))

	return nil
}

// GetETFHoldings retrieves the current holdings composition of an ETF
func (s *ETFService) GetETFHoldings(symbol string) ([]ETFHolding, error) {
	s.logger.Debug("Retrieving ETF holdings", zap.String("symbol", symbol))

	// In a real implementation, this would fetch from a holdings database
	// For now, return mock data based on ETF type
	asset, err := s.assetService.GetAssetBySymbol(context.Background(), symbol)
	if err != nil {
		return nil, fmt.Errorf("failed to get ETF asset: %w", err)
	}

	attributes := map[string]interface{}(asset.Attributes)

	// Generate sample holdings based on ETF characteristics
	holdings := s.generateSampleHoldings(symbol, attributes)

	s.logger.Debug("Retrieved ETF holdings",
		zap.String("symbol", symbol),
		zap.Int("holdings_count", len(holdings)))

	return holdings, nil
}

// ValidateETFOrder validates an ETF order against ETF-specific rules
func (s *ETFService) ValidateETFOrder(symbol string, quantity float64, price float64) error {
	s.logger.Debug("Validating ETF order",
		zap.String("symbol", symbol),
		zap.Float64("quantity", quantity),
		zap.Float64("price", price))

	// Get ETF configuration
	config, err := s.assetService.GetAssetConfiguration(context.Background(), types.AssetTypeETF)
	if err != nil {
		return fmt.Errorf("failed to get ETF configuration: %w", err)
	}

	// Validate order size
	if quantity < config.MinOrderSize {
		return fmt.Errorf("order quantity %.2f is below minimum %.2f", quantity, config.MinOrderSize)
	}

	if quantity > config.MaxOrderSize {
		return fmt.Errorf("order quantity %.2f exceeds maximum %.2f", quantity, config.MaxOrderSize)
	}

	// Get ETF metrics for additional validation
	metrics, err := s.GetETFMetrics(symbol)
	if err != nil {
		s.logger.Warn("Failed to get ETF metrics for validation", zap.Error(err))
		return nil // Don't fail order if metrics unavailable
	}

	// Validate against NAV (warn if significant premium/discount)
	if metrics.NAV > 0 && price > 0 {
		premium := ((price - metrics.NAV) / metrics.NAV) * 100
		if math.Abs(premium) > 5.0 { // 5% threshold
			s.logger.Warn("ETF order has significant premium/discount",
				zap.String("symbol", symbol),
				zap.Float64("premium", premium),
				zap.Float64("nav", metrics.NAV),
				zap.Float64("price", price))
		}
	}

	s.logger.Debug("ETF order validation passed", zap.String("symbol", symbol))
	return nil
}

// Helper methods

func (s *ETFService) calculateETFMetrics(metrics *ETFMetrics, attributes map[string]interface{}) {
	metrics.NAV = s.getFloatAttribute(attributes, "nav")
	metrics.AUM = s.getFloatAttribute(attributes, "aum")
	metrics.DividendYield = s.getFloatAttribute(attributes, "dividend_yield")
	metrics.TrackingError = s.getFloatAttribute(attributes, "tracking_error")

	// Calculate premium/discount
	if metrics.NAV > 0 && metrics.MarketPrice > 0 {
		metrics.Premium = ((metrics.MarketPrice - metrics.NAV) / metrics.NAV) * 100
	}

	// Get dates
	if creationDate, ok := attributes["last_creation_date"].(string); ok {
		if parsed, err := time.Parse(time.RFC3339, creationDate); err == nil {
			metrics.LastCreationDate = parsed
		}
	}

	if redemptionDate, ok := attributes["last_redemption_date"].(string); ok {
		if parsed, err := time.Parse(time.RFC3339, redemptionDate); err == nil {
			metrics.LastRedemptionDate = parsed
		}
	}
}

func (s *ETFService) calculateLiquidityMetrics(metrics *ETFMetrics, symbol string) {
	// In a real implementation, this would calculate from market data
	metrics.Liquidity = LiquidityMetrics{
		BidAskSpread:        0.01, // 1 cent
		AverageVolume:       1000000,
		MedianVolume:        800000,
		VolumeWeightedPrice: metrics.MarketPrice,
		LiquidityScore:      8.5,  // Out of 10
		MarketImpact:        0.05, // 5 basis points
	}
}

func (s *ETFService) calculatePerformanceMetrics(metrics *ETFMetrics, symbol string) {
	// In a real implementation, this would calculate from historical data
	metrics.PerformanceMetrics = ETFPerformance{
		OneDay:         0.12,
		OneWeek:        0.85,
		OneMonth:       2.34,
		ThreeMonth:     5.67,
		SixMonth:       8.91,
		YTD:            12.45,
		OneYear:        15.67,
		ThreeYear:      8.23,
		FiveYear:       9.87,
		TenYear:        11.23,
		SinceInception: 9.45,
	}
}

func (s *ETFService) calculateRiskMetrics(metrics *ETFMetrics, symbol string) {
	// In a real implementation, this would calculate from historical data
	metrics.RiskMetrics = ETFRiskMetrics{
		Beta:               1.02,
		Alpha:              0.15,
		Volatility:         16.5,
		SharpeRatio:        0.95,
		MaxDrawdown:        -18.7,
		VaR95:              -2.1,
		VaR99:              -3.8,
		CorrelationToIndex: 0.98,
	}
}

func (s *ETFService) calculateTaxEfficiency(metrics *ETFMetrics, symbol string) {
	// In a real implementation, this would calculate from distribution history
	metrics.TaxEfficiency = TaxEfficiencyMetrics{
		TaxEfficiencyRatio:       0.92,
		CapitalGainsDistribution: 0.15,
		DividendDistribution:     2.34,
		LastDistributionDate:     time.Now().AddDate(0, -3, 0),
		TurnoverRatio:            25.5,
	}
}

func (s *ETFService) getETFPriceHistory(symbol string, days int) ([]float64, error) {
	// In a real implementation, this would query historical pricing data
	// For now, return mock data
	prices := make([]float64, days)
	basePrice := 100.0
	for i := 0; i < days; i++ {
		// Simulate price movement
		prices[i] = basePrice * (1.0 + (float64(i%10)-5)/1000)
	}
	return prices, nil
}

func (s *ETFService) getBenchmarkPriceHistory(benchmark string, days int) ([]float64, error) {
	// In a real implementation, this would query benchmark index data
	// For now, return mock data
	prices := make([]float64, days)
	basePrice := 4500.0 // S&P 500 level
	for i := 0; i < days; i++ {
		// Simulate benchmark movement
		prices[i] = basePrice * (1.0 + (float64(i%8)-4)/1000)
	}
	return prices, nil
}

func (s *ETFService) calculateTrackingError(etfPrices, benchmarkPrices []float64) float64 {
	if len(etfPrices) != len(benchmarkPrices) || len(etfPrices) < 2 {
		return 0.0
	}

	// Calculate daily returns
	etfReturns := make([]float64, len(etfPrices)-1)
	benchmarkReturns := make([]float64, len(benchmarkPrices)-1)

	for i := 1; i < len(etfPrices); i++ {
		etfReturns[i-1] = (etfPrices[i] - etfPrices[i-1]) / etfPrices[i-1]
		benchmarkReturns[i-1] = (benchmarkPrices[i] - benchmarkPrices[i-1]) / benchmarkPrices[i-1]
	}

	// Calculate tracking differences
	differences := make([]float64, len(etfReturns))
	for i := 0; i < len(etfReturns); i++ {
		differences[i] = etfReturns[i] - benchmarkReturns[i]
	}

	// Calculate standard deviation of differences
	mean := 0.0
	for _, diff := range differences {
		mean += diff
	}
	mean /= float64(len(differences))

	variance := 0.0
	for _, diff := range differences {
		variance += math.Pow(diff-mean, 2)
	}
	variance /= float64(len(differences) - 1)

	// Annualize tracking error (assuming daily data)
	trackingError := math.Sqrt(variance) * math.Sqrt(252) * 100 // Convert to percentage

	return trackingError
}

func (s *ETFService) validateCreationRedemption(operation *CreationRedemptionOperation) error {
	if operation.Symbol == "" {
		return fmt.Errorf("symbol is required")
	}

	if operation.OperationType != "creation" && operation.OperationType != "redemption" {
		return fmt.Errorf("operation type must be 'creation' or 'redemption'")
	}

	if operation.Units <= 0 {
		return fmt.Errorf("units must be positive")
	}

	if operation.SharesPerUnit <= 0 {
		return fmt.Errorf("shares per unit must be positive")
	}

	if operation.AuthorizedParticipant == "" {
		return fmt.Errorf("authorized participant is required")
	}

	return nil
}

func (s *ETFService) generateSampleHoldings(symbol string, attributes map[string]interface{}) []ETFHolding {
	// Generate sample holdings based on ETF type
	holdings := []ETFHolding{
		{
			Symbol:      "AAPL",
			Name:        "Apple Inc.",
			Weight:      7.2,
			Shares:      150000,
			MarketValue: 25000000,
			Sector:      "Technology",
			Country:     "US",
		},
		{
			Symbol:      "MSFT",
			Name:        "Microsoft Corporation",
			Weight:      6.8,
			Shares:      120000,
			MarketValue: 23500000,
			Sector:      "Technology",
			Country:     "US",
		},
		{
			Symbol:      "GOOGL",
			Name:        "Alphabet Inc.",
			Weight:      4.1,
			Shares:      80000,
			MarketValue: 14200000,
			Sector:      "Technology",
			Country:     "US",
		},
		{
			Symbol:      "AMZN",
			Name:        "Amazon.com Inc.",
			Weight:      3.9,
			Shares:      75000,
			MarketValue: 13500000,
			Sector:      "Consumer Discretionary",
			Country:     "US",
		},
		{
			Symbol:      "TSLA",
			Name:        "Tesla Inc.",
			Weight:      2.1,
			Shares:      45000,
			MarketValue: 7300000,
			Sector:      "Consumer Discretionary",
			Country:     "US",
		},
	}

	return holdings
}

// Helper methods for attribute parsing
func (s *ETFService) getStringAttribute(attributes map[string]interface{}, key string) string {
	if val, ok := attributes[key].(string); ok {
		return val
	}
	return ""
}

func (s *ETFService) getFloatAttribute(attributes map[string]interface{}, key string) float64 {
	if val, ok := attributes[key].(float64); ok {
		return val
	}
	return 0.0
}

func (s *ETFService) getIntAttribute(attributes map[string]interface{}, key string) int {
	if val, ok := attributes[key].(float64); ok {
		return int(val)
	}
	return 0
}

func (s *ETFService) getInt64Attribute(attributes map[string]interface{}, key string) int64 {
	if val, ok := attributes[key].(float64); ok {
		return int64(val)
	}
	return 0
}
