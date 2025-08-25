package app

import (
	"github.com/abdoElHodaky/tradSys/internal/architecture/coordination"
	"github.com/abdoElHodaky/tradSys/internal/exchange/connectors"
	exchange_lazy "github.com/abdoElHodaky/tradSys/internal/exchange/connectors/lazy"
	"github.com/abdoElHodaky/tradSys/internal/performance"
	performance_lazy "github.com/abdoElHodaky/tradSys/internal/performance/lazy"
	"github.com/abdoElHodaky/tradSys/internal/risk/validator/plugin"
	risk_lazy "github.com/abdoElHodaky/tradSys/internal/risk/validator/plugin/lazy"
	"github.com/abdoElHodaky/tradSys/internal/strategy"
	strategy_lazy "github.com/abdoElHodaky/tradSys/internal/strategy/lazy"
	"github.com/abdoElHodaky/tradSys/internal/trading/market_data/historical"
	historical_lazy "github.com/abdoElHodaky/tradSys/internal/trading/market_data/historical/lazy"
	"github.com/abdoElHodaky/tradSys/internal/trading/matching/algorithm/plugin"
	matching_lazy "github.com/abdoElHodaky/tradSys/internal/trading/matching/algorithm/plugin/lazy"
	"github.com/abdoElHodaky/tradSys/internal/trading/order_management"
	order_lazy "github.com/abdoElHodaky/tradSys/internal/trading/order_management/lazy"
	"github.com/abdoElHodaky/tradSys/internal/ws"
	ws_lazy "github.com/abdoElHodaky/tradSys/internal/ws/lazy"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// LazyIntegrationModule provides the fx module for lazy loading integration
var LazyIntegrationModule = fx.Options(
	coordination.Module,
	fx.Provide(
		NewLazyHistoricalDataService,
		NewLazyOrderService,
		NewLazyMatchingAlgorithmLoader,
		NewLazyWebSocketServer,
		NewLazyConnectorManager,
		NewLazyValidatorRegistry,
		NewLazyStrategyManager,
		NewLazyConnectionPool,
	),
)

// NewLazyHistoricalDataService creates a new lazy-loaded historical data service
func NewLazyHistoricalDataService(
	coordinator *coordination.ComponentCoordinator,
	config historical.Config,
	logger *zap.Logger,
) (*historical_lazy.LazyHistoricalDataService, error) {
	return historical_lazy.NewLazyHistoricalDataService(coordinator, config, logger)
}

// NewLazyOrderService creates a new lazy-loaded order service
func NewLazyOrderService(
	coordinator *coordination.ComponentCoordinator,
	config order_management.OrderServiceConfig,
	logger *zap.Logger,
) (*order_lazy.LazyOrderService, error) {
	return order_lazy.NewLazyOrderService(coordinator, config, logger)
}

// NewLazyMatchingAlgorithmLoader creates a new lazy-loaded matching algorithm loader
func NewLazyMatchingAlgorithmLoader(
	coordinator *coordination.ComponentCoordinator,
	lockManager *coordination.LockManager,
	config plugin.LoaderConfig,
	logger *zap.Logger,
) (*matching_lazy.LazyPluginLoader, error) {
	return matching_lazy.NewLazyPluginLoader(coordinator, lockManager, config, logger)
}

// NewLazyWebSocketServer creates a new lazy-loaded WebSocket server
func NewLazyWebSocketServer(
	coordinator *coordination.ComponentCoordinator,
	config ws.WebSocketConfig,
	logger *zap.Logger,
) (*ws_lazy.LazyOptimizedWebSocketServer, error) {
	return ws_lazy.NewLazyOptimizedWebSocketServer(coordinator, config, logger)
}

// NewLazyConnectorManager creates a new lazy-loaded connector manager
func NewLazyConnectorManager(
	coordinator *coordination.ComponentCoordinator,
	lockManager *coordination.LockManager,
	factory connectors.ConnectorFactory,
	config connectors.ConnectorConfig,
	logger *zap.Logger,
) (*exchange_lazy.LazyConnectorManager, error) {
	return exchange_lazy.NewLazyConnectorManager(coordinator, lockManager, factory, config, logger)
}

// NewLazyValidatorRegistry creates a new lazy-loaded validator registry
func NewLazyValidatorRegistry(
	coordinator *coordination.ComponentCoordinator,
	lockManager *coordination.LockManager,
	config plugin.RegistryConfig,
	logger *zap.Logger,
) (*risk_lazy.LazyValidatorRegistry, error) {
	return risk_lazy.NewLazyValidatorRegistry(coordinator, lockManager, config, logger)
}

// NewLazyStrategyManager creates a new lazy-loaded strategy manager
func NewLazyStrategyManager(
	coordinator *coordination.ComponentCoordinator,
	lockManager *coordination.LockManager,
	factory strategy.StrategyFactory,
	logger *zap.Logger,
) (*strategy_lazy.LazyStrategyManager, error) {
	return strategy_lazy.NewLazyStrategyManager(coordinator, lockManager, factory, logger)
}

// NewLazyConnectionPool creates a new lazy-loaded connection pool
func NewLazyConnectionPool(
	coordinator *coordination.ComponentCoordinator,
	config performance.PoolConfig,
	logger *zap.Logger,
) (*performance_lazy.LazyConnectionPool, error) {
	return performance_lazy.NewLazyConnectionPool(coordinator, config, logger)
}

