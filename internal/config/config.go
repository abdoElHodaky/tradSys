package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// Config represents the application configuration
type Config struct {
	// Server configuration
	Server struct {
		Host string `mapstructure:"host"`
		Port int    `mapstructure:"port"`
	} `mapstructure:"server"`

	// Database configuration
	Database struct {
		Host     string `mapstructure:"host"`
		Port     int    `mapstructure:"port"`
		User     string `mapstructure:"user"`
		Password string `mapstructure:"password"`
		Name     string `mapstructure:"name"`
		SSLMode  string `mapstructure:"sslmode"`
	} `mapstructure:"database"`

	// WebSocket configuration
	WebSocket struct {
		Host           string `mapstructure:"host"`
		Port           int    `mapstructure:"port"`
		Path           string `mapstructure:"path"`
		MaxConnections int    `mapstructure:"max_connections"`
	} `mapstructure:"websocket"`

	// PeerJS configuration
	PeerJS struct {
		Host string `mapstructure:"host"`
		Port int    `mapstructure:"port"`
		Path string `mapstructure:"path"`
	} `mapstructure:"peerjs"`

	// Market data configuration
	MarketData struct {
		Sources []string `mapstructure:"sources"`
		Symbols []string `mapstructure:"symbols"`
	} `mapstructure:"market_data"`

	// Risk management configuration
	Risk struct {
		MaxOrderValue     float64 `mapstructure:"max_order_value"`
		MaxPositionValue  float64 `mapstructure:"max_position_value"`
		MaxDrawdown       float64 `mapstructure:"max_drawdown"`
		MaxLeverage       float64 `mapstructure:"max_leverage"`
		StopLossPercent   float64 `mapstructure:"stop_loss_percent"`
		TakeProfitPercent float64 `mapstructure:"take_profit_percent"`
	} `mapstructure:"risk"`

	// Monitoring configuration
	Monitoring struct {
		PrometheusPort int    `mapstructure:"prometheus_port"`
		LogLevel       string `mapstructure:"log_level"`
	} `mapstructure:"monitoring"`

	// Authentication configuration
	Auth struct {
		JWTSecret     string `mapstructure:"jwt_secret"`
		TokenDuration int    `mapstructure:"token_duration"` // in minutes
	} `mapstructure:"auth"`
}

var (
	config *Config
	once   sync.Once
)

// LoadConfig loads the configuration from the specified file
func LoadConfig(configPath string) (*Config, error) {
	var err error

	once.Do(func() {
		config = &Config{}

		// Set default values
		setDefaults()

		// Initialize viper
		v := viper.New()
		v.SetConfigName("config")
		v.SetConfigType("yaml")

		// Add config path
		if configPath != "" {
			v.AddConfigPath(configPath)
		} else {
			v.AddConfigPath(".")
			v.AddConfigPath("./config")
			v.AddConfigPath("/etc/tradsys")
		}

		// Read environment variables
		v.AutomaticEnv()
		v.SetEnvPrefix("TRADSYS")

		// Read config file
		if err = v.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
				err = fmt.Errorf("failed to read config file: %w", err)
				return
			}
			// Config file not found, using defaults and environment variables
			err = nil
		}

		// Unmarshal config
		if err = v.Unmarshal(config); err != nil {
			err = fmt.Errorf("failed to unmarshal config: %w", err)
			return
		}
	})

	return config, err
}

// GetConfig returns the current configuration
func GetConfig() *Config {
	if config == nil {
		_, err := LoadConfig("")
		if err != nil {
			panic(fmt.Sprintf("failed to load config: %v", err))
		}
	}
	return config
}

// SaveConfig saves the configuration to a file
func SaveConfig(config *Config, path string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Marshal config to JSON
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// setDefaults sets default values for the configuration
func setDefaults() {
	// Server defaults
	config.Server.Host = "0.0.0.0"
	config.Server.Port = 8080

	// Database defaults
	config.Database.Host = "localhost"
	config.Database.Port = 5432
	config.Database.User = "postgres"
	config.Database.Name = "tradsys"
	config.Database.SSLMode = "disable"

	// WebSocket defaults
	config.WebSocket.Host = "0.0.0.0"
	config.WebSocket.Port = 8081
	config.WebSocket.Path = "/ws"
	config.WebSocket.MaxConnections = 1000

	// PeerJS defaults
	config.PeerJS.Host = "0.0.0.0"
	config.PeerJS.Port = 9000
	config.PeerJS.Path = "/peerjs"

	// Market data defaults
	config.MarketData.Sources = []string{"internal"}
	config.MarketData.Symbols = []string{"BTC/USD", "ETH/USD"}

	// Risk defaults
	config.Risk.MaxOrderValue = 10000.0
	config.Risk.MaxPositionValue = 50000.0
	config.Risk.MaxDrawdown = 0.1
	config.Risk.MaxLeverage = 5.0
	config.Risk.StopLossPercent = 0.05
	config.Risk.TakeProfitPercent = 0.1

	// Monitoring defaults
	config.Monitoring.PrometheusPort = 9090
	config.Monitoring.LogLevel = "info"

	// Auth defaults
	config.Auth.TokenDuration = 60 // 1 hour
}

// InitLogger initializes the logger based on the configuration
func InitLogger(cfg *Config) (*zap.Logger, error) {
	var logger *zap.Logger
	var err error

	switch cfg.Monitoring.LogLevel {
	case "debug":
		logger, err = zap.NewDevelopment()
	case "info", "warn", "error":
		logger, err = zap.NewProduction()
	default:
		logger, err = zap.NewProduction()
	}

	if err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}

	return logger, nil
}
