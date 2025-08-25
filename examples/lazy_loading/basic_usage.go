package main

import (
	"context"
	"fmt"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/architecture/coordination"
	"github.com/abdoElHodaky/tradSys/internal/trading/market_data/historical"
	historical_lazy "github.com/abdoElHodaky/tradSys/internal/trading/market_data/historical/lazy"
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

	// Create a lazy historical data service
	lazyService, err := historical_lazy.NewLazyHistoricalDataService(
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

	// Create a context
	ctx := context.Background()

	// Use the service - it will be initialized on first use
	fmt.Println("Getting historical data...")
	data, err := lazyService.GetHistoricalData(
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

	// Get cache stats
	stats, err := lazyService.GetCacheStats(ctx)
	if err != nil {
		logger.Error("Failed to get cache stats", zap.Error(err))
	} else {
		fmt.Printf("Cache stats: %+v\n", stats)
	}

	// Shutdown the service when done
	err = lazyService.Shutdown(ctx)
	if err != nil {
		logger.Error("Failed to shutdown service", zap.Error(err))
	}

	// Shutdown the coordinator
	err = coordinator.Shutdown(ctx)
	if err != nil {
		logger.Error("Failed to shutdown coordinator", zap.Error(err))
	}
}

