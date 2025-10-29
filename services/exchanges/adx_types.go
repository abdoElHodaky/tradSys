// 🎯 **ADX Service Types**
// Generated using TradSys Code Splitting Standards
//
// This file contains type definitions, constants, and data structures
// for the Abu Dhabi Exchange (ADX) Service component. All types follow the established
// naming conventions and include comprehensive documentation for Islamic finance integration.
//
// Performance Requirements: Standard latency, Islamic compliance focus
// File size limit: 300 lines

package exchanges

import (
	"fmt"
	"sync"
	"time"
)

// ComplianceLevel defines Sharia compliance levels
type ComplianceLevel int

const (
	ComplianceLevelHalal ComplianceLevel = iota
	ComplianceLevelDoubtful
	ComplianceLevelHaram
	ComplianceLevelUnderReview
)

// ADXService provides Abu Dhabi Exchange integration with Islamic finance focus
type ADXService struct {
	exchangeID         string
	region             string
	assetTypes         []AssetType
	tradingHours       *TradingSchedule
	islamicCompliance  *IslamicCompliance
	uaeCompliance      *UAECompliance
	shariaBoards       []*ShariaBoard
	zakatCalculator    *ZakatCalculator
	languageSupport    []string
	connector          *ADXConnector
	marketData         *ADXMarketData
	orderManager       *ADXOrderManager
	riskEngine         *ADXRiskEngine
	sukukService       *SukukService
	islamicFundService *IslamicFundService
	performanceMonitor *PerformanceMonitor
	mu                 sync.RWMutex
}

// IslamicCompliance handles Sharia compliance for ADX
type IslamicCompliance struct {
	shariaRules     map[string]ShariaRule
	screeningEngine *ScreeningEngine
	complianceDB    *ComplianceDatabase
	auditTrail      *IslamicAuditTrail
	mu              sync.RWMutex
}

// ShariaRule represents an Islamic finance rule
type ShariaRule struct {
	RuleID          string
	Description     string
	ShariaBoard     string
	AssetTypes      []AssetType
	Validator       func(interface{}) bool
	ComplianceLevel ComplianceLevel
	LastUpdated     time.Time
}

// ShariaBoard represents a Sharia supervisory board
type ShariaBoard struct {
	ID          string
	Name        string
	Country     string
	Scholars    []ShariaScholar
	Methodology string
	IsActive    bool
	LastReview  time.Time
}

// ShariaScholar represents a Sharia scholar
type ShariaScholar struct {
	Name           string
	Qualification  string
	Specialization []string
	IsActive       bool
}

// ZakatCalculator calculates Zakat for Islamic investments
type ZakatCalculator struct {
	zakatRates     map[AssetType]float64
	nisabThreshold float64
	currency       string
	mu             sync.RWMutex
}

// UAECompliance handles UAE regulatory compliance
type UAECompliance struct {
	regulatoryRules map[string]ComplianceRule
	adgmRules       map[string]ComplianceRule
	difcRules       map[string]ComplianceRule
	sca             *SCACompliance // Securities and Commodities Authority
	mu              sync.RWMutex
}

// SCACompliance handles SCA (Securities and Commodities Authority) compliance
type SCACompliance struct {
	rules       map[string]ComplianceRule
	reportingReq ReportingRequirements
	licensing   LicensingRequirements
}

// ADXConnector handles connection to Abu Dhabi Exchange
type ADXConnector struct {
	endpoint        string
	apiKey          string
	islamicEndpoint string
	connectionPool  *ConnectionPool
	rateLimiter     *RateLimiter
	retryPolicy     *RetryPolicy
	healthChecker   *HealthChecker
	mu              sync.RWMutex
}

// ADXMarketData handles Islamic-focused market data from ADX
type ADXMarketData struct {
	realTimeFeeds     map[string]*DataFeed
	islamicFeeds      map[string]*IslamicDataFeed
	sukukPricing      *SukukPricingEngine
	islamicIndices    *IslamicIndexCalculator
	historicalData    *HistoricalDataStore
	complianceData    *ComplianceDataStore
	mu                sync.RWMutex
}

// IslamicDataFeed represents Islamic-compliant data feed
type IslamicDataFeed struct {
	Symbol           string
	AssetType        AssetType
	ComplianceStatus ComplianceLevel
	ShariaBoard      string
	LastScreened     time.Time
	IsActive         bool
}

// SukukService handles Sukuk (Islamic bonds) trading
type SukukService struct {
	sukukTypes      map[string]SukukType
	pricingEngine   *SukukPricingEngine
	yieldCalculator *IslamicYieldCalculator
	riskAssessment  *SukukRiskEngine
	mu              sync.RWMutex
}

// SukukType defines types of Sukuk
type SukukType struct {
	TypeID      string
	Name        string
	Structure   string
	Underlying  string
	Maturity    time.Duration
	MinAmount   float64
	Currency    string
	IsActive    bool
}

// ADXOrderManager handles order management for ADX
type ADXOrderManager struct {
	orders          map[string]*ADXOrder
	islamicOrders   map[string]*IslamicOrder
	orderValidator  *IslamicOrderValidator
	executionEngine *ADXExecutionEngine
	mu              sync.RWMutex
}

// ADXOrder represents an order on ADX
type ADXOrder struct {
	OrderID         string
	Symbol          string
	AssetType       AssetType
	Side            OrderSide
	Quantity        float64
	Price           float64
	OrderType       OrderType
	TimeInForce     TimeInForce
	IslamicFlag     bool
	ComplianceCheck bool
	Status          OrderStatus
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// IslamicOrder represents an Islamic-compliant order
type IslamicOrder struct {
	*ADXOrder
	ShariaApproval   bool
	ComplianceLevel  ComplianceLevel
	ShariaBoard      string
	ZakatApplicable  bool
	ScreeningResults *ScreeningResults
}

// ADXRiskEngine handles risk management for ADX
type ADXRiskEngine struct {
	riskLimits      map[string]RiskLimit
	islamicRisks    map[string]IslamicRisk
	riskCalculator  *RiskCalculator
	complianceRisk  *ComplianceRiskEngine
	mu              sync.RWMutex
}

// IslamicFundService handles Islamic mutual funds
type IslamicFundService struct {
	funds           map[string]*IslamicFund
	fundManager     *IslamicFundManager
	performanceCalc *IslamicPerformanceCalculator
	mu              sync.RWMutex
}

// IslamicFund represents an Islamic mutual fund
type IslamicFund struct {
	FundID          string
	Name            string
	FundType        string
	ShariaBoard     string
	ComplianceLevel ComplianceLevel
	NAV             float64
	TotalAssets     float64
	InceptionDate   time.Time
	IsActive        bool
}

// PerformanceMonitor monitors ADX service performance
type PerformanceMonitor struct {
	metrics         map[string]*PerformanceMetric
	islamicMetrics  map[string]*IslamicMetric
	alertManager    *AlertManager
	reportGenerator *ReportGenerator
	mu              sync.RWMutex
}

// PerformanceMetric represents a performance metric
type PerformanceMetric struct {
	MetricID    string
	Name        string
	Value       float64
	Unit        string
	Timestamp   time.Time
	IsIslamic   bool
}

// IslamicMetric represents Islamic-specific metrics
type IslamicMetric struct {
	*PerformanceMetric
	ComplianceScore float64
	ShariaRating    string
	ZakatImpact     float64
}

// Configuration constants
const (
	DefaultConnectionTimeout = 30 * time.Second
	DefaultRequestTimeout    = 10 * time.Second
	DefaultRetryAttempts     = 3
	DefaultRateLimit         = 1000 // requests per minute
	
	// Islamic finance constants
	DefaultNisabThreshold = 85.0 // grams of gold equivalent
	DefaultZakatRate      = 0.025 // 2.5%
	
	// ADX specific constants
	ADXExchangeID = "ADX"
	ADXRegion     = "UAE"
	ADXTimezone   = "Asia/Dubai"
)

// Error definitions
var (
	ErrInvalidShariaCompliance = fmt.Errorf("invalid Sharia compliance")
	ErrSukukNotFound          = fmt.Errorf("Sukuk not found")
	ErrIslamicOrderRejected   = fmt.Errorf("Islamic order rejected")
	ErrComplianceCheckFailed  = fmt.Errorf("compliance check failed")
	ErrZakatCalculationFailed = fmt.Errorf("Zakat calculation failed")
	ErrADXConnectionFailed    = fmt.Errorf("ADX connection failed")
	ErrInvalidAssetType       = fmt.Errorf("invalid asset type")
	ErrShariaRuleViolation    = fmt.Errorf("Sharia rule violation")
)
