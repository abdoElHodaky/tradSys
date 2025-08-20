package config

import (
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Config contains application configuration
type Config struct {
	// Service configuration
	Service struct {
		Name    string `json:"name"`
		Version string `json:"version"`
		Address string `json:"address"`
	} `json:"service"`

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
}

// NewConfig creates a new configuration with default values
func NewConfig(logger *zap.Logger) (*Config, error) {
	// Create a default config
	cfg := &Config{}

	// Set default values
	cfg.Service.Name = "tradsys"
	cfg.Service.Version = "1.0.0"
	cfg.Service.Address = ":8080"

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

	logger.Info("Configuration initialized with default values")

	return cfg, nil
}

// Module provides the configuration module for fx
var Module = fx.Options(
	fx.Provide(NewConfig),
)

