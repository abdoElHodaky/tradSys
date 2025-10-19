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
	
	"github.com/abdoElHodaky/tradSys/internal/trading/core"
	"github.com/abdoElHodaky/tradSys/internal/trading/connectivity"
	"github.com/abdoElHodaky/tradSys/internal/trading/compliance"
	"github.com/abdoElHodaky/tradSys/internal/trading/strategies"
	"github.com/abdoElHodaky/tradSys/internal/api/handlers"
	"github.com/abdoElHodaky/tradSys/internal/config"
)

const (
	// Application metadata
	AppName    = "TradSys - High-Frequency Trading System"
	AppVersion = "2.0.0"
	AppAuthor  = "TradSys Team"
)

func main() {
	log.Printf("Starting %s v%s", AppName, AppVersion)

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize unified trading system
	tradingSystem, err := initializeTradingSystem(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize trading system: %v", err)
	}

	// Setup HTTP server with Gin
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	// Setup API routes
	api := router.Group("/api/v1")
	handlers.SetupRoutes(api, tradingSystem)

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": AppName,
			"version": AppVersion,
			"time":    time.Now().UTC(),
		})
	})

	// Ready check endpoint
	router.GET("/ready", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ready",
			"components": gin.H{
				"core":         "ready",
				"connectivity": "ready",
				"compliance":   "ready",
				"strategies":   "ready",
			},
		})
	})

	// Metrics endpoint
	router.GET("/metrics", func(c *gin.Context) {
		// Prometheus metrics would be served here
		c.String(http.StatusOK, "# TradSys Metrics\n")
	})

	// Create HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Server starting on port %d", cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

// initializeTradingSystem initializes all trading system components
func initializeTradingSystem(cfg *config.Config) (*TradingSystem, error) {
	// Initialize core trading engine
	coreEngine, err := core.NewEngine(cfg.Core)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize core engine: %w", err)
	}

	// Initialize connectivity
	connManager, err := connectivity.NewManager(cfg.Connectivity)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize connectivity: %w", err)
	}

	// Initialize compliance engine
	complianceEngine, err := compliance.NewEngine(cfg.Compliance)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize compliance: %w", err)
	}

	// Initialize strategy engine
	strategyEngine, err := strategies.NewEngine(cfg.Strategies)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize strategies: %w", err)
	}

	return &TradingSystem{
		Core:        coreEngine,
		Connectivity: connManager,
		Compliance:  complianceEngine,
		Strategies:  strategyEngine,
	}, nil
}

// TradingSystem represents the unified trading system
type TradingSystem struct {
	Core        *core.Engine
	Connectivity *connectivity.Manager
	Compliance  *compliance.Engine
	Strategies  *strategies.Engine
}
