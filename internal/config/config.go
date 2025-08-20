package config

import (
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Config contains application configuration
type Config struct {
	// Service configuration
	Service struct {
		Name        string `json:"name"`
		Version     string `json:"version"`
		Address     string `json:"address"`
		Environment string `json:"environment"`
	} `json:"service"`

	// Gateway configuration
	Gateway struct {
		Address            string `json:"address"`
		ReadTimeout        int    `json:"readTimeout"`
		WriteTimeout       int    `json:"writeTimeout"`
		MaxHeaderBytes     int    `json:"maxHeaderBytes"`
		RateLimitRequests  int    `json:"rateLimitRequests"`
		RateLimitBurst     int    `json:"rateLimitBurst"`
		CircuitBreakerThreshold int `json:"circuitBreakerThreshold"`
		CircuitBreakerTimeout   int `json:"circuitBreakerTimeout"`
	} `json:"gateway"`

	// Registry configuration
	Registry struct {
		Type      string   `json:"type"`
		Addresses []string `json:"addresses"`
	} `json:"registry"`

	// Broker configuration
	Broker struct {
		Type      string   `json:"type"`
		Addresses []string `json:"addresses"`
	} `json:"broker"`

	// Database configuration
	Database struct {
		Type     string `json:"type"`
		Address  string `json:"address"`
		Username string `json:"username"`
		Password string `json:"password"`
		Database string `json:"database"`
	} `json:"database"`

	// Tracing configuration
	Tracing struct {
		Enabled bool   `json:"enabled"`
		Type    string `json:"type"`
		Address string `json:"address"`
	} `json:"tracing"`

	// Metrics configuration
	Metrics struct {
		Enabled bool   `json:"enabled"`
		Address string `json:"address"`
	} `json:"metrics"`

	// Resilience configuration
	Resilience struct {
		CircuitBreakerEnabled bool `json:"circuitBreakerEnabled"`
		RateLimitingEnabled   bool `json:"rateLimitingEnabled"`
	} `json:"resilience"`

	// Auth configuration
	Auth struct {
		JWTSecret     string `json:"jwtSecret"`
		TokenExpiry   int    `json:"tokenExpiry"`
		RefreshExpiry int    `json:"refreshExpiry"`
	} `json:"auth"`
}

// NewConfig creates a new configuration with default values
func NewConfig(logger *zap.Logger) (*Config, error) {
	// Create a default config
	cfg := &Config{}

	// Set default values
	cfg.Service.Name = "tradsys"
	cfg.Service.Version = "1.0.0"
	cfg.Service.Address = ":8080"
	cfg.Service.Environment = "development"

	cfg.Gateway.Address = ":8000"
	cfg.Gateway.ReadTimeout = 5000
	cfg.Gateway.WriteTimeout = 10000
	cfg.Gateway.MaxHeaderBytes = 1 << 20 // 1MB
	cfg.Gateway.RateLimitRequests = 100
	cfg.Gateway.RateLimitBurst = 200
	cfg.Gateway.CircuitBreakerThreshold = 5
	cfg.Gateway.CircuitBreakerTimeout = 30

	cfg.Registry.Type = "mdns"
	cfg.Registry.Addresses = []string{}

	cfg.Broker.Type = "http"
	cfg.Broker.Addresses = []string{}

	cfg.Database.Type = "postgres"
	cfg.Database.Address = "localhost:5432"
	cfg.Database.Username = "postgres"
	cfg.Database.Password = "postgres"
	cfg.Database.Database = "tradsys"

	cfg.Tracing.Enabled = false
	cfg.Tracing.Type = "jaeger"
	cfg.Tracing.Address = "localhost:6831"

	cfg.Metrics.Enabled = false
	cfg.Metrics.Address = ":9090"

	cfg.Resilience.CircuitBreakerEnabled = false
	cfg.Resilience.RateLimitingEnabled = false

	cfg.Auth.JWTSecret = "default-jwt-secret-change-in-production"
	cfg.Auth.TokenExpiry = 3600
	cfg.Auth.RefreshExpiry = 86400

	logger.Info("Configuration initialized with default values")

	return cfg, nil
}

// Module provides the configuration module for fx
var Module = fx.Options(
	fx.Provide(NewConfig),
)

