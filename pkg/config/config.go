package config

import (
	"errors"
	"fmt"
	"gopkg.in/yaml.v2"
	"os"
	"time"
)

// Config represents the main application configuration
type Config struct {
	Server    ServerConfig    `json:"server" yaml:"server"`
	Database  DatabaseConfig  `json:"database" yaml:"database"`
	Redis     RedisConfig     `json:"redis" yaml:"redis"`
	Matching  MatchingConfig  `json:"matching" yaml:"matching"`
	Risk      RiskConfig      `json:"risk" yaml:"risk"`
	Auth      AuthConfig      `json:"auth" yaml:"auth"`
	Logging   LoggingConfig   `json:"logging" yaml:"logging"`
	Metrics   MetricsConfig   `json:"metrics" yaml:"metrics"`
	WebSocket WebSocketConfig `json:"websocket" yaml:"websocket"`
	GRPC      GRPCConfig      `json:"grpc" yaml:"grpc"`
	Exchange  ExchangeConfig  `json:"exchange" yaml:"exchange"`
}

// ServerConfig contains HTTP server configuration
type ServerConfig struct {
	Host            string        `json:"host" yaml:"host"`
	Port            int           `json:"port" yaml:"port"`
	ReadTimeout     time.Duration `json:"read_timeout" yaml:"read_timeout"`
	WriteTimeout    time.Duration `json:"write_timeout" yaml:"write_timeout"`
	IdleTimeout     time.Duration `json:"idle_timeout" yaml:"idle_timeout"`
	MaxHeaderBytes  int           `json:"max_header_bytes" yaml:"max_header_bytes"`
	EnableTLS       bool          `json:"enable_tls" yaml:"enable_tls"`
	TLSCertFile     string        `json:"tls_cert_file" yaml:"tls_cert_file"`
	TLSKeyFile      string        `json:"tls_key_file" yaml:"tls_key_file"`
	EnableCORS      bool          `json:"enable_cors" yaml:"enable_cors"`
	CORSOrigins     []string      `json:"cors_origins" yaml:"cors_origins"`
	RateLimitRPS    int           `json:"rate_limit_rps" yaml:"rate_limit_rps"`
	RateLimitBurst  int           `json:"rate_limit_burst" yaml:"rate_limit_burst"`
	ShutdownTimeout time.Duration `json:"shutdown_timeout" yaml:"shutdown_timeout"`
}

// DatabaseConfig contains database configuration
type DatabaseConfig struct {
	Driver          string        `json:"driver" yaml:"driver"`
	Host            string        `json:"host" yaml:"host"`
	Port            int           `json:"port" yaml:"port"`
	Database        string        `json:"database" yaml:"database"`
	Username        string        `json:"username" yaml:"username"`
	Password        string        `json:"password" yaml:"password"`
	SSLMode         string        `json:"ssl_mode" yaml:"ssl_mode"`
	MaxOpenConns    int           `json:"max_open_conns" yaml:"max_open_conns"`
	MaxIdleConns    int           `json:"max_idle_conns" yaml:"max_idle_conns"`
	ConnMaxLifetime time.Duration `json:"conn_max_lifetime" yaml:"conn_max_lifetime"`
	ConnMaxIdleTime time.Duration `json:"conn_max_idle_time" yaml:"conn_max_idle_time"`
	EnableMigration bool          `json:"enable_migration" yaml:"enable_migration"`
	MigrationPath   string        `json:"migration_path" yaml:"migration_path"`
}

// RedisConfig contains Redis configuration
type RedisConfig struct {
	Host         string        `json:"host" yaml:"host"`
	Port         int           `json:"port" yaml:"port"`
	Password     string        `json:"password" yaml:"password"`
	Database     int           `json:"database" yaml:"database"`
	PoolSize     int           `json:"pool_size" yaml:"pool_size"`
	MinIdleConns int           `json:"min_idle_conns" yaml:"min_idle_conns"`
	DialTimeout  time.Duration `json:"dial_timeout" yaml:"dial_timeout"`
	ReadTimeout  time.Duration `json:"read_timeout" yaml:"read_timeout"`
	WriteTimeout time.Duration `json:"write_timeout" yaml:"write_timeout"`
	IdleTimeout  time.Duration `json:"idle_timeout" yaml:"idle_timeout"`
	EnableTLS    bool          `json:"enable_tls" yaml:"enable_tls"`
}

// MatchingConfig contains matching engine configuration
type MatchingConfig struct {
	EngineType         string        `json:"engine_type" yaml:"engine_type"`
	MaxOrdersPerSymbol int           `json:"max_orders_per_symbol" yaml:"max_orders_per_symbol"`
	TickSize           float64       `json:"tick_size" yaml:"tick_size"`
	ProcessingTimeout  time.Duration `json:"processing_timeout" yaml:"processing_timeout"`
	EnableMetrics      bool          `json:"enable_metrics" yaml:"enable_metrics"`
	PoolSize           int           `json:"pool_size" yaml:"pool_size"`
	BufferSize         int           `json:"buffer_size" yaml:"buffer_size"`
	WorkerCount        int           `json:"worker_count" yaml:"worker_count"`
	EnableProfiling    bool          `json:"enable_profiling" yaml:"enable_profiling"`
	MaxLatency         time.Duration `json:"max_latency" yaml:"max_latency"`
	EnableOrderBook    bool          `json:"enable_order_book" yaml:"enable_order_book"`
	OrderBookDepth     int           `json:"order_book_depth" yaml:"order_book_depth"`
}

// RiskConfig contains risk management configuration
type RiskConfig struct {
	EnableRiskCheck     bool          `json:"enable_risk_check" yaml:"enable_risk_check"`
	MaxPositionSize     float64       `json:"max_position_size" yaml:"max_position_size"`
	MaxDailyVolume      float64       `json:"max_daily_volume" yaml:"max_daily_volume"`
	MaxLeverage         float64       `json:"max_leverage" yaml:"max_leverage"`
	MarginRequirement   float64       `json:"margin_requirement" yaml:"margin_requirement"`
	StopLossThreshold   float64       `json:"stop_loss_threshold" yaml:"stop_loss_threshold"`
	MaxDrawdown         float64       `json:"max_drawdown" yaml:"max_drawdown"`
	RiskCheckTimeout    time.Duration `json:"risk_check_timeout" yaml:"risk_check_timeout"`
	EnablePositionLimit bool          `json:"enable_position_limit" yaml:"enable_position_limit"`
	EnableVolumeLimit   bool          `json:"enable_volume_limit" yaml:"enable_volume_limit"`
}

// AuthConfig contains authentication configuration
type AuthConfig struct {
	JWTSecret          string        `json:"jwt_secret" yaml:"jwt_secret"`
	JWTExpiration      time.Duration `json:"jwt_expiration" yaml:"jwt_expiration"`
	RefreshExpiration  time.Duration `json:"refresh_expiration" yaml:"refresh_expiration"`
	EnableRefreshToken bool          `json:"enable_refresh_token" yaml:"enable_refresh_token"`
	PasswordMinLength  int           `json:"password_min_length" yaml:"password_min_length"`
	EnableTwoFactor    bool          `json:"enable_two_factor" yaml:"enable_two_factor"`
	SessionTimeout     time.Duration `json:"session_timeout" yaml:"session_timeout"`
	MaxLoginAttempts   int           `json:"max_login_attempts" yaml:"max_login_attempts"`
	LockoutDuration    time.Duration `json:"lockout_duration" yaml:"lockout_duration"`
	EnableAPIKeys      bool          `json:"enable_api_keys" yaml:"enable_api_keys"`
	APIKeyExpiration   time.Duration `json:"api_key_expiration" yaml:"api_key_expiration"`
}

// LoggingConfig contains logging configuration
type LoggingConfig struct {
	Level            string `json:"level" yaml:"level"`
	Format           string `json:"format" yaml:"format"`
	Output           string `json:"output" yaml:"output"`
	Filename         string `json:"filename" yaml:"filename"`
	MaxSize          int    `json:"max_size" yaml:"max_size"`
	MaxBackups       int    `json:"max_backups" yaml:"max_backups"`
	MaxAge           int    `json:"max_age" yaml:"max_age"`
	Compress         bool   `json:"compress" yaml:"compress"`
	EnableColor      bool   `json:"enable_color" yaml:"enable_color"`
	EnableCaller     bool   `json:"enable_caller" yaml:"enable_caller"`
	EnableStacktrace bool   `json:"enable_stacktrace" yaml:"enable_stacktrace"`
}

// MetricsConfig contains metrics configuration
type MetricsConfig struct {
	Enabled       bool              `json:"enabled" yaml:"enabled"`
	Provider      string            `json:"provider" yaml:"provider"`
	Address       string            `json:"address" yaml:"address"`
	Port          int               `json:"port" yaml:"port"`
	Path          string            `json:"path" yaml:"path"`
	Interval      time.Duration     `json:"interval" yaml:"interval"`
	EnableRuntime bool              `json:"enable_runtime" yaml:"enable_runtime"`
	EnableGC      bool              `json:"enable_gc" yaml:"enable_gc"`
	EnableMemory  bool              `json:"enable_memory" yaml:"enable_memory"`
	EnableCPU     bool              `json:"enable_cpu" yaml:"enable_cpu"`
	Tags          map[string]string `json:"tags" yaml:"tags"`
}

// WebSocketConfig contains WebSocket configuration
type WebSocketConfig struct {
	Enabled           bool          `json:"enabled" yaml:"enabled"`
	Host              string        `json:"host" yaml:"host"`
	Port              int           `json:"port" yaml:"port"`
	Path              string        `json:"path" yaml:"path"`
	ReadBufferSize    int           `json:"read_buffer_size" yaml:"read_buffer_size"`
	WriteBufferSize   int           `json:"write_buffer_size" yaml:"write_buffer_size"`
	HandshakeTimeout  time.Duration `json:"handshake_timeout" yaml:"handshake_timeout"`
	ReadTimeout       time.Duration `json:"read_timeout" yaml:"read_timeout"`
	WriteTimeout      time.Duration `json:"write_timeout" yaml:"write_timeout"`
	PingPeriod        time.Duration `json:"ping_period" yaml:"ping_period"`
	PongWait          time.Duration `json:"pong_wait" yaml:"pong_wait"`
	MaxMessageSize    int64         `json:"max_message_size" yaml:"max_message_size"`
	EnableCompression bool          `json:"enable_compression" yaml:"enable_compression"`
	MaxConnections    int           `json:"max_connections" yaml:"max_connections"`
}

// GRPCConfig contains gRPC configuration
type GRPCConfig struct {
	Enabled           bool          `json:"enabled" yaml:"enabled"`
	Host              string        `json:"host" yaml:"host"`
	Port              int           `json:"port" yaml:"port"`
	EnableTLS         bool          `json:"enable_tls" yaml:"enable_tls"`
	TLSCertFile       string        `json:"tls_cert_file" yaml:"tls_cert_file"`
	TLSKeyFile        string        `json:"tls_key_file" yaml:"tls_key_file"`
	MaxRecvMsgSize    int           `json:"max_recv_msg_size" yaml:"max_recv_msg_size"`
	MaxSendMsgSize    int           `json:"max_send_msg_size" yaml:"max_send_msg_size"`
	ConnectionTimeout time.Duration `json:"connection_timeout" yaml:"connection_timeout"`
	KeepAliveTime     time.Duration `json:"keep_alive_time" yaml:"keep_alive_time"`
	KeepAliveTimeout  time.Duration `json:"keep_alive_timeout" yaml:"keep_alive_timeout"`
	EnableReflection  bool          `json:"enable_reflection" yaml:"enable_reflection"`
	EnableHealthCheck bool          `json:"enable_health_check" yaml:"enable_health_check"`
}

// ExchangeConfig contains exchange-specific configuration
type ExchangeConfig struct {
	Name                    string             `json:"name" yaml:"name"`
	Symbols                 []SymbolConfig     `json:"symbols" yaml:"symbols"`
	TradingHours            TradingHoursConfig `json:"trading_hours" yaml:"trading_hours"`
	Fees                    FeesConfig         `json:"fees" yaml:"fees"`
	Limits                  LimitsConfig       `json:"limits" yaml:"limits"`
	EnableHalts             bool               `json:"enable_halts" yaml:"enable_halts"`
	EnableCircuitBreaker    bool               `json:"enable_circuit_breaker" yaml:"enable_circuit_breaker"`
	CircuitBreakerThreshold float64            `json:"circuit_breaker_threshold" yaml:"circuit_breaker_threshold"`
}

// SymbolConfig contains symbol-specific configuration
type SymbolConfig struct {
	Symbol      string  `json:"symbol" yaml:"symbol"`
	BaseAsset   string  `json:"base_asset" yaml:"base_asset"`
	QuoteAsset  string  `json:"quote_asset" yaml:"quote_asset"`
	MinPrice    float64 `json:"min_price" yaml:"min_price"`
	MaxPrice    float64 `json:"max_price" yaml:"max_price"`
	TickSize    float64 `json:"tick_size" yaml:"tick_size"`
	MinQuantity float64 `json:"min_quantity" yaml:"min_quantity"`
	MaxQuantity float64 `json:"max_quantity" yaml:"max_quantity"`
	StepSize    float64 `json:"step_size" yaml:"step_size"`
	MinNotional float64 `json:"min_notional" yaml:"min_notional"`
	Enabled     bool    `json:"enabled" yaml:"enabled"`
}

// TradingHoursConfig contains trading hours configuration
type TradingHoursConfig struct {
	Timezone    string   `json:"timezone" yaml:"timezone"`
	MarketOpen  string   `json:"market_open" yaml:"market_open"`
	MarketClose string   `json:"market_close" yaml:"market_close"`
	Weekends    []string `json:"weekends" yaml:"weekends"`
	Holidays    []string `json:"holidays" yaml:"holidays"`
	PreMarket   bool     `json:"pre_market" yaml:"pre_market"`
	PostMarket  bool     `json:"post_market" yaml:"post_market"`
}

// FeesConfig contains fee configuration
type FeesConfig struct {
	MakerFee    float64 `json:"maker_fee" yaml:"maker_fee"`
	TakerFee    float64 `json:"taker_fee" yaml:"taker_fee"`
	WithdrawFee float64 `json:"withdraw_fee" yaml:"withdraw_fee"`
	DepositFee  float64 `json:"deposit_fee" yaml:"deposit_fee"`
}

// LimitsConfig contains various limits configuration
type LimitsConfig struct {
	MaxOrderSize    float64 `json:"max_order_size" yaml:"max_order_size"`
	MinOrderSize    float64 `json:"min_order_size" yaml:"min_order_size"`
	MaxDailyVolume  float64 `json:"max_daily_volume" yaml:"max_daily_volume"`
	MaxOpenOrders   int     `json:"max_open_orders" yaml:"max_open_orders"`
	MaxOrdersPerSec int     `json:"max_orders_per_sec" yaml:"max_orders_per_sec"`
	MaxCancelPerSec int     `json:"max_cancel_per_sec" yaml:"max_cancel_per_sec"`
}

// Environment represents the application environment
type Environment string

const (
	EnvironmentDevelopment Environment = "development"
	EnvironmentStaging     Environment = "staging"
	EnvironmentProduction  Environment = "production"
	EnvironmentTest        Environment = "test"
)

// GetEnvironment returns the current environment
func (c *Config) GetEnvironment() Environment {
	// This would typically be set from an environment variable
	return EnvironmentDevelopment
}

// IsDevelopment returns true if running in development mode
func (c *Config) IsDevelopment() bool {
	return c.GetEnvironment() == EnvironmentDevelopment
}

// IsProduction returns true if running in production mode
func (c *Config) IsProduction() bool {
	return c.GetEnvironment() == EnvironmentProduction
}

// IsTest returns true if running in test mode
func (c *Config) IsTest() bool {
	return c.GetEnvironment() == EnvironmentTest
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return ErrInvalidPort
	}

	if c.Database.Driver == "" {
		return ErrMissingDatabaseDriver
	}

	if c.Auth.JWTSecret == "" {
		return ErrMissingJWTSecret
	}

	if c.Matching.MaxOrdersPerSymbol <= 0 {
		return ErrInvalidMatchingConfig
	}

	return nil
}

// GetDatabaseDSN returns the database connection string
func (c *Config) GetDatabaseDSN() string {
	switch c.Database.Driver {
	case "postgres":
		return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			c.Database.Host, c.Database.Port, c.Database.Username,
			c.Database.Password, c.Database.Database, c.Database.SSLMode)
	case "mysql":
		return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
			c.Database.Username, c.Database.Password,
			c.Database.Host, c.Database.Port, c.Database.Database)
	default:
		return ""
	}
}

// GetRedisAddr returns the Redis address
func (c *Config) GetRedisAddr() string {
	return fmt.Sprintf("%s:%d", c.Redis.Host, c.Redis.Port)
}

// GetServerAddr returns the server address
func (c *Config) GetServerAddr() string {
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
}

// GetGRPCAddr returns the gRPC server address
func (c *Config) GetGRPCAddr() string {
	return fmt.Sprintf("%s:%d", c.GRPC.Host, c.GRPC.Port)
}

// GetWebSocketAddr returns the WebSocket server address
func (c *Config) GetWebSocketAddr() string {
	return fmt.Sprintf("%s:%d", c.WebSocket.Host, c.WebSocket.Port)
}

// Configuration errors
var (
	ErrInvalidPort           = errors.New("invalid port number")
	ErrMissingDatabaseDriver = errors.New("missing database driver")
	ErrMissingJWTSecret      = errors.New("missing JWT secret")
	ErrInvalidMatchingConfig = errors.New("invalid matching engine configuration")
)

// Default configurations
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host:            "0.0.0.0",
			Port:            8080,
			ReadTimeout:     30 * time.Second,
			WriteTimeout:    30 * time.Second,
			IdleTimeout:     120 * time.Second,
			MaxHeaderBytes:  1 << 20, // 1MB
			EnableCORS:      true,
			CORSOrigins:     []string{"*"},
			RateLimitRPS:    1000,
			RateLimitBurst:  2000,
			ShutdownTimeout: 30 * time.Second,
		},
		Database: DatabaseConfig{
			Driver:          "postgres",
			Host:            "localhost",
			Port:            5432,
			Database:        "tradingsystem",
			Username:        "postgres",
			Password:        "password",
			SSLMode:         "disable",
			MaxOpenConns:    25,
			MaxIdleConns:    5,
			ConnMaxLifetime: 5 * time.Minute,
			ConnMaxIdleTime: 5 * time.Minute,
			EnableMigration: true,
			MigrationPath:   "./migrations",
		},
		Redis: RedisConfig{
			Host:         "localhost",
			Port:         6379,
			Database:     0,
			PoolSize:     10,
			MinIdleConns: 2,
			DialTimeout:  5 * time.Second,
			ReadTimeout:  3 * time.Second,
			WriteTimeout: 3 * time.Second,
			IdleTimeout:  5 * time.Minute,
		},
		Matching: MatchingConfig{
			EngineType:         "hft",
			MaxOrdersPerSymbol: 10000,
			TickSize:           0.01,
			ProcessingTimeout:  100 * time.Millisecond,
			EnableMetrics:      true,
			PoolSize:           100,
			BufferSize:         1000,
			WorkerCount:        4,
			MaxLatency:         1 * time.Millisecond,
			EnableOrderBook:    true,
			OrderBookDepth:     20,
		},
		Risk: RiskConfig{
			EnableRiskCheck:     true,
			MaxPositionSize:     1000000,
			MaxDailyVolume:      10000000,
			MaxLeverage:         10.0,
			MarginRequirement:   0.1,
			StopLossThreshold:   0.05,
			MaxDrawdown:         0.2,
			RiskCheckTimeout:    50 * time.Millisecond,
			EnablePositionLimit: true,
			EnableVolumeLimit:   true,
		},
		Auth: AuthConfig{
			JWTExpiration:      24 * time.Hour,
			RefreshExpiration:  7 * 24 * time.Hour,
			EnableRefreshToken: true,
			PasswordMinLength:  8,
			SessionTimeout:     30 * time.Minute,
			MaxLoginAttempts:   5,
			LockoutDuration:    15 * time.Minute,
			EnableAPIKeys:      true,
			APIKeyExpiration:   30 * 24 * time.Hour,
		},
		Logging: LoggingConfig{
			Level:            "info",
			Format:           "json",
			Output:           "stdout",
			EnableColor:      false,
			EnableCaller:     true,
			EnableStacktrace: false,
		},
		Metrics: MetricsConfig{
			Enabled:       true,
			Provider:      "prometheus",
			Address:       "0.0.0.0",
			Port:          9090,
			Path:          "/metrics",
			Interval:      15 * time.Second,
			EnableRuntime: true,
			EnableGC:      true,
			EnableMemory:  true,
			EnableCPU:     true,
		},
		WebSocket: WebSocketConfig{
			Enabled:           true,
			Host:              "0.0.0.0",
			Port:              8081,
			Path:              "/ws",
			ReadBufferSize:    1024,
			WriteBufferSize:   1024,
			HandshakeTimeout:  10 * time.Second,
			ReadTimeout:       60 * time.Second,
			WriteTimeout:      10 * time.Second,
			PingPeriod:        54 * time.Second,
			PongWait:          60 * time.Second,
			MaxMessageSize:    512,
			EnableCompression: true,
			MaxConnections:    1000,
		},
		GRPC: GRPCConfig{
			Enabled:           true,
			Host:              "0.0.0.0",
			Port:              9000,
			MaxRecvMsgSize:    4 * 1024 * 1024, // 4MB
			MaxSendMsgSize:    4 * 1024 * 1024, // 4MB
			ConnectionTimeout: 5 * time.Second,
			KeepAliveTime:     30 * time.Second,
			KeepAliveTimeout:  5 * time.Second,
			EnableReflection:  true,
			EnableHealthCheck: true,
		},
	}
}

// LoadConfig loads configuration from a YAML file
func LoadConfig(configPath string) (*Config, error) {
	// If no config path provided, return default config
	if configPath == "" {
		return DefaultConfig(), nil
	}

	// Read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		// If file doesn't exist, return default config
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
}
