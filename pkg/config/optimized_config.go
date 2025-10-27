package config

import (
	"fmt"
	"time"

	"go.uber.org/zap"
)

// OptimizedConfig represents the unified configuration structure
type OptimizedConfig struct {
	// Core system configuration
	System   SystemConfig   `yaml:"system" json:"system"`
	Trading  TradingConfig  `yaml:"trading" json:"trading"`
	Risk     RiskConfig     `yaml:"risk" json:"risk"`
	Network  NetworkConfig  `yaml:"network" json:"network"`
	Database DatabaseConfig `yaml:"database" json:"database"`
	
	// Performance and monitoring
	Performance PerformanceConfig `yaml:"performance" json:"performance"`
	Monitoring  MonitoringConfig  `yaml:"monitoring" json:"monitoring"`
	
	// Feature flags
	Features FeatureConfig `yaml:"features" json:"features"`
}

// SystemConfig contains core system settings
type SystemConfig struct {
	Environment     string        `yaml:"environment" json:"environment"`
	ServiceName     string        `yaml:"service_name" json:"service_name"`
	Version         string        `yaml:"version" json:"version"`
	LogLevel        string        `yaml:"log_level" json:"log_level"`
	GracefulTimeout time.Duration `yaml:"graceful_timeout" json:"graceful_timeout"`
	MaxConcurrency  int           `yaml:"max_concurrency" json:"max_concurrency"`
}

// TradingConfig contains trading-specific settings
type TradingConfig struct {
	MaxOrdersPerSecond int           `yaml:"max_orders_per_second" json:"max_orders_per_second"`
	MaxLatency         time.Duration `yaml:"max_latency" json:"max_latency"`
	TargetLatency      time.Duration `yaml:"target_latency" json:"target_latency"`
	OrderBookDepth     int           `yaml:"order_book_depth" json:"order_book_depth"`
	TickSize           float64       `yaml:"tick_size" json:"tick_size"`
	MinOrderSize       float64       `yaml:"min_order_size" json:"min_order_size"`
	MaxOrderSize       float64       `yaml:"max_order_size" json:"max_order_size"`
}



// NetworkConfig contains network settings
type NetworkConfig struct {
	HTTPPort           int           `yaml:"http_port" json:"http_port"`
	GRPCPort           int           `yaml:"grpc_port" json:"grpc_port"`
	WebSocketPort      int           `yaml:"websocket_port" json:"websocket_port"`
	TCPNoDelay         bool          `yaml:"tcp_no_delay" json:"tcp_no_delay"`
	SocketBufferSize   int           `yaml:"socket_buffer_size" json:"socket_buffer_size"`
	ConnectionPoolSize int           `yaml:"connection_pool_size" json:"connection_pool_size"`
	KeepAliveInterval  time.Duration `yaml:"keep_alive_interval" json:"keep_alive_interval"`
	ReadTimeout        time.Duration `yaml:"read_timeout" json:"read_timeout"`
	WriteTimeout       time.Duration `yaml:"write_timeout" json:"write_timeout"`
}



// PerformanceConfig contains performance optimization settings
type PerformanceConfig struct {
	BufferSize         int    `yaml:"buffer_size" json:"buffer_size"`
	WorkerPoolSize     int    `yaml:"worker_pool_size" json:"worker_pool_size"`
	MemoryPoolSize     int    `yaml:"memory_pool_size" json:"memory_pool_size"`
	GCTargetPercentage int    `yaml:"gc_target_percentage" json:"gc_target_percentage"`
	MaxMemoryUsage     uint64 `yaml:"max_memory_usage" json:"max_memory_usage"`
	EnableProfiling    bool   `yaml:"enable_profiling" json:"enable_profiling"`
	CPUProfilePath     string `yaml:"cpu_profile_path" json:"cpu_profile_path"`
	MemProfilePath     string `yaml:"mem_profile_path" json:"mem_profile_path"`
}

// MonitoringConfig contains monitoring and observability settings
type MonitoringConfig struct {
	MetricsInterval     time.Duration `yaml:"metrics_interval" json:"metrics_interval"`
	HealthCheckInterval time.Duration `yaml:"health_check_interval" json:"health_check_interval"`
	EnableMetrics       bool          `yaml:"enable_metrics" json:"enable_metrics"`
	EnableTracing       bool          `yaml:"enable_tracing" json:"enable_tracing"`
	MetricsPort         int           `yaml:"metrics_port" json:"metrics_port"`
	TracingEndpoint     string        `yaml:"tracing_endpoint" json:"tracing_endpoint"`
	LogFormat           string        `yaml:"log_format" json:"log_format"`
}

// FeatureConfig contains feature flags
type FeatureConfig struct {
	EnableHFTMode          bool `yaml:"enable_hft_mode" json:"enable_hft_mode"`
	EnableRealTimeRisk     bool `yaml:"enable_realtime_risk" json:"enable_realtime_risk"`
	EnableAdvancedMatching bool `yaml:"enable_advanced_matching" json:"enable_advanced_matching"`
	EnableWebSocket        bool `yaml:"enable_websocket" json:"enable_websocket"`
	EnableGRPC             bool `yaml:"enable_grpc" json:"enable_grpc"`
	EnableCaching          bool `yaml:"enable_caching" json:"enable_caching"`
	EnableCompression      bool `yaml:"enable_compression" json:"enable_compression"`
}

// NewOptimizedConfig creates a new optimized configuration with defaults
func NewOptimizedConfig() *OptimizedConfig {
	return &OptimizedConfig{
		System: SystemConfig{
			Environment:     "development",
			ServiceName:     "tradSys",
			Version:         "1.0.0",
			LogLevel:        "info",
			GracefulTimeout: 30 * time.Second,
			MaxConcurrency:  1000,
		},
		Trading: TradingConfig{
			MaxOrdersPerSecond: 10000,
			MaxLatency:         100 * time.Microsecond,
			TargetLatency:      50 * time.Microsecond,
			OrderBookDepth:     100,
			TickSize:           0.01,
			MinOrderSize:       1.0,
			MaxOrderSize:       1000000.0,
		},
		Risk: RiskConfig{
			EnableRiskCheck:     true,
			MaxPositionSize:     1000000.0,
			MaxDailyVolume:      10000000.0,
			MaxLeverage:         10.0,
			MarginRequirement:   0.1,
			StopLossThreshold:   0.05,
			MaxDrawdown:         0.2,
			RiskCheckTimeout:    10 * time.Millisecond,
			EnablePositionLimit: true,
			EnableVolumeLimit:   true,
		},
		Network: NetworkConfig{
			HTTPPort:           8080,
			GRPCPort:           9090,
			WebSocketPort:      8081,
			TCPNoDelay:         true,
			SocketBufferSize:   65536,
			ConnectionPoolSize: 100,
			KeepAliveInterval:  30 * time.Second,
			ReadTimeout:        5 * time.Second,
			WriteTimeout:       5 * time.Second,
		},
		Database: DatabaseConfig{
			Driver:          "postgres",
			Host:            "localhost",
			Port:            5432,
			Database:        "tradSys",
			Username:        "postgres",
			Password:        "",
			SSLMode:         "disable",
			MaxOpenConns:    100,
			MaxIdleConns:    10,
			ConnMaxLifetime: time.Hour,
			ConnMaxIdleTime: 30 * time.Minute,
			EnableMigration: true,
			MigrationPath:   "./migrations",
		},
		Performance: PerformanceConfig{
			BufferSize:         8192,
			WorkerPoolSize:     10,
			MemoryPoolSize:     1000,
			GCTargetPercentage: 10,
			MaxMemoryUsage:     1024 * 1024 * 1024, // 1GB
			EnableProfiling:    false,
			CPUProfilePath:     "/tmp/cpu.prof",
			MemProfilePath:     "/tmp/mem.prof",
		},
		Monitoring: MonitoringConfig{
			MetricsInterval:     time.Second,
			HealthCheckInterval: 10 * time.Second,
			EnableMetrics:       true,
			EnableTracing:       false,
			MetricsPort:         9091,
			TracingEndpoint:     "",
			LogFormat:           "json",
		},
		Features: FeatureConfig{
			EnableHFTMode:          true,
			EnableRealTimeRisk:     true,
			EnableAdvancedMatching: true,
			EnableWebSocket:        true,
			EnableGRPC:             true,
			EnableCaching:          true,
			EnableCompression:      false,
		},
	}
}

// Validate validates the configuration
func (c *OptimizedConfig) Validate() error {
	if c.System.ServiceName == "" {
		return fmt.Errorf("system.service_name is required")
	}
	
	if c.Trading.MaxLatency <= 0 {
		return fmt.Errorf("trading.max_latency must be positive")
	}
	
	if c.Trading.TargetLatency <= 0 {
		return fmt.Errorf("trading.target_latency must be positive")
	}
	
	if c.Trading.TargetLatency > c.Trading.MaxLatency {
		return fmt.Errorf("trading.target_latency cannot be greater than max_latency")
	}
	
	if c.Risk.MaxPositionSize <= 0 {
		return fmt.Errorf("risk.max_position_size must be positive")
	}
	
	if c.Risk.MaxLeverage <= 0 {
		return fmt.Errorf("risk.max_leverage must be positive")
	}
	
	if c.Network.HTTPPort <= 0 || c.Network.HTTPPort > 65535 {
		return fmt.Errorf("network.http_port must be between 1 and 65535")
	}
	
	if c.Database.MaxOpenConns <= 0 {
		return fmt.Errorf("database.max_open_conns must be positive")
	}
	
	return nil
}

// GetTradingConfig returns the trading configuration
func (c *OptimizedConfig) GetTradingConfig() TradingConfig {
	return c.Trading
}

// GetRiskConfig returns the risk configuration
func (c *OptimizedConfig) GetRiskConfig() RiskConfig {
	return c.Risk
}

// GetNetworkConfig returns the network configuration
func (c *OptimizedConfig) GetNetworkConfig() NetworkConfig {
	return c.Network
}

// GetPerformanceConfig returns the performance configuration
func (c *OptimizedConfig) GetPerformanceConfig() PerformanceConfig {
	return c.Performance
}

// IsHFTModeEnabled returns whether HFT mode is enabled
func (c *OptimizedConfig) IsHFTModeEnabled() bool {
	return c.Features.EnableHFTMode
}

// IsRealTimeRiskEnabled returns whether real-time risk is enabled
func (c *OptimizedConfig) IsRealTimeRiskEnabled() bool {
	return c.Features.EnableRealTimeRisk
}

// GetLogger creates a logger based on configuration
func (c *OptimizedConfig) GetLogger() (*zap.Logger, error) {
	var config zap.Config
	
	if c.System.Environment == "production" {
		config = zap.NewProductionConfig()
	} else {
		config = zap.NewDevelopmentConfig()
	}
	
	// Set log level
	switch c.System.LogLevel {
	case "debug":
		config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	case "info":
		config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	case "warn":
		config.Level = zap.NewAtomicLevelAt(zap.WarnLevel)
	case "error":
		config.Level = zap.NewAtomicLevelAt(zap.ErrorLevel)
	default:
		config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	}
	
	// Set output format
	if c.Monitoring.LogFormat == "json" {
		config.Encoding = "json"
	} else {
		config.Encoding = "console"
	}
	
	return config.Build()
}
