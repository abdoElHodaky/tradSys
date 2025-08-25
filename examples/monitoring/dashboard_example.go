package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/architecture/coordination"
	"github.com/abdoElHodaky/tradSys/internal/monitoring"
	"github.com/abdoElHodaky/tradSys/internal/performance"
	performance_lazy "github.com/abdoElHodaky/tradSys/internal/performance/lazy"
	"github.com/abdoElHodaky/tradSys/internal/trading/market_data/historical"
	historical_lazy "github.com/abdoElHodaky/tradSys/internal/trading/market_data/historical/lazy"
	"github.com/abdoElHodaky/tradSys/internal/trading/order_management"
	order_lazy "github.com/abdoElHodaky/tradSys/internal/trading/order_management/lazy"
	"go.uber.org/zap"
)

func main() {
	// Create a logger
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// Create a component coordinator with default configuration
	coordinator := coordination.NewComponentCoordinator(
		coordination.DefaultCoordinatorConfig(),
		logger,
	)

	// Create a context
	ctx := context.Background()

	// Create multiple lazy-loaded services
	historicalService := createHistoricalService(coordinator, logger)
	orderService := createOrderService(coordinator, logger)
	connectionPool := createConnectionPool(coordinator, logger)

	// Create the dashboard
	dashboard := monitoring.NewLazyComponentDashboard(coordinator, logger)

	// Start the dashboard in a goroutine
	go func() {
		err := dashboard.Start(":8080")
		if err != nil {
			logger.Error("Failed to start dashboard", zap.Error(err))
		}
	}()

	fmt.Println("Dashboard started at http://localhost:8080")
	fmt.Println("Press Ctrl+C to exit")

	// Simulate some activity
	go simulateActivity(ctx, historicalService, orderService, connectionPool, logger)

	// Wait for interrupt signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	// Shutdown
	fmt.Println("Shutting down...")
	
	// Stop the dashboard
	dashboard.Stop(ctx)

	// Shutdown services
	historicalService.Shutdown(ctx)
	orderService.Shutdown(ctx)
	connectionPool.Shutdown(ctx)

	// Shutdown the coordinator
	coordinator.Shutdown(ctx)
}

func createHistoricalService(coordinator *coordination.ComponentCoordinator, logger *zap.Logger) *historical_lazy.LazyHistoricalDataService {
	service, err := historical_lazy.NewLazyHistoricalDataService(
		coordinator,
		historical.Config{
			MaxCacheSize: 100 * 1024 * 1024, // 100MB
			CacheTTL:     time.Hour,
		},
		logger,
	)
	if err != nil {
		logger.Fatal("Failed to create lazy historical data service", zap.Error(err))
	}
	return service
}

func createOrderService(coordinator *coordination.ComponentCoordinator, logger *zap.Logger) *order_lazy.LazyOrderService {
	service, err := order_lazy.NewLazyOrderService(
		coordinator,
		order_management.OrderServiceConfig{
			MaxOrders: 10000,
		},
		logger,
	)
	if err != nil {
		logger.Fatal("Failed to create lazy order service", zap.Error(err))
	}
	return service
}

func createConnectionPool(coordinator *coordination.ComponentCoordinator, logger *zap.Logger) *performance_lazy.LazyConnectionPool {
	pool, err := performance_lazy.NewLazyConnectionPool(
		coordinator,
		performance.PoolConfig{
			MaxPoolSize:    100,
			InitialPoolSize: 10,
			IdleTimeout:    time.Minute * 5,
		},
		logger,
	)
	if err != nil {
		logger.Fatal("Failed to create lazy connection pool", zap.Error(err))
	}
	return pool
}

func simulateActivity(
	ctx context.Context,
	historicalService *historical_lazy.LazyHistoricalDataService,
	orderService *order_lazy.LazyOrderService,
	connectionPool *performance_lazy.LazyConnectionPool,
	logger *zap.Logger,
) {
	// Create a ticker for periodic activity
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	// Create symbols
	symbols := []string{"BTC-USD", "ETH-USD", "SOL-USD", "ADA-USD", "DOT-USD"}

	for {
		select {
		case <-ticker.C:
			// Simulate historical data requests
			for _, symbol := range symbols {
				_, err := historicalService.GetHistoricalData(
					ctx,
					symbol,
					time.Now().Add(-24*time.Hour),
					time.Now(),
					"1h",
				)
				if err != nil {
					logger.Error("Failed to get historical data", zap.Error(err))
				}
			}

			// Simulate connection pool usage
			conn, err := connectionPool.GetConnection(ctx, "example.com")
			if err != nil {
				logger.Error("Failed to get connection", zap.Error(err))
			} else {
				// Use the connection
				time.Sleep(1 * time.Second)

				// Release the connection
				err = connectionPool.ReleaseConnection(ctx, conn)
				if err != nil {
					logger.Error("Failed to release connection", zap.Error(err))
				}
			}

			// Log memory usage
			memoryManager := historicalService.GetCoordinator().GetMemoryManager()
			logger.Info("Memory usage",
				zap.Int64("total_usage", memoryManager.GetMemoryUsage()),
				zap.Int64("limit", memoryManager.GetMemoryLimit()),
				zap.String("pressure", memoryManager.GetMemoryPressureLevel().String()),
			)
		case <-ctx.Done():
			return
		}
	}
}

