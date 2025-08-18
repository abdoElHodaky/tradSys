package main

import (
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/abdoElHodaky/tradSys/internal/api/handlers"
	"github.com/abdoElHodaky/tradSys/internal/db"
	"github.com/abdoElHodaky/tradSys/internal/db/repositories"
	"github.com/abdoElHodaky/tradSys/internal/marketdata"
	"github.com/abdoElHodaky/tradSys/internal/orders"
	"github.com/abdoElHodaky/tradSys/internal/peerjs"
	"github.com/abdoElHodaky/tradSys/internal/risk"
	"github.com/abdoElHodaky/tradSys/internal/ws"
	marketdatapb "github.com/abdoElHodaky/tradSys/proto/marketdata"
	orderspb "github.com/abdoElHodaky/tradSys/proto/orders"
	riskpb "github.com/abdoElHodaky/tradSys/proto/risk"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func main() {
	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	// Initialize database
	dbConfig := db.DefaultConfig()
	database, err := db.Connect(dbConfig, logger)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}

	// Initialize database with optimizations
	if err := db.InitializeDatabase(database, logger); err != nil {
		logger.Fatal("Failed to initialize database", zap.Error(err))
	}

	// Initialize repositories
	orderRepo := repositories.NewOrderRepository(database, logger)
	marketDataRepo := repositories.NewMarketDataRepository(database, logger)
	riskRepo := repositories.NewRiskRepository(database, logger)
	_ = repositories.NewStrategyRepository(database, logger) // Will be used in future

	// Initialize services
	marketDataService := marketdata.NewService(logger, marketDataRepo)
	orderService := orders.NewService(logger, orderRepo)
	riskService := risk.NewService(logger, riskRepo)

	// Initialize WebSocket servers
	// Legacy WebSocket server
	wsServer := ws.NewServer(logger)
	go wsServer.Run()
	
	// Enhanced WebSocket server with binary message support
	enhancedWsOptions := ws.DefaultEnhancedServerOptions()
	enhancedWsServer := ws.NewEnhancedServer(logger, enhancedWsOptions)
	
	// Initialize PeerJS server
	peerServer := peerjs.NewPeerServer(logger)
	peerServer.StartCleanupTask(5*time.Minute, 10*time.Minute)

	// Start gRPC server
	go startGRPCServer(logger, marketDataService, orderService, riskService)

	// Start REST API server
	go startRESTServer(logger, wsServer, enhancedWsServer, peerServer)

	// Wait for termination signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigCh
	logger.Info("Received signal, shutting down", zap.String("signal", sig.String()))
}

func startGRPCServer(logger *zap.Logger, marketDataService marketdatapb.MarketDataServiceServer, orderService orderspb.OrderServiceServer, riskService riskpb.RiskServiceServer) {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		logger.Fatal("Failed to listen", zap.Error(err))
	}

	grpcServer := grpc.NewServer()
	marketdatapb.RegisterMarketDataServiceServer(grpcServer, marketDataService)
	orderspb.RegisterOrderServiceServer(grpcServer, orderService)
	riskpb.RegisterRiskServiceServer(grpcServer, riskService)

	logger.Info("Starting gRPC server on :50051")
	if err := grpcServer.Serve(lis); err != nil {
		logger.Fatal("Failed to serve gRPC", zap.Error(err))
	}
}

func startRESTServer(logger *zap.Logger, wsServer *ws.Server, enhancedWsServer *ws.EnhancedServer, peerServer *peerjs.PeerServer) {
	router := gin.Default()

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Metrics endpoint
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Legacy WebSocket endpoint
	router.GET("/ws", func(c *gin.Context) {
		wsServer.ServeWs(c.Writer, c.Request)
	})
	
	// Enhanced WebSocket endpoint with binary message support
	router.GET("/ws/v2", func(c *gin.Context) {
		enhancedWsServer.ServeWs(c.Writer, c.Request)
	})
	
	// PeerJS WebSocket endpoint
	router.GET("/peerjs/ws", func(c *gin.Context) {
		peerServer.HandleConnection(c.Writer, c.Request)
	})
	
	// PeerJS stats endpoint
	router.GET("/peerjs/stats", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"peer_count": peerServer.GetPeerCount(),
		})
	})
	
	// Register PeerJS handler
	peerJSHandler := handlers.NewPeerJSHandler(logger)
	peerJSHandler.RegisterRoutes(router)

	logger.Info("Starting REST server on :8080")
	if err := router.Run(":8080"); err != nil {
		logger.Fatal("Failed to start REST server", zap.Error(err))
	}
}
