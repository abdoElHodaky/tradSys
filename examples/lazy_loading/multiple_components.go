package main

import (
	"context"
	"fmt"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/architecture/coordination"
	"github.com/abdoElHodaky/tradSys/internal/performance"
	performance_lazy "github.com/abdoElHodaky/tradSys/internal/performance/lazy"
	"github.com/abdoElHodaky/tradSys/internal/trading/market_data/historical"
	historical_lazy "github.com/abdoElHodaky/tradSys/internal/trading/market_data/historical/lazy"
	"github.com/abdoElHodaky/tradSys/internal/trading/order_management"
	order_lazy "github.com/abdoElHodaky/tradSys/internal/trading/order_management/lazy"
	"github.com/abdoElHodaky/tradSys/proto/orders"
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

	// Use the historical data service
	fmt.Println("Getting historical data...")
	data, err := historicalService.GetHistoricalData(
		ctx,
		"BTC-USD",
		time.Now().Add(-24*time.Hour),
		time.Now(),
		"1h",
	)
	if err != nil {
		logger.Error("Failed to get historical data", zap.Error(err))
	} else {
		fmt.Printf("Got %d data points\n", len(data))
	}

	// Use the order service
	fmt.Println("Creating an order...")
	order := &orders.Order{
		Symbol:   "BTC-USD",
		Side:     "buy",
		Type:     "limit",
		Quantity: "1.0",
		Price:    "50000.0",
	}
	orderResponse, err := orderService.CreateOrder(ctx, order)
	if err != nil {
		logger.Error("Failed to create order", zap.Error(err))
	} else {
		fmt.Printf("Created order: %s\n", orderResponse.OrderId)
	}

	// Use the connection pool
	fmt.Println("Getting a connection...")
	conn, err := connectionPool.GetConnection(ctx, "example.com")
	if err != nil {
		logger.Error("Failed to get connection", zap.Error(err))
	} else {
		fmt.Println("Got connection")
		// Use the connection...

		// Release the connection when done
		err = connectionPool.ReleaseConnection(ctx, conn)
		if err != nil {
			logger.Error("Failed to release connection", zap.Error(err))
		}
	}

	// Get memory usage information
	memoryManager := coordinator.GetMemoryManager()
	fmt.Printf("Total memory usage: %d bytes\n", memoryManager.GetMemoryUsage())
	fmt.Printf("Memory limit: %d bytes\n", memoryManager.GetMemoryLimit())
	fmt.Printf("Memory pressure level: %v\n", memoryManager.GetMemoryPressureLevel())

	// Shutdown all services
	fmt.Println("Shutting down services...")
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

