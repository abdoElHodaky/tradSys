// Package common provides unified interfaces and base implementations for all TradSys v3 services
package common

import (
	"context"
	"fmt"
	"sync"
	"time"

	basecommon "github.com/abdoElHodaky/tradSys/pkg/common"
)

// ServiceInterface is an alias to the unified service interface
// This maintains backward compatibility while using the new unified interface
type ServiceInterface = basecommon.ServiceInterface

// HealthStatus is an alias to the unified health status
type HealthStatus = basecommon.HealthStatus

// BaseService provides a standard implementation of ServiceInterface
type BaseService struct {
	name       string
	version    string
	config     interface{}
	logger     Logger
	metrics    MetricsCollector
	validator  Validator
	mu         sync.RWMutex
	isRunning  bool
	startTime  time.Time
	stopTime   time.Time
}

// NewBaseService creates a new base service instance
func NewBaseService(name, version string, logger Logger, metrics MetricsCollector, validator Validator) *BaseService {
	return &BaseService{
		name:      name,
		version:   version,
		logger:    logger,
		metrics:   metrics,
		validator: validator,
	}
}

// Initialize initializes the base service
func (bs *BaseService) Initialize(ctx context.Context) error {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	
	bs.logger.Info("Initializing service", "service", bs.name, "version", bs.version)
	bs.startTime = time.Now()
	
	return nil
}

// Start starts the base service
func (bs *BaseService) Start(ctx context.Context) error {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	
	if bs.isRunning {
		return NewServiceError(bs.name, "SERVICE_ALREADY_RUNNING", "Service is already running")
	}
	
	bs.logger.Info("Starting service", "service", bs.name)
	bs.isRunning = true
	bs.startTime = time.Now()
	
	return nil
}

// Stop stops the base service
func (bs *BaseService) Stop(ctx context.Context) error {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	
	if !bs.isRunning {
		return NewServiceError(bs.name, "SERVICE_NOT_RUNNING", "Service is not running")
	}
	
	bs.logger.Info("Stopping service", "service", bs.name)
	bs.isRunning = false
	bs.stopTime = time.Now()
	
	return nil
}

// Health returns the health status of the service
func (bs *BaseService) Health(ctx context.Context) *HealthStatus {
	bs.mu.RLock()
	defer bs.mu.RUnlock()
	
	status := "healthy"
	if !bs.isRunning {
		status = "stopped"
	}
	
	return &HealthStatus{
		Status:    status,
		Message:   fmt.Sprintf("Service %s (v%s) is %s", bs.name, bs.version, status),
		Timestamp: time.Now(),
		Details: map[string]string{
			"service":     bs.name,
			"version":     bs.version,
			"is_running":  fmt.Sprintf("%v", bs.isRunning),
			"uptime":      time.Since(bs.startTime).String(),
			"start_time":  bs.startTime.Format(time.RFC3339),
			"stop_time":   bs.stopTime.Format(time.RFC3339),
		},
	}
}

// Status returns the detailed status of the service
func (bs *BaseService) Status(ctx context.Context) *ServiceStatus {
	bs.mu.RLock()
	defer bs.mu.RUnlock()
	
	return &ServiceStatus{
		Service:     bs.name,
		Version:     bs.version,
		IsRunning:   bs.isRunning,
		StartTime:   bs.startTime,
		StopTime:    bs.stopTime,
		Uptime:      time.Since(bs.startTime),
		Config:      bs.config,
		Timestamp:   time.Now(),
	}
}

// Configure configures the service
func (bs *BaseService) Configure(config interface{}) error {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	
	if err := bs.validator.Validate(config); err != nil {
		return NewServiceError(bs.name, "INVALID_CONFIG", "Invalid configuration").WithDetail("error", err.Error())
	}
	
	bs.config = config
	bs.logger.Info("Service configured", "service", bs.name)
	
	return nil
}

// GetConfig returns the current configuration
func (bs *BaseService) GetConfig() interface{} {
	bs.mu.RLock()
	defer bs.mu.RUnlock()
	
	return bs.config
}

// Name returns the service name
func (bs *BaseService) Name() string {
	return bs.name
}

// Version returns the service version
func (bs *BaseService) Version() string {
	return bs.version
}

// IsRunning returns whether the service is running
func (bs *BaseService) IsRunning() bool {
	bs.mu.RLock()
	defer bs.mu.RUnlock()
	
	return bs.isRunning
}

// Domain-Specific Service Interfaces

// ExchangeService defines the interface for exchange services
type ExchangeService interface {
	ServiceInterface
	
	// Exchange-specific methods
	SubmitOrder(ctx context.Context, order *Order) (*OrderResponse, error)
	CancelOrder(ctx context.Context, orderID string) (*OrderResponse, error)
	GetOrder(ctx context.Context, orderID string) (*Order, error)
	GetMarketData(ctx context.Context, symbol string) (*MarketData, error)
	GetTradingStatus(ctx context.Context) (*TradingStatus, error)
	
	// Exchange information
	GetExchangeInfo() *ExchangeInfo
	GetSupportedAssets() []AssetType
}

// AssetService defines the interface for asset management services
type AssetService interface {
	ServiceInterface
	
	// Asset-specific methods
	GetAsset(ctx context.Context, id string) (*Asset, error)
	SearchAssets(ctx context.Context, query *SearchQuery) ([]*Asset, error)
	ValidateAsset(ctx context.Context, asset *Asset) error
	RegisterAsset(ctx context.Context, asset *Asset) error
	UpdateAsset(ctx context.Context, asset *Asset) error
	
	// Asset information
	GetSupportedAssetTypes() []AssetType
	GetAssetCount() int64
}

// ComplianceService defines the interface for compliance services
type ComplianceService interface {
	ServiceInterface
	
	// Compliance-specific methods
	ValidateOrder(ctx context.Context, order *Order) (*ComplianceResult, error)
	CheckRegulation(ctx context.Context, request *RegulationRequest) (*RegulationResult, error)
	AuditTransaction(ctx context.Context, transaction *Transaction) error
	GenerateReport(ctx context.Context, request *ReportRequest) (*ComplianceReport, error)
	
	// Compliance information
	GetSupportedRegulations() []string
	GetComplianceRules() []ComplianceRule
}

// RiskService defines the interface for risk assessment services
type RiskService interface {
	ServiceInterface
	
	// Risk-specific methods
	AssessOrder(ctx context.Context, order *Order) (*RiskAssessment, error)
	CalculateVaR(ctx context.Context, portfolio *Portfolio) (*VaRResult, error)
	CalculatePortfolioRisk(ctx context.Context, portfolio *Portfolio) (*PortfolioRisk, error)
	GetRiskLimits(ctx context.Context, userID string) (*RiskLimits, error)
	
	// Risk information
	GetRiskModels() []string
	GetRiskMetrics() []string
}

// AuthenticationService defines the interface for authentication services
type AuthenticationService interface {
	ServiceInterface
	
	// Authentication-specific methods
	Authenticate(ctx context.Context, credentials *Credentials) (*AuthResult, error)
	ValidateToken(ctx context.Context, token string) (*TokenClaims, error)
	RefreshToken(ctx context.Context, refreshToken string) (*AuthResult, error)
	Logout(ctx context.Context, token string) error
	
	// Authentication information
	GetSupportedMethods() []string
	GetTokenExpiry() time.Duration
}

// MarketDataService defines the interface for market data services
type MarketDataService interface {
	ServiceInterface
	
	// Market data-specific methods
	GetRealTimeData(ctx context.Context, symbol string) (*MarketData, error)
	GetHistoricalData(ctx context.Context, symbol string, period *TimePeriod) ([]*MarketData, error)
	Subscribe(ctx context.Context, symbols []string, callback func(*MarketDataUpdate)) error
	Unsubscribe(ctx context.Context, symbols []string) error
	
	// Market data information
	GetSupportedSymbols() []string
	GetDataSources() []string
}

// OrderManagementService defines the interface for order management services
type OrderManagementService interface {
	ServiceInterface
	
	// Order management-specific methods
	CreateOrder(ctx context.Context, order *Order) (*OrderResponse, error)
	UpdateOrder(ctx context.Context, orderID string, updates *OrderUpdates) (*OrderResponse, error)
	CancelOrder(ctx context.Context, orderID string) (*OrderResponse, error)
	GetOrder(ctx context.Context, orderID string) (*Order, error)
	ListOrders(ctx context.Context, filter *OrderFilter) ([]*Order, error)
	
	// Order management information
	GetSupportedOrderTypes() []OrderType
	GetOrderStatistics() *OrderStatistics
}

// Supporting Types

// ServiceStatus represents the detailed status of a service
type ServiceStatus struct {
	Service     string      `json:"service"`
	Version     string      `json:"version"`
	IsRunning   bool        `json:"is_running"`
	StartTime   time.Time   `json:"start_time"`
	StopTime    time.Time   `json:"stop_time,omitempty"`
	Uptime      time.Duration `json:"uptime"`
	Config      interface{} `json:"config,omitempty"`
	Timestamp   time.Time   `json:"timestamp"`
}

// ExchangeInfo represents exchange information
type ExchangeInfo struct {
	ID           string        `json:"id"`
	Name         string        `json:"name"`
	Region       string        `json:"region"`
	Timezone     *time.Location `json:"timezone"`
	TradingHours *TradingSchedule `json:"trading_hours"`
	AssetTypes   []AssetType   `json:"asset_types"`
}

// SearchQuery represents an asset search query
type SearchQuery struct {
	Query       string      `json:"query"`
	AssetTypes  []AssetType `json:"asset_types,omitempty"`
	Exchanges   []string    `json:"exchanges,omitempty"`
	Limit       int         `json:"limit,omitempty"`
	Offset      int         `json:"offset,omitempty"`
}

// ComplianceResult represents a compliance validation result
type ComplianceResult struct {
	IsCompliant bool     `json:"is_compliant"`
	Violations  []string `json:"violations,omitempty"`
	Warnings    []string `json:"warnings,omitempty"`
	Score       float64  `json:"score"`
	Details     map[string]interface{} `json:"details,omitempty"`
}

// RegulationRequest represents a regulation check request
type RegulationRequest struct {
	Type        string      `json:"type"`
	Data        interface{} `json:"data"`
	Jurisdiction string     `json:"jurisdiction"`
}

// RegulationResult represents a regulation check result
type RegulationResult struct {
	IsCompliant bool     `json:"is_compliant"`
	Regulation  string   `json:"regulation"`
	Violations  []string `json:"violations,omitempty"`
	Details     map[string]interface{} `json:"details,omitempty"`
}

// ReportRequest represents a compliance report request
type ReportRequest struct {
	Type      string     `json:"type"`
	StartDate time.Time  `json:"start_date"`
	EndDate   time.Time  `json:"end_date"`
	Filters   map[string]interface{} `json:"filters,omitempty"`
}

// ComplianceReport represents a compliance report
type ComplianceReport struct {
	Type        string    `json:"type"`
	Period      string    `json:"period"`
	GeneratedAt time.Time `json:"generated_at"`
	Data        interface{} `json:"data"`
	Summary     map[string]interface{} `json:"summary"`
}

// RiskAssessment represents a risk assessment result
type RiskAssessment struct {
	RiskLevel   string  `json:"risk_level"`
	RiskScore   float64 `json:"risk_score"`
	Factors     []string `json:"factors"`
	Recommendations []string `json:"recommendations"`
	Details     map[string]interface{} `json:"details,omitempty"`
}

// VaRResult represents a Value at Risk calculation result
type VaRResult struct {
	VaR95       float64   `json:"var_95"`
	VaR99       float64   `json:"var_99"`
	Currency    string    `json:"currency"`
	Period      string    `json:"period"`
	CalculatedAt time.Time `json:"calculated_at"`
	Method      string    `json:"method"`
}

// PortfolioRisk represents portfolio risk metrics
type PortfolioRisk struct {
	TotalRisk     float64 `json:"total_risk"`
	SystemicRisk  float64 `json:"systemic_risk"`
	SpecificRisk  float64 `json:"specific_risk"`
	Volatility    float64 `json:"volatility"`
	Beta          float64 `json:"beta"`
	Sharpe        float64 `json:"sharpe"`
	MaxDrawdown   float64 `json:"max_drawdown"`
}

// RiskLimits represents risk limits for a user
type RiskLimits struct {
	UserID          string  `json:"user_id"`
	MaxPosition     float64 `json:"max_position"`
	MaxNotional     float64 `json:"max_notional"`
	MaxLeverage     float64 `json:"max_leverage"`
	MaxDrawdown     float64 `json:"max_drawdown"`
	DailyLossLimit  float64 `json:"daily_loss_limit"`
}

// Credentials represents authentication credentials
type Credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Method   string `json:"method"`
	Extra    map[string]interface{} `json:"extra,omitempty"`
}

// AuthResult represents authentication result
type AuthResult struct {
	Token        string    `json:"token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	User         *User     `json:"user"`
}

// TokenClaims represents token claims
type TokenClaims struct {
	UserID    string    `json:"user_id"`
	Username  string    `json:"username"`
	Roles     []string  `json:"roles"`
	ExpiresAt time.Time `json:"expires_at"`
	IssuedAt  time.Time `json:"issued_at"`
}

// User represents a user
type User struct {
	ID       string   `json:"id"`
	Username string   `json:"username"`
	Email    string   `json:"email"`
	Roles    []string `json:"roles"`
	Status   string   `json:"status"`
}

// TimePeriod represents a time period
type TimePeriod struct {
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Interval  string    `json:"interval"`
}

// MarketDataUpdate represents a market data update
type MarketDataUpdate struct {
	Symbol    string    `json:"symbol"`
	Price     float64   `json:"price"`
	Volume    int64     `json:"volume"`
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"`
}

// OrderUpdates represents order updates
type OrderUpdates struct {
	Quantity *float64 `json:"quantity,omitempty"`
	Price    *float64 `json:"price,omitempty"`
	Status   *string  `json:"status,omitempty"`
}

// OrderFilter represents order filtering criteria
type OrderFilter struct {
	UserID     string      `json:"user_id,omitempty"`
	Symbol     string      `json:"symbol,omitempty"`
	Status     []string    `json:"status,omitempty"`
	AssetType  *AssetType  `json:"asset_type,omitempty"`
	StartDate  *time.Time  `json:"start_date,omitempty"`
	EndDate    *time.Time  `json:"end_date,omitempty"`
	Limit      int         `json:"limit,omitempty"`
	Offset     int         `json:"offset,omitempty"`
}

// OrderStatistics represents order statistics
type OrderStatistics struct {
	TotalOrders     int64   `json:"total_orders"`
	ActiveOrders    int64   `json:"active_orders"`
	CompletedOrders int64   `json:"completed_orders"`
	CancelledOrders int64   `json:"cancelled_orders"`
	AverageSize     float64 `json:"average_size"`
	TotalVolume     float64 `json:"total_volume"`
}
