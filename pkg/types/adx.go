// 🎯 **ADX Service Types**
// Generated using TradSys Code Splitting Standards
//
// This file contains type definitions, constants, and data structures
// for the Abu Dhabi Exchange (ADX) Service component. All types follow the established
// naming conventions and include comprehensive documentation for Islamic finance integration.
//
// Performance Requirements: Standard latency, Islamic compliance focus
// File size limit: 300 lines

package types

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





// TradingSchedule represents trading hours and schedules
type TradingSchedule struct {
	Timezone     string                   `json:"timezone"`
	MarketHours  map[string]*TradingHours `json:"market_hours"` // day of week -> hours
	Holidays     []time.Time              `json:"holidays"`
	SpecialHours map[string]*TradingHours `json:"special_hours"` // special dates
	LastUpdated  time.Time                `json:"last_updated"`
}

// IslamicAuditTrail tracks Islamic finance transactions for audit
type IslamicAuditTrail struct {
	records     []AuditRecord
	checkpoints map[string]time.Time
	mu          sync.RWMutex
}

// AuditRecord represents a single audit record
type AuditRecord struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Details   string    `json:"details"`
	Timestamp time.Time `json:"timestamp"`
}

// ReportingRequirements defines Islamic regulatory reporting standards
type ReportingRequirements struct {
	ReportTypes    []string
	Frequency      string
	Authorities    []string
	Templates      map[string]ReportTemplate
	Deadlines      map[string]time.Time
}



// LicensingRequirements defines licensing for Islamic finance operations
type LicensingRequirements struct {
	RequiredLicenses []string
	Authorities      []string
	ExpiryDates      map[string]time.Time
	ComplianceLevel  string
}

// SukukPricingEngine handles Sukuk pricing
type SukukPricingEngine struct {
	pricingModels map[string]PricingModel
	mu            sync.RWMutex
}

// PricingModel represents a pricing model
type PricingModel struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	Parameters map[string]float64 `json:"parameters"`
}

// IslamicIndexCalculator calculates Islamic indices
type IslamicIndexCalculator struct {
	indices map[string]IndexDefinition
	mu      sync.RWMutex
}

// IndexDefinition defines an Islamic index
type IndexDefinition struct {
	Name        string    `json:"name"`
	Components  []string  `json:"components"`
	Weights     []float64 `json:"weights"`
	LastUpdated time.Time `json:"last_updated"`
}

// ComplianceDataStore stores compliance data
type ComplianceDataStore struct {
	data map[string]interface{}
	mu   sync.RWMutex
}

// IslamicOrderValidator validates Islamic orders
type IslamicOrderValidator struct {
	rules map[string]ValidationRule
	mu    sync.RWMutex
}

// ScreeningResults represents screening results
type ScreeningResults struct {
	AssetID     string    `json:"asset_id"`
	IsCompliant bool      `json:"is_compliant"`
	Reasons     []string  `json:"reasons"`
	Score       float64   `json:"score"`
	Timestamp   time.Time `json:"timestamp"`
}

// ShariaScholar represents a Sharia scholar
type ShariaScholar struct {
	Name           string
	Qualification  string
	Specialization []string
	IsActive       bool
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
	rules        map[string]ComplianceRule
	reportingReq ReportingRequirements
	licensing    LicensingRequirements
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
	realTimeFeeds  map[string]*DataFeed
	islamicFeeds   map[string]*IslamicDataFeed
	sukukPricing   *SukukPricingEngine
	islamicIndices *IslamicIndexCalculator
	historicalData *HistoricalDataStore
	complianceData *ComplianceDataStore
	mu             sync.RWMutex
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

// IslamicYieldCalculator calculates yields for Islamic instruments
type IslamicYieldCalculator struct {
	models map[string]YieldModel
	mu     sync.RWMutex
}

// YieldModel represents a yield calculation model
type YieldModel struct {
	Name       string                 `json:"name"`
	Type       string                 `json:"type"`
	Parameters map[string]float64     `json:"parameters"`
	Formula    string                 `json:"formula"`
}

// SukukRiskEngine assesses risk for Sukuk instruments
type SukukRiskEngine struct {
	riskModels map[string]RiskModel
	mu         sync.RWMutex
}

// RiskModel represents a risk assessment model
type RiskModel struct {
	Name        string             `json:"name"`
	Type        string             `json:"type"`
	Factors     []string           `json:"factors"`
	Weights     map[string]float64 `json:"weights"`
	Threshold   float64            `json:"threshold"`
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
	TypeID     string
	Name       string
	Structure  string
	Underlying string
	Maturity   time.Duration
	MinAmount  float64
	Currency   string
	IsActive   bool
}

// ADXExecutionEngine handles order execution for ADX
type ADXExecutionEngine struct {
	executors   map[string]OrderExecutor
	strategies  map[string]ExecutionStrategy
	mu          sync.RWMutex
}

// OrderExecutor represents an order execution interface
type OrderExecutor interface {
	Execute(order *ADXOrder) (*Trade, error)
	Cancel(orderID string) error
	GetStatus(orderID string) (OrderStatus, error)
}

// ExecutionStrategy represents an execution strategy
type ExecutionStrategy struct {
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	Parameters  map[string]interface{} `json:"parameters"`
	IsActive    bool                   `json:"is_active"`
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

// IslamicRisk represents Islamic finance risk factors
type IslamicRisk struct {
	RiskID      string             `json:"risk_id"`
	Type        string             `json:"type"`
	Level       string             `json:"level"`
	Description string             `json:"description"`
	Factors     []string           `json:"factors"`
	Mitigation  map[string]string  `json:"mitigation"`
	Score       float64            `json:"score"`
}

// ComplianceRiskEngine assesses compliance risks
type ComplianceRiskEngine struct {
	rules       map[string]ComplianceRule
	assessments map[string]RiskAssessment
	mu          sync.RWMutex
}

// RiskAssessment represents a risk assessment result
type RiskAssessment struct {
	AssetID     string             `json:"asset_id"`
	RiskScore   float64            `json:"risk_score"`
	RiskLevel   string             `json:"risk_level"`
	Factors     []string           `json:"factors"`
	Timestamp   time.Time          `json:"timestamp"`
	Details     map[string]interface{} `json:"details"`
}

// ADXRiskEngine handles risk management for ADX
type ADXRiskEngine struct {
	riskLimits     map[string]RiskLimit
	islamicRisks   map[string]IslamicRisk
	riskCalculator *RiskCalculator
	complianceRisk *ComplianceRiskEngine
	mu             sync.RWMutex
}

// IslamicFundManager manages Islamic mutual funds
type IslamicFundManager struct {
	funds       map[string]*IslamicFund
	strategies  map[string]FundStrategy
	allocations map[string]AssetAllocation
	mu          sync.RWMutex
}

// FundStrategy represents a fund management strategy
type FundStrategy struct {
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	Objective   string                 `json:"objective"`
	Parameters  map[string]interface{} `json:"parameters"`
	IsActive    bool                   `json:"is_active"`
}

// AssetAllocation represents asset allocation for a fund
type AssetAllocation struct {
	AssetType   string  `json:"asset_type"`
	Percentage  float64 `json:"percentage"`
	MinWeight   float64 `json:"min_weight"`
	MaxWeight   float64 `json:"max_weight"`
}

// IslamicPerformanceCalculator calculates performance metrics for Islamic instruments
type IslamicPerformanceCalculator struct {
	metrics    map[string]PerformanceMetric
	benchmarks map[string]Benchmark
	mu         sync.RWMutex
}

// PerformanceMetric represents a performance metric
type PerformanceMetric struct {
	Name        string    `json:"name"`
	Value       float64   `json:"value"`
	Period      string    `json:"period"`
	Benchmark   string    `json:"benchmark"`
	Timestamp   time.Time `json:"timestamp"`
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
	DefaultNisabThreshold = 85.0  // grams of gold equivalent
	DefaultZakatRate      = 0.025 // 2.5%

	// ADX specific constants
	ADXExchangeID = "ADX"
	ADXRegion     = "UAE"
	ADXTimezone   = "Asia/Dubai"
)

// Error definitions
var (
	ErrInvalidShariaCompliance = fmt.Errorf("invalid Sharia compliance")
	ErrSukukNotFound           = fmt.Errorf("Sukuk not found")
	ErrIslamicOrderRejected    = fmt.Errorf("Islamic order rejected")
	ErrComplianceCheckFailed   = fmt.Errorf("compliance check failed")
	ErrZakatCalculationFailed  = fmt.Errorf("Zakat calculation failed")
	ErrADXConnectionFailed     = fmt.Errorf("ADX connection failed")
	ErrInvalidAssetType        = fmt.Errorf("invalid asset type")
	ErrShariaRuleViolation     = fmt.Errorf("Sharia rule violation")
)
