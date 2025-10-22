package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	"github.com/abdoElHodaky/tradSys/internal/api/handlers"
	"github.com/abdoElHodaky/tradSys/internal/config"
	"github.com/abdoElHodaky/tradSys/internal/core/matching"
	"github.com/abdoElHodaky/tradSys/internal/risk"
	"github.com/abdoElHodaky/tradSys/internal/core/settlement"
	"github.com/abdoElHodaky/tradSys/internal/connectivity"
	"github.com/abdoElHodaky/tradSys/internal/compliance"
	"github.com/abdoElHodaky/tradSys/internal/strategies"
	"github.com/abdoElHodaky/tradSys/internal/gateway"
	"github.com/abdoElHodaky/tradSys/internal/orders"
	"github.com/abdoElHodaky/tradSys/internal/marketdata"
	"github.com/abdoElHodaky/tradSys/internal/ws"
	orders_proto "github.com/abdoElHodaky/tradSys/proto/orders"
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


	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	// Load configuration (unified config merged into main config)
	cfg, err := config.LoadConfig("config/tradsys.yaml")

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
	
	// Load configuration
	cfg, err := config.LoadConfig("config/tradsys.yaml")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}
	
	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()
	
	// Create and start gateway server
	gatewayServer := gateway.NewServer(gateway.ServerParams{
		Logger: logger,
		Config: cfg,
	})
	
	// Start the gateway server
	if err := gatewayServer.Start(); err != nil {
		log.Fatalf("Failed to start gateway server: %v", err)
	}
}

func runOrderService() {
	log.Printf("Starting TradSys Order Service v%s", AppVersion)
	
	// Load configuration
	cfg, err := config.LoadConfig("config/tradsys.yaml")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}
	
	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()
	
	// Create order handler for gRPC
	orderHandler := orders.NewHandler(orders.HandlerParams{
		Logger: logger,
	})
	
	// Start gRPC server for order service
	grpcServer := grpc.NewServer()
	orders_proto.RegisterOrderServiceServer(grpcServer, orderHandler)
	
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Service.GRPCPort))
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	
	log.Printf("Order service listening on port %d", cfg.Service.GRPCPort)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

func runRiskService() {
	log.Printf("Starting TradSys Risk Service v%s", AppVersion)
	
	// Load configuration
	cfg, err := config.LoadConfig("config/tradsys.yaml")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}
	
	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()
	
	// Create risk service
	riskService := risk.NewService(risk.ServiceParams{
		Logger: logger,
	})
	
	// Start gRPC server for risk service
	grpcServer := grpc.NewServer()
	risk.RegisterRiskServiceServer(grpcServer, riskService)
	
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Service.GRPCPort+1))
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	
	log.Printf("Risk service listening on port %d", cfg.Service.GRPCPort+1)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

func runMarketDataService() {
	log.Printf("Starting TradSys Market Data Service v%s", AppVersion)
	
	// Load configuration
	cfg, err := config.LoadConfig("config/tradsys.yaml")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}
	
	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()
	
	// Create market data service
	mdService := marketdata.NewService(marketdata.ServiceParams{
		Logger: logger,
		Config: cfg,
	})
	
	// Start the market data service
	ctx := context.Background()
	if err := mdService.Start(ctx); err != nil {
		log.Fatalf("Failed to start market data service: %v", err)
	}
	
	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	
	log.Println("Shutting down market data service...")
	
	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	if err := mdService.Stop(shutdownCtx); err != nil {
		log.Printf("Error during market data service shutdown: %v", err)
	}
	
	log.Println("Market data service exited")
}

func runWebSocketService() {
	log.Printf("Starting TradSys WebSocket Service v%s", AppVersion)
	
	// Load configuration
	cfg, err := config.LoadConfig("config/tradsys.yaml")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}
	
	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()
	
	// Create WebSocket server
	wsServer := ws.NewServer(ws.ServerParams{
		Logger: logger,
		Config: cfg,
	})
	
	// Start the WebSocket server
	if err := wsServer.Start(); err != nil {
		log.Fatalf("Failed to start WebSocket server: %v", err)
	}
}

// initializeTradingSystem initializes all trading system components
func initializeTradingSystem(cfg *config.Config) (*TradingSystem, error) {
	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}

	// Initialize matching engine
	matchingEngine := matching.NewEngine(logger)

	// Initialize risk engine
	riskEngine := risk.NewEngine(logger)

	// Initialize settlement processor
	settlementProcessor, err := settlement.NewProcessor(logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize settlement processor: %w", err)
	}

	// Initialize connectivity manager
	connManager, err := connectivity.NewManager(cfg.Connectivity, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize connectivity: %w", err)
	}

	// Initialize compliance engine
	complianceEngine, err := compliance.NewEngine(cfg.Compliance, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize compliance: %w", err)
	}

	// Initialize strategies engine
	strategyEngine, err := strategies.NewEngine(cfg.Strategies, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize strategies: %w", err)
	}

	return &TradingSystem{
		MatchingEngine:      matchingEngine,
		RiskEngine:         riskEngine,
		SettlementProcessor: settlementProcessor,
		Connectivity:       connManager,
		Compliance:         complianceEngine,
		Strategies:         strategyEngine,
		Logger:             logger,
	}, nil
}

// TradingSystem represents the unified trading system
type TradingSystem struct {
	MatchingEngine      *order_matching.Engine
	RiskEngine         *risk.Service
	SettlementProcessor *settlement.Processor
	ConnectivityManager *connectivity.Manager
	ComplianceEngine   *compliance.Engine
	StrategiesEngine   *strategies.Engine
	Logger             *zap.Logger
}

// GetMatchingEngine returns the matching engine (implements TradingSystemInterface)
func (ts *TradingSystem) GetMatchingEngine() *order_matching.Engine {
	return ts.MatchingEngine
}

// GetPerformanceMetrics returns performance metrics (implements TradingSystemInterface)
func (ts *TradingSystem) GetPerformanceMetrics() map[string]interface{} {
	return map[string]interface{}{
		"uptime":           time.Since(time.Now()).String(),
		"orders_processed": 0, // TODO: implement actual metrics
		"trades_executed":  0,
		"risk_checks":      0,
		"latency_avg":      "0ms",
	}
}
