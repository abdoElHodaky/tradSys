package config

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
	
	"github.com/abdoElHodaky/tradSys/internal/trading/memory"
)

// HFTConfigManager manages configuration for HFT components
type HFTConfigManager struct {
	// Configuration sources
	viper      *viper.Viper
	configPath string
	env        string
	
	// Current configuration
	config     atomic.Value // *HFTManagerConfig
	
	// Hot reload
	watcher    *fsnotify.Watcher
	reloadChan chan struct{}
	
	// Callbacks
	callbacks  []func(*HFTManagerConfig)
	cbLock     sync.RWMutex
	
	// Control
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
}

// Type aliases for imported configs
type HFTMemoryConfig = memory.HFTMemoryConfig

// HFTMonitoringConfig contains monitoring configuration
type HFTMonitoringConfig struct {
	// Metrics collection
	EnablePrometheus    bool          `yaml:"enable_prometheus" default:"true"`
	EnableCustomMetrics bool          `yaml:"enable_custom_metrics" default:"true"`
	MetricsInterval     time.Duration `yaml:"metrics_interval" default:"10s"`
	
	// Health checks
	EnableHealthChecks  bool          `yaml:"enable_health_checks" default:"true"`
	HealthCheckInterval time.Duration `yaml:"health_check_interval" default:"30s"`
	
	// Alerting
	EnableAlerting      bool          `yaml:"enable_alerting" default:"true"`
	AlertThresholds     AlertThresholds `yaml:"alert_thresholds"`
	
	// Performance monitoring
	EnablePerformanceMonitoring bool          `yaml:"enable_performance_monitoring" default:"true"`
	PerformanceInterval         time.Duration `yaml:"performance_interval" default:"5s"`
	
	// Dashboard
	EnableDashboard bool   `yaml:"enable_dashboard" default:"true"`
	DashboardPort   int    `yaml:"dashboard_port" default:"9090"`
	DashboardPath   string `yaml:"dashboard_path" default:"/dashboard"`
}

// AlertThresholds contains alerting thresholds
type AlertThresholds struct {
	MaxLatency        time.Duration `yaml:"max_latency" default:"100ms"`
	MaxErrorRate      float64       `yaml:"max_error_rate" default:"0.01"`      // 1%
	MaxMemoryUsage    int64         `yaml:"max_memory_usage" default:"1073741824"` // 1GB
	MaxGCPauseTime    time.Duration `yaml:"max_gc_pause_time" default:"10ms"`
	MinThroughput     int64         `yaml:"min_throughput" default:"1000"`     // requests/sec
}

// HFTManagerConfig contains all HFT configuration for the manager
type HFTManagerConfig struct {
	// Environment
	Environment string `yaml:"environment" default:"development"`
	
	// Memory settings
	Memory HFTMemoryConfig `yaml:"memory"`
	
	// Database settings
	Database struct {
		Driver     string `yaml:"driver" default:"sqlite3"`
		DSN        string `yaml:"dsn" default:"trading.db"`
		MaxConns   int    `yaml:"max_conns" default:"10"`
		EnableWAL  bool   `yaml:"enable_wal" default:"true"`
	} `yaml:"database"`
	
	// WebSocket settings
	WebSocket struct {
		Port            int           `yaml:"port" default:"8080"`
		ReadBufferSize  int           `yaml:"read_buffer_size" default:"4096"`
		WriteBufferSize int           `yaml:"write_buffer_size" default:"4096"`
		PingInterval    time.Duration `yaml:"ping_interval" default:"30s"`
		BinaryProtocol  bool          `yaml:"binary_protocol" default:"true"`
	} `yaml:"websocket"`
	
	// Monitoring settings
	Monitoring HFTMonitoringConfig `yaml:"monitoring"`
	
	// Security settings
	Security struct {
		JWTSecret     string        `yaml:"jwt_secret"`
		TokenExpiry   time.Duration `yaml:"token_expiry" default:"24h"`
		EnableTLS     bool          `yaml:"enable_tls" default:"false"`
		TLSCertFile   string        `yaml:"tls_cert_file"`
		TLSKeyFile    string        `yaml:"tls_key_file"`
	} `yaml:"security"`
	
	// Rate limiting
	RateLimit struct {
		RequestsPerSecond int           `yaml:"requests_per_second" default:"1000"`
		BurstSize        int           `yaml:"burst_size" default:"100"`
		WindowSize       time.Duration `yaml:"window_size" default:"1s"`
	} `yaml:"rate_limit"`
	
	// Circuit breaker
	CircuitBreaker struct {
		MaxFailures      int           `yaml:"max_failures" default:"5"`
		ResetTimeout     time.Duration `yaml:"reset_timeout" default:"30s"`
		FailureRatio     float64       `yaml:"failure_ratio" default:"0.5"`
	} `yaml:"circuit_breaker"`
	
	// Timeouts
	Timeouts struct {
		HTTPRead    time.Duration `yaml:"http_read" default:"5s"`
		HTTPWrite   time.Duration `yaml:"http_write" default:"10s"`
		GRPCRequest time.Duration `yaml:"grpc_request" default:"10s"`
	} `yaml:"timeouts"`
	
	// GC configuration
	GC HFTGCConfig `yaml:"gc"`
	
	// Gin configuration
	Gin HFTGinConfig `yaml:"gin"`
}

// NewHFTConfigManager creates a new configuration manager
func NewHFTConfigManager(configPath string, env string) (*HFTConfigManager, error) {
	// Create watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create config watcher: %w", err)
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	mgr := &HFTConfigManager{
		viper:      viper.New(),
		configPath: configPath,
		env:        env,
		watcher:    watcher,
		reloadChan: make(chan struct{}, 1),
		ctx:        ctx,
		cancel:     cancel,
	}
	
	// Initialize viper
	mgr.viper.SetConfigFile(configPath)
	mgr.viper.SetEnvPrefix("HFT")
	mgr.viper.AutomaticEnv()
	
	// Set defaults
	mgr.setDefaults()
	
	// Load initial configuration
	if err := mgr.loadConfig(); err != nil {
		return nil, err
	}
	
	// Start watching for changes
	if err := mgr.startWatcher(); err != nil {
		return nil, err
	}
	
	return mgr, nil
}

// setDefaults sets default configuration values
func (m *HFTConfigManager) setDefaults() {
	// Environment
	m.viper.SetDefault("environment", "development")
	
	// Database defaults
	m.viper.SetDefault("database.driver", "sqlite3")
	m.viper.SetDefault("database.dsn", "trading.db")
	m.viper.SetDefault("database.max_conns", 10)
	m.viper.SetDefault("database.enable_wal", true)
	
	// WebSocket defaults
	m.viper.SetDefault("websocket.port", 8080)
	m.viper.SetDefault("websocket.read_buffer_size", 4096)
	m.viper.SetDefault("websocket.write_buffer_size", 4096)
	m.viper.SetDefault("websocket.ping_interval", "30s")
	m.viper.SetDefault("websocket.binary_protocol", true)
	
	// Security defaults
	m.viper.SetDefault("security.token_expiry", "24h")
	m.viper.SetDefault("security.enable_tls", false)
	
	// Rate limiting defaults
	m.viper.SetDefault("rate_limit.requests_per_second", 1000)
	m.viper.SetDefault("rate_limit.burst_size", 100)
	m.viper.SetDefault("rate_limit.window_size", "1s")
	
	// Circuit breaker defaults
	m.viper.SetDefault("circuit_breaker.max_failures", 5)
	m.viper.SetDefault("circuit_breaker.reset_timeout", "30s")
	m.viper.SetDefault("circuit_breaker.failure_ratio", 0.5)
	
	// Timeout defaults
	m.viper.SetDefault("timeouts.http_read", "5s")
	m.viper.SetDefault("timeouts.http_write", "10s")
	m.viper.SetDefault("timeouts.grpc_request", "10s")
	
	// GC defaults
	m.viper.SetDefault("gc.gc_percent", 200)
	m.viper.SetDefault("gc.memory_limit", 2147483648) // 2GB
	m.viper.SetDefault("gc.enable_memory_limit", true)
	m.viper.SetDefault("gc.enable_gc_monitoring", true)
	m.viper.SetDefault("gc.gc_stats_interval", "30s")
	
	// Memory defaults
	m.viper.SetDefault("memory.enable_object_pools", true)
	m.viper.SetDefault("memory.enable_buffer_pools", true)
	m.viper.SetDefault("memory.enable_string_pools", true)
	m.viper.SetDefault("memory.max_heap_size", 2147483648) // 2GB
	m.viper.SetDefault("memory.gc_target_percentage", 200)
	m.viper.SetDefault("memory.enable_memory_monitoring", true)
	m.viper.SetDefault("memory.monitoring_interval", "10s")
	
	// Monitoring defaults
	m.viper.SetDefault("monitoring.enable_prometheus", true)
	m.viper.SetDefault("monitoring.enable_custom_metrics", true)
	m.viper.SetDefault("monitoring.metrics_interval", "10s")
	m.viper.SetDefault("monitoring.enable_health_checks", true)
	m.viper.SetDefault("monitoring.health_check_interval", "30s")
	m.viper.SetDefault("monitoring.enable_alerting", true)
	m.viper.SetDefault("monitoring.enable_performance_monitoring", true)
	m.viper.SetDefault("monitoring.performance_interval", "5s")
	m.viper.SetDefault("monitoring.enable_dashboard", true)
	m.viper.SetDefault("monitoring.dashboard_port", 9090)
	m.viper.SetDefault("monitoring.dashboard_path", "/dashboard")
	
	// Alert thresholds
	m.viper.SetDefault("monitoring.alert_thresholds.max_latency", "100ms")
	m.viper.SetDefault("monitoring.alert_thresholds.max_error_rate", 0.01)
	m.viper.SetDefault("monitoring.alert_thresholds.max_memory_usage", 1073741824) // 1GB
	m.viper.SetDefault("monitoring.alert_thresholds.max_gc_pause_time", "10ms")
	m.viper.SetDefault("monitoring.alert_thresholds.min_throughput", 1000)
}

// loadConfig loads configuration from file and environment
func (m *HFTConfigManager) loadConfig() error {
	// Read config file if it exists
	if _, err := os.Stat(m.configPath); err == nil {
		if err := m.viper.ReadInConfig(); err != nil {
			return fmt.Errorf("failed to read config file: %w", err)
		}
	}
	
	// Create new config
	config := &HFTManagerConfig{}
	
	// Unmarshal config
	if err := m.viper.Unmarshal(config); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}
	
	// Set environment
	config.Environment = m.env
	
	// Store config
	m.config.Store(config)
	
	// Notify callbacks
	m.notifyCallbacks(config)
	
	return nil
}

// startWatcher starts watching for configuration changes
func (m *HFTConfigManager) startWatcher() error {
	// Watch config file directory
	configDir := filepath.Dir(m.configPath)
	if err := m.watcher.Add(configDir); err != nil {
		return fmt.Errorf("failed to watch config directory: %w", err)
	}
	
	// Start watcher goroutine
	m.wg.Add(1)
	go m.watchLoop()
	
	return nil
}

// watchLoop watches for configuration file changes
func (m *HFTConfigManager) watchLoop() {
	defer m.wg.Done()
	
	for {
		select {
		case <-m.ctx.Done():
			return
		case event, ok := <-m.watcher.Events:
			if !ok {
				return
			}
			
			// Check if it's our config file
			if event.Name == m.configPath && (event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create) {
				// Debounce rapid changes
				select {
				case m.reloadChan <- struct{}{}:
				default:
				}
			}
		case err, ok := <-m.watcher.Errors:
			if !ok {
				return
			}
			fmt.Printf("Config watcher error: %v\n", err)
		case <-m.reloadChan:
			// Wait a bit to avoid rapid reloads
			time.Sleep(100 * time.Millisecond)
			
			// Reload configuration
			if err := m.loadConfig(); err != nil {
				fmt.Printf("Failed to reload config: %v\n", err)
			} else {
				fmt.Println("Configuration reloaded successfully")
			}
		}
	}
}

// notifyCallbacks notifies all registered callbacks
func (m *HFTConfigManager) notifyCallbacks(config *HFTManagerConfig) {
	m.cbLock.RLock()
	defer m.cbLock.RUnlock()
	
	for _, callback := range m.callbacks {
		go callback(config)
	}
}

// GetConfig returns the current configuration
func (m *HFTConfigManager) GetConfig() *HFTManagerConfig {
	return m.config.Load().(*HFTManagerConfig)
}

// RegisterCallback registers a callback for configuration changes
func (m *HFTConfigManager) RegisterCallback(callback func(*HFTManagerConfig)) {
	m.cbLock.Lock()
	defer m.cbLock.Unlock()
	
	m.callbacks = append(m.callbacks, callback)
}

// Close closes the configuration manager
func (m *HFTConfigManager) Close() error {
	m.cancel()
	m.wg.Wait()
	return m.watcher.Close()
}

// ValidateHFTConfig validates the HFT configuration
func ValidateHFTConfig(config *HFTManagerConfig) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}
	
	// Validate environment
	validEnvs := map[string]bool{
		"development": true,
		"staging":     true,
		"production":  true,
	}
	if !validEnvs[config.Environment] {
		return fmt.Errorf("invalid environment: %s", config.Environment)
	}
	
	// Validate WebSocket port
	if config.WebSocket.Port <= 0 || config.WebSocket.Port > 65535 {
		return fmt.Errorf("invalid websocket port: %d", config.WebSocket.Port)
	}
	
	// Validate buffer sizes
	if config.WebSocket.ReadBufferSize <= 0 {
		return fmt.Errorf("invalid read buffer size: %d", config.WebSocket.ReadBufferSize)
	}
	if config.WebSocket.WriteBufferSize <= 0 {
		return fmt.Errorf("invalid write buffer size: %d", config.WebSocket.WriteBufferSize)
	}
	
	// Validate rate limiting
	if config.RateLimit.RequestsPerSecond <= 0 {
		return fmt.Errorf("invalid requests per second: %d", config.RateLimit.RequestsPerSecond)
	}
	if config.RateLimit.BurstSize <= 0 {
		return fmt.Errorf("invalid burst size: %d", config.RateLimit.BurstSize)
	}
	
	// Validate circuit breaker
	if config.CircuitBreaker.MaxFailures <= 0 {
		return fmt.Errorf("invalid max failures: %d", config.CircuitBreaker.MaxFailures)
	}
	if config.CircuitBreaker.FailureRatio <= 0 || config.CircuitBreaker.FailureRatio > 1 {
		return fmt.Errorf("invalid failure ratio: %f", config.CircuitBreaker.FailureRatio)
	}
	
	// Validate GC configuration
	if err := ValidateGCConfig(&config.GC); err != nil {
		return fmt.Errorf("invalid GC config: %w", err)
	}
	
	return nil
}

// LoadConfigFromFile loads configuration from a YAML file
func LoadConfigFromFile(path string) (*HFTManagerConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	
	var config HFTManagerConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	
	return &config, nil
}

// SaveConfigToFile saves configuration to a YAML file
func SaveConfigToFile(config *HFTManagerConfig, path string) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	
	return nil
}
