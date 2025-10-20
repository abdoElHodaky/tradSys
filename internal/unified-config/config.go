package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the unified configuration structure
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Trading  TradingConfig  `yaml:"trading"`
	Risk     RiskConfig     `yaml:"risk"`
	Logging  LoggingConfig  `yaml:"logging"`
}

// ServerConfig contains server-related configuration
type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

// DatabaseConfig contains database-related configuration
type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Database string `yaml:"database"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// TradingConfig contains trading-related configuration
type TradingConfig struct {
	MaxOrderSize     float64 `yaml:"max_order_size"`
	MaxPositionSize  float64 `yaml:"max_position_size"`
	DefaultLeverage  float64 `yaml:"default_leverage"`
	CommissionRate   float64 `yaml:"commission_rate"`
}

// RiskConfig contains risk management configuration
type RiskConfig struct {
	MaxDailyLoss     float64 `yaml:"max_daily_loss"`
	MaxTotalLoss     float64 `yaml:"max_total_loss"`
	MinMarginLevel   float64 `yaml:"min_margin_level"`
	MarginCallLevel  float64 `yaml:"margin_call_level"`
	LiquidationLevel float64 `yaml:"liquidation_level"`
}

// LoggingConfig contains logging configuration
type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

// Load loads configuration from the specified file
func Load(configPath string) (*Config, error) {
	if configPath == "" {
		configPath = "config/tradsys.yaml"
	}

	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file not found: %s", configPath)
	}

	// Read the file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// LoadFromDir loads configuration from a directory
func LoadFromDir(dir string) (*Config, error) {
	configPath := filepath.Join(dir, "tradsys.yaml")
	return Load(configPath)
}

// Default returns a default configuration
func Default() *Config {
	return &Config{
		Server: ServerConfig{
			Host: "localhost",
			Port: 8080,
		},
		Database: DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			Database: "tradsys",
			Username: "tradsys",
			Password: "password",
		},
		Trading: TradingConfig{
			MaxOrderSize:    1000000.0,
			MaxPositionSize: 5000000.0,
			DefaultLeverage: 1.0,
			CommissionRate:  0.001,
		},
		Risk: RiskConfig{
			MaxDailyLoss:     10000.0,
			MaxTotalLoss:     50000.0,
			MinMarginLevel:   120.0,
			MarginCallLevel:  120.0,
			LiquidationLevel: 100.0,
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
		},
	}
}
