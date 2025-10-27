// Package assets implements Phase 4: Unified Asset System for TradSys v3
// Provides unified multi-asset support across EGX and ADX exchanges
package assets

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/services/exchanges"
)

// UnifiedAssetSystem provides unified asset management across exchanges
type UnifiedAssetSystem struct {
	assetRegistry      *AssetRegistry
	pricingEngine      *UnifiedPricingEngine
	portfolioManager   *CrossExchangePortfolioManager
	analyticsEngine    *UnifiedAnalyticsEngine
	complianceManager  *UnifiedComplianceManager
	licensingManager   *UnifiedLicensingManager
	configManager      *UnifiedConfigManager
	reportingEngine    *UnifiedReportingEngine
	performanceMonitor *UnifiedPerformanceMonitor
	mu                 sync.RWMutex
}

// AssetRegistry maintains registry of all assets across exchanges
type AssetRegistry struct {
	assets           map[string]*UnifiedAsset
	assetsByExchange map[string]map[string]*UnifiedAsset
	assetsByType     map[exchanges.AssetType][]*UnifiedAsset
	searchIndex      *AssetSearchIndex
	mu               sync.RWMutex
}

// UnifiedAsset represents a unified asset across exchanges
type UnifiedAsset struct {
	ID             string
	Symbol         string
	Name           string
	AssetType      exchanges.AssetType
	Exchange       string
	Region         string
	Currency       string
	ISIN           string
	Sector         string
	Industry       string
	MarketCap      float64
	IslamicInfo    *IslamicAssetInfo
	ComplianceInfo *UnifiedComplianceInfo
	TradingInfo    *TradingInfo
	PricingInfo    *PricingInfo
	AnalyticsInfo  *AnalyticsInfo
	Metadata       map[string]interface{}
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// IslamicAssetInfo contains Islamic finance information
type IslamicAssetInfo struct {
	IsHalal            bool
	ComplianceLevel    exchanges.ComplianceLevel
	ShariaBoard        string
	LastScreened       time.Time
	ComplianceScore    float64
	DebtEquityRatio    float64
	BusinessActivities []string
	Restrictions       []string
	ZakatApplicable    bool
}

// UnifiedComplianceInfo contains compliance information across jurisdictions
type UnifiedComplianceInfo struct {
	EgyptianCompliance *EgyptianComplianceInfo
	UAECompliance      *UAEComplianceInfo
	IslamicCompliance  *IslamicComplianceInfo
	GlobalCompliance   *GlobalComplianceInfo
	LastUpdated        time.Time
}

// TradingInfo contains trading-related information
type TradingInfo struct {
	TradingHours   *exchanges.TradingSchedule
	MinOrderSize   float64
	MaxOrderSize   float64
	TickSize       float64
	LotSize        int
	SettlementDays int
	IsActive       bool
	TradingStatus  string
}

// PricingInfo contains pricing and market data information
type PricingInfo struct {
	CurrentPrice     float64
	PreviousClose    float64
	DayChange        float64
	DayChangePercent float64
	Volume           int64
	MarketCap        float64
	PE               float64
	DividendYield    float64
	LastUpdated      time.Time
}

// AnalyticsInfo contains analytics and performance information
type AnalyticsInfo struct {
	Volatility     float64
	Beta           float64
	Sharpe         float64
	MaxDrawdown    float64
	Performance1D  float64
	Performance1W  float64
	Performance1M  float64
	Performance1Y  float64
	LastCalculated time.Time
}

// UnifiedPricingEngine provides unified pricing across exchanges
type UnifiedPricingEngine struct {
	pricingModels  map[string]PricingModel
	dataAggregator *DataAggregator
	priceCache     *PriceCache
	realTimeFeeds  map[string]*RealTimeFeed
	mu             sync.RWMutex
}

// CrossExchangePortfolioManager manages portfolios across exchanges
type CrossExchangePortfolioManager struct {
	portfolios      map[string]*UnifiedPortfolio
	positionManager *PositionManager
	riskManager     *CrossExchangeRiskManager
	rebalancer      *PortfolioRebalancer
	mu              sync.RWMutex
}

// UnifiedPortfolio represents a portfolio across multiple exchanges
type UnifiedPortfolio struct {
	UserID      string
	PortfolioID string
	Name        string
	Currency    string
	TotalValue  float64
	CashBalance float64
	Positions   []*UnifiedPosition
	Performance *PortfolioPerformance
	RiskMetrics *PortfolioRiskMetrics
	IslamicInfo *IslamicPortfolioInfo
	LastUpdated time.Time
}

// UnifiedPosition represents a position across exchanges
type UnifiedPosition struct {
	AssetID       string
	Symbol        string
	Exchange      string
	AssetType     exchanges.AssetType
	Quantity      float64
	AverageCost   float64
	CurrentPrice  float64
	MarketValue   float64
	UnrealizedPnL float64
	RealizedPnL   float64
	DayChange     float64
	Weight        float64
	LastUpdated   time.Time
}

// UnifiedAnalyticsEngine provides analytics across exchanges
type UnifiedAnalyticsEngine struct {
	analyticsModels map[string]AnalyticsModel
	dataProcessor   *AnalyticsDataProcessor
	reportGenerator *AnalyticsReportGenerator
	mlEngine        *MachineLearningEngine
	mu              sync.RWMutex
}

// UnifiedComplianceManager manages compliance across jurisdictions
type UnifiedComplianceManager struct {
	complianceRules map[string]ComplianceRuleSet
	auditTrail      *UnifiedAuditTrail
	reportingEngine *ComplianceReportingEngine
	alertManager    *ComplianceAlertManager
	mu              sync.RWMutex
}

// UnifiedLicensingManager manages enterprise licensing
type UnifiedLicensingManager struct {
	licenseValidator *LicenseValidator
	quotaManager     *QuotaManager
	billingEngine    *BillingEngine
	usageTracker     *UsageTracker
	mu               sync.RWMutex
}

// UnifiedConfigManager manages configuration across all services
type UnifiedConfigManager struct {
	configs         map[string]*ServiceConfig
	configStore     *ConfigStore
	configValidator *ConfigValidator
	changeNotifier  *ConfigChangeNotifier
	mu              sync.RWMutex
}

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

	log.Printf("Portfolio created successfully: %s for user %s", portfolio.PortfolioID, userID)
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

// validateAsset validates a unified asset
func (uas *UnifiedAssetSystem) validateAsset(asset *UnifiedAsset) error {
	if asset.Symbol == "" {
		return fmt.Errorf("asset symbol is required")
	}
	if asset.Exchange == "" {
		return fmt.Errorf("asset exchange is required")
	}
	if asset.AssetType < 0 {
		return fmt.Errorf("invalid asset type")
	}
	if asset.Currency == "" {
		return fmt.Errorf("asset currency is required")
	}
	return nil
}

// updateAssetPricing updates pricing information for an asset
func (uas *UnifiedAssetSystem) updateAssetPricing(ctx context.Context, asset *UnifiedAsset) error {
	pricing, err := uas.pricingEngine.GetCurrentPricing(asset.Symbol, asset.Exchange)
	if err != nil {
		return err
	}

	asset.PricingInfo = pricing
	asset.UpdatedAt = time.Now()

	return nil
}

// updatePortfolioValues updates portfolio values and metrics
func (uas *UnifiedAssetSystem) updatePortfolioValues(ctx context.Context, portfolio *UnifiedPortfolio) error {
	totalValue := portfolio.CashBalance

	for _, position := range portfolio.Positions {
		// Get current price
		pricing, err := uas.pricingEngine.GetCurrentPricing(position.Symbol, position.Exchange)
		if err != nil {
			log.Printf("Warning: Failed to get pricing for %s: %v", position.Symbol, err)
			continue
		}

		// Update position values
		position.CurrentPrice = pricing.CurrentPrice
		position.MarketValue = position.Quantity * position.CurrentPrice
		position.UnrealizedPnL = position.MarketValue - (position.Quantity * position.AverageCost)
		position.DayChange = (position.CurrentPrice - pricing.PreviousClose) / pricing.PreviousClose * 100
		position.LastUpdated = time.Now()

		totalValue += position.MarketValue
	}

	// Update portfolio totals
	portfolio.TotalValue = totalValue
	portfolio.LastUpdated = time.Now()

	// Calculate weights
	for _, position := range portfolio.Positions {
		if totalValue > 0 {
			position.Weight = position.MarketValue / totalValue * 100
		}
	}

	// Update performance metrics
	if err := uas.updatePortfolioPerformance(portfolio); err != nil {
		log.Printf("Warning: Failed to update portfolio performance: %v", err)
	}

	return nil
}

// updatePortfolioPerformance updates portfolio performance metrics
func (uas *UnifiedAssetSystem) updatePortfolioPerformance(portfolio *UnifiedPortfolio) error {
	// Calculate performance metrics
	// This would involve historical data analysis
	// For now, we'll set placeholder values

	if portfolio.Performance == nil {
		portfolio.Performance = &PortfolioPerformance{}
	}

	// Update performance metrics
	portfolio.Performance.LastUpdated = time.Now()

	return nil
}

// GetSystemMetrics returns unified system metrics
func (uas *UnifiedAssetSystem) GetSystemMetrics() *SystemMetrics {
	uas.mu.RLock()
	defer uas.mu.RUnlock()

	return &SystemMetrics{
		TotalAssets:     uas.assetRegistry.GetAssetCount(),
		TotalPortfolios: uas.portfolioManager.GetPortfolioCount(),
		ActiveUsers:     uas.licensingManager.GetActiveUserCount(),
		SystemUptime:    uas.performanceMonitor.GetUptime(),
		Timestamp:       time.Now(),
	}
}

// Shutdown gracefully shuts down the unified asset system
func (uas *UnifiedAssetSystem) Shutdown(ctx context.Context) error {
	log.Printf("Shutting down Unified Asset System...")

	// Shutdown components
	uas.performanceMonitor.Stop()
	uas.pricingEngine.Shutdown()
	uas.analyticsEngine.Shutdown()
	uas.complianceManager.Shutdown()
	uas.licensingManager.Shutdown()

	log.Printf("Unified Asset System shutdown complete")
	return nil
}

// Supporting types and structures

// AssetSearchQuery represents an asset search query
type AssetSearchQuery struct {
	UserID       string
	Query        string
	AssetTypes   []exchanges.AssetType
	Exchanges    []string
	Sectors      []string
	IslamicOnly  bool
	MinMarketCap float64
	MaxMarketCap float64
	Limit        int
	Offset       int
}

// AnalyticsRequest represents an analytics request
type AnalyticsRequest struct {
	UserID      string
	PortfolioID string
	AssetIDs    []string
	Metrics     []string
	TimeRange   TimeRange
	Benchmarks  []string
}

// ComplianceRequest represents a compliance request
type ComplianceRequest struct {
	UserID        string
	PortfolioID   string
	AssetIDs      []string
	Jurisdictions []string
	ReportType    string
}

// TimeRange represents a time range for analytics
type TimeRange struct {
	StartDate time.Time
	EndDate   time.Time
}

// SystemMetrics represents unified system metrics
type SystemMetrics struct {
	TotalAssets     int
	TotalPortfolios int
	ActiveUsers     int
	SystemUptime    time.Duration
	Timestamp       time.Time
}

// PortfolioPerformance represents portfolio performance metrics
type PortfolioPerformance struct {
	TotalReturn      float64
	AnnualizedReturn float64
	Volatility       float64
	SharpeRatio      float64
	MaxDrawdown      float64
	Beta             float64
	Alpha            float64
	LastUpdated      time.Time
}

// PortfolioRiskMetrics represents portfolio risk metrics
type PortfolioRiskMetrics struct {
	VaR95             float64
	VaR99             float64
	ExpectedShortfall float64
	ConcentrationRisk float64
	CurrencyRisk      float64
	LastCalculated    time.Time
}

// IslamicPortfolioInfo represents Islamic portfolio information
type IslamicPortfolioInfo struct {
	IsCompliant       bool
	ComplianceScore   float64
	ZakatDue          float64
	NonCompliantValue float64
	LastScreened      time.Time
}
