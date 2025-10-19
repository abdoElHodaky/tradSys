package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/abdoElHodaky/tradSys/internal/api/handlers"
	unifiedconfig "github.com/abdoElHodaky/tradSys/internal/unified-config"
	// "github.com/abdoElHodaky/tradSys/internal/core/matching"
	// "github.com/abdoElHodaky/tradSys/internal/core/risk"
	// "github.com/abdoElHodaky/tradSys/internal/core/settlement"
	// "github.com/abdoElHodaky/tradSys/internal/connectivity"
	// "github.com/abdoElHodaky/tradSys/internal/compliance"
	// "github.com/abdoElHodaky/tradSys/internal/strategies"
	// "github.com/abdoElHodaky/tradSys/internal/common"
)

const (
	// Application metadata
	AppName    = "TradSys - High-Frequency Trading System"
	AppVersion = "2.0.0"
	AppAuthor  = "TradSys Team"
)

func main() {
	// Parse command line arguments for subcommands
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	// Set up flag parsing for the subcommand
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	switch command {
	case "server":
		runServer()
	case "gateway":
		runGateway()
	case "orders":
		runOrderService()
	case "risk":
		runRiskService()
	case "marketdata":
		runMarketDataService()
	case "ws":
		runWebSocketService()
	case "version":
		printVersion()
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Printf("Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Printf("%s v%s\n", AppName, AppVersion)
	fmt.Printf("Usage: %s <command> [options]\n\n", os.Args[0])
	fmt.Println("Commands:")
	fmt.Println("  server     - Run unified trading server (default)")
	fmt.Println("  gateway    - Run API gateway service")
	fmt.Println("  orders     - Run order management service")
	fmt.Println("  risk       - Run risk management service")
	fmt.Println("  marketdata - Run market data service")
	fmt.Println("  ws         - Run WebSocket service")
	fmt.Println("  version    - Show version information")
	fmt.Println("  help       - Show this help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  tradsys server                    # Run full trading server")
	fmt.Println("  tradsys server --port 8080        # Run server on specific port")
	fmt.Println("  tradsys gateway --config custom.yaml  # Run gateway with custom config")
}

func printVersion() {
	fmt.Printf("%s v%s\n", AppName, AppVersion)
	fmt.Printf("Author: %s\n", AppAuthor)
}

func runServer() {
	log.Printf("Starting %s v%s", AppName, AppVersion)

	// Load unified configuration
	cfg, err := unifiedconfig.Load()
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

// Individual service runners
func runGateway() {
	log.Printf("Starting TradSys Gateway Service v%s", AppVersion)
	// TODO: Implement gateway service startup
	log.Println("Gateway service functionality will be implemented")
}

func runOrderService() {
	log.Printf("Starting TradSys Order Service v%s", AppVersion)
	// TODO: Implement order service startup
	log.Println("Order service functionality will be implemented")
}

func runRiskService() {
	log.Printf("Starting TradSys Risk Service v%s", AppVersion)
	// TODO: Implement risk service startup
	log.Println("Risk service functionality will be implemented")
}

func runMarketDataService() {
	log.Printf("Starting TradSys Market Data Service v%s", AppVersion)
	// TODO: Implement market data service startup
	log.Println("Market data service functionality will be implemented")
}

func runWebSocketService() {
	log.Printf("Starting TradSys WebSocket Service v%s", AppVersion)
	// TODO: Implement WebSocket service startup
	log.Println("WebSocket service functionality will be implemented")
}

// initializeTradingSystem initializes all trading system components
func initializeTradingSystem(cfg *unifiedconfig.Config) (*TradingSystem, error) {
	// Initialize matching engine
	// matchingEngine := matching.NewEngine(nil)

	// Initialize risk engine (placeholder - using interface{} for now)
	// var riskEngine interface{} = "risk-engine-placeholder"

	// Initialize settlement processor
	// settlementProcessor, err := settlement.NewProcessor()
	// if err != nil {
	//	return nil, fmt.Errorf("failed to initialize settlement processor: %w", err)
	// }

	// Initialize connectivity (placeholder)
	// connManager, err := connectivity.NewManager(cfg.Connectivity)
	// if err != nil {
	//     return nil, fmt.Errorf("failed to initialize connectivity: %w", err)
	// }

	// Initialize compliance (placeholder)
	// complianceEngine, err := compliance.NewEngine(cfg.Compliance)
	// if err != nil {
	//     return nil, fmt.Errorf("failed to initialize compliance: %w", err)
	// }

	// Initialize strategies (placeholder)
	// strategyEngine, err := strategies.NewEngine(cfg.Strategies)
	// if err != nil {
	//     return nil, fmt.Errorf("failed to initialize strategies: %w", err)
	// }

	return &TradingSystem{
		// MatchingEngine:      matchingEngine,
		// RiskEngine:         riskEngine,
		// SettlementProcessor: settlementProcessor,
		// Connectivity: connManager,
		// Compliance:  complianceEngine,
		// Strategies:  strategyEngine,
	}, nil
}

// TradingSystem represents the unified trading system
type TradingSystem struct {
	// MatchingEngine      interface{}
	// RiskEngine         interface{}
	// SettlementProcessor interface{}
	// Connectivity *connectivity.Manager
	// Compliance  *compliance.Engine
	// Strategies  *strategies.Engine
}
