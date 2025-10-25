// Package common provides unified types for all TradSys v3 services
package common

import (
	"time"
)

// AssetType represents different types of assets
type AssetType int

const (
	AssetTypeStock AssetType = iota
	AssetTypeBond
	AssetTypeETF
	AssetTypeREIT
	AssetTypeMutualFund
	AssetTypeCommodity
	AssetTypeCrypto
	AssetTypeForex
	AssetTypeGovernmentBond
	AssetTypeCorporateBond
	AssetTypeIslamicInstrument
	AssetTypeSukuk
	AssetTypeIslamicFund
	AssetTypeIslamicREIT
)

// String returns the string representation of AssetType
func (at AssetType) String() string {
	switch at {
	case AssetTypeStock:
		return "STOCK"
	case AssetTypeBond:
		return "BOND"
	case AssetTypeETF:
		return "ETF"
	case AssetTypeREIT:
		return "REIT"
	case AssetTypeMutualFund:
		return "MUTUAL_FUND"
	case AssetTypeCommodity:
		return "COMMODITY"
	case AssetTypeCrypto:
		return "CRYPTO"
	case AssetTypeForex:
		return "FOREX"
	case AssetTypeGovernmentBond:
		return "GOVERNMENT_BOND"
	case AssetTypeCorporateBond:
		return "CORPORATE_BOND"
	case AssetTypeIslamicInstrument:
		return "ISLAMIC_INSTRUMENT"
	case AssetTypeSukuk:
		return "SUKUK"
	case AssetTypeIslamicFund:
		return "ISLAMIC_FUND"
	case AssetTypeIslamicREIT:
		return "ISLAMIC_REIT"
	default:
		return "UNKNOWN"
	}
}

// OrderType represents different types of orders
type OrderType int

const (
	OrderTypeMarket OrderType = iota
	OrderTypeLimit
	OrderTypeStop
	OrderTypeStopLimit
	OrderTypeTrailingStop
	OrderTypeIceberg
	OrderTypeFillOrKill
	OrderTypeImmediateOrCancel
	OrderTypeGoodTillCancelled
	OrderTypeGoodTillDate
)

// String returns the string representation of OrderType
func (ot OrderType) String() string {
	switch ot {
	case OrderTypeMarket:
		return "MARKET"
	case OrderTypeLimit:
		return "LIMIT"
	case OrderTypeStop:
		return "STOP"
	case OrderTypeStopLimit:
		return "STOP_LIMIT"
	case OrderTypeTrailingStop:
		return "TRAILING_STOP"
	case OrderTypeIceberg:
		return "ICEBERG"
	case OrderTypeFillOrKill:
		return "FILL_OR_KILL"
	case OrderTypeImmediateOrCancel:
		return "IMMEDIATE_OR_CANCEL"
	case OrderTypeGoodTillCancelled:
		return "GOOD_TILL_CANCELLED"
	case OrderTypeGoodTillDate:
		return "GOOD_TILL_DATE"
	default:
		return "UNKNOWN"
	}
}

// OrderSide represents the side of an order
type OrderSide int

const (
	OrderSideBuy OrderSide = iota
	OrderSideSell
)

// String returns the string representation of OrderSide
func (os OrderSide) String() string {
	switch os {
	case OrderSideBuy:
		return "BUY"
	case OrderSideSell:
		return "SELL"
	default:
		return "UNKNOWN"
	}
}

// OrderStatus represents the status of an order
type OrderStatus int

const (
	OrderStatusPending OrderStatus = iota
	OrderStatusNew
	OrderStatusPartiallyFilled
	OrderStatusFilled
	OrderStatusCancelled
	OrderStatusRejected
	OrderStatusExpired
	OrderStatusSuspended
)

// String returns the string representation of OrderStatus
func (os OrderStatus) String() string {
	switch os {
	case OrderStatusPending:
		return "PENDING"
	case OrderStatusNew:
		return "NEW"
	case OrderStatusPartiallyFilled:
		return "PARTIALLY_FILLED"
	case OrderStatusFilled:
		return "FILLED"
	case OrderStatusCancelled:
		return "CANCELLED"
	case OrderStatusRejected:
		return "REJECTED"
	case OrderStatusExpired:
		return "EXPIRED"
	case OrderStatusSuspended:
		return "SUSPENDED"
	default:
		return "UNKNOWN"
	}
}

// Core Business Types

// Asset represents a tradeable asset
type Asset struct {
	ID           string                 `json:"id"`
	Symbol       string                 `json:"symbol"`
	Name         string                 `json:"name"`
	AssetType    AssetType              `json:"asset_type"`
	Exchange     string                 `json:"exchange"`
	Region       string                 `json:"region"`
	Currency     string                 `json:"currency"`
	ISIN         string                 `json:"isin,omitempty"`
	Sector       string                 `json:"sector,omitempty"`
	Industry     string                 `json:"industry,omitempty"`
	MarketCap    float64                `json:"market_cap,omitempty"`
	IsActive     bool                   `json:"is_active"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

// Order represents a trading order
type Order struct {
	ID          string                 `json:"id"`
	UserID      string                 `json:"user_id"`
	Symbol      string                 `json:"symbol"`
	AssetType   AssetType              `json:"asset_type"`
	Exchange    string                 `json:"exchange"`
	Type        OrderType              `json:"type"`
	Side        OrderSide              `json:"side"`
	Quantity    float64                `json:"quantity"`
	Price       float64                `json:"price,omitempty"`
	StopPrice   float64                `json:"stop_price,omitempty"`
	Status      OrderStatus            `json:"status"`
	FilledQty   float64                `json:"filled_qty"`
	AvgPrice    float64                `json:"avg_price"`
	Commission  float64                `json:"commission"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	ExpiresAt   *time.Time             `json:"expires_at,omitempty"`
}

// OrderResponse represents the response to an order submission
type OrderResponse struct {
	OrderID     string      `json:"order_id"`
	Status      OrderStatus `json:"status"`
	Message     string      `json:"message,omitempty"`
	FilledQty   float64     `json:"filled_qty"`
	AvgPrice    float64     `json:"avg_price"`
	Commission  float64     `json:"commission"`
	Timestamp   time.Time   `json:"timestamp"`
}

// MarketData represents market data for an asset
type MarketData struct {
	Symbol        string    `json:"symbol"`
	AssetType     AssetType `json:"asset_type"`
	Exchange      string    `json:"exchange"`
	Price         float64   `json:"price"`
	Bid           float64   `json:"bid"`
	Ask           float64   `json:"ask"`
	Volume        int64     `json:"volume"`
	High          float64   `json:"high"`
	Low           float64   `json:"low"`
	Open          float64   `json:"open"`
	Close         float64   `json:"close"`
	Change        float64   `json:"change"`
	ChangePercent float64   `json:"change_percent"`
	Timestamp     time.Time `json:"timestamp"`
}

// Portfolio represents a user's portfolio
type Portfolio struct {
	ID          string      `json:"id"`
	UserID      string      `json:"user_id"`
	Name        string      `json:"name"`
	Currency    string      `json:"currency"`
	TotalValue  float64     `json:"total_value"`
	CashBalance float64     `json:"cash_balance"`
	Positions   []*Position `json:"positions"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

// Position represents a position in a portfolio
type Position struct {
	ID            string    `json:"id"`
	PortfolioID   string    `json:"portfolio_id"`
	Symbol        string    `json:"symbol"`
	AssetType     AssetType `json:"asset_type"`
	Exchange      string    `json:"exchange"`
	Quantity      float64   `json:"quantity"`
	AverageCost   float64   `json:"average_cost"`
	CurrentPrice  float64   `json:"current_price"`
	MarketValue   float64   `json:"market_value"`
	UnrealizedPnL float64   `json:"unrealized_pnl"`
	RealizedPnL   float64   `json:"realized_pnl"`
	DayChange     float64   `json:"day_change"`
	Weight        float64   `json:"weight"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// Transaction represents a financial transaction
type Transaction struct {
	ID          string                 `json:"id"`
	UserID      string                 `json:"user_id"`
	OrderID     string                 `json:"order_id,omitempty"`
	Type        string                 `json:"type"`
	Symbol      string                 `json:"symbol,omitempty"`
	AssetType   AssetType              `json:"asset_type,omitempty"`
	Exchange    string                 `json:"exchange,omitempty"`
	Quantity    float64                `json:"quantity,omitempty"`
	Price       float64                `json:"price,omitempty"`
	Amount      float64                `json:"amount"`
	Currency    string                 `json:"currency"`
	Commission  float64                `json:"commission"`
	Status      string                 `json:"status"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	SettledAt   *time.Time             `json:"settled_at,omitempty"`
}

// TradingSchedule represents trading schedule information
type TradingSchedule struct {
	MarketOpen      time.Time        `json:"market_open"`
	MarketClose     time.Time        `json:"market_close"`
	PreMarketOpen   time.Time        `json:"pre_market_open,omitempty"`
	PostMarketClose time.Time        `json:"post_market_close,omitempty"`
	TradingSessions []TradingSession `json:"trading_sessions"`
	Holidays        []time.Time      `json:"holidays"`
	Timezone        *time.Location   `json:"timezone"`
}

// TradingSession represents a trading session
type TradingSession struct {
	Name       string      `json:"name"`
	StartTime  time.Time   `json:"start_time"`
	EndTime    time.Time   `json:"end_time"`
	AssetTypes []AssetType `json:"asset_types"`
}

// TradingStatus represents the current trading status
type TradingStatus struct {
	Exchange    string          `json:"exchange"`
	IsOpen      bool            `json:"is_open"`
	CurrentTime time.Time       `json:"current_time"`
	NextOpen    time.Time       `json:"next_open,omitempty"`
	NextClose   time.Time       `json:"next_close,omitempty"`
	Session     *TradingSession `json:"session,omitempty"`
	Message     string          `json:"message,omitempty"`
}

// ComplianceRule represents a compliance rule
type ComplianceRule struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Type        string      `json:"type"`
	Severity    string      `json:"severity"`
	AssetTypes  []AssetType `json:"asset_types,omitempty"`
	Exchanges   []string    `json:"exchanges,omitempty"`
	Regions     []string    `json:"regions,omitempty"`
	IsActive    bool        `json:"is_active"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

// Utility Interfaces

// Validator defines the interface for validation
type Validator interface {
	Validate(interface{}) error
}

// MetricsCollector defines the interface for metrics collection
type MetricsCollector interface {
	Counter(name string, tags map[string]string) Counter
	Gauge(name string, tags map[string]string) Gauge
	Histogram(name string, tags map[string]string) Histogram
	Timer(name string, tags map[string]string) Timer
}

// Counter represents a counter metric
type Counter interface {
	Inc()
	Add(delta float64)
}

// Gauge represents a gauge metric
type Gauge interface {
	Set(value float64)
	Add(delta float64)
	Sub(delta float64)
}

// Histogram represents a histogram metric
type Histogram interface {
	Observe(value float64)
}

// Timer represents a timer metric
type Timer interface {
	Record(duration time.Duration)
	Start() func()
}

// Configuration Types

// BaseConfig represents base configuration for all services
type BaseConfig struct {
	ServiceName    string        `yaml:"service_name" json:"service_name"`
	Version        string        `yaml:"version" json:"version"`
	Environment    string        `yaml:"environment" json:"environment"`
	LogLevel       string        `yaml:"log_level" json:"log_level"`
	MetricsEnabled bool          `yaml:"metrics_enabled" json:"metrics_enabled"`
	HealthCheck    HealthConfig  `yaml:"health_check" json:"health_check"`
	Timeouts       TimeoutConfig `yaml:"timeouts" json:"timeouts"`
}

// HealthConfig represents health check configuration
type HealthConfig struct {
	Enabled  bool          `yaml:"enabled" json:"enabled"`
	Interval time.Duration `yaml:"interval" json:"interval"`
	Timeout  time.Duration `yaml:"timeout" json:"timeout"`
	Port     int           `yaml:"port" json:"port"`
	Path     string        `yaml:"path" json:"path"`
}

// TimeoutConfig represents timeout configuration
type TimeoutConfig struct {
	Read    time.Duration `yaml:"read" json:"read"`
	Write   time.Duration `yaml:"write" json:"write"`
	Idle    time.Duration `yaml:"idle" json:"idle"`
	Connect time.Duration `yaml:"connect" json:"connect"`
}

// DatabaseConfig represents database configuration
type DatabaseConfig struct {
	Host            string        `yaml:"host" json:"host"`
	Port            int           `yaml:"port" json:"port"`
	Database        string        `yaml:"database" json:"database"`
	Username        string        `yaml:"username" json:"username"`
	Password        string        `yaml:"password" json:"password"`
	SSLMode         string        `yaml:"ssl_mode" json:"ssl_mode"`
	MaxConnections  int           `yaml:"max_connections" json:"max_connections"`
	MaxIdleConns    int           `yaml:"max_idle_conns" json:"max_idle_conns"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime" json:"conn_max_lifetime"`
}

// RedisConfig represents Redis configuration
type RedisConfig struct {
	Host        string        `yaml:"host" json:"host"`
	Port        int           `yaml:"port" json:"port"`
	Password    string        `yaml:"password" json:"password"`
	Database    int           `yaml:"database" json:"database"`
	MaxRetries  int           `yaml:"max_retries" json:"max_retries"`
	PoolSize    int           `yaml:"pool_size" json:"pool_size"`
	PoolTimeout time.Duration `yaml:"pool_timeout" json:"pool_timeout"`
}

// HTTPConfig represents HTTP server configuration
type HTTPConfig struct {
	Host         string        `yaml:"host" json:"host"`
	Port         int           `yaml:"port" json:"port"`
	ReadTimeout  time.Duration `yaml:"read_timeout" json:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout" json:"write_timeout"`
	IdleTimeout  time.Duration `yaml:"idle_timeout" json:"idle_timeout"`
	TLS          *TLSConfig    `yaml:"tls" json:"tls"`
}

// TLSConfig represents TLS configuration
type TLSConfig struct {
	Enabled  bool   `yaml:"enabled" json:"enabled"`
	CertFile string `yaml:"cert_file" json:"cert_file"`
	KeyFile  string `yaml:"key_file" json:"key_file"`
	CAFile   string `yaml:"ca_file" json:"ca_file"`
}

// GRPCConfig represents gRPC server configuration
type GRPCConfig struct {
	Host               string        `yaml:"host" json:"host"`
	Port               int           `yaml:"port" json:"port"`
	MaxRecvMsgSize     int           `yaml:"max_recv_msg_size" json:"max_recv_msg_size"`
	MaxSendMsgSize     int           `yaml:"max_send_msg_size" json:"max_send_msg_size"`
	ConnectionTimeout  time.Duration `yaml:"connection_timeout" json:"connection_timeout"`
	MaxConnectionIdle  time.Duration `yaml:"max_connection_idle" json:"max_connection_idle"`
	MaxConnectionAge   time.Duration `yaml:"max_connection_age" json:"max_connection_age"`
	KeepAliveTime      time.Duration `yaml:"keep_alive_time" json:"keep_alive_time"`
	KeepAliveTimeout   time.Duration `yaml:"keep_alive_timeout" json:"keep_alive_timeout"`
	TLS                *TLSConfig    `yaml:"tls" json:"tls"`
}

// WebSocketConfig represents WebSocket configuration
type WebSocketConfig struct {
	Host            string        `yaml:"host" json:"host"`
	Port            int           `yaml:"port" json:"port"`
	Path            string        `yaml:"path" json:"path"`
	MaxConnections  int           `yaml:"max_connections" json:"max_connections"`
	ReadBufferSize  int           `yaml:"read_buffer_size" json:"read_buffer_size"`
	WriteBufferSize int           `yaml:"write_buffer_size" json:"write_buffer_size"`
	PingInterval    time.Duration `yaml:"ping_interval" json:"ping_interval"`
	PongTimeout     time.Duration `yaml:"pong_timeout" json:"pong_timeout"`
	WriteTimeout    time.Duration `yaml:"write_timeout" json:"write_timeout"`
}

// Helper Functions

// GetAllAssetTypes returns all supported asset types
func GetAllAssetTypes() []AssetType {
	return []AssetType{
		AssetTypeStock,
		AssetTypeBond,
		AssetTypeETF,
		AssetTypeREIT,
		AssetTypeMutualFund,
		AssetTypeCommodity,
		AssetTypeCrypto,
		AssetTypeForex,
		AssetTypeGovernmentBond,
		AssetTypeCorporateBond,
		AssetTypeIslamicInstrument,
		AssetTypeSukuk,
		AssetTypeIslamicFund,
		AssetTypeIslamicREIT,
	}
}

// IsIslamicAsset checks if an asset type is Islamic
func IsIslamicAsset(assetType AssetType) bool {
	islamicAssets := []AssetType{
		AssetTypeIslamicInstrument,
		AssetTypeSukuk,
		AssetTypeIslamicFund,
		AssetTypeIslamicREIT,
	}
	
	for _, islamic := range islamicAssets {
		if islamic == assetType {
			return true
		}
	}
	return false
}

// GetAssetTypeFromString converts string to AssetType
func GetAssetTypeFromString(s string) AssetType {
	switch s {
	case "STOCK":
		return AssetTypeStock
	case "BOND":
		return AssetTypeBond
	case "ETF":
		return AssetTypeETF
	case "REIT":
		return AssetTypeREIT
	case "MUTUAL_FUND":
		return AssetTypeMutualFund
	case "COMMODITY":
		return AssetTypeCommodity
	case "CRYPTO":
		return AssetTypeCrypto
	case "FOREX":
		return AssetTypeForex
	case "GOVERNMENT_BOND":
		return AssetTypeGovernmentBond
	case "CORPORATE_BOND":
		return AssetTypeCorporateBond
	case "ISLAMIC_INSTRUMENT":
		return AssetTypeIslamicInstrument
	case "SUKUK":
		return AssetTypeSukuk
	case "ISLAMIC_FUND":
		return AssetTypeIslamicFund
	case "ISLAMIC_REIT":
		return AssetTypeIslamicREIT
	default:
		return AssetTypeStock // Default fallback
	}
}

// GetOrderTypeFromString converts string to OrderType
func GetOrderTypeFromString(s string) OrderType {
	switch s {
	case "MARKET":
		return OrderTypeMarket
	case "LIMIT":
		return OrderTypeLimit
	case "STOP":
		return OrderTypeStop
	case "STOP_LIMIT":
		return OrderTypeStopLimit
	case "TRAILING_STOP":
		return OrderTypeTrailingStop
	case "ICEBERG":
		return OrderTypeIceberg
	case "FILL_OR_KILL":
		return OrderTypeFillOrKill
	case "IMMEDIATE_OR_CANCEL":
		return OrderTypeImmediateOrCancel
	case "GOOD_TILL_CANCELLED":
		return OrderTypeGoodTillCancelled
	case "GOOD_TILL_DATE":
		return OrderTypeGoodTillDate
	default:
		return OrderTypeMarket // Default fallback
	}
}

// GetOrderSideFromString converts string to OrderSide
func GetOrderSideFromString(s string) OrderSide {
	switch s {
	case "BUY":
		return OrderSideBuy
	case "SELL":
		return OrderSideSell
	default:
		return OrderSideBuy // Default fallback
	}
}
