package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	
	"github.com/abdoElHodaky/tradSys/internal/hft/app"
	"github.com/abdoElHodaky/tradSys/internal/hft/config"
	"github.com/abdoElHodaky/tradSys/internal/hft/memory"
	"github.com/abdoElHodaky/tradSys/internal/hft/metrics"
	"github.com/abdoElHodaky/tradSys/internal/monitoring"
	"github.com/abdoElHodaky/tradSys/internal/api/handlers"
	"github.com/abdoElHodaky/tradSys/internal/ws/manager"
)

const (
	// Application metadata
	AppName    = "HFT Trading System"
	AppVersion = "2.0.0"
	AppAuthor  = "TradSys Team"
)

func main() {
	// Print startup banner
	printBanner()
	
	// Load configuration
	cfg, err := loadConfiguration()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}
	
	// Initialize application
	app, err := initializeApplication(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize application: %v", err)
	}
	
	// Start application
	if err := startApplication(app, cfg); err != nil {
		log.Fatalf("Failed to start application: %v", err)
	}
	
	// Wait for shutdown signal
	waitForShutdown(app)
	
	log.Println("HFT Trading System shutdown complete")
}

// printBanner prints the application startup banner
func printBanner() {
	banner := `
╔══════════════════════════════════════════════════════════════╗
║                    HFT TRADING SYSTEM v2.0                  ║
║                                                              ║
║  🚀 High-Frequency Trading Platform                         ║
║  ⚡ Microsecond-level latency optimization                  ║
║  📊 Enterprise-grade monitoring & analytics                 ║
║  🔒 Production-ready security & compliance                  ║
║                                                              ║
║  Performance Targets:                                        ║
║  • Order Processing: < 100μs (99th percentile)             ║
║  • WebSocket Latency: < 50μs (99th percentile)             ║
║  • Database Queries: < 1ms (95th percentile)               ║
║  • Throughput: > 100,000 orders/second                     ║
║                                                              ║
╚══════════════════════════════════════════════════════════════╝
`
	fmt.Print(banner)
	fmt.Printf("Starting %s v%s...\n\n", AppName, AppVersion)
}

// loadConfiguration loads and validates application configuration
func loadConfiguration() (*config.HFTConfig, error) {
	log.Println("Loading configuration...")
	
	// Determine configuration file path
	configPath := os.Getenv("HFT_CONFIG_PATH")
	if configPath == "" {
		configPath = "configs/hft-config.yaml"
	}
	
	// Load configuration
	configManager, err := config.NewHFTConfigManager(configPath, getEnvironment())
	if err != nil {
		return nil, fmt.Errorf("failed to create config manager: %w", err)
	}
	
	cfg := configManager.GetConfig()
	
	// Validate configuration
	if err := config.ValidateHFTConfig(cfg); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}
	
	log.Printf("Configuration loaded successfully (environment: %s)", cfg.Environment)
	return cfg, nil
}

// initializeApplication initializes all application components
func initializeApplication(cfg *config.HFTConfig) (*app.HFTApplication, error) {
	log.Println("Initializing application components...")
	
	// Initialize GC tuning first
	if err := config.OptimizeGCForHFT(&cfg.GC); err != nil {
		return nil, fmt.Errorf("failed to optimize GC: %w", err)
	}
	log.Println("✓ GC optimization configured")
	
	// Initialize metrics system
	metrics.InitMetrics()
	log.Println("✓ Metrics system initialized")
	
	// Initialize memory manager
	memory.InitMemoryManager(&cfg.Memory)
	log.Println("✓ Memory manager initialized")
	
	// Initialize monitoring
	monitoring.InitProductionMonitor(&cfg.Monitoring)
	log.Println("✓ Production monitoring initialized")
	
	// Create main application
	hftApp := app.NewHFTApplication(cfg)
	if err := hftApp.Initialize(); err != nil {
		return nil, fmt.Errorf("failed to initialize HFT application: %w", err)
	}
	log.Println("✓ HFT application initialized")
	
	log.Println("All components initialized successfully")
	return hftApp, nil
}

// startApplication starts all application services
func startApplication(hftApp *app.HFTApplication, cfg *config.HFTConfig) error {
	log.Println("Starting application services...")
	
	// Start HFT application
	if err := hftApp.Start(); err != nil {
		return fmt.Errorf("failed to start HFT application: %w", err)
	}
	log.Println("✓ HFT application started")
	
	// Setup and start HTTP server
	router := setupRouter(hftApp, cfg)
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.WebSocket.Port),
		Handler:      router,
		ReadTimeout:  cfg.Timeouts.HTTPRead,
		WriteTimeout: cfg.Timeouts.HTTPWrite,
		IdleTimeout:  30 * time.Second,
	}
	
	// Start server in goroutine
	go func() {
		log.Printf("✓ HTTP server starting on port %d", cfg.WebSocket.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()
	
	// Start WebSocket manager
	wsManager := hftApp.GetWebSocketManager()
	go func() {
		if err := wsManager.Start(); err != nil {
			log.Printf("WebSocket manager error: %v", err)
		}
	}()
	log.Println("✓ WebSocket manager started")
	
	log.Println("All services started successfully")
	return nil
}

// setupRouter configures the HTTP router with all endpoints
func setupRouter(hftApp *app.HFTApplication, cfg *config.HFTConfig) *gin.Engine {
	// Set Gin mode based on environment
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	
	// Create router with HFT optimizations
	router := config.NewHFTGinEngine(&cfg.Gin)
	
	// Add global middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	
	// Health check endpoints
	router.GET("/health", func(c *gin.Context) {
		status := hftApp.GetMonitor().GetHealthStatus()
		if status.Status == "healthy" {
			c.JSON(http.StatusOK, status)
		} else {
			c.JSON(http.StatusServiceUnavailable, status)
		}
	})
	
	router.GET("/ready", func(c *gin.Context) {
		if hftApp.IsRunning() {
			c.JSON(http.StatusOK, gin.H{"status": "ready"})
		} else {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not ready"})
		}
	})
	
	// Metrics endpoint
	router.GET("/metrics", func(c *gin.Context) {
		metrics := map[string]interface{}{
			"application": hftApp.GetMetrics().GetStats(),
			"memory":      hftApp.GetMemoryManager().GetMemoryStats(),
			"performance": hftApp.GetMonitor().GetPerformanceMetrics(),
		}
		c.JSON(http.StatusOK, metrics)
	})
	
	// API routes
	apiV1 := router.Group("/api/v1")
	{
		// Fast order endpoints (HFT optimized)
		orders := apiV1.Group("/orders")
		{
			orders.POST("/", handlers.FastCreateOrder)
			orders.GET("/:id", handlers.FastGetOrder)
			orders.PUT("/:id", handlers.FastUpdateOrder)
			orders.DELETE("/:id", handlers.FastCancelOrder)
			orders.GET("/", handlers.FastListOrders)
		}
		
		// WebSocket endpoint
		apiV1.GET("/ws", func(c *gin.Context) {
			hftApp.GetWebSocketManager().HandleWebSocket(c.Writer, c.Request)
		})
	}
	
	// Admin routes
	admin := router.Group("/admin")
	{
		admin.GET("/stats", func(c *gin.Context) {
			stats := map[string]interface{}{
				"uptime":     hftApp.GetUptime(),
				"version":    AppVersion,
				"components": map[string]bool{
					"metrics":    hftApp.GetMetrics() != nil,
					"monitoring": hftApp.GetMonitor() != nil,
					"websocket":  hftApp.GetWebSocketManager() != nil,
					"memory":     hftApp.GetMemoryManager() != nil,
				},
			}
			c.JSON(http.StatusOK, stats)
		})
		
		admin.POST("/gc", func(c *gin.Context) {
			hftApp.GetMemoryManager().ForceGC()
			c.JSON(http.StatusOK, gin.H{"message": "GC triggered"})
		})
	}
	
	return router
}

// waitForShutdown waits for shutdown signal and gracefully shuts down the application
func waitForShutdown(hftApp *app.HFTApplication) {
	// Create channel to receive OS signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	
	// Wait for signal
	sig := <-sigChan
	log.Printf("Received signal: %v, initiating graceful shutdown...", sig)
	
	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	// Shutdown application components
	shutdownComponents(ctx, hftApp)
}

// shutdownComponents gracefully shuts down all application components
func shutdownComponents(ctx context.Context, hftApp *app.HFTApplication) {
	log.Println("Shutting down application components...")
	
	// Stop HFT application
	if err := hftApp.Stop(); err != nil {
		log.Printf("Error stopping HFT application: %v", err)
	} else {
		log.Println("✓ HFT application stopped")
	}
	
	// Stop monitoring
	if monitoring.GlobalProductionMonitor != nil {
		monitoring.GlobalProductionMonitor.Close()
		log.Println("✓ Monitoring stopped")
	}
	
	// Stop memory manager
	if memory.GlobalMemoryManager != nil {
		memory.GlobalMemoryManager.Close()
		log.Println("✓ Memory manager stopped")
	}
	
	log.Println("Graceful shutdown completed")
}

// getEnvironment returns the current environment
func getEnvironment() string {
	env := os.Getenv("HFT_ENVIRONMENT")
	if env == "" {
		env = "development"
	}
	return env
}
