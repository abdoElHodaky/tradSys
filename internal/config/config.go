package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

// Config contains application configuration
type Config struct {
	Environment string
	Server      ServerConfig
	Database    DatabaseConfig
	JWT         JWTConfig
	Services    ServicesConfig
	Logging     LoggingConfig
	
	// Trading-specific configuration (merged from unified-config)
	Trading    TradingConfig
	Risk       RiskConfig
	
	// Additional fields for microservices architecture
	Service    ServiceConfig
	Gateway    GatewayConfig
	Registry   RegistryConfig
	Broker     BrokerConfig
	Tracing    TracingConfig
	Metrics    MetricsConfig
	Resilience ResilienceConfig
	Auth       AuthConfig
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
}

// JWTConfig represents the JWT configuration
type JWTConfig struct {
	SecretKey     string
	TokenDuration time.Duration
	Issuer        string
}

// ServicesConfig represents the services configuration
type ServicesConfig struct {
	MarketDataURL string
	OrdersURL     string
	RiskURL       string
}

// LoggingConfig represents the logging configuration
type LoggingConfig struct {
	Level      string
	OutputPath string
}

// ServiceConfig represents the service configuration
type ServiceConfig struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Address     string `json:"address"`
	Environment string `json:"environment"`
}

// GatewayConfig represents the gateway configuration
type GatewayConfig struct {
	Address                 string `json:"address"`
	ReadTimeout             int    `json:"readTimeout"`
	WriteTimeout            int    `json:"writeTimeout"`
	MaxHeaderBytes          int    `json:"maxHeaderBytes"`
	RateLimitRequests       int    `json:"rateLimitRequests"`
	RateLimitBurst          int    `json:"rateLimitBurst"`
	CircuitBreakerThreshold int    `json:"circuitBreakerThreshold"`
	CircuitBreakerTimeout   int    `json:"circuitBreakerTimeout"`
}

// RegistryConfig represents the registry configuration
type RegistryConfig struct {
	Type      string   `json:"type"`
	Addresses []string `json:"addresses"`
}

// BrokerConfig represents the broker configuration
type BrokerConfig struct {
	Type      string   `json:"type"`
	Addresses []string `json:"addresses"`
}

// TracingConfig represents the tracing configuration
type TracingConfig struct {
	Enabled bool   `json:"enabled"`
	Type    string `json:"type"`
	Address string `json:"address"`
}

// MetricsConfig represents the metrics configuration
type MetricsConfig struct {
	Enabled bool   `json:"enabled"`
	Address string `json:"address"`
}

// ResilienceConfig represents the resilience configuration
type ResilienceConfig struct {
	CircuitBreakerEnabled bool `json:"circuitBreakerEnabled"`
	RateLimitingEnabled   bool `json:"rateLimitingEnabled"`
}

// AuthConfig represents the auth configuration
type AuthConfig struct {
	JWTSecret     string `json:"jwtSecret"`
	TokenExpiry   int    `json:"tokenExpiry"`
	RefreshExpiry int    `json:"refreshExpiry"`
}

// TradingConfig contains trading-related configuration (merged from unified-config)
type TradingConfig struct {
	MaxOrderSize     float64 `json:"max_order_size" yaml:"max_order_size"`
	MaxPositionSize  float64 `json:"max_position_size" yaml:"max_position_size"`
	DefaultLeverage  float64 `json:"default_leverage" yaml:"default_leverage"`
	CommissionRate   float64 `json:"commission_rate" yaml:"commission_rate"`
}

// RiskConfig contains risk management configuration (merged from unified-config)
type RiskConfig struct {
	MaxDailyLoss     float64 `json:"max_daily_loss" yaml:"max_daily_loss"`
	MaxTotalLoss     float64 `json:"max_total_loss" yaml:"max_total_loss"`
	MinMarginLevel   float64 `json:"min_margin_level" yaml:"min_margin_level"`
	MarginCallLevel  float64 `json:"margin_call_level" yaml:"margin_call_level"`
	LiquidationLevel float64 `json:"liquidation_level" yaml:"liquidation_level"`
}

// LoadConfig loads the application configuration
func LoadConfig(configPath string, logger *zap.Logger) (*Config, error) {
	// Set default configuration values
	config := &Config{
		Environment: "development",
		Server: ServerConfig{
			Port:         8080,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  120 * time.Second,
		},
		Database: DatabaseConfig{
			Driver:   "sqlite3",
			Host:     "localhost",
			Port:     5432,
			Username: "postgres",
			Password: "postgres",
			Database: "trading",
			SSLMode:  "disable",
		},
		JWT: JWTConfig{
			SecretKey:     "default-secret-key",
			TokenDuration: 24 * time.Hour,
			Issuer:        "trading-system",
		},
		Services: ServicesConfig{
			MarketDataURL: "localhost:50051",
			OrdersURL:     "localhost:50051",
			RiskURL:       "localhost:50051",
		},
		Logging: LoggingConfig{
			Level:      "info",
			OutputPath: "stdout",
		},
		Service: ServiceConfig{
			Name:        "tradsys",
			Version:     "1.0.0",
			Address:     ":8080",
			Environment: "development",
		},
		Gateway: GatewayConfig{
			Address:                 ":8000",
			ReadTimeout:             5000,
			WriteTimeout:            10000,
			MaxHeaderBytes:          1 << 20, // 1MB
			RateLimitRequests:       100,
			RateLimitBurst:          200,
			CircuitBreakerThreshold: 5,
			CircuitBreakerTimeout:   30,
		},
		Registry: RegistryConfig{
			Type:      "mdns",
			Addresses: []string{},
		},
		Broker: BrokerConfig{
			Type:      "http",
			Addresses: []string{},
		},
		Tracing: TracingConfig{
			Enabled: false,
			Type:    "jaeger",
			Address: "localhost:6831",
		},
		Metrics: MetricsConfig{
			Enabled: false,
			Address: ":9090",
		},
		Resilience: ResilienceConfig{
			CircuitBreakerEnabled: false,
			RateLimitingEnabled:   false,
		},
		Auth: AuthConfig{
			JWTSecret:     "default-jwt-secret-change-in-production",
			TokenExpiry:   3600,
			RefreshExpiry: 86400,
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
	}

	// Initialize Viper
	v := viper.New()

	// Set configuration file path
	if configPath != "" {
		// Get the directory and file name from the config path
		dir, file := filepath.Split(configPath)
		ext := filepath.Ext(file)
		name := strings.TrimSuffix(file, ext)

		// Set the configuration file properties
		v.AddConfigPath(dir)
		v.SetConfigName(name)
		v.SetConfigType(strings.TrimPrefix(ext, "."))
	} else {
		// Set default configuration file properties
		v.AddConfigPath(".")
		v.AddConfigPath("./config")
		v.AddConfigPath("/etc/trading")
		v.SetConfigName("config")
		v.SetConfigType("yaml")
	}

	// Read environment variables
	v.AutomaticEnv()
	v.SetEnvPrefix("TRADING")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Read configuration file
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			logger.Warn("Config file not found, using default values and environment variables")
		} else {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	} else {
		logger.Info("Using config file", zap.String("file", v.ConfigFileUsed()))
	}

	// Override configuration with environment variables
	if env := os.Getenv("TRADING_ENVIRONMENT"); env != "" {
		config.Environment = env
	} else if v.IsSet("environment") {
		config.Environment = v.GetString("environment")
	}

	// Server configuration
	if v.IsSet("server.port") {
		config.Server.Port = v.GetInt("server.port")
	}
	if v.IsSet("server.read_timeout") {
		config.Server.ReadTimeout = v.GetDuration("server.read_timeout")
	}
	if v.IsSet("server.write_timeout") {
		config.Server.WriteTimeout = v.GetDuration("server.write_timeout")
	}
	if v.IsSet("server.idle_timeout") {
		config.Server.IdleTimeout = v.GetDuration("server.idle_timeout")
	}

	// Database configuration
	if v.IsSet("database.driver") {
		config.Database.Driver = v.GetString("database.driver")
	}
	if v.IsSet("database.host") {
		config.Database.Host = v.GetString("database.host")
	}
	if v.IsSet("database.port") {
		config.Database.Port = v.GetInt("database.port")
	}
	if v.IsSet("database.username") {
		config.Database.Username = v.GetString("database.username")
	}
	if v.IsSet("database.password") {
		config.Database.Password = v.GetString("database.password")
	}
	if v.IsSet("database.database") {
		config.Database.Database = v.GetString("database.database")
	}
	if v.IsSet("database.ssl_mode") {
		config.Database.SSLMode = v.GetString("database.ssl_mode")
	}

	// JWT configuration
	if v.IsSet("jwt.secret_key") {
		config.JWT.SecretKey = v.GetString("jwt.secret_key")
	}
	if v.IsSet("jwt.token_duration") {
		config.JWT.TokenDuration = v.GetDuration("jwt.token_duration")
	}
	if v.IsSet("jwt.issuer") {
		config.JWT.Issuer = v.GetString("jwt.issuer")
	}

	// Services configuration
	if v.IsSet("services.market_data_url") {
		config.Services.MarketDataURL = v.GetString("services.market_data_url")
	}
	if v.IsSet("services.orders_url") {
		config.Services.OrdersURL = v.GetString("services.orders_url")
	}
	if v.IsSet("services.risk_url") {
		config.Services.RiskURL = v.GetString("services.risk_url")
	}

	// Logging configuration
	if v.IsSet("logging.level") {
		config.Logging.Level = v.GetString("logging.level")
	}
	if v.IsSet("logging.output_path") {
		config.Logging.OutputPath = v.GetString("logging.output_path")
	}

	return config, nil
}

// NewConfig creates a new configuration with default values
func NewConfig(logger *zap.Logger) (*Config, error) {
	return LoadConfig("", logger)
}

// Module provides the configuration module for fx
var Module = fx.Options(
	fx.Provide(NewConfig),
)

// Load loads configuration from the specified YAML file (merged from unified-config)
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

// LoadFromDir loads configuration from a directory (merged from unified-config)
func LoadFromDir(dir string) (*Config, error) {
	configPath := filepath.Join(dir, "tradsys.yaml")
	return Load(configPath)
}

// Default returns a default configuration (merged from unified-config)
func Default() *Config {
	return &Config{
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
			Database: "tradsys",
			Username: "tradsys",
			Password: "password",
			SSLMode:  "disable",
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
			Level:      "info",
			OutputPath: "stdout",
		},
	}
}
