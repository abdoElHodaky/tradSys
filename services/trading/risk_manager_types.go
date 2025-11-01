// Package trading provides risk management for TradSys v3
package trading

import (
	"context"
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/pkg/interfaces"
	"github.com/abdoElHodaky/tradSys/pkg/types"
)

// RiskManager provides comprehensive risk management
type RiskManager struct {
	config     *RiskManagerConfig
	riskStore  RiskStore
	calculator *RiskCalculator
	mu         sync.RWMutex
}

// RiskManagerConfig holds risk management configuration
type RiskManagerConfig struct {
	MaxPositionSize     float64
	MaxDailyVolume      float64
	MaxOrderValue       float64
	VolatilityThreshold float64
	ConcentrationLimit  float64
	EnableRealTimeCheck bool
	RiskCheckTimeout    time.Duration
}

// RiskStore interface for risk data persistence
type RiskStore interface {
	GetUserRisk(ctx context.Context, userID string) (*UserRiskProfile, error)
	UpdateUserRisk(ctx context.Context, profile *UserRiskProfile) error
	GetPositions(ctx context.Context, userID string) ([]*Position, error)
	GetDailyVolume(ctx context.Context, userID string, date time.Time) (float64, error)
	SaveRiskCheck(ctx context.Context, check *RiskCheckRecord) error
}

// RiskCalculator performs risk calculations
type RiskCalculator struct {
	volatilityCache map[string]*VolatilityData
	cacheTTL        time.Duration
	mu              sync.RWMutex
}

// UserRiskProfile represents a user's risk profile
type UserRiskProfile struct {
	UserID             string    `json:"user_id"`
	RiskTolerance      string    `json:"risk_tolerance"` // LOW, MEDIUM, HIGH
	MaxPositionSize    float64   `json:"max_position_size"`
	MaxDailyVolume     float64   `json:"max_daily_volume"`
	ConcentrationLimit float64   `json:"concentration_limit"`
	IsActive           bool      `json:"is_active"`
	LastUpdated        time.Time `json:"last_updated"`
}

// Position represents a trading position
type Position struct {
	UserID       string             `json:"user_id"`
	Symbol       string             `json:"symbol"`
	AssetType    types.AssetType    `json:"asset_type"`
	Exchange     types.ExchangeType `json:"exchange"`
	Quantity     float64            `json:"quantity"`
	AveragePrice float64            `json:"average_price"`
	MarketValue  float64            `json:"market_value"`
	UnrealizedPL float64            `json:"unrealized_pl"`
	LastUpdated  time.Time          `json:"last_updated"`
}

// RiskCheckResult represents the result of a risk check
type RiskCheckResult struct {
	Approved        bool                   `json:"approved"`
	Reason          string                 `json:"reason"`
	RiskScore       float64                `json:"risk_score"`
	Violations      []RiskViolation        `json:"violations"`
	Recommendations []string               `json:"recommendations"`
	CheckedAt       time.Time              `json:"checked_at"`
	Metadata        map[string]interface{} `json:"metadata"`
}

// RiskViolation represents a risk rule violation
type RiskViolation struct {
	Rule        string  `json:"rule"`
	Description string  `json:"description"`
	Severity    string  `json:"severity"` // LOW, MEDIUM, HIGH, CRITICAL
	Value       float64 `json:"value"`
	Limit       float64 `json:"limit"`
}

// RiskCheckRecord represents a risk check record for audit
type RiskCheckRecord struct {
	ID        string            `json:"id"`
	UserID    string            `json:"user_id"`
	OrderID   string            `json:"order_id"`
	Result    *RiskCheckResult  `json:"result"`
	Order     *interfaces.Order `json:"order"`
	CheckedAt time.Time         `json:"checked_at"`
}

// VolatilityData represents volatility information for an asset
type VolatilityData struct {
	Symbol      string    `json:"symbol"`
	Volatility  float64   `json:"volatility"`
	LastUpdated time.Time `json:"last_updated"`
}

// RiskMetrics represents aggregated risk metrics
type RiskMetrics struct {
	UserID             string    `json:"user_id"`
	TotalExposure      float64   `json:"total_exposure"`
	ConcentrationRatio float64   `json:"concentration_ratio"`
	DailyVolumeUsed    float64   `json:"daily_volume_used"`
	PositionCount      int       `json:"position_count"`
	AverageRiskScore   float64   `json:"average_risk_score"`
	LastCalculated     time.Time `json:"last_calculated"`
}

// RiskAlert represents a risk alert
type RiskAlert struct {
	ID           string                 `json:"id"`
	UserID       string                 `json:"user_id"`
	AlertType    string                 `json:"alert_type"`
	Severity     string                 `json:"severity"`
	Message      string                 `json:"message"`
	Triggered    time.Time              `json:"triggered"`
	Acknowledged bool                   `json:"acknowledged"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// RiskLimits represents various risk limits
type RiskLimits struct {
	MaxPositionSize    float64 `json:"max_position_size"`
	MaxDailyVolume     float64 `json:"max_daily_volume"`
	MaxOrderValue      float64 `json:"max_order_value"`
	ConcentrationLimit float64 `json:"concentration_limit"`
	VolatilityLimit    float64 `json:"volatility_limit"`
}

// PortfolioRisk represents portfolio-level risk metrics
type PortfolioRisk struct {
	UserID         string    `json:"user_id"`
	TotalValue     float64   `json:"total_value"`
	VaR95          float64   `json:"var_95"`
	VaR99          float64   `json:"var_99"`
	Beta           float64   `json:"beta"`
	Sharpe         float64   `json:"sharpe"`
	MaxDrawdown    float64   `json:"max_drawdown"`
	Volatility     float64   `json:"volatility"`
	LastCalculated time.Time `json:"last_calculated"`
}

// RiskEvent represents a risk-related event
type RiskEvent struct {
	ID          string                 `json:"id"`
	UserID      string                 `json:"user_id"`
	EventType   string                 `json:"event_type"`
	Description string                 `json:"description"`
	Impact      string                 `json:"impact"`
	Timestamp   time.Time              `json:"timestamp"`
	Data        map[string]interface{} `json:"data"`
}

// RiskToleranceLevel represents risk tolerance levels
type RiskToleranceLevel string

const (
	RiskToleranceLow        RiskToleranceLevel = "LOW"
	RiskToleranceMedium     RiskToleranceLevel = "MEDIUM"
	RiskToleranceHigh       RiskToleranceLevel = "HIGH"
	RiskToleranceAggressive RiskToleranceLevel = "AGGRESSIVE"
)

// ViolationSeverity represents violation severity levels
type ViolationSeverity string

const (
	SeverityLow      ViolationSeverity = "LOW"
	SeverityMedium   ViolationSeverity = "MEDIUM"
	SeverityHigh     ViolationSeverity = "HIGH"
	SeverityCritical ViolationSeverity = "CRITICAL"
)

// AlertType represents different types of risk alerts
type AlertType string

const (
	AlertTypePositionLimit AlertType = "POSITION_LIMIT"
	AlertTypeVolumeLimit   AlertType = "VOLUME_LIMIT"
	AlertTypeConcentration AlertType = "CONCENTRATION"
	AlertTypeVolatility    AlertType = "VOLATILITY"
	AlertTypeMarginCall    AlertType = "MARGIN_CALL"
	AlertTypeRiskThreshold AlertType = "RISK_THRESHOLD"
)

// RiskCheckType represents different types of risk checks
type RiskCheckType string

const (
	CheckTypePreTrade  RiskCheckType = "PRE_TRADE"
	CheckTypePostTrade RiskCheckType = "POST_TRADE"
	CheckTypeRealTime  RiskCheckType = "REAL_TIME"
	CheckTypeEndOfDay  RiskCheckType = "END_OF_DAY"
)

// RiskStatus represents the status of risk management
type RiskStatus string

const (
	StatusActive    RiskStatus = "ACTIVE"
	StatusInactive  RiskStatus = "INACTIVE"
	StatusSuspended RiskStatus = "SUSPENDED"
	StatusBlocked   RiskStatus = "BLOCKED"
)
