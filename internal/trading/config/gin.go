package config

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	
	"github.com/abdoElHodaky/tradSys/internal/api/handlers"
	"github.com/abdoElHodaky/tradSys/internal/trading/middleware"
)

// HFTGinConfig contains HFT-specific Gin configuration
type HFTGinConfig struct {
	// Performance settings
	DisableConsoleColor     bool `yaml:"disable_console_color" default:"true"`
	DisableRouteLogging     bool `yaml:"disable_route_logging" default:"true"`
	HandleMethodNotAllowed  bool `yaml:"handle_method_not_allowed" default:"false"`
	RedirectTrailingSlash   bool `yaml:"redirect_trailing_slash" default:"false"`
	RedirectFixedPath       bool `yaml:"redirect_fixed_path" default:"false"`
	
	// Memory settings
	MaxMultipartMemory      int64 `yaml:"max_multipart_memory" default:"1048576"` // 1MB
	
	// Middleware settings
	EnableRecovery          bool `yaml:"enable_recovery" default:"true"`
	EnableLogger            bool `yaml:"enable_logger" default:"false"`
	EnableCORS              bool `yaml:"enable_cors" default:"true"`
	EnableAuth              bool `yaml:"enable_auth" default:"true"`
	
	// Timeout settings
	ReadTimeout             time.Duration `yaml:"read_timeout" default:"10s"`
	WriteTimeout            time.Duration `yaml:"write_timeout" default:"10s"`
	IdleTimeout             time.Duration `yaml:"idle_timeout" default:"60s"`
	
	// Trust settings
	TrustedProxies          []string `yaml:"trusted_proxies"`
	ForwardedByClientIP     bool     `yaml:"forwarded_by_client_ip" default:"false"`
}

// NewHFTGinEngine creates a Gin engine optimized for HFT workloads
func NewHFTGinEngine(config *HFTGinConfig) *gin.Engine {
	if config == nil {
		config = &HFTGinConfig{
			DisableConsoleColor:    true,
			DisableRouteLogging:    true,
			HandleMethodNotAllowed: false,
			RedirectTrailingSlash:  false,
			RedirectFixedPath:      false,
			MaxMultipartMemory:     1048576, // 1MB
			EnableRecovery:         true,
			EnableLogger:           false,
			EnableCORS:             true,
			EnableAuth:             true,
			ReadTimeout:            10 * time.Second,
			WriteTimeout:           10 * time.Second,
			IdleTimeout:            60 * time.Second,
			ForwardedByClientIP:    false,
		}
	}
	
	// Set Gin to release mode for maximum performance
	gin.SetMode(gin.ReleaseMode)
	
	// Disable console color for performance
	if config.DisableConsoleColor {
		gin.DisableConsoleColor()
	}
	
	// Create new engine with no default middleware
	engine := gin.New()
	
	// Configure engine settings for HFT performance
	engine.HandleMethodNotAllowed = config.HandleMethodNotAllowed
	engine.RedirectTrailingSlash = config.RedirectTrailingSlash
	engine.RedirectFixedPath = config.RedirectFixedPath
	engine.MaxMultipartMemory = config.MaxMultipartMemory
	engine.ForwardedByClientIP = config.ForwardedByClientIP
	
	// Set trusted proxies if configured
	if len(config.TrustedProxies) > 0 {
		engine.SetTrustedProxies(config.TrustedProxies)
	} else {
		// Disable trusted proxy feature for performance
		engine.SetTrustedProxies(nil)
	}
	
	// Add HFT-optimized middleware stack
	addHFTMiddleware(engine, config)
	
	return engine
}

// addHFTMiddleware adds HFT-optimized middleware to the engine
func addHFTMiddleware(engine *gin.Engine, config *HFTGinConfig) {
	// Recovery middleware (always first for safety)
	if config.EnableRecovery {
		engine.Use(middleware.HFTRecoveryMiddleware())
	}
	
	// Logger middleware (minimal logging for HFT)
	if config.EnableLogger {
		engine.Use(middleware.HFTLoggerMiddleware())
	}
	
	// CORS middleware (optimized for HFT)
	if config.EnableCORS {
		engine.Use(middleware.HFTCORSMiddleware())
	}
	
	// Authentication middleware (fast JWT validation)
	if config.EnableAuth {
		engine.Use(middleware.HFTAuthMiddleware())
	}
}

// HFTServerConfig contains HTTP server configuration for HFT
type HFTServerConfig struct {
	Address         string        `yaml:"address" default:":8080"`
	ReadTimeout     time.Duration `yaml:"read_timeout" default:"10s"`
	WriteTimeout    time.Duration `yaml:"write_timeout" default:"10s"`
	IdleTimeout     time.Duration `yaml:"idle_timeout" default:"60s"`
	MaxHeaderBytes  int           `yaml:"max_header_bytes" default:"1048576"` // 1MB
	
	// Keep-alive settings
	SetKeepAlivesEnabled bool          `yaml:"set_keep_alives_enabled" default:"true"`
	ReadHeaderTimeout    time.Duration `yaml:"read_header_timeout" default:"5s"`
	
	// TLS settings (if needed)
	TLSEnabled  bool   `yaml:"tls_enabled" default:"false"`
	CertFile    string `yaml:"cert_file"`
	KeyFile     string `yaml:"key_file"`
}

// NewHFTServer creates an HTTP server optimized for HFT workloads
func NewHFTServer(engine *gin.Engine, config *HFTServerConfig) *http.Server {
	if config == nil {
		config = &HFTServerConfig{
			Address:              ":8080",
			ReadTimeout:          10 * time.Second,
			WriteTimeout:         10 * time.Second,
			IdleTimeout:          60 * time.Second,
			MaxHeaderBytes:       1048576, // 1MB
			SetKeepAlivesEnabled: true,
			ReadHeaderTimeout:    5 * time.Second,
			TLSEnabled:           false,
		}
	}
	
	server := &http.Server{
		Addr:              config.Address,
		Handler:           engine,
		ReadTimeout:       config.ReadTimeout,
		WriteTimeout:      config.WriteTimeout,
		IdleTimeout:       config.IdleTimeout,
		MaxHeaderBytes:    config.MaxHeaderBytes,
		ReadHeaderTimeout: config.ReadHeaderTimeout,
	}
	
	// Configure keep-alives
	server.SetKeepAlivesEnabled(config.SetKeepAlivesEnabled)
	
	return server
}

// HFTRouteConfig contains route configuration for HFT endpoints
type HFTRouteConfig struct {
	EnableProfiling    bool `yaml:"enable_profiling" default:"false"`
	EnableMetrics      bool `yaml:"enable_metrics" default:"true"`
	EnableHealthCheck  bool `yaml:"enable_health_check" default:"true"`
	EnableSwagger      bool `yaml:"enable_swagger" default:"false"`
	
	// API versioning
	APIVersion         string `yaml:"api_version" default:"v1"`
	EnableVersioning   bool   `yaml:"enable_versioning" default:"true"`
}

// SetupHFTRoutes sets up HFT-optimized routes
func SetupHFTRoutes(engine *gin.Engine, config *HFTRouteConfig) {
	if config == nil {
		config = &HFTRouteConfig{
			EnableProfiling:   false,
			EnableMetrics:     true,
			EnableHealthCheck: true,
			EnableSwagger:     false,
			APIVersion:        "v1",
			EnableVersioning:  true,
		}
	}
	
	// Health check endpoint (no auth required)
	if config.EnableHealthCheck {
		engine.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"status":    "ok",
				"timestamp": time.Now().Unix(),
			})
		})
		
		// Readiness check
		engine.GET("/ready", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"status": "ready",
				"timestamp": time.Now().Unix(),
			})
		})
	}
	
	// Metrics endpoint
	if config.EnableMetrics {
		engine.GET("/metrics", gin.WrapH(promhttp.Handler()))
	}
	
	// Profiling endpoints (only in development)
	if config.EnableProfiling {
		pprof.Register(engine)
	}
	
	// API routes with versioning
	if config.EnableVersioning {
		setupVersionedRoutes(engine, config.APIVersion)
	} else {
		setupDirectRoutes(engine)
	}
}

// setupVersionedRoutes sets up versioned API routes
func setupVersionedRoutes(engine *gin.Engine, version string) {
	api := engine.Group(fmt.Sprintf("/api/%s", version))
	
	// Fast order endpoints
	orders := api.Group("/orders")
	{
		fastHandler := handlers.NewFastOrderHandler()
		orders.POST("", fastHandler.FastCreateOrder)
		orders.GET("/:id", fastHandler.FastGetOrder)
		orders.PUT("/:id", fastHandler.FastUpdateOrder)
		orders.DELETE("/:id", fastHandler.FastCancelOrder)
		orders.GET("", fastHandler.FastListOrders)
	}
	
	// Market data endpoints
	market := api.Group("/market")
	{
		// These would be implemented with fast handlers
		market.GET("/price/:symbol", func(c *gin.Context) {
			// Fast price lookup
			c.JSON(200, gin.H{"message": "price endpoint"})
		})
		
		market.GET("/depth/:symbol", func(c *gin.Context) {
			// Fast order book depth
			c.JSON(200, gin.H{"message": "depth endpoint"})
		})
	}
	
	// WebSocket endpoint
	api.GET("/ws", func(c *gin.Context) {
		// WebSocket upgrade would be handled here
		c.JSON(200, gin.H{"message": "websocket endpoint"})
	})
}

// setupDirectRoutes sets up direct API routes (no versioning)
func setupDirectRoutes(engine *gin.Engine) {
	// Fast order endpoints
	orders := engine.Group("/orders")
	{
		fastHandler := handlers.NewFastOrderHandler()
		orders.POST("", fastHandler.FastCreateOrder)
		orders.GET("/:id", fastHandler.FastGetOrder)
		orders.PUT("/:id", fastHandler.FastUpdateOrder)
		orders.DELETE("/:id", fastHandler.FastCancelOrder)
		orders.GET("", fastHandler.FastListOrders)
	}
	
	// Market data endpoints
	market := engine.Group("/market")
	{
		market.GET("/price/:symbol", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "price endpoint"})
		})
		
		market.GET("/depth/:symbol", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "depth endpoint"})
		})
	}
	
	// WebSocket endpoint
	engine.GET("/ws", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "websocket endpoint"})
	})
}

// HFTGinOptions contains all HFT Gin configuration options
type HFTGinOptions struct {
	GinConfig    *HFTGinConfig
	ServerConfig *HFTServerConfig
	RouteConfig  *HFTRouteConfig
}

// NewHFTGinWithOptions creates a complete HFT-optimized Gin setup
func NewHFTGinWithOptions(options *HFTGinOptions) (*gin.Engine, *http.Server) {
	if options == nil {
		options = &HFTGinOptions{}
	}
	
	// Create HFT-optimized Gin engine
	engine := NewHFTGinEngine(options.GinConfig)
	
	// Setup HFT routes
	SetupHFTRoutes(engine, options.RouteConfig)
	
	// Create HFT-optimized server
	server := NewHFTServer(engine, options.ServerConfig)
	
	return engine, server
}

// ValidateHFTGinConfig validates the HFT Gin configuration
func ValidateHFTGinConfig(config *HFTGinConfig) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}
	
	if config.MaxMultipartMemory <= 0 {
		return fmt.Errorf("max_multipart_memory must be positive")
	}
	
	if config.ReadTimeout <= 0 {
		return fmt.Errorf("read_timeout must be positive")
	}
	
	if config.WriteTimeout <= 0 {
		return fmt.Errorf("write_timeout must be positive")
	}
	
	if config.IdleTimeout <= 0 {
		return fmt.Errorf("idle_timeout must be positive")
	}
	
	return nil
}

// GetHFTGinStats returns performance statistics for the Gin engine
func GetHFTGinStats(engine *gin.Engine) map[string]interface{} {
	stats := make(map[string]interface{})
	
	// Get route information
	routes := engine.Routes()
	stats["total_routes"] = len(routes)
	
	// Count routes by method
	methodCounts := make(map[string]int)
	for _, route := range routes {
		methodCounts[route.Method]++
	}
	stats["routes_by_method"] = methodCounts
	
	// Get middleware count (approximate)
	stats["middleware_count"] = len(engine.Handlers)
	
	// Get engine configuration
	stats["handle_method_not_allowed"] = engine.HandleMethodNotAllowed
	stats["redirect_trailing_slash"] = engine.RedirectTrailingSlash
	stats["redirect_fixed_path"] = engine.RedirectFixedPath
	stats["max_multipart_memory"] = engine.MaxMultipartMemory
	
	return stats
}
