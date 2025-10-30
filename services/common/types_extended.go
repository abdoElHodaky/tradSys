package common

import (
	"time"
)

// Market Data Types

// Quote represents a market quote
type Quote struct {
	Symbol    string    `json:"symbol"`
	Exchange  string    `json:"exchange"`
	BidPrice  float64   `json:"bid_price"`
	BidSize   float64   `json:"bid_size"`
	AskPrice  float64   `json:"ask_price"`
	AskSize   float64   `json:"ask_size"`
	LastPrice float64   `json:"last_price"`
	LastSize  float64   `json:"last_size"`
	Volume    float64   `json:"volume"`
	Timestamp time.Time `json:"timestamp"`
}

// Candle represents OHLCV data
type Candle struct {
	Symbol    string    `json:"symbol"`
	Exchange  string    `json:"exchange"`
	TimeFrame TimeFrame `json:"time_frame"`
	Open      float64   `json:"open"`
	High      float64   `json:"high"`
	Low       float64   `json:"low"`
	Close     float64   `json:"close"`
	Volume    float64   `json:"volume"`
	VWAP      float64   `json:"vwap,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// OrderBook represents order book data
type OrderBook struct {
	Symbol    string          `json:"symbol"`
	Exchange  string          `json:"exchange"`
	Bids      []OrderBookItem `json:"bids"`
	Asks      []OrderBookItem `json:"asks"`
	Timestamp time.Time       `json:"timestamp"`
}

// OrderBookItem represents a single order book entry
type OrderBookItem struct {
	Price    float64 `json:"price"`
	Quantity float64 `json:"quantity"`
	Count    int     `json:"count,omitempty"`
}

// Tick represents tick data
type Tick struct {
	Symbol    string    `json:"symbol"`
	Exchange  string    `json:"exchange"`
	Price     float64   `json:"price"`
	Size      float64   `json:"size"`
	Side      OrderSide `json:"side"`
	Timestamp time.Time `json:"timestamp"`
}

// Portfolio represents a user's portfolio
type Portfolio struct {
	ID              string                 `json:"id"`
	UserID          string                 `json:"user_id"`
	Name            string                 `json:"name"`
	TotalValue      float64                `json:"total_value"`
	CashBalance     float64                `json:"cash_balance"`
	TotalPL         float64                `json:"total_pl"`
	DayPL           float64                `json:"day_pl"`
	Positions       []Position             `json:"positions"`
	Currency        string                 `json:"currency"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

// Account represents a user account
type Account struct {
	ID              string                 `json:"id"`
	UserID          string                 `json:"user_id"`
	AccountType     string                 `json:"account_type"`
	AccountNumber   string                 `json:"account_number"`
	BrokerID        string                 `json:"broker_id"`
	Currency        string                 `json:"currency"`
	Balance         float64                `json:"balance"`
	AvailableBalance float64               `json:"available_balance"`
	MarginUsed      float64                `json:"margin_used"`
	MarginAvailable float64                `json:"margin_available"`
	IsActive        bool                   `json:"is_active"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
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
	Host         string        `yaml:"host" json:"host"`
	Port         int           `yaml:"port" json:"port"`
	Database     string        `yaml:"database" json:"database"`
	Username     string        `yaml:"username" json:"username"`
	Password     string        `yaml:"password" json:"password"`
	SSLMode      string        `yaml:"ssl_mode" json:"ssl_mode"`
	MaxConns     int           `yaml:"max_conns" json:"max_conns"`
	MaxIdleConns int           `yaml:"max_idle_conns" json:"max_idle_conns"`
	MaxLifetime  time.Duration `yaml:"max_lifetime" json:"max_lifetime"`
}

// RedisConfig represents Redis configuration
type RedisConfig struct {
	Host         string        `yaml:"host" json:"host"`
	Port         int           `yaml:"port" json:"port"`
	Password     string        `yaml:"password" json:"password"`
	Database     int           `yaml:"database" json:"database"`
	PoolSize     int           `yaml:"pool_size" json:"pool_size"`
	MinIdleConns int           `yaml:"min_idle_conns" json:"min_idle_conns"`
	MaxRetries   int           `yaml:"max_retries" json:"max_retries"`
	DialTimeout  time.Duration `yaml:"dial_timeout" json:"dial_timeout"`
	ReadTimeout  time.Duration `yaml:"read_timeout" json:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout" json:"write_timeout"`
}

// MessageQueueConfig represents message queue configuration
type MessageQueueConfig struct {
	Type     string `yaml:"type" json:"type"`
	Host     string `yaml:"host" json:"host"`
	Port     int    `yaml:"port" json:"port"`
	Username string `yaml:"username" json:"username"`
	Password string `yaml:"password" json:"password"`
	VHost    string `yaml:"vhost" json:"vhost"`
}

// APIConfig represents API configuration
type APIConfig struct {
	Host           string        `yaml:"host" json:"host"`
	Port           int           `yaml:"port" json:"port"`
	ReadTimeout    time.Duration `yaml:"read_timeout" json:"read_timeout"`
	WriteTimeout   time.Duration `yaml:"write_timeout" json:"write_timeout"`
	MaxHeaderBytes int           `yaml:"max_header_bytes" json:"max_header_bytes"`
	TLS            TLSConfig     `yaml:"tls" json:"tls"`
	CORS           CORSConfig    `yaml:"cors" json:"cors"`
	RateLimit      RateLimitConfig `yaml:"rate_limit" json:"rate_limit"`
}

// TLSConfig represents TLS configuration
type TLSConfig struct {
	Enabled  bool   `yaml:"enabled" json:"enabled"`
	CertFile string `yaml:"cert_file" json:"cert_file"`
	KeyFile  string `yaml:"key_file" json:"key_file"`
}

// CORSConfig represents CORS configuration
type CORSConfig struct {
	Enabled          bool     `yaml:"enabled" json:"enabled"`
	AllowedOrigins   []string `yaml:"allowed_origins" json:"allowed_origins"`
	AllowedMethods   []string `yaml:"allowed_methods" json:"allowed_methods"`
	AllowedHeaders   []string `yaml:"allowed_headers" json:"allowed_headers"`
	ExposedHeaders   []string `yaml:"exposed_headers" json:"exposed_headers"`
	AllowCredentials bool     `yaml:"allow_credentials" json:"allow_credentials"`
	MaxAge           int      `yaml:"max_age" json:"max_age"`
}

// RateLimitConfig represents rate limiting configuration
type RateLimitConfig struct {
	Enabled     bool          `yaml:"enabled" json:"enabled"`
	RequestsPerSecond int     `yaml:"requests_per_second" json:"requests_per_second"`
	BurstSize   int           `yaml:"burst_size" json:"burst_size"`
	WindowSize  time.Duration `yaml:"window_size" json:"window_size"`
}

// Event represents a system event
type Event struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Source    string                 `json:"source"`
	Subject   string                 `json:"subject"`
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
	Version   string                 `json:"version"`
}

// Notification represents a user notification
type Notification struct {
	ID        string                 `json:"id"`
	UserID    string                 `json:"user_id"`
	Type      string                 `json:"type"`
	Title     string                 `json:"title"`
	Message   string                 `json:"message"`
	Priority  string                 `json:"priority"`
	Channel   string                 `json:"channel"`
	IsRead    bool                   `json:"is_read"`
	Data      map[string]interface{} `json:"data,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
	ReadAt    *time.Time             `json:"read_at,omitempty"`
}

// AuditLog represents an audit log entry
type AuditLog struct {
	ID        string                 `json:"id"`
	UserID    string                 `json:"user_id"`
	Action    string                 `json:"action"`
	Resource  string                 `json:"resource"`
	Details   map[string]interface{} `json:"details"`
	IPAddress string                 `json:"ip_address"`
	UserAgent string                 `json:"user_agent"`
	Timestamp time.Time              `json:"timestamp"`
}
