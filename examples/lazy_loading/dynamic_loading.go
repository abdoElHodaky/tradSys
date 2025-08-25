package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/architecture/coordination"
	"github.com/abdoElHodaky/tradSys/internal/exchange/connectors"
	exchange_lazy "github.com/abdoElHodaky/tradSys/internal/exchange/connectors/lazy"
	"github.com/abdoElHodaky/tradSys/proto/marketdata"
	"go.uber.org/zap"
)

// MockConnectorFactory is a mock implementation of the ConnectorFactory
type MockConnectorFactory struct{}

// CreateConnector creates a mock connector
func (f *MockConnectorFactory) CreateConnector(
	exchangeName string,
	config connectors.ConnectorConfig,
	logger *zap.Logger,
) (connectors.ExchangeConnector, error) {
	return &MockConnector{
		name:   exchangeName,
		logger: logger,
	}, nil
}

// MockConnector is a mock implementation of the ExchangeConnector
type MockConnector struct {
	name   string
	logger *zap.Logger
	mu     sync.Mutex
}

// Initialize initializes the connector
func (c *MockConnector) Initialize(ctx context.Context) error {
	c.logger.Info("Initializing connector", zap.String("exchange", c.name))
	return nil
}

// GetMarketData gets market data
func (c *MockConnector) GetMarketData(ctx context.Context, symbol string) (*marketdata.MarketDataResponse, error) {
	c.logger.Info("Getting market data",
		zap.String("exchange", c.name),
		zap.String("symbol", symbol),
	)
	return &marketdata.MarketDataResponse{
		Symbol: symbol,
		Price:  "50000.0",
		Volume: "100.0",
		Time:   time.Now().Unix(),
	}, nil
}

// SubscribeMarketData subscribes to market data
func (c *MockConnector) SubscribeMarketData(
	ctx context.Context,
	symbol string,
	callback func(*marketdata.MarketDataResponse),
) error {
	c.logger.Info("Subscribing to market data",
		zap.String("exchange", c.name),
		zap.String("symbol", symbol),
	)
	return nil
}

// UnsubscribeMarketData unsubscribes from market data
func (c *MockConnector) UnsubscribeMarketData(ctx context.Context, symbol string) error {
	c.logger.Info("Unsubscribing from market data",
		zap.String("exchange", c.name),
		zap.String("symbol", symbol),
	)
	return nil
}

// Shutdown shuts down the connector
func (c *MockConnector) Shutdown(ctx context.Context) error {
	c.logger.Info("Shutting down connector", zap.String("exchange", c.name))
	return nil
}

func main() {
	// Create a logger
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// Create a component coordinator with default configuration
	coordinator := coordination.NewComponentCoordinator(
		coordination.DefaultCoordinatorConfig(),
		logger,
	)

	// Create a lock manager
	lockManager := coordination.NewLockManager(
		coordination.DefaultLockManagerConfig(),
		logger,
	)

	// Create a context
	ctx := context.Background()

	// Create a mock connector factory
	factory := &MockConnectorFactory{}

	// Create a lazy connector manager
	connectorManager, err := exchange_lazy.NewLazyConnectorManager(
		coordinator,
		lockManager,
		factory,
		connectors.ConnectorConfig{},
		logger,
	)
	if err != nil {
		logger.Fatal("Failed to create lazy connector manager", zap.Error(err))
	}

	// Dynamically load and use connectors
	exchanges := []string{"binance", "coinbase", "kraken"}
	symbols := []string{"BTC-USD", "ETH-USD", "SOL-USD"}

	for _, exchange := range exchanges {
		fmt.Printf("Loading connector for %s...\n", exchange)
		connector, err := connectorManager.GetConnector(ctx, exchange)
		if err != nil {
			logger.Error("Failed to get connector", zap.String("exchange", exchange), zap.Error(err))
			continue
		}

		for _, symbol := range symbols {
			fmt.Printf("Getting market data for %s on %s...\n", symbol, exchange)
			data, err := connector.GetMarketData(ctx, symbol)
			if err != nil {
				logger.Error("Failed to get market data",
					zap.String("exchange", exchange),
					zap.String("symbol", symbol),
					zap.Error(err),
				)
				continue
			}
			fmt.Printf("%s on %s: Price=%s, Volume=%s\n", symbol, exchange, data.Price, data.Volume)
		}
	}

	// List active connectors
	activeConnectors := connectorManager.ListActiveConnectors()
	fmt.Printf("Active connectors: %v\n", activeConnectors)

	// Release connectors when done
	for _, exchange := range exchanges {
		fmt.Printf("Releasing connector for %s...\n", exchange)
		err := connectorManager.ReleaseConnector(ctx, exchange)
		if err != nil {
			logger.Error("Failed to release connector", zap.String("exchange", exchange), zap.Error(err))
		}
	}

	// Shutdown the connector manager
	connectorManager.ShutdownAll(ctx)

	// Shutdown the coordinator
	coordinator.Shutdown(ctx)
}

