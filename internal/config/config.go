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
)

// Config contains application configuration
type Config struct {
	Environment string
	Server      ServerConfig
	Database    DatabaseConfig
	JWT         JWTConfig
	Services    ServicesConfig
	Logging     LoggingConfig
	Registry    RegistryConfig
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

// RegistryConfig represents the service registry configuration
type RegistryConfig struct {
	Type             string
	Addresses        []string
	TTL              time.Duration
	RegisterInterval time.Duration
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
		Registry: RegistryConfig{
			Type:             "mdns",
			Addresses:        []string{},
			TTL:              30 * time.Second,
			RegisterInterval: 15 * time.Second,
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

	// Registry configuration
	if v.IsSet("registry.type") {
		config.Registry.Type = v.GetString("registry.type")
	}
	if v.IsSet("registry.addresses") {
		config.Registry.Addresses = v.GetStringSlice("registry.addresses")
	}
	if v.IsSet("registry.ttl") {
		config.Registry.TTL = v.GetDuration("registry.ttl")
	}
	if v.IsSet("registry.register_interval") {
		config.Registry.RegisterInterval = v.GetDuration("registry.register_interval")
	}

	return config, nil
}

=======
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

>>>>>>> pr-15
