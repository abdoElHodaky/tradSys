package unifiedconfig

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the unified configuration for TradSys
type Config struct {
	Server       ServerConfig       `yaml:"server"`
	Core         CoreConfig         `yaml:"core"`
	Connectivity ConnectivityConfig `yaml:"connectivity"`
	Compliance   ComplianceConfig   `yaml:"compliance"`
	Strategies   StrategiesConfig   `yaml:"strategies"`
	Database     DatabaseConfig     `yaml:"database"`
	Redis        RedisConfig        `yaml:"redis"`
	Logging      LoggingConfig      `yaml:"logging"`
	Metrics      MetricsConfig      `yaml:"metrics"`
	Security     SecurityConfig     `yaml:"security"`
}

// ServerConfig contains HTTP server configuration
type ServerConfig struct {
	Port         int           `yaml:"port"`
	Host         string        `yaml:"host"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
	IdleTimeout  time.Duration `yaml:"idle_timeout"`
	TLS          TLSConfig     `yaml:"tls"`
}

// CoreConfig contains core trading engine configuration
type CoreConfig struct {
	OrderMatching OrderMatchingConfig `yaml:"order_matching"`
	RiskEngine    RiskEngineConfig    `yaml:"risk_engine"`
	Settlement    SettlementConfig    `yaml:"settlement"`
}

// ConnectivityConfig contains exchange connectivity configuration
type ConnectivityConfig struct {
	Exchanges []ExchangeConfig `yaml:"exchanges"`
	Timeout   time.Duration    `yaml:"timeout"`
	Retries   int              `yaml:"retries"`
}

// ComplianceConfig contains compliance engine configuration
type ComplianceConfig struct {
	Rules       []ComplianceRule `yaml:"rules"`
	AuditLog    AuditLogConfig   `yaml:"audit_log"`
	Reporting   ReportingConfig  `yaml:"reporting"`
}

// StrategiesConfig contains algorithmic strategies configuration
type StrategiesConfig struct {
	Enabled    []string          `yaml:"enabled"`
	Parameters map[string]interface{} `yaml:"parameters"`
}

// DatabaseConfig contains database configuration
type DatabaseConfig struct {
	Driver   string `yaml:"driver"`
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Database string `yaml:"database"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	SSLMode  string `yaml:"ssl_mode"`
	MaxConns int    `yaml:"max_connections"`
}

// RedisConfig contains Redis configuration
type RedisConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Password string `yaml:"password"`
	Database int    `yaml:"database"`
}

// LoggingConfig contains logging configuration
type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
	Output string `yaml:"output"`
}

// MetricsConfig contains metrics configuration
type MetricsConfig struct {
	Enabled    bool   `yaml:"enabled"`
	Port       int    `yaml:"port"`
	Path       string `yaml:"path"`
	Namespace  string `yaml:"namespace"`
}

// SecurityConfig contains security configuration
type SecurityConfig struct {
	JWT       JWTConfig       `yaml:"jwt"`
	RateLimit RateLimitConfig `yaml:"rate_limit"`
}

// Supporting configuration types
type TLSConfig struct {
	Enabled  bool   `yaml:"enabled"`
	CertFile string `yaml:"cert_file"`
	KeyFile  string `yaml:"key_file"`
}

type OrderMatchingConfig struct {
	Algorithm string        `yaml:"algorithm"`
	Timeout   time.Duration `yaml:"timeout"`
}

type RiskEngineConfig struct {
	MaxPositionSize float64       `yaml:"max_position_size"`
	MaxOrderValue   float64       `yaml:"max_order_value"`
	CheckTimeout    time.Duration `yaml:"check_timeout"`
}

type SettlementConfig struct {
	BatchSize int           `yaml:"batch_size"`
	Timeout   time.Duration `yaml:"timeout"`
}

type ExchangeConfig struct {
	Name     string `yaml:"name"`
	Endpoint string `yaml:"endpoint"`
	APIKey   string `yaml:"api_key"`
	Secret   string `yaml:"secret"`
	Enabled  bool   `yaml:"enabled"`
}

type ComplianceRule struct {
	Name        string      `yaml:"name"`
	Type        string      `yaml:"type"`
	Parameters  interface{} `yaml:"parameters"`
	Enabled     bool        `yaml:"enabled"`
}

type AuditLogConfig struct {
	Enabled bool   `yaml:"enabled"`
	Path    string `yaml:"path"`
}

type ReportingConfig struct {
	Enabled   bool          `yaml:"enabled"`
	Frequency time.Duration `yaml:"frequency"`
	Output    string        `yaml:"output"`
}

type JWTConfig struct {
	Secret     string        `yaml:"secret"`
	Expiration time.Duration `yaml:"expiration"`
}

type RateLimitConfig struct {
	Enabled bool `yaml:"enabled"`
	RPS     int  `yaml:"rps"`
}

// Load loads configuration from file and environment variables
func Load() (*Config, error) {
	configPath := os.Getenv("TRADSYS_CONFIG_PATH")
	if configPath == "" {
		configPath = "config/tradsys-config.yaml"
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", configPath, err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Override with environment variables
	overrideWithEnv(&config)

	// Validate configuration
	if err := validate(&config); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return &config, nil
}

// overrideWithEnv overrides configuration with environment variables
func overrideWithEnv(config *Config) {
	if port := os.Getenv("TRADSYS_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			config.Server.Port = p
		}
	}

	if host := os.Getenv("TRADSYS_HOST"); host != "" {
		config.Server.Host = host
	}

	if logLevel := os.Getenv("TRADSYS_LOG_LEVEL"); logLevel != "" {
		config.Logging.Level = logLevel
	}

	if dbHost := os.Getenv("TRADSYS_DB_HOST"); dbHost != "" {
		config.Database.Host = dbHost
	}

	if dbPassword := os.Getenv("TRADSYS_DB_PASSWORD"); dbPassword != "" {
		config.Database.Password = dbPassword
	}

	if redisHost := os.Getenv("TRADSYS_REDIS_HOST"); redisHost != "" {
		config.Redis.Host = redisHost
	}

	if jwtSecret := os.Getenv("TRADSYS_JWT_SECRET"); jwtSecret != "" {
		config.Security.JWT.Secret = jwtSecret
	}
}

// validate validates the configuration
func validate(config *Config) error {
	if config.Server.Port <= 0 || config.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", config.Server.Port)
	}

	if config.Database.Driver == "" {
		return fmt.Errorf("database driver is required")
	}

	if config.Security.JWT.Secret == "" {
		return fmt.Errorf("JWT secret is required")
	}

	return nil
}

// GetDefault returns a default configuration
func GetDefault() *Config {
	return &Config{
		Server: ServerConfig{
			Port:         8080,
			Host:         "0.0.0.0",
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  120 * time.Second,
		},
		Core: CoreConfig{
			OrderMatching: OrderMatchingConfig{
				Algorithm: "fifo",
				Timeout:   100 * time.Microsecond,
			},
			RiskEngine: RiskEngineConfig{
				MaxPositionSize: 1000000,
				MaxOrderValue:   100000,
				CheckTimeout:    10 * time.Microsecond,
			},
			Settlement: SettlementConfig{
				BatchSize: 100,
				Timeout:   1 * time.Millisecond,
			},
		},
		Database: DatabaseConfig{
			Driver:   "postgres",
			Host:     "localhost",
			Port:     5432,
			Database: "tradsys",
			Username: "tradsys",
			SSLMode:  "disable",
			MaxConns: 10,
		},
		Redis: RedisConfig{
			Host:     "localhost",
			Port:     6379,
			Database: 0,
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
			Output: "stdout",
		},
		Metrics: MetricsConfig{
			Enabled:   true,
			Port:      9090,
			Path:      "/metrics",
			Namespace: "tradsys",
		},
		Security: SecurityConfig{
			JWT: JWTConfig{
				Expiration: 24 * time.Hour,
			},
			RateLimit: RateLimitConfig{
				Enabled: true,
				RPS:     1000,
			},
		},
	}
}
