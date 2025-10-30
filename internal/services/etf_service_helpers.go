package services

import (
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/db/models"
	"go.uber.org/zap"
)

// Helper methods for ETF calculations and operations

// calculateETFMetrics calculates derived ETF metrics from attributes
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

// calculateLiquidityMetrics calculates liquidity metrics for an ETF
func (s *ETFService) calculateLiquidityMetrics(metrics *ETFMetrics, symbol string) {
	// In a real implementation, this would calculate from market data
	metrics.Liquidity = LiquidityMetrics{
		BidAskSpread:        0.01, // 1 cent
		AverageVolume:       1000000,
		MedianVolume:        800000,
		VolumeWeightedPrice: metrics.MarketPrice,
		LiquidityScore:      8.5, // Out of 10
		MarketImpact:        0.05, // 5 basis points
	}
}

// calculatePerformanceMetrics calculates performance metrics for an ETF
func (s *ETFService) calculatePerformanceMetrics(metrics *ETFMetrics, symbol string) {
	// In a real implementation, this would calculate from historical data
	metrics.PerformanceMetrics = ETFPerformance{
		OneDay:         0.12,
		OneWeek:        0.85,
		OneMonth:       2.34,
		ThreeMonth:     5.67,
		SixMonth:       8.91,
		YTD:           12.45,
		OneYear:       15.67,
		ThreeYear:     8.23,
		FiveYear:      9.87,
		TenYear:       11.23,
		SinceInception: 9.45,
	}
}

// calculateRiskMetrics calculates risk metrics for an ETF
func (s *ETFService) calculateRiskMetrics(metrics *ETFMetrics, symbol string) {
	// In a real implementation, this would calculate from historical data
	metrics.RiskMetrics = ETFRiskMetrics{
		Beta:               1.02,
		Alpha:              0.15,
		Volatility:         16.5,
		SharpeRatio:        0.95,
		MaxDrawdown:        -18.7,
		VaR95:             -2.1,
		VaR99:             -3.8,
		CorrelationToIndex: 0.98,
	}
}

// calculateTaxEfficiency calculates tax efficiency metrics for an ETF
func (s *ETFService) calculateTaxEfficiency(metrics *ETFMetrics, symbol string) {
	// In a real implementation, this would calculate from distribution history
	metrics.TaxEfficiency = TaxEfficiencyMetrics{
		TaxEfficiencyRatio:       0.92,
		CapitalGainsDistribution: 0.15,
		DividendDistribution:     2.34,
		LastDistributionDate:     time.Now().AddDate(0, -3, 0),
		TurnoverRatio:           25.5,
	}
}

// getETFPriceHistory retrieves historical price data for an ETF
func (s *ETFService) getETFPriceHistory(symbol string, days int) ([]float64, error) {
	// In a real implementation, this would query historical pricing data
	// For now, return mock data
	prices := make([]float64, days)
	basePrice := 100.0
	
	for i := 0; i < days; i++ {
		// Simulate price movement with random walk
		change := (rand.Float64() - 0.5) * 2.0 // -1% to +1%
		basePrice *= (1.0 + change/100.0)
		prices[i] = basePrice
	}
	
	return prices, nil
}

// getBenchmarkPriceHistory retrieves historical price data for a benchmark
func (s *ETFService) getBenchmarkPriceHistory(benchmarkIndex string, days int) ([]float64, error) {
	// In a real implementation, this would query benchmark pricing data
	// For now, return mock data
	prices := make([]float64, days)
	basePrice := 1000.0 // Typical index level
	
	for i := 0; i < days; i++ {
		// Simulate index movement with random walk
		change := (rand.Float64() - 0.5) * 1.5 // -0.75% to +0.75%
		basePrice *= (1.0 + change/100.0)
		prices[i] = basePrice
	}
	
	return prices, nil
}

// calculateTrackingError calculates tracking error between ETF and benchmark
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
	trackingError := math.Sqrt(variance) * math.Sqrt(252) * 100 // 252 trading days per year
	
	return trackingError
}

// validateCreationRedemption validates creation/redemption operations
func (s *ETFService) validateCreationRedemption(operation *CreationRedemptionOperation) error {
	if operation.Symbol == "" {
		return fmt.Errorf("symbol is required")
	}
	
	if operation.OperationType != "creation" && operation.OperationType != "redemption" {
		return fmt.Errorf("invalid operation type: %s", operation.OperationType)
	}
	
	if operation.Units <= 0 {
		return fmt.Errorf("units must be positive")
	}
	
	if operation.AuthorizedParticipant == "" {
		return fmt.Errorf("authorized participant is required")
	}
	
	return nil
}

// processCreation processes ETF creation operations
func (s *ETFService) processCreation(operation *CreationRedemptionOperation, asset *models.AssetMetadata) error {
	// In a real implementation, this would:
	// 1. Validate authorized participant
	// 2. Check creation unit requirements
	// 3. Process underlying securities
	// 4. Update ETF shares outstanding
	// 5. Record the transaction
	
	s.logger.Info("Processing ETF creation", 
		zap.String("symbol", operation.Symbol),
		zap.Int("units", operation.Units))
	
	// Mock processing
	operation.Status = "completed"
	operation.Timestamp = time.Now()
	
	return nil
}

// processRedemption processes ETF redemption operations
func (s *ETFService) processRedemption(operation *CreationRedemptionOperation, asset *models.AssetMetadata) error {
	// In a real implementation, this would:
	// 1. Validate authorized participant
	// 2. Check redemption unit requirements
	// 3. Deliver underlying securities
	// 4. Update ETF shares outstanding
	// 5. Record the transaction
	
	s.logger.Info("Processing ETF redemption", 
		zap.String("symbol", operation.Symbol),
		zap.Int("units", operation.Units))
	
	// Mock processing
	operation.Status = "completed"
	operation.Timestamp = time.Now()
	
	return nil
}

// getMockHoldings returns mock ETF holdings data
func (s *ETFService) getMockHoldings(symbol string) []ETFHolding {
	// In a real implementation, this would query holdings database
	holdings := []ETFHolding{
		{
			Symbol:      "AAPL",
			Name:        "Apple Inc.",
			Weight:      6.8,
			Shares:      150000,
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

// generateRecommendations generates recommendations based on ETF metrics
func (s *ETFService) generateRecommendations(result *ETFAnalysisResult, metrics *ETFMetrics) {
	// Low expense ratio recommendation
	if metrics.ExpenseRatio < 0.20 {
		result.Recommendations = append(result.Recommendations, 
			"Low expense ratio makes this ETF cost-effective for long-term investing")
	}
	
	// High liquidity recommendation
	if metrics.Liquidity.LiquidityScore > 8.0 {
		result.Recommendations = append(result.Recommendations, 
			"High liquidity score indicates easy entry and exit")
	}
	
	// Low tracking error recommendation
	if metrics.TrackingError < 0.50 {
		result.Recommendations = append(result.Recommendations, 
			"Low tracking error indicates effective index replication")
	}
	
	// Performance recommendation
	if metrics.PerformanceMetrics.OneYear > 10.0 {
		result.Recommendations = append(result.Recommendations, 
			"Strong one-year performance indicates good momentum")
	}
}

// generateWarnings generates warnings based on ETF metrics
func (s *ETFService) generateWarnings(result *ETFAnalysisResult, metrics *ETFMetrics) {
	// High expense ratio warning
	if metrics.ExpenseRatio > 0.75 {
		result.Warnings = append(result.Warnings, 
			"High expense ratio may impact long-term returns")
	}
	
	// High tracking error warning
	if metrics.TrackingError > 1.0 {
		result.Warnings = append(result.Warnings, 
			"High tracking error indicates poor index replication")
	}
	
	// Low liquidity warning
	if metrics.Liquidity.LiquidityScore < 5.0 {
		result.Warnings = append(result.Warnings, 
			"Low liquidity may result in wider bid-ask spreads")
	}
	
	// Premium/discount warning
	if math.Abs(metrics.Premium) > 2.0 {
		result.Warnings = append(result.Warnings, 
			fmt.Sprintf("ETF trading at %.2f%% premium/discount to NAV", metrics.Premium))
	}
}

// meetsScreeningCriteria checks if ETF meets screening criteria
func (s *ETFService) meetsScreeningCriteria(metrics *ETFMetrics, criteria *ETFScreeningCriteria) bool {
	// Check AUM
	if criteria.MinAUM > 0 && metrics.AUM < criteria.MinAUM {
		return false
	}
	
	// Check expense ratio
	if criteria.MaxExpenseRatio > 0 && metrics.ExpenseRatio > criteria.MaxExpenseRatio {
		return false
	}
	
	// Check liquidity
	if criteria.MinLiquidity > 0 && metrics.Liquidity.LiquidityScore < criteria.MinLiquidity {
		return false
	}
	
	// Check tracking error
	if criteria.MaxTrackingError > 0 && metrics.TrackingError > criteria.MaxTrackingError {
		return false
	}
	
	return true
}

// calculateETFRankings calculates rankings for ETF comparison
func (s *ETFService) calculateETFRankings(result *ETFComparisonResult) {
	// Simple ranking based on expense ratio (lower is better)
	type etfScore struct {
		symbol string
		score  float64
	}
	
	var scores []etfScore
	for symbol, metrics := range result.Metrics {
		// Calculate composite score (lower expense ratio + higher performance)
		score := metrics.PerformanceMetrics.OneYear - (metrics.ExpenseRatio * 10)
		scores = append(scores, etfScore{symbol: symbol, score: score})
	}
	
	// Sort by score (descending)
	for i := 0; i < len(scores)-1; i++ {
		for j := i + 1; j < len(scores); j++ {
			if scores[j].score > scores[i].score {
				scores[i], scores[j] = scores[j], scores[i]
			}
		}
	}
	
	// Assign rankings
	for i, score := range scores {
		result.Rankings[score.symbol] = i + 1
		if i == 0 {
			result.BestPerformer = score.symbol
		}
		if i == len(scores)-1 {
			result.WorstPerformer = score.symbol
		}
	}
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
