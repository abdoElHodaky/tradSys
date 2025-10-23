package config

import (
	"fmt"
	"time"
)

// HFTConfig represents high-frequency trading configuration
type HFTConfig struct {
	// Performance settings
	MaxOrdersPerSecond int           `yaml:"max_orders_per_second"`
	MaxLatency         time.Duration `yaml:"max_latency"`
	TargetLatency      time.Duration `yaml:"target_latency"`
	BufferSize         int           `yaml:"buffer_size"`
	WorkerPoolSize     int           `yaml:"worker_pool_size"`

	// Memory management
	MemoryPoolSize     int    `yaml:"memory_pool_size"`
	GCTargetPercentage int    `yaml:"gc_target_percentage"`
	MaxMemoryUsage     uint64 `yaml:"max_memory_usage"`

	// Network settings
	TCPNoDelay         bool          `yaml:"tcp_no_delay"`
	SocketBufferSize   int           `yaml:"socket_buffer_size"`
	ConnectionPoolSize int           `yaml:"connection_pool_size"`
	KeepAliveInterval  time.Duration `yaml:"keep_alive_interval"`

	// Risk management
	MaxPositionSize  float64       `yaml:"max_position_size"`
	MaxOrderSize     float64       `yaml:"max_order_size"`
	RiskCheckLatency time.Duration `yaml:"risk_check_latency"`

	// Monitoring
	MetricsInterval     time.Duration `yaml:"metrics_interval"`
	HealthCheckInterval time.Duration `yaml:"health_check_interval"`
	LogLevel            string        `yaml:"log_level"`
	EnableProfiling     bool          `yaml:"enable_profiling"`
}

// Default returns default HFT configuration
func Default() *HFTConfig {
	return &HFTConfig{
		MaxOrdersPerSecond:  10000,
		MaxLatency:          time.Microsecond * 100,
		TargetLatency:       time.Microsecond * 50,
		BufferSize:          8192,
		WorkerPoolSize:      10,
		MemoryPoolSize:      1000,
		GCTargetPercentage:  10,
		MaxMemoryUsage:      1024 * 1024 * 1024, // 1GB
		TCPNoDelay:          true,
		SocketBufferSize:    65536,
		ConnectionPoolSize:  100,
		KeepAliveInterval:   time.Second * 30,
		MaxPositionSize:     1000000.0,
		MaxOrderSize:        100000.0,
		RiskCheckLatency:    time.Microsecond * 10,
		MetricsInterval:     time.Second * 1,
		HealthCheckInterval: time.Second * 5,
		LogLevel:            "info",
		EnableProfiling:     false,
	}
}

// Validate validates the HFT configuration
func (c *HFTConfig) Validate() error {
	if c.MaxOrdersPerSecond <= 0 {
		return ErrInvalidMaxOrdersPerSecond
	}
	if c.MaxLatency <= 0 {
		return ErrInvalidMaxLatency
	}
	if c.BufferSize <= 0 {
		return ErrInvalidBufferSize
	}
	if c.WorkerPoolSize <= 0 {
		return ErrInvalidWorkerPoolSize
	}
	return nil
}

// Optimize optimizes configuration for performance
func (c *HFTConfig) Optimize() {
	// Optimize for low latency
	if c.TargetLatency < time.Microsecond*10 {
		c.BufferSize = 16384
		c.WorkerPoolSize = 20
		c.MemoryPoolSize = 2000
		c.GCTargetPercentage = 5
	}

	// Optimize TCP settings
	c.TCPNoDelay = true
	c.SocketBufferSize = 131072 // 128KB
}

// LegacyConfig represents the legacy application configuration
// Renamed to avoid conflict with UnifiedConfig
type LegacyConfig struct {
	// HFT contains high-frequency trading configuration
	HFT HFTConfig `yaml:"hft"`

	// Server configuration
	Server struct {
		Host string `yaml:"host"`
		Port int    `yaml:"port"`
	} `yaml:"server"`

	// Database configuration
	Database struct {
		Driver string `yaml:"driver"`
		DSN    string `yaml:"dsn"`
	} `yaml:"database"`

	// JWT configuration
	JWT struct {
		Secret     string        `yaml:"secret"`
		Expiration time.Duration `yaml:"expiration"`
	} `yaml:"jwt"`
}

// Errors
var (
	ErrInvalidMaxOrdersPerSecond = fmt.Errorf("invalid max orders per second")
	ErrInvalidMaxLatency         = fmt.Errorf("invalid max latency")
	ErrInvalidBufferSize         = fmt.Errorf("invalid buffer size")
	ErrInvalidWorkerPoolSize     = fmt.Errorf("invalid worker pool size")
)
