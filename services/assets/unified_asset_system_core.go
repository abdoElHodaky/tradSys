// Package assets implements Phase 4: Unified Asset System for TradSys v3
// Provides unified multi-asset support across EGX and ADX exchanges
package assets

import (
	"context"
	"fmt"
	"log"

	"github.com/abdoElHodaky/tradSys/services/exchanges"
)

// NewUnifiedAssetSystem creates a new unified asset system
func NewUnifiedAssetSystem() *UnifiedAssetSystem {
	system := &UnifiedAssetSystem{
		assetRegistry:      NewAssetRegistry(),
		pricingEngine:      NewUnifiedPricingEngine(),
		portfolioManager:   NewCrossExchangePortfolioManager(),
		analyticsEngine:    NewUnifiedAnalyticsEngine(),
		complianceManager:  NewUnifiedComplianceManager(),
		licensingManager:   NewUnifiedLicensingManager(),
		configManager:      NewUnifiedConfigManager(),
		reportingEngine:    NewUnifiedReportingEngine(),
		performanceMonitor: NewUnifiedPerformanceMonitor(),
	}

	// Initialize system components
	system.initialize()

	return system
}

// initialize sets up the unified asset system
func (uas *UnifiedAssetSystem) initialize() {
	log.Printf("Initializing Unified Asset System")

	// Initialize asset registry
	uas.assetRegistry.Initialize()

	// Initialize pricing engine
	uas.pricingEngine.Initialize()

	// Initialize portfolio manager
	uas.portfolioManager.Initialize()

	// Initialize analytics engine
	uas.analyticsEngine.Initialize()

	// Initialize compliance manager
	uas.complianceManager.Initialize()

	// Initialize licensing manager
	uas.licensingManager.Initialize()

	// Initialize configuration manager
	uas.configManager.Initialize()

	// Start performance monitoring
	go uas.performanceMonitor.Start()

	log.Printf("Unified Asset System initialized successfully")
}

// RegisterAsset registers a new asset in the unified system
func (uas *UnifiedAssetSystem) RegisterAsset(ctx context.Context, asset *UnifiedAsset) error {
	// Validate asset
	if err := uas.validateAsset(asset); err != nil {
		return fmt.Errorf("asset validation failed: %w", err)
	}

	// Check compliance
	if err := uas.complianceManager.ValidateAsset(asset); err != nil {
		return fmt.Errorf("compliance validation failed: %w", err)
	}

	// Register in asset registry
	if err := uas.assetRegistry.RegisterAsset(asset); err != nil {
		return fmt.Errorf("asset registration failed: %w", err)
	}

	// Initialize pricing
	if err := uas.pricingEngine.InitializeAssetPricing(asset); err != nil {
		log.Printf("Warning: Failed to initialize pricing for asset %s: %v", asset.Symbol, err)
	}

	log.Printf("Asset registered successfully: %s (%s)", asset.Symbol, asset.Exchange)
	return nil
}

// GetAsset retrieves a unified asset by symbol and exchange
func (uas *UnifiedAssetSystem) GetAsset(ctx context.Context, symbol, exchange string) (*UnifiedAsset, error) {
	asset, err := uas.assetRegistry.GetAsset(symbol, exchange)
	if err != nil {
		return nil, fmt.Errorf("failed to get asset: %w", err)
	}

	// Update pricing information
	if err := uas.updateAssetPricing(ctx, asset); err != nil {
		log.Printf("Warning: Failed to update pricing for asset %s: %v", asset.Symbol, err)
	}

	return asset, nil
}

// SearchAssets searches for assets across exchanges
func (uas *UnifiedAssetSystem) SearchAssets(ctx context.Context, query *AssetSearchQuery) ([]*UnifiedAsset, error) {
	results, err := uas.assetRegistry.SearchAssets(query)
	if err != nil {
		return nil, fmt.Errorf("asset search failed: %w", err)
	}

	// Apply licensing filters
	filteredResults, err := uas.licensingManager.FilterAssetsByLicense(ctx, results, query.UserID)
	if err != nil {
		return nil, fmt.Errorf("license filtering failed: %w", err)
	}

	return filteredResults, nil
}

// CreatePortfolio creates a new unified portfolio
func (uas *UnifiedAssetSystem) CreatePortfolio(ctx context.Context, userID, name, currency string) (*UnifiedPortfolio, error) {
	// Validate license
	if err := uas.licensingManager.ValidatePortfolioCreation(ctx, userID); err != nil {
		return nil, fmt.Errorf("license validation failed: %w", err)
	}

	// Create portfolio
	portfolio, err := uas.portfolioManager.CreatePortfolio(userID, name, currency)
	if err != nil {
		return nil, fmt.Errorf("portfolio creation failed: %w", err)
	}

	log.Printf("Portfolio created successfully: %s for user %s", portfolio.ID, userID)
	return portfolio, nil
}

// GetPortfolio retrieves a unified portfolio
func (uas *UnifiedAssetSystem) GetPortfolio(ctx context.Context, userID, portfolioID string) (*UnifiedPortfolio, error) {
	// Validate access
	if err := uas.licensingManager.ValidatePortfolioAccess(ctx, userID, portfolioID); err != nil {
		return nil, fmt.Errorf("access validation failed: %w", err)
	}

	// Get portfolio
	portfolio, err := uas.portfolioManager.GetPortfolio(portfolioID)
	if err != nil {
		return nil, fmt.Errorf("failed to get portfolio: %w", err)
	}

	// Update portfolio values
	if err := uas.updatePortfolioValues(ctx, portfolio); err != nil {
		log.Printf("Warning: Failed to update portfolio values: %v", err)
	}

	return portfolio, nil
}

// GetCrossExchangeAnalytics provides analytics across exchanges
func (uas *UnifiedAssetSystem) GetCrossExchangeAnalytics(ctx context.Context, userID string, request *AnalyticsRequest) (*AnalyticsReport, error) {
	// Validate license
	if err := uas.licensingManager.ValidateAnalyticsAccess(ctx, userID); err != nil {
		return nil, fmt.Errorf("license validation failed: %w", err)
	}

	// Generate analytics
	report, err := uas.analyticsEngine.GenerateReport(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("analytics generation failed: %w", err)
	}

	return report, nil
}

// GetUnifiedCompliance provides compliance information across jurisdictions
func (uas *UnifiedAssetSystem) GetUnifiedCompliance(ctx context.Context, userID string, request *ComplianceRequest) (*ComplianceReport, error) {
	// Validate access
	if err := uas.licensingManager.ValidateComplianceAccess(ctx, userID); err != nil {
		return nil, fmt.Errorf("access validation failed: %w", err)
	}

	// Generate compliance report
	report, err := uas.complianceManager.GenerateReport(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("compliance report generation failed: %w", err)
	}

	return report, nil
}

// GetSystemMetrics returns unified system metrics
func (uas *UnifiedAssetSystem) GetSystemMetrics(ctx context.Context) (*SystemMetrics, error) {
	return uas.performanceMonitor.GetMetrics(), nil
}

// Shutdown gracefully shuts down the unified asset system
func (uas *UnifiedAssetSystem) Shutdown(ctx context.Context) error {
	log.Printf("Shutting down Unified Asset System")

	// Stop performance monitoring
	uas.performanceMonitor.Stop()

	// Shutdown components
	if err := uas.pricingEngine.Shutdown(ctx); err != nil {
		log.Printf("Warning: Failed to shutdown pricing engine: %v", err)
	}

	if err := uas.portfolioManager.Shutdown(ctx); err != nil {
		log.Printf("Warning: Failed to shutdown portfolio manager: %v", err)
	}

	if err := uas.analyticsEngine.Shutdown(ctx); err != nil {
		log.Printf("Warning: Failed to shutdown analytics engine: %v", err)
	}

	log.Printf("Unified Asset System shutdown complete")
	return nil
}

// validateAsset validates a unified asset
func (uas *UnifiedAssetSystem) validateAsset(asset *UnifiedAsset) error {
	if asset.Symbol == "" {
		return fmt.Errorf("asset symbol is required")
	}
	if asset.Exchange == "" {
		return fmt.Errorf("asset exchange is required")
	}
	if asset.Currency == "" {
		return fmt.Errorf("asset currency is required")
	}
	return nil
}

// updateAssetPricing updates pricing information for an asset
func (uas *UnifiedAssetSystem) updateAssetPricing(ctx context.Context, asset *UnifiedAsset) error {
	price, err := uas.pricingEngine.GetCurrentPrice(asset.Symbol, asset.Exchange)
	if err != nil {
		return err
	}

	if asset.PricingInfo == nil {
		asset.PricingInfo = &AssetPricingInfo{}
	}

	asset.PricingInfo.CurrentPrice = price
	// Additional pricing updates would go here

	return nil
}

// updatePortfolioValues updates portfolio values with current market prices
func (uas *UnifiedAssetSystem) updatePortfolioValues(ctx context.Context, portfolio *UnifiedPortfolio) error {
	for _, position := range portfolio.Assets {
		price, err := uas.pricingEngine.GetCurrentPrice(position.Symbol, position.Exchange)
		if err != nil {
			log.Printf("Warning: Failed to get price for %s: %v", position.Symbol, err)
			continue
		}

		position.CurrentPrice = price
		position.MarketValue = position.Quantity * price
		position.UnrealizedPnL = (price - position.AveragePrice) * position.Quantity
	}

	return nil
}

// Constructor functions for components (placeholder implementations)
func NewAssetRegistry() *AssetRegistry {
	return &AssetRegistry{
		assets:           make(map[string]*UnifiedAsset),
		assetsByExchange: make(map[string]map[string]*UnifiedAsset),
		assetsByType:     make(map[exchanges.AssetType][]*UnifiedAsset),
		searchIndex:      &AssetSearchIndex{},
	}
}

func NewUnifiedPricingEngine() *UnifiedPricingEngine {
	return &UnifiedPricingEngine{
		priceProviders: make(map[string]PriceProvider),
	}
}

func NewCrossExchangePortfolioManager() *CrossExchangePortfolioManager {
	return &CrossExchangePortfolioManager{
		portfolios: make(map[string]*UnifiedPortfolio),
	}
}

func NewUnifiedAnalyticsEngine() *UnifiedAnalyticsEngine {
	return &UnifiedAnalyticsEngine{
		analyticsProviders: make(map[string]AnalyticsProvider),
	}
}

func NewUnifiedComplianceManager() *UnifiedComplianceManager {
	return &UnifiedComplianceManager{
		complianceRules: make(map[string]*ComplianceRuleSet),
	}
}

func NewUnifiedLicensingManager() *UnifiedLicensingManager {
	return &UnifiedLicensingManager{
		licenses: make(map[string]*LicenseInfo),
	}
}

func NewUnifiedConfigManager() *UnifiedConfigManager {
	return &UnifiedConfigManager{
		configs: make(map[string]*ServiceConfig),
	}
}

func NewUnifiedReportingEngine() *UnifiedReportingEngine {
	return &UnifiedReportingEngine{
		reportTemplates: make(map[string]*ReportTemplate),
	}
}

func NewUnifiedPerformanceMonitor() *UnifiedPerformanceMonitor {
	return &UnifiedPerformanceMonitor{
		metrics: &SystemMetrics{},
	}
}
