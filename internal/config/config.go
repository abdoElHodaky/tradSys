package config

import (
	"time"
)

// Config represents the application configuration
type Config struct {
	Environment string
	Server      ServerConfig
	Database    DatabaseConfig
	JWT         JWTConfig
	Services    ServicesConfig
	Logging     LoggingConfig
	Registry    RegistryConfig
	Broker      BrokerConfig
	Service     ServiceConfig
	Resilience  ResilienceConfig
}

// ServerConfig represents the server configuration
type ServerConfig struct {
	Port         int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

// DatabaseConfig represents the database configuration
type DatabaseConfig struct {
	Driver   string
	Host     string
	Port     int
	Username string
	Password string
	Database string
	SSLMode  string
	MaxConns int
	MaxIdle  int
}

// JWTConfig represents the JWT configuration
type JWTConfig struct {
	Secret        string
	ExpiryMinutes int
}

// ServicesConfig represents the services configuration
type ServicesConfig struct {
	OrderService      ServiceEndpoint
	MarketDataService ServiceEndpoint
	RiskService       ServiceEndpoint
}

// ServiceEndpoint represents a service endpoint configuration
type ServiceEndpoint struct {
	URL      string
	Timeout  time.Duration
	Retries  int
	Insecure bool
}

// LoggingConfig represents the logging configuration
type LoggingConfig struct {
	Level      string
	Format     string
	OutputPath string
}

// RegistryConfig represents the service registry configuration
type RegistryConfig struct {
	Type      string
	Address   string
	Namespace string
}

// BrokerConfig represents the message broker configuration
type BrokerConfig struct {
	Type      string
	Addresses []string
}

// ServiceConfig represents the microservice configuration
type ServiceConfig struct {
	Name      string
	Version   string
	ID        string
	Namespace string
}

// ResilienceConfig represents the resilience configuration
type ResilienceConfig struct {
	CircuitBreaker CircuitBreakerConfig
	RateLimiter    RateLimiterConfig
	Retry          RetryConfig
	Bulkhead       BulkheadConfig
	Timeout        TimeoutConfig
}

// CircuitBreakerConfig represents the circuit breaker configuration
type CircuitBreakerConfig struct {
	Enabled       bool
	Threshold     int
	Timeout       time.Duration
	HalfOpenLimit int
}

// RateLimiterConfig represents the rate limiter configuration
type RateLimiterConfig struct {
	Enabled bool
	Limit   int
	Burst   int
	Period  time.Duration
}

// RetryConfig represents the retry configuration
type RetryConfig struct {
	Enabled     bool
	MaxAttempts int
	Delay       time.Duration
	MaxDelay    time.Duration
	Multiplier  float64
}

// BulkheadConfig represents the bulkhead configuration
type BulkheadConfig struct {
	Enabled      bool
	MaxConcurrent int
	QueueSize    int
}

// TimeoutConfig represents the timeout configuration
type TimeoutConfig struct {
	Enabled bool
	Default time.Duration
}

// LoadConfig loads the configuration from the specified file
func LoadConfig(path string) (*Config, error) {
	// Implementation would load from file
	// For now, return default config
	return DefaultConfig(), nil
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		Environment: "development",
		Server: ServerConfig{
			Port:         8080,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  120 * time.Second,
		},
		Database: DatabaseConfig{
			Driver:   "postgres",
			Host:     "localhost",
			Port:     5432,
			Username: "postgres",
			Password: "postgres",
			Database: "tradsys",
			SSLMode:  "disable",
			MaxConns: 10,
			MaxIdle:  5,
		},
		JWT: JWTConfig{
			Secret:        "default-secret-key",
			ExpiryMinutes: 60,
		},
		Services: ServicesConfig{
			OrderService: ServiceEndpoint{
				URL:      "http://localhost:8081",
				Timeout:  5 * time.Second,
				Retries:  3,
				Insecure: true,
			},
			MarketDataService: ServiceEndpoint{
				URL:      "http://localhost:8082",
				Timeout:  5 * time.Second,
				Retries:  3,
				Insecure: true,
			},
			RiskService: ServiceEndpoint{
				URL:      "http://localhost:8083",
				Timeout:  5 * time.Second,
				Retries:  3,
				Insecure: true,
			},
		},
		Logging: LoggingConfig{
			Level:      "info",
			Format:     "json",
			OutputPath: "stdout",
		},
		Registry: RegistryConfig{
			Type:      "mdns",
			Address:   "",
			Namespace: "tradsys",
		},
		Broker: BrokerConfig{
			Type:      "nats",
			Addresses: []string{"nats://localhost:4222"},
		},
		Service: ServiceConfig{
			Name:      "tradsys",
			Version:   "1.0.0",
			ID:        "tradsys-1",
			Namespace: "tradsys",
		},
		Resilience: ResilienceConfig{
			CircuitBreaker: CircuitBreakerConfig{
				Enabled:       true,
				Threshold:     5,
				Timeout:       30 * time.Second,
				HalfOpenLimit: 2,
			},
			RateLimiter: RateLimiterConfig{
				Enabled: true,
				Limit:   100,
				Burst:   10,
				Period:  1 * time.Second,
			},
			Retry: RetryConfig{
				Enabled:     true,
				MaxAttempts: 3,
				Delay:       100 * time.Millisecond,
				MaxDelay:    1 * time.Second,
				Multiplier:  2.0,
			},
			Bulkhead: BulkheadConfig{
				Enabled:      true,
				MaxConcurrent: 20,
				QueueSize:    50,
			},
			Timeout: TimeoutConfig{
				Enabled: true,
				Default: 5 * time.Second,
			},
		},
	}
}

