// 🎯 **ADX Service Core**
// Generated using TradSys Code Splitting Standards
//
// This file contains the main service struct, constructor, and core API methods
// for the Abu Dhabi Exchange (ADX) Service component. It follows the established patterns for
// service initialization, lifecycle management, and primary business operations with Islamic finance focus.
//
// Performance Requirements: Standard latency, Islamic compliance integration
// File size limit: 410 lines

package exchanges

import (
	"context"
	"fmt"
	"log"
	"time"
)

// NewADXService creates a new ADX service instance with Islamic finance capabilities
func NewADXService() *ADXService {
	service := &ADXService{
		exchangeID:      ADXExchangeID,
		region:          ADXRegion,
		assetTypes:      []AssetType{AssetTypeEquity, AssetTypeSukuk, AssetTypeIslamicFund},
		languageSupport: []string{"en", "ar"},
		
		// Initialize components
		connector:          NewADXConnector(),
		marketData:         NewADXMarketData(),
		orderManager:       NewADXOrderManager(),
		riskEngine:         NewADXRiskEngine(),
		islamicCompliance:  NewIslamicCompliance(),
		uaeCompliance:      NewUAECompliance(),
		zakatCalculator:    NewZakatCalculator(),
		sukukService:       NewSukukService(),
		islamicFundService: NewIslamicFundService(),
		performanceMonitor: NewPerformanceMonitor(),
	}

	// Initialize service components
	service.initialize()

	return service
}

// initialize sets up the ADX service components
func (adx *ADXService) initialize() {
	log.Printf("Initializing ADX Service for Abu Dhabi Exchange with Islamic finance focus")

	// Initialize connector
	if err := adx.connector.Connect(); err != nil {
		log.Printf("Failed to connect to ADX: %v", err)
	}

	// Start Islamic market data feeds
	adx.marketData.StartIslamicFeeds()

	// Initialize Islamic compliance engine
	adx.islamicCompliance.LoadShariaRules()

	// Initialize UAE compliance
	adx.uaeCompliance.LoadRegulatoryRules()

	// Start Sukuk service
	adx.sukukService.Initialize()

	// Start Islamic fund service
	adx.islamicFundService.Initialize()

	// Start performance monitoring
	go adx.performanceMonitor.Start()

	log.Printf("ADX Service initialized successfully with Islamic finance capabilities")
}

// SubmitOrder submits an order to ADX with Islamic compliance checking
func (adx *ADXService) SubmitOrder(ctx context.Context, order *Order) (*OrderResponse, error) {
	startTime := time.Now()

	// Validate order
	if err := adx.validateOrder(order); err != nil {
		return nil, fmt.Errorf("order validation failed: %w", err)
	}

	// Check Islamic compliance if required
	if adx.isIslamicAsset(order.AssetType) {
		if err := adx.islamicCompliance.ValidateOrder(order); err != nil {
			return nil, fmt.Errorf("Islamic compliance validation failed: %w", err)
		}
	}

	// Check UAE compliance
	if err := adx.uaeCompliance.ValidateOrder(order); err != nil {
		return nil, fmt.Errorf("UAE compliance validation failed: %w", err)
	}

	// Risk assessment with Islamic considerations
	if err := adx.riskEngine.AssessOrder(order); err != nil {
		return nil, fmt.Errorf("risk assessment failed: %w", err)
	}

	// Submit to ADX
	response, err := adx.orderManager.SubmitOrder(ctx, order)
	if err != nil {
		return nil, fmt.Errorf("order submission failed: %w", err)
	}

	// Record performance metrics
	latency := time.Since(startTime)
	adx.performanceMonitor.RecordOrderLatency(latency)

	log.Printf("Order submitted to ADX: %s, Latency: %v", response.OrderID, latency)
	return response, nil
}

// GetSukukData retrieves Sukuk market data
func (adx *ADXService) GetSukukData(ctx context.Context, symbol string) (*SukukData, error) {
	// Validate Sukuk symbol
	if !adx.isValidADXSymbol(symbol) {
		return nil, fmt.Errorf("invalid ADX Sukuk symbol: %s", symbol)
	}

	// Get Sukuk data
	data, err := adx.sukukService.GetSukukData(symbol)
	if err != nil {
		return nil, fmt.Errorf("failed to get Sukuk data: %w", err)
	}

	return data, nil
}

// GetIslamicFundData retrieves Islamic fund data
func (adx *ADXService) GetIslamicFundData(ctx context.Context, fundID string) (*IslamicFundData, error) {
	// Get Islamic fund data
	data, err := adx.islamicFundService.GetFundData(fundID)
	if err != nil {
		return nil, fmt.Errorf("failed to get Islamic fund data: %w", err)
	}

	return data, nil
}

// GetMarketData retrieves market data with Islamic filtering
func (adx *ADXService) GetMarketData(ctx context.Context, symbol string, islamicOnly bool) (*MarketData, error) {
	// Validate symbol
	if !adx.isValidADXSymbol(symbol) {
		return nil, fmt.Errorf("invalid ADX symbol: %s", symbol)
	}

	// Get market data
	data, err := adx.marketData.GetMarketData(symbol)
	if err != nil {
		return nil, fmt.Errorf("failed to get market data: %w", err)
	}

	// Apply Islamic filtering if requested
	if islamicOnly {
		if !adx.islamicCompliance.IsCompliant(symbol) {
			return nil, fmt.Errorf("symbol %s is not Sharia compliant", symbol)
		}
	}

	return data, nil
}

// CalculateZakat calculates Zakat for Islamic investments
func (adx *ADXService) CalculateZakat(ctx context.Context, portfolio *Portfolio) (*ZakatCalculation, error) {
	// Validate portfolio
	if portfolio == nil {
		return nil, fmt.Errorf("portfolio cannot be nil")
	}

	// Calculate Zakat
	calculation, err := adx.zakatCalculator.Calculate(portfolio)
	if err != nil {
		return nil, fmt.Errorf("Zakat calculation failed: %w", err)
	}

	return calculation, nil
}

// GetComplianceReport generates Islamic compliance report
func (adx *ADXService) GetComplianceReport(ctx context.Context, portfolioID string) (*ComplianceReport, error) {
	// Generate compliance report
	report, err := adx.islamicCompliance.GenerateReport(portfolioID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate compliance report: %w", err)
	}

	return report, nil
}

// GetTradingHours returns ADX trading hours
func (adx *ADXService) GetTradingHours() *TradingSchedule {
	adx.mu.RLock()
	defer adx.mu.RUnlock()
	return adx.tradingHours
}

// IsMarketOpen checks if ADX market is currently open
func (adx *ADXService) IsMarketOpen() bool {
	now := time.Now()
	schedule := adx.GetTradingHours()
	
	if schedule == nil {
		return false
	}

	return schedule.IsOpen(now)
}

// GetSupportedAssetTypes returns supported asset types
func (adx *ADXService) GetSupportedAssetTypes() []AssetType {
	adx.mu.RLock()
	defer adx.mu.RUnlock()
	
	// Return copy to prevent modification
	assetTypes := make([]AssetType, len(adx.assetTypes))
	copy(assetTypes, adx.assetTypes)
	return assetTypes
}

// GetShariaBoards returns available Sharia boards
func (adx *ADXService) GetShariaBoards() []*ShariaBoard {
	adx.mu.RLock()
	defer adx.mu.RUnlock()
	
	// Return copy to prevent modification
	boards := make([]*ShariaBoard, len(adx.shariaBoards))
	copy(boards, adx.shariaBoards)
	return boards
}

// GetPerformanceMetrics returns service performance metrics
func (adx *ADXService) GetPerformanceMetrics() *PerformanceReport {
	return adx.performanceMonitor.GenerateReport()
}

// HealthCheck performs health check on ADX service
func (adx *ADXService) HealthCheck(ctx context.Context) *HealthStatus {
	status := &HealthStatus{
		Service:   "ADX",
		Timestamp: time.Now(),
		Status:    "healthy",
		Details:   make(map[string]interface{}),
	}

	// Check connector health
	if !adx.connector.IsHealthy() {
		status.Status = "unhealthy"
		status.Details["connector"] = "connection failed"
	}

	// Check market data health
	if !adx.marketData.IsHealthy() {
		status.Status = "degraded"
		status.Details["market_data"] = "data feed issues"
	}

	// Check Islamic compliance engine
	if !adx.islamicCompliance.IsHealthy() {
		status.Status = "degraded"
		status.Details["islamic_compliance"] = "compliance engine issues"
	}

	// Check UAE compliance
	if !adx.uaeCompliance.IsHealthy() {
		status.Status = "degraded"
		status.Details["uae_compliance"] = "regulatory compliance issues"
	}

	return status
}

// Shutdown gracefully shuts down the ADX service
func (adx *ADXService) Shutdown(ctx context.Context) error {
	log.Printf("Shutting down ADX Service")

	// Stop performance monitoring
	adx.performanceMonitor.Stop()

	// Shutdown Islamic fund service
	if err := adx.islamicFundService.Shutdown(); err != nil {
		log.Printf("Error shutting down Islamic fund service: %v", err)
	}

	// Shutdown Sukuk service
	if err := adx.sukukService.Shutdown(); err != nil {
		log.Printf("Error shutting down Sukuk service: %v", err)
	}

	// Stop market data feeds
	adx.marketData.Stop()

	// Disconnect from ADX
	if err := adx.connector.Disconnect(); err != nil {
		log.Printf("Error disconnecting from ADX: %v", err)
	}

	log.Printf("ADX Service shutdown complete")
	return nil
}

// GetExchangeInfo returns ADX exchange information
func (adx *ADXService) GetExchangeInfo() *ExchangeInfo {
	adx.mu.RLock()
	defer adx.mu.RUnlock()

	return &ExchangeInfo{
		ExchangeID:      adx.exchangeID,
		Name:            "Abu Dhabi Securities Exchange",
		Region:          adx.region,
		Timezone:        ADXTimezone,
		AssetTypes:      adx.GetSupportedAssetTypes(),
		LanguageSupport: adx.languageSupport,
		IslamicFocus:    true,
		TradingHours:    adx.tradingHours,
	}
}

// UpdateTradingHours updates the trading schedule
func (adx *ADXService) UpdateTradingHours(schedule *TradingSchedule) error {
	if schedule == nil {
		return fmt.Errorf("trading schedule cannot be nil")
	}

	adx.mu.Lock()
	defer adx.mu.Unlock()

	adx.tradingHours = schedule
	log.Printf("ADX trading hours updated")
	return nil
}

// AddShariaBoard adds a new Sharia board
func (adx *ADXService) AddShariaBoard(board *ShariaBoard) error {
	if board == nil {
		return fmt.Errorf("Sharia board cannot be nil")
	}

	adx.mu.Lock()
	defer adx.mu.Unlock()

	adx.shariaBoards = append(adx.shariaBoards, board)
	log.Printf("Sharia board added: %s", board.Name)
	return nil
}

// GetServiceStatus returns current service status
func (adx *ADXService) GetServiceStatus() *ServiceStatus {
	return &ServiceStatus{
		Service:            "ADX",
		Status:             "running",
		Uptime:             adx.performanceMonitor.GetUptime(),
		ConnectionStatus:   adx.connector.GetStatus(),
		IslamicCompliance:  adx.islamicCompliance.GetStatus(),
		UAECompliance:      adx.uaeCompliance.GetStatus(),
		LastHealthCheck:    time.Now(),
	}
}
