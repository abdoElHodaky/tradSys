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

	"github.com/abdoElHodaky/tradSys/internal/core/matching"
	"github.com/abdoElHodaky/tradSys/internal/trading/types"
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
	performanceMonitor *ADXPerformanceMonitor
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
	riskLimits     map[string]RiskLimit
	islamicRisks   map[string]IslamicRisk
	riskCalculator *RiskCalculator
	complianceRisk *ComplianceRiskEngine
	mu             sync.RWMutex
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

// ADXPerformanceMonitor monitors ADX service performance
type ADXPerformanceMonitor struct {
	metrics         map[string]*PerformanceMetric
	islamicMetrics  map[string]*IslamicMetric
	alertManager    *AlertManager
	reportGenerator *ReportGenerator
	mu              sync.RWMutex
}

// PerformanceMetric represents a performance metric
type PerformanceMetric struct {
	MetricID  string
	Name      string
	Value     float64
	Unit      string
	Timestamp time.Time
	IsIslamic bool
}

// IslamicMetric represents Islamic-specific metrics
type IslamicMetric struct {
	*PerformanceMetric
	ComplianceScore float64
	ShariaRating    string
	ZakatImpact     float64
}

// Islamic Finance Infrastructure Types

// ScreeningEngine handles Sharia compliance screening
type ScreeningEngine struct {
	rules       map[string]ShariaRule
	validators  []ComplianceValidator
	cache       map[string]bool
	mu          sync.RWMutex
}

// ComplianceDatabase stores Islamic compliance data
type ComplianceDatabase struct {
	rules           map[string]ShariaRule
	auditRecords    []AuditRecord
	complianceCache map[string]ComplianceStatus
	mu              sync.RWMutex
}

// IslamicAuditTrail tracks Islamic finance transactions for audit
type IslamicAuditTrail struct {
	records     []AuditRecord
	checkpoints map[string]time.Time
	mu          sync.RWMutex
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

// SukukPricingEngine calculates prices for Islamic bonds (Sukuk)
type SukukPricingEngine struct {
	pricingModels map[string]PricingModel
	yieldCurves   map[string]YieldCurve
	riskFactors   []RiskFactor
	mu            sync.RWMutex
}

// IslamicIndexCalculator computes Sharia-compliant indices
type IslamicIndexCalculator struct {
	indices       map[string]IslamicIndex
	constituents  map[string][]string
	weights       map[string]float64
	screeningData map[string]ScreeningResults
	mu            sync.RWMutex
}

// ComplianceDataStore stores compliance-related market data
type ComplianceDataStore struct {
	complianceData map[string]ComplianceRecord
	historicalData map[string][]ComplianceRecord
	alerts         []ComplianceAlert
	mu             sync.RWMutex
}

// IslamicOrderValidator validates orders against Islamic finance rules
type IslamicOrderValidator struct {
	rules       []ValidationRule
	screenings  map[string]ScreeningResults
	exceptions  []string
	mu          sync.RWMutex
}

// ScreeningResults contains results of Sharia compliance screening
type ScreeningResults struct {
	Symbol          string
	IsCompliant     bool
	ComplianceScore float64
	Violations      []string
	Recommendations []string
	LastUpdated     time.Time
}

// Supporting types for Islamic finance infrastructure

// ComplianceValidator validates specific compliance aspects
type ComplianceValidator struct {
	Name        string
	Type        string
	Rules       []string
	IsActive    bool
}

// AuditRecord represents an audit trail record
type AuditRecord struct {
	ID          string
	Timestamp   time.Time
	Action      string
	UserID      string
	Details     map[string]interface{}
	ComplianceStatus string
}

// ComplianceStatus represents compliance state
type ComplianceStatus struct {
	Status      string
	Score       float64
	LastChecked time.Time
	Issues      []string
}

// ReportTemplate defines structure for regulatory reports
type ReportTemplate struct {
	Name        string
	Format      string
	Fields      []string
	Validations []string
}

// PricingModel for Sukuk pricing
type PricingModel struct {
	Name       string
	Type       string
	Parameters map[string]float64
	IsActive   bool
}

// YieldCurve for Islamic bond pricing
type YieldCurve struct {
	Currency    string
	Points      []YieldPoint
	LastUpdated time.Time
}

// YieldPoint represents a point on yield curve
type YieldPoint struct {
	Maturity time.Duration
	Yield    float64
}

// RiskFactor for pricing calculations
type RiskFactor struct {
	Name   string
	Value  float64
	Weight float64
}

// IslamicIndex represents a Sharia-compliant index
type IslamicIndex struct {
	Symbol      string
	Name        string
	Value       float64
	Change      float64
	LastUpdated time.Time
}

// ComplianceRecord stores compliance data
type ComplianceRecord struct {
	Symbol      string
	Timestamp   time.Time
	Status      string
	Score       float64
	Details     map[string]interface{}
}

// ComplianceAlert represents compliance alerts
type ComplianceAlert struct {
	ID          string
	Symbol      string
	Type        string
	Severity    string
	Message     string
	Timestamp   time.Time
	IsResolved  bool
}

// ValidationRule for Islamic order validation
type ValidationRule struct {
	Name        string
	Type        string
	Condition   string
	Action      string
	IsActive    bool
}

// Additional Islamic Finance Types

// IslamicYieldCalculator calculates yields for Islamic instruments
type IslamicYieldCalculator struct {
	yieldModels map[string]YieldModel
	benchmarks  map[string]float64
	mu          sync.RWMutex
}

// SukukRiskEngine assesses risk for Sukuk instruments
type SukukRiskEngine struct {
	riskModels   map[string]RiskModel
	riskFactors  []RiskFactor
	riskLimits   map[string]float64
	mu           sync.RWMutex
}

// ADXExecutionEngine handles order execution for ADX
type ADXExecutionEngine struct {
	orderQueue    []types.Order
	executionRules map[string]ExecutionRule
	latencyTracker *LatencyTracker
	mu             sync.RWMutex
}

// RiskLimit defines risk limits for trading
type RiskLimit struct {
	Symbol      string
	MaxPosition float64
	MaxValue    float64
	MaxLoss     float64
	IsActive    bool
}

// IslamicRisk represents Islamic-specific risk factors
type IslamicRisk struct {
	ShariaRisk      float64
	ComplianceRisk  float64
	ReputationRisk  float64
	RegulatoryRisk  float64
}

// RiskCalculator calculates various risk metrics
type RiskCalculator struct {
	models      map[string]RiskModel
	parameters  map[string]float64
	mu          sync.RWMutex
}

// ComplianceRiskEngine assesses compliance-related risks
type ComplianceRiskEngine struct {
	riskRules    []ComplianceRiskRule
	riskScores   map[string]float64
	alerts       []RiskAlert
	mu           sync.RWMutex
}

// Supporting types for additional functionality

// YieldModel for yield calculations
type YieldModel struct {
	Name       string
	Type       string
	Parameters map[string]float64
	IsActive   bool
}

// RiskModel for risk calculations
type RiskModel struct {
	Name       string
	Type       string
	Factors    []string
	Weights    map[string]float64
	IsActive   bool
}

// Order is available in pkg/types

// ExecutionRule defines order execution rules
type ExecutionRule struct {
	Name      string
	Condition string
	Action    string
	Priority  int
	IsActive  bool
}

// LatencyTracker tracks execution latency
type LatencyTracker struct {
	measurements map[string]time.Duration
	averages     map[string]time.Duration
	mu           sync.RWMutex
}

// ComplianceRiskRule defines compliance risk rules
type ComplianceRiskRule struct {
	ID          string
	Name        string
	Type        string
	Severity    string
	Condition   string
	Action      string
	IsActive    bool
}

// RiskAlert represents risk alerts
type RiskAlert struct {
	ID        string
	Type      string
	Severity  string
	Message   string
	Symbol    string
	Timestamp time.Time
	IsActive  bool
}

// Additional Missing Types for ADX Core

// IslamicFundManager manages Islamic investment funds
type IslamicFundManager struct {
	funds       map[string]*IslamicFund
	performance map[string]*PerformanceData
	compliance  *matching.ComplianceEngine
	mu          sync.RWMutex
}

// IslamicPerformanceCalculator calculates performance metrics for Islamic instruments
type IslamicPerformanceCalculator struct {
	benchmarks    map[string]float64
	calculations  map[string]*PerformanceResult
	riskMetrics   map[string]*RiskMetrics
	mu            sync.RWMutex
}

// AlertManager manages trading and compliance alerts
type AlertManager struct {
	alerts      []Alert
	subscribers map[string][]AlertSubscriber
	rules       map[string]AlertRule
	mu          sync.RWMutex
}

// ReportGenerator generates various reports for Islamic finance
type ReportGenerator struct {
	templates   map[string]*ReportTemplate
	generators  map[string]ReportFunc
	cache       map[string]*GeneratedReport
	mu          sync.RWMutex
}

// SukukData represents Sukuk (Islamic bond) data
type SukukData struct {
	ID           string
	Name         string
	Issuer       string
	Structure    string
	Maturity     time.Time
	FaceValue    float64
	CurrentPrice float64
	YieldRate    float64
	Rating       string
	IsActive     bool
}

// IslamicFundData represents Islamic fund data
type IslamicFundData struct {
	ID             string
	Name           string
	Manager        string
	Strategy       string
	NAV            float64
	TotalAssets    float64
	InceptionDate  time.Time
	ExpenseRatio   float64
	MinInvestment  float64
	IsActive       bool
}

// Portfolio represents an investment portfolio
type Portfolio struct {
	ID          string
	UserID      string
	Name        string
	Holdings    map[string]*Holding
	TotalValue  float64
	Cash        float64
	Performance *PerformanceData
	IsIslamic   bool
}

// ZakatCalculation represents Zakat calculation data
type ZakatCalculation struct {
	UserID        string
	Year          int
	TotalWealth   float64
	Nisab         float64
	ZakatDue      float64
	Assets        map[string]float64
	Liabilities   map[string]float64
	CalculatedAt  time.Time
}

// ComplianceReport represents compliance reporting data
type ComplianceReport struct {
	ID           string
	Type         string
	Period       string
	GeneratedAt  time.Time
	Data         map[string]interface{}
	Status       string
	Violations   []ComplianceViolation
	Recommendations []string
}

// Supporting types for the above

// PerformanceData represents performance metrics
type PerformanceData struct {
	Returns    map[string]float64
	Volatility float64
	SharpeRatio float64
	MaxDrawdown float64
}

// PerformanceResult represents calculation results
type PerformanceResult struct {
	Period   string
	Return   float64
	Risk     float64
	Metrics  map[string]float64
}

// RiskMetrics represents risk calculation results
type RiskMetrics struct {
	VaR         float64
	CVaR        float64
	Beta        float64
	Correlation map[string]float64
}

// Alert represents system alerts
type Alert struct {
	ID        string
	Type      string
	Severity  string
	Message   string
	Timestamp time.Time
	IsRead    bool
}

// AlertSubscriber represents alert subscribers
type AlertSubscriber struct {
	ID       string
	UserID   string
	Type     string
	Channels []string
}

// AlertRule represents alert rules
type AlertRule struct {
	ID        string
	Name      string
	Condition string
	Action    string
	IsActive  bool
}

// ReportFunc represents report generation function
type ReportFunc func(data interface{}) (*GeneratedReport, error)

// GeneratedReport represents a generated report
type GeneratedReport struct {
	ID          string
	Type        string
	Data        []byte
	Format      string
	GeneratedAt time.Time
}

// Holding represents a portfolio holding
type Holding struct {
	Symbol   string
	Quantity float64
	Price    float64
	Value    float64
}

// ComplianceViolation represents compliance violations
type ComplianceViolation struct {
	ID          string
	Type        string
	Severity    string
	Description string
	Symbol      string
	Timestamp   time.Time
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
