package config

import (
	"time"

	"github.com/abdoElHodaky/tradSys/internal/trading/memory"
)

// UnifiedConfig represents the main application configuration
// This replaces all other Config structs to eliminate redeclaration errors
type UnifiedConfig struct {
	// System configuration
	System SystemConfig `yaml:"system"`
	
	// Server configuration
	Server ServerConfig `yaml:"server"`
	
	// Database configuration
	Database DatabaseConfig `yaml:"database"`
	
	// Trading configuration
	Trading TradingConfig `yaml:"trading"`
	
	// Risk management configuration
	Risk RiskConfig `yaml:"risk"`
	
	// API configuration
	API APIConfig `yaml:"api"`
	
	// Performance configuration
	Performance PerformanceConfig `yaml:"performance"`
	
	// Monitoring configuration
	Monitoring MonitoringConfig `yaml:"monitoring"`
	
	// External integrations
	Exchanges ExchangesConfig `yaml:"exchanges"`
	
	// WebSocket configuration
	WebSocket WebSocketConfig `yaml:"websocket"`
	
	// JWT configuration
	JWT JWTConfig `yaml:"jwt"`
	
	// HFT configuration (for backward compatibility)
	HFT UnifiedHFTConfig `yaml:"hft"`
	
	// Gateway configuration
	Gateway GatewayConfig `yaml:"gateway"`
	
	// Broker configuration
	Broker BrokerConfig `yaml:"broker"`
	
	// Service configuration
	Service ServiceConfig `yaml:"service"`
	
	// Connectivity configuration
	Connectivity ConnectivityConfig `yaml:"connectivity"`
	
	// Compliance configuration
	Compliance ComplianceConfig `yaml:"compliance"`
	
	// Strategies configuration
	Strategies StrategiesConfig `yaml:"strategies"`
	
	// Tracing configuration
	Tracing TracingConfig `yaml:"tracing"`
	
	// Metrics configuration
	Metrics MetricsConfig `yaml:"metrics"`
	
	// Resilience configuration
	Resilience ResilienceConfig `yaml:"resilience"`
	
	// Registry configuration
	Registry RegistryConfig `yaml:"registry"`
}

// SystemConfig contains core system settings
type SystemConfig struct {
	Environment string `yaml:"environment" default:"development"`
	LogLevel    string `yaml:"log_level" default:"info"`
	Debug       bool   `yaml:"debug" default:"false"`
	Version     string `yaml:"version" default:"1.0.0"`
}

// ServerConfig contains HTTP server settings
type ServerConfig struct {
	Host         string        `yaml:"host" default:"localhost"`
	Port         int           `yaml:"port" default:"8080"`
	ReadTimeout  time.Duration `yaml:"read_timeout" default:"30s"`
	WriteTimeout time.Duration `yaml:"write_timeout" default:"30s"`
	IdleTimeout  time.Duration `yaml:"idle_timeout" default:"120s"`
}

// DatabaseConfig contains database connection settings
type DatabaseConfig struct {
	Driver       string `yaml:"driver" default:"sqlite"`
	Host         string `yaml:"host" default:"localhost"`
	Port         int    `yaml:"port" default:"5432"`
	Database     string `yaml:"database" default:"tradsys"`
	Username     string `yaml:"username"`
	Password     string `yaml:"password"`
	SSLMode      string `yaml:"ssl_mode" default:"disable"`
	MaxOpenConns int    `yaml:"max_open_conns" default:"25"`
	MaxIdleConns int    `yaml:"max_idle_conns" default:"5"`
	DSN          string `yaml:"dsn"` // For backward compatibility
}

// TradingConfig contains trading engine settings
type TradingConfig struct {
	Engine      TradingEngineConfig `yaml:"engine"`
	OrderBook   OrderBookConfig     `yaml:"order_book"`
	Execution   ExecutionConfig     `yaml:"execution"`
	Settlement  SettlementConfig    `yaml:"settlement"`
}

// TradingEngineConfig contains trading engine specific settings
type TradingEngineConfig struct {
	MaxOrdersPerSecond int           `yaml:"max_orders_per_second" default:"1000"`
	OrderTimeout       time.Duration `yaml:"order_timeout" default:"30s"`
	EnablePaperTrading bool          `yaml:"enable_paper_trading" default:"true"`
}

// OrderBookConfig contains order book settings
type OrderBookConfig struct {
	MaxDepth     int  `yaml:"max_depth" default:"100"`
	EnableL2Data bool `yaml:"enable_l2_data" default:"true"`
}

// ExecutionConfig contains order execution settings
type ExecutionConfig struct {
	MaxSlippage      float64       `yaml:"max_slippage" default:"0.01"`
	ExecutionTimeout time.Duration `yaml:"execution_timeout" default:"5s"`
}

// SettlementConfig contains settlement processing settings
type SettlementConfig struct {
	AutoSettle      bool          `yaml:"auto_settle" default:"true"`
	SettleTimeout   time.Duration `yaml:"settle_timeout" default:"60s"`
	BatchSize       int           `yaml:"batch_size" default:"100"`
}

// RiskConfig contains risk management settings
type RiskConfig struct {
	MaxPositionSize    float64 `yaml:"max_position_size" default:"10000"`
	MaxDailyLoss       float64 `yaml:"max_daily_loss" default:"1000"`
	EnableRiskChecks   bool    `yaml:"enable_risk_checks" default:"true"`
	CircuitBreaker     CircuitBreakerConfig `yaml:"circuit_breaker"`
}

// CircuitBreakerConfig contains circuit breaker settings
type CircuitBreakerConfig struct {
	Enabled              bool          `yaml:"enabled" default:"true"`
	PriceChangeThreshold float64       `yaml:"price_change_threshold" default:"0.05"`
	CooldownPeriod       time.Duration `yaml:"cooldown_period" default:"300s"`
}

// APIConfig contains API server settings
type APIConfig struct {
	RateLimit   RateLimitConfig `yaml:"rate_limit"`
	CORS        CORSConfig      `yaml:"cors"`
	EnableDocs  bool            `yaml:"enable_docs" default:"true"`
}

// RateLimitConfig contains rate limiting settings
type RateLimitConfig struct {
	RequestsPerMinute int           `yaml:"requests_per_minute" default:"60"`
	BurstSize         int           `yaml:"burst_size" default:"10"`
	CleanupInterval   time.Duration `yaml:"cleanup_interval" default:"60s"`
}

// CORSConfig contains CORS settings
type CORSConfig struct {
	AllowedOrigins []string `yaml:"allowed_origins"`
	AllowedMethods []string `yaml:"allowed_methods"`
	AllowedHeaders []string `yaml:"allowed_headers"`
}

// PerformanceConfig contains performance optimization settings
type PerformanceConfig struct {
	EnableProfiling   bool `yaml:"enable_profiling" default:"false"`
	EnableMetrics     bool `yaml:"enable_metrics" default:"true"`
	GCTargetPercent   int  `yaml:"gc_target_percent" default:"100"`
	MaxProcs          int  `yaml:"max_procs" default:"0"` // 0 = use all CPUs
}

// MonitoringConfig contains monitoring and observability settings
type MonitoringConfig struct {
	EnablePrometheus    bool          `yaml:"enable_prometheus" default:"true"`
	EnableCustomMetrics bool          `yaml:"enable_custom_metrics" default:"true"`
	MetricsInterval     time.Duration `yaml:"metrics_interval" default:"10s"`
	EnableHealthChecks  bool          `yaml:"enable_health_checks" default:"true"`
	HealthCheckInterval time.Duration `yaml:"health_check_interval" default:"30s"`
	EnableTracing       bool          `yaml:"enable_tracing" default:"false"`
}

// ExchangesConfig contains exchange integration settings
type ExchangesConfig struct {
	Binance  ExchangeConfig `yaml:"binance"`
	Coinbase ExchangeConfig `yaml:"coinbase"`
	Kraken   ExchangeConfig `yaml:"kraken"`
}

// ExchangeConfig contains individual exchange settings
type ExchangeConfig struct {
	Enabled   bool   `yaml:"enabled" default:"false"`
	APIKey    string `yaml:"api_key"`
	APISecret string `yaml:"api_secret"`
	Sandbox   bool   `yaml:"sandbox" default:"true"`
	RateLimit int    `yaml:"rate_limit" default:"10"`
}

// WebSocketConfig contains WebSocket server settings
type WebSocketConfig struct {
	Port                int           `yaml:"port" default:"8081"`
	ReadBufferSize      int           `yaml:"read_buffer_size" default:"1024"`
	WriteBufferSize     int           `yaml:"write_buffer_size" default:"1024"`
	HandshakeTimeout    time.Duration `yaml:"handshake_timeout" default:"10s"`
	EnableCompression   bool          `yaml:"enable_compression" default:"true"`
	CompressionLevel    int           `yaml:"compression_level" default:"6"`
	MaxMessageSize      int64         `yaml:"max_message_size" default:"512"`
	PingPeriod          time.Duration `yaml:"ping_period" default:"54s"`
	PongWait            time.Duration `yaml:"pong_wait" default:"60s"`
	WriteWait           time.Duration `yaml:"write_wait" default:"10s"`
}

// JWTConfig contains JWT authentication settings
type JWTConfig struct {
	Secret        string        `yaml:"secret"`
	SecretKey     string        `yaml:"secret_key"`     // Alias for Secret for backward compatibility
	Expiration    time.Duration `yaml:"expiration" default:"24h"`
	TokenDuration time.Duration `yaml:"token_duration" default:"24h"` // Alias for Expiration
	Issuer        string        `yaml:"issuer" default:"tradsys"`
}

// GatewayConfig contains API gateway settings
type GatewayConfig struct {
	Address         string        `yaml:"address" default:":8080"`
	ReadTimeout     time.Duration `yaml:"read_timeout" default:"30s"`
	WriteTimeout    time.Duration `yaml:"write_timeout" default:"30s"`
	IdleTimeout     time.Duration `yaml:"idle_timeout" default:"120s"`
	EnableMetrics   bool          `yaml:"enable_metrics" default:"true"`
	EnableCORS      bool          `yaml:"enable_cors" default:"true"`
	EnableRateLimit bool          `yaml:"enable_rate_limit" default:"true"`
}

// UnifiedHFTConfig contains high-frequency trading settings (for backward compatibility)
type UnifiedHFTConfig struct {
	Memory      UnifiedHFTMemoryConfig      `yaml:"memory"`
	Monitoring  UnifiedHFTMonitoringConfig  `yaml:"monitoring"`
	Performance UnifiedHFTPerformanceConfig `yaml:"performance"`
}

// UnifiedHFTMemoryConfig is an alias for backward compatibility
type UnifiedHFTMemoryConfig = memory.HFTMemoryConfig

// UnifiedHFTMonitoringConfig contains HFT monitoring settings
type UnifiedHFTMonitoringConfig struct {
	EnablePrometheus    bool          `yaml:"enable_prometheus" default:"true"`
	EnableCustomMetrics bool          `yaml:"enable_custom_metrics" default:"true"`
	MetricsInterval     time.Duration `yaml:"metrics_interval" default:"10s"`
	EnableHealthChecks  bool          `yaml:"enable_health_checks" default:"true"`
	HealthCheckInterval time.Duration `yaml:"health_check_interval" default:"30s"`
}

// UnifiedHFTPerformanceConfig contains HFT performance settings
type UnifiedHFTPerformanceConfig struct {
	EnableOptimizations bool `yaml:"enable_optimizations" default:"true"`
	MemoryPoolSize      int  `yaml:"memory_pool_size" default:"1000"`
	BatchSize           int  `yaml:"batch_size" default:"100"`
}

// BrokerConfig contains message broker settings
type BrokerConfig struct {
	Type      string   `yaml:"type" default:"nats"`
	Addresses []string `yaml:"addresses"`
	Username  string   `yaml:"username"`
	Password  string   `yaml:"password"`
	TLS       bool     `yaml:"tls" default:"false"`
}

// ServiceConfig contains service-specific settings
type ServiceConfig struct {
	Name     string `yaml:"name" default:"tradsys"`
	Version  string `yaml:"version" default:"1.0.0"`
	GRPCPort int    `yaml:"grpc_port" default:"9090"`
}

// ConnectivityConfig contains connectivity settings
type ConnectivityConfig struct {
	Enabled bool `yaml:"enabled" default:"true"`
}

// ComplianceConfig contains compliance settings
type ComplianceConfig struct {
	Enabled bool `yaml:"enabled" default:"true"`
}

// StrategiesConfig contains strategies settings
type StrategiesConfig struct {
	Enabled bool `yaml:"enabled" default:"true"`
}

// TracingConfig contains tracing settings
type TracingConfig struct {
	Enabled bool `yaml:"enabled" default:"false"`
}

// MetricsConfig contains metrics settings
type MetricsConfig struct {
	Enabled bool `yaml:"enabled" default:"true"`
}

// ResilienceConfig contains resilience settings
type ResilienceConfig struct {
	CircuitBreakerEnabled bool `yaml:"circuit_breaker_enabled" default:"true"`
	RateLimitingEnabled   bool `yaml:"rate_limiting_enabled" default:"true"`
}

// RegistryConfig contains service registry settings
type RegistryConfig struct {
	Enabled bool   `yaml:"enabled" default:"false"`
	Type    string `yaml:"type" default:"consul"`
}

// Global configuration instance
var GlobalConfig *UnifiedConfig

// LoadConfig loads the configuration from file
func LoadConfig(configPath string) (*UnifiedConfig, error) {
	// Implementation will be added in next step
	config := &UnifiedConfig{}
	
	// Set defaults
	config.System.Environment = "development"
	config.System.LogLevel = "info"
	config.Server.Host = "localhost"
	config.Server.Port = 8080
	config.Database.Driver = "sqlite"
	config.Database.Database = "tradsys.db"
	config.Service.Name = "tradsys"
	config.Service.Version = "1.0.0"
	config.Service.GRPCPort = 9090
	config.Gateway.Address = ":8080"
	config.Connectivity.Enabled = true
	config.Compliance.Enabled = true
	config.Strategies.Enabled = true
	config.Tracing.Enabled = false
	config.Metrics.Enabled = true
	config.Resilience.CircuitBreakerEnabled = true
	config.Resilience.RateLimitingEnabled = true
	config.Registry.Enabled = false
	config.Registry.Type = "consul"
	
	GlobalConfig = config
	return config, nil
}

// GetConfig returns the global configuration instance
func GetConfig() *UnifiedConfig {
	if GlobalConfig == nil {
		GlobalConfig = &UnifiedConfig{}
	}
	return GlobalConfig
}
