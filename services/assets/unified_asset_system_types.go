// Package assets implements Phase 4: Unified Asset System for TradSys v3
// Provides unified multi-asset support across EGX and ADX exchanges
package assets

import (
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
	ID               string
	Symbol           string
	Name             string
	AssetType        exchanges.AssetType
	Exchange         string
	Region           string
	Currency         string
	ISIN             string
	Sector           string
	Industry         string
	MarketCap        float64
	IslamicCompliant bool
	TradingHours     *TradingHours
	PricingInfo      *AssetPricingInfo
	RiskMetrics      *AssetRiskMetrics
	ComplianceInfo   *AssetComplianceInfo
	LicensingInfo    *AssetLicensingInfo
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// AssetSearchIndex provides fast asset search capabilities
type AssetSearchIndex struct {
	symbolIndex   map[string][]*UnifiedAsset
	nameIndex     map[string][]*UnifiedAsset
	sectorIndex   map[string][]*UnifiedAsset
	industryIndex map[string][]*UnifiedAsset
	mu            sync.RWMutex
}

// UnifiedPricingEngine provides unified pricing across exchanges
type UnifiedPricingEngine struct {
	priceProviders    map[string]PriceProvider
	priceCache        *PriceCache
	pricingRules      *PricingRuleEngine
	arbitrageDetector *ArbitrageDetector
	mu                sync.RWMutex
}

// CrossExchangePortfolioManager manages portfolios across exchanges
type CrossExchangePortfolioManager struct {
	portfolios      map[string]*UnifiedPortfolio
	positionManager *CrossExchangePositionManager
	riskManager     *CrossExchangeRiskManager
	rebalancer      *CrossExchangeRebalancer
	mu              sync.RWMutex
}

// UnifiedAnalyticsEngine provides unified analytics across exchanges
type UnifiedAnalyticsEngine struct {
	analyticsProviders map[string]AnalyticsProvider
	metricsCalculator  *UnifiedMetricsCalculator
	benchmarkManager   *BenchmarkManager
	reportGenerator    *AnalyticsReportGenerator
	mu                 sync.RWMutex
}

// UnifiedComplianceManager manages compliance across exchanges
type UnifiedComplianceManager struct {
	complianceRules map[string]*ComplianceRuleSet
	screeningEngine *ComplianceScreeningEngine
	reportingEngine *ComplianceReportingEngine
	auditTrail      *ComplianceAuditTrail
	mu              sync.RWMutex
}

// UnifiedLicensingManager manages licensing across exchanges
type UnifiedLicensingManager struct {
	licenses         map[string]*LicenseInfo
	licenseValidator *LicenseValidator
	usageTracker     *LicenseUsageTracker
	renewalManager   *LicenseRenewalManager
	mu               sync.RWMutex
}

// UnifiedConfigManager manages configuration across exchanges
type UnifiedConfigManager struct {
	configs         map[string]*ServiceConfig
	configStore     *ConfigStore
	configValidator *ConfigValidator
	changeNotifier  *ConfigChangeNotifier
	mu              sync.RWMutex
}

// UnifiedReportingEngine provides unified reporting across exchanges
type UnifiedReportingEngine struct {
	reportTemplates map[string]*ReportTemplate
	reportGenerator *ReportGenerator
	reportScheduler *ReportScheduler
	reportDelivery  *ReportDeliveryManager
	mu              sync.RWMutex
}

// UnifiedPerformanceMonitor monitors system performance
type UnifiedPerformanceMonitor struct {
	metrics        *SystemMetrics
	alertManager   *AlertManager
	performanceLog *PerformanceLog
	healthChecker  *HealthChecker
	mu             sync.RWMutex
}

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

// Supporting interface types
type PriceProvider interface {
	GetPrice(symbol, exchange string) (float64, error)
	Subscribe(symbol, exchange string) error
}

type AnalyticsProvider interface {
	CalculateMetrics(request *AnalyticsRequest) (map[string]interface{}, error)
}

// Additional missing types and interfaces
type AnalyticsReport struct {
	ReportID    string
	UserID      string
	PortfolioID string
	Metrics     map[string]interface{}
	GeneratedAt time.Time
}

type ComplianceReport struct {
	ReportID        string
	UserID          string
	PortfolioID     string
	IsCompliant     bool
	Violations      []string
	Recommendations []string
	GeneratedAt     time.Time
}

type ComplianceRuleSet struct {
	RuleID      string
	Name        string
	Description string
	Rules       []ComplianceRule
}

type ComplianceRule struct {
	ID        string
	Type      string
	Condition string
	Action    string
	Severity  string
}

type LicenseInfo struct {
	LicenseID   string
	UserID      string
	Type        string
	Status      string
	ExpiryDate  time.Time
	Permissions []string
}

type ServiceConfig struct {
	ConfigID    string
	ServiceName string
	Settings    map[string]interface{}
	UpdatedAt   time.Time
}

type ReportTemplate struct {
	TemplateID string
	Name       string
	Type       string
	Format     string
	Template   string
}

// Additional component types
type PriceCache struct {
	cache map[string]float64
	mu    sync.RWMutex
}

type PricingRuleEngine struct {
	rules map[string]PricingRule
}

type PricingRule struct {
	RuleID    string
	Condition string
	Action    string
}

type ArbitrageDetector struct {
	threshold float64
	alerts    chan ArbitrageAlert
}

type ArbitrageAlert struct {
	Symbol    string
	Exchange1 string
	Exchange2 string
	PriceDiff float64
	Timestamp time.Time
}

type CrossExchangePositionManager struct {
	positions map[string]*Position
	mu        sync.RWMutex
}

type Position struct {
	PositionID    string
	Symbol        string
	Exchange      string
	Quantity      float64
	AveragePrice  float64
	CurrentPrice  float64
	UnrealizedPnL float64
}

type CrossExchangeRiskManager struct {
	riskLimits map[string]RiskLimit
	mu         sync.RWMutex
}

type RiskLimit struct {
	LimitID  string
	Type     string
	Value    float64
	Currency string
}

type CrossExchangeRebalancer struct {
	strategies map[string]RebalanceStrategy
	mu         sync.RWMutex
}

type RebalanceStrategy struct {
	StrategyID string
	Name       string
	Rules      []RebalanceRule
}

type RebalanceRule struct {
	RuleID    string
	Condition string
	Action    string
	Weight    float64
}

type UnifiedMetricsCalculator struct {
	calculators map[string]MetricsCalculator
}

type MetricsCalculator interface {
	Calculate(data interface{}) (float64, error)
}

type BenchmarkManager struct {
	benchmarks map[string]Benchmark
	mu         sync.RWMutex
}

type Benchmark struct {
	BenchmarkID string
	Name        string
	Symbol      string
	Returns     []float64
}

type AnalyticsReportGenerator struct {
	templates map[string]ReportTemplate
}

type ComplianceScreeningEngine struct {
	rules map[string]ScreeningRule
}

type ScreeningRule struct {
	RuleID   string
	Type     string
	Criteria string
	Action   string
}

type ComplianceReportingEngine struct {
	templates map[string]ComplianceTemplate
}

type ComplianceTemplate struct {
	TemplateID string
	Name       string
	Format     string
	Fields     []string
}

type ComplianceAuditTrail struct {
	entries []AuditEntry
	mu      sync.RWMutex
}

type AuditEntry struct {
	EntryID   string
	UserID    string
	Action    string
	Details   string
	Timestamp time.Time
}

type LicenseValidator struct {
	validationRules map[string]ValidationRule
}

type ValidationRule struct {
	RuleID    string
	Type      string
	Condition string
}

type LicenseUsageTracker struct {
	usage map[string]UsageMetrics
	mu    sync.RWMutex
}

type UsageMetrics struct {
	UserID     string
	LicenseID  string
	UsageCount int64
	LastUsed   time.Time
}

type LicenseRenewalManager struct {
	renewals map[string]RenewalInfo
	mu       sync.RWMutex
}

type RenewalInfo struct {
	LicenseID        string
	RenewalDate      time.Time
	NotificationSent bool
}

type ConfigStore struct {
	configs map[string]interface{}
	mu      sync.RWMutex
}

type ConfigValidator struct {
	validationRules map[string]ConfigValidationRule
}

type ConfigValidationRule struct {
	RuleID   string
	Field    string
	Type     string
	Required bool
	MinValue interface{}
	MaxValue interface{}
}

type ConfigChangeNotifier struct {
	subscribers []ConfigSubscriber
	mu          sync.RWMutex
}

type ConfigSubscriber interface {
	OnConfigChange(configID string, oldValue, newValue interface{})
}

type ReportGenerator struct {
	generators map[string]ReportGeneratorFunc
}

type ReportGeneratorFunc func(data interface{}) ([]byte, error)

type ReportScheduler struct {
	schedules map[string]ReportSchedule
	mu        sync.RWMutex
}

type ReportSchedule struct {
	ScheduleID string
	ReportType string
	Frequency  string
	NextRun    time.Time
}

type ReportDeliveryManager struct {
	deliveryMethods map[string]DeliveryMethod
}

type DeliveryMethod interface {
	Deliver(report []byte, recipient string) error
}

type AlertManager struct {
	alerts chan Alert
	rules  map[string]AlertRule
}

type Alert struct {
	AlertID   string
	Type      string
	Message   string
	Severity  string
	Timestamp time.Time
}

type AlertRule struct {
	RuleID    string
	Condition string
	Action    string
	Severity  string
}

type PerformanceLog struct {
	entries []PerformanceEntry
	mu      sync.RWMutex
}

type PerformanceEntry struct {
	EntryID   string
	Metric    string
	Value     float64
	Timestamp time.Time
}

type HealthChecker struct {
	checks map[string]HealthCheck
}

type HealthCheck interface {
	Check() (bool, error)
}

// Additional supporting types
type AssetPricingInfo struct {
	CurrentPrice     float64
	PreviousClose    float64
	DayChange        float64
	DayChangePercent float64
	Volume           int64
	LastUpdated      time.Time
}

type AssetRiskMetrics struct {
	Beta           float64
	Volatility     float64
	VaR95          float64
	Correlation    map[string]float64
	LastCalculated time.Time
}

type AssetComplianceInfo struct {
	IslamicCompliant bool
	ComplianceScore  float64
	Restrictions     []string
	LastScreened     time.Time
}

type AssetLicensingInfo struct {
	RequiredLicenses []string
	LicenseStatus    string
	ExpiryDate       time.Time
	LastChecked      time.Time
}

type UnifiedPortfolio struct {
	ID          string
	UserID      string
	Name        string
	Description string
	Assets      map[string]*PortfolioPosition
	Performance *PortfolioPerformance
	RiskMetrics *PortfolioRiskMetrics
	IslamicInfo *IslamicPortfolioInfo
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type PortfolioPosition struct {
	AssetID       string
	Symbol        string
	Exchange      string
	Quantity      float64
	AveragePrice  float64
	CurrentPrice  float64
	MarketValue   float64
	UnrealizedPnL float64
	Weight        float64
	LastUpdated   time.Time
}
