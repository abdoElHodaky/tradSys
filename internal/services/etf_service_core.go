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

// NewETFService creates a new ETF service instance
func NewETFService(db *gorm.DB, assetService *AssetService, logger *zap.Logger) *ETFService {
	return &ETFService{
		db:           db,
		assetService: assetService,
		logger:       logger,
	}
}

// CreateETF creates a new ETF with initial metadata
func (s *ETFService) CreateETF(symbol, benchmarkIndex string, creationUnitSize int, expenseRatio float64) error {
	s.logger.Info("Creating new ETF", zap.String("symbol", symbol))

	// Create asset metadata
	assetAttributes := models.AssetAttributes{
		"benchmark_index":     benchmarkIndex,
		"creation_unit_size":  creationUnitSize,
		"expense_ratio":       expenseRatio,
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
		AssetType:       types.AssetTypeETF,
		TradingEnabled:  true,
		MinOrderSize:    1.0,
		MaxOrderSize:    1000000.0,
		RiskMultiplier:  1.0,
		SettlementDays:  2,
		TradingHours:    "09:30-16:00",
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

	// Get ETF asset
	asset, err := s.assetService.GetAssetBySymbol(context.Background(), operation.Symbol)
	if err != nil {
		return fmt.Errorf("failed to get ETF asset: %w", err)
	}

	// Process operation based on type
	switch operation.OperationType {
	case "creation":
		return s.processCreation(operation, asset)
	case "redemption":
		return s.processRedemption(operation, asset)
	default:
		return fmt.Errorf("invalid operation type: %s", operation.OperationType)
	}
}

// GetETFHoldings returns the current holdings of an ETF
func (s *ETFService) GetETFHoldings(symbol string) ([]ETFHolding, error) {
	s.logger.Debug("Retrieving ETF holdings", zap.String("symbol", symbol))

	// Get asset metadata
	asset, err := s.assetService.GetAssetBySymbol(context.Background(), symbol)
	if err != nil {
		return nil, fmt.Errorf("failed to get ETF asset: %w", err)
	}

	if asset.AssetType != types.AssetTypeETF {
		return nil, fmt.Errorf("asset %s is not an ETF", symbol)
	}

	// For now, return mock holdings
	// In a real implementation, this would fetch from a holdings database
	holdings := s.getMockHoldings(symbol)

	s.logger.Debug("Retrieved ETF holdings", 
		zap.String("symbol", symbol), 
		zap.Int("holdings_count", len(holdings)))

	return holdings, nil
}

// AnalyzeETF performs comprehensive ETF analysis
func (s *ETFService) AnalyzeETF(request *ETFAnalysisRequest) (*ETFAnalysisResult, error) {
	s.logger.Info("Analyzing ETF", zap.String("symbol", request.Symbol))

	// Get ETF metrics
	metrics, err := s.GetETFMetrics(request.Symbol)
	if err != nil {
		return nil, fmt.Errorf("failed to get ETF metrics: %w", err)
	}

	// Perform analysis
	result := &ETFAnalysisResult{
		Symbol:       request.Symbol,
		AnalysisDate: time.Now(),
		Metrics:      *metrics,
		Recommendations: []string{},
		Warnings:       []string{},
	}

	// Add recommendations based on metrics
	s.generateRecommendations(result, metrics)
	s.generateWarnings(result, metrics)

	s.logger.Info("ETF analysis completed", 
		zap.String("symbol", request.Symbol),
		zap.Int("recommendations", len(result.Recommendations)),
		zap.Int("warnings", len(result.Warnings)))

	return result, nil
}

// ScreenETFs screens ETFs based on criteria
func (s *ETFService) ScreenETFs(criteria *ETFScreeningCriteria) ([]string, error) {
	s.logger.Info("Screening ETFs with criteria")

	// This would typically query a database of ETFs
	// For now, return a mock list
	etfs := []string{"SPY", "QQQ", "VTI", "IWM", "EFA"}
	
	var filteredETFs []string
	
	for _, etf := range etfs {
		metrics, err := s.GetETFMetrics(etf)
		if err != nil {
			s.logger.Warn("Failed to get metrics for ETF", zap.String("symbol", etf), zap.Error(err))
			continue
		}
		
		if s.meetsScreeningCriteria(metrics, criteria) {
			filteredETFs = append(filteredETFs, etf)
		}
	}

	s.logger.Info("ETF screening completed", 
		zap.Int("total_etfs", len(etfs)),
		zap.Int("filtered_etfs", len(filteredETFs)))

	return filteredETFs, nil
}

// CompareETFs compares multiple ETFs
func (s *ETFService) CompareETFs(symbols []string) (*ETFComparisonResult, error) {
	s.logger.Info("Comparing ETFs", zap.Strings("symbols", symbols))

	result := &ETFComparisonResult{
		ETFs:           symbols,
		ComparisonDate: time.Now(),
		Metrics:        make(map[string]ETFMetrics),
		Rankings:       make(map[string]int),
	}

	// Get metrics for each ETF
	for _, symbol := range symbols {
		metrics, err := s.GetETFMetrics(symbol)
		if err != nil {
			s.logger.Warn("Failed to get metrics for ETF", zap.String("symbol", symbol), zap.Error(err))
			continue
		}
		result.Metrics[symbol] = *metrics
	}

	// Calculate rankings
	s.calculateETFRankings(result)

	s.logger.Info("ETF comparison completed", 
		zap.Int("etfs_compared", len(result.Metrics)))

	return result, nil
}
