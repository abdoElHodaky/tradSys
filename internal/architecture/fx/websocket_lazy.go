package fx

import (
	"github.com/abdoElHodaky/tradSys/internal/architecture/fx/lazy"
	"github.com/abdoElHodaky/tradSys/internal/transport/websocket"
	"github.com/abdoElHodaky/tradSys/internal/trading/market_data"
	"github.com/abdoElHodaky/tradSys/internal/trading/orders"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// LazyWebSocketModule provides lazily loaded WebSocket components
var LazyWebSocketModule = fx.Options(
	// Provide the WebSocket hub (always loaded eagerly)
	fx.Provide(NewWebSocketHub),
	
	// Provide lazily loaded WebSocket components
	provideLazyWebSocketHandler,
	provideLazyMarketDataHandler,
	provideLazyOrderHandler,
	
	// Register lifecycle hooks
	fx.Invoke(registerLazyWebSocketHooks),
)

// provideLazyWebSocketHandler provides a lazily loaded WebSocket handler
func provideLazyWebSocketHandler(logger *zap.Logger, metrics *lazy.LazyLoadingMetrics) *lazy.LazyProvider {
	return lazy.NewLazyProvider(
		"websocket-handler",
		func(hub *websocket.Hub, logger *zap.Logger) (*websocket.WebSocketHandler, error) {
			logger.Info("Lazily initializing WebSocket handler")
			config := websocket.DefaultWebSocketHandlerConfig()
			return websocket.NewWebSocketHandler(hub, logger, config), nil
		},
		logger,
		metrics,
	)
}

// provideLazyMarketDataHandler provides a lazily loaded market data handler
func provideLazyMarketDataHandler(logger *zap.Logger, metrics *lazy.LazyLoadingMetrics) *lazy.LazyProvider {
	return lazy.NewLazyProvider(
		"websocket-market-data-handler",
		func(
			hub *websocket.Hub,
			logger *zap.Logger,
			marketDataService *market_data.MarketDataService,
		) (*websocket.MarketDataHandler, error) {
			logger.Info("Lazily initializing WebSocket market data handler")
			return websocket.NewMarketDataHandler(hub, logger, marketDataService), nil
		},
		logger,
		metrics,
	)
}

// provideLazyOrderHandler provides a lazily loaded order handler
func provideLazyOrderHandler(logger *zap.Logger, metrics *lazy.LazyLoadingMetrics) *lazy.LazyProvider {
	return lazy.NewLazyProvider(
		"websocket-order-handler",
		func(
			hub *websocket.Hub,
			logger *zap.Logger,
			orderService *orders.OrderService,
		) (*websocket.OrderHandler, error) {
			logger.Info("Lazily initializing WebSocket order handler")
			return websocket.NewOrderHandler(hub, logger, orderService), nil
		},
		logger,
		metrics,
	)
}

// registerLazyWebSocketHooks registers lifecycle hooks for the lazy WebSocket components
func registerLazyWebSocketHooks(
	lc fx.Lifecycle,
	logger *zap.Logger,
	handlerProvider *lazy.LazyProvider,
	marketDataHandlerProvider *lazy.LazyProvider,
	orderHandlerProvider *lazy.LazyProvider,
) {
	logger.Info("Registering lazy WebSocket component hooks")
}

// GetWebSocketHandler gets the WebSocket handler, initializing it if necessary
func GetWebSocketHandler(provider *lazy.LazyProvider) (*websocket.WebSocketHandler, error) {
	instance, err := provider.Get()
	if err != nil {
		return nil, err
	}
	return instance.(*websocket.WebSocketHandler), nil
}

// GetMarketDataHandler gets the market data handler, initializing it if necessary
func GetMarketDataHandler(provider *lazy.LazyProvider) (*websocket.MarketDataHandler, error) {
	instance, err := provider.Get()
	if err != nil {
		return nil, err
	}
	return instance.(*websocket.MarketDataHandler), nil
}

// GetOrderHandler gets the order handler, initializing it if necessary
func GetOrderHandler(provider *lazy.LazyProvider) (*websocket.OrderHandler, error) {
	instance, err := provider.Get()
	if err != nil {
		return nil, err
	}
	return instance.(*websocket.OrderHandler), nil
}

