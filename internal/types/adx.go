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

	"github.com/abdoElHodaky/tradSys/internal/monitoring"
	"github.com/abdoElHodaky/tradSys/internal/services/common"
	"github.com/abdoElHodaky/tradSys/internal/services/exchanges"
)

// ComplianceLevel is defined in sharia.go to avoid duplication

// ADXService provides Abu Dhabi Exchange integration with Islamic finance focus
type ADXService struct {
	exchangeID         string
	region             string
	assetTypes         []common.AssetType
	tradingHours       *common.TradingSchedule
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
	auditTrail      *exchanges.IslamicAuditTrail
	mu              sync.RWMutex
}

// ShariaRule is defined in sharia.go to avoid duplication

// ShariaBoard is defined in sharia.go to avoid duplication

// ShariaScholar represents a Sharia scholar
type ShariaScholar struct {
	Name           string
	Qualification  string
	Specialization []string
	IsActive       bool
}

// ZakatCalculator is defined in sharia.go to avoid duplication

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
	reportingReq exchanges.ReportingRequirements
	licensing    exchanges.LicensingRequirements
}

// ADXConnector handles connection to Abu Dhabi Exchange
type ADXConnector struct {
	endpoint        string
	apiKey          string
	islamicEndpoint string
	connectionPool  *ConnectionPool
	rateLimiter     *RateLimiter
	retryPolicy     *RetryPolicy
	healthChecker   *monitoring.HealthChecker
	mu              sync.RWMutex
}

// ADXMarketData handles Islamic-focused market data from ADX
type ADXMarketData struct {
	realTimeFeeds  map[string]*DataFeed
	islamicFeeds   map[string]*IslamicDataFeed
	sukukPricing   *exchanges.SukukPricingEngine
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

// IslamicFundService is available in pkg/types

// IslamicFund is available in pkg/types

// PerformanceMonitor is available in pkg/types

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
