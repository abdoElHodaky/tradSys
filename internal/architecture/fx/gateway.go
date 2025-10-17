package fx

import (
	"context"

	"github.com/abdoElHodaky/tradSys/internal/architecture/discovery"
	"github.com/abdoElHodaky/tradSys/internal/architecture/gateway"
	"github.com/abdoElHodaky/tradSys/internal/common"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// GatewayModule provides API gateway components
var GatewayModule = fx.Options(
	// Provide the API gateway
	fx.Provide(NewAPIGateway),
	
	// Register lifecycle hooks
	fx.Invoke(registerGatewayHooks),
)

// NewAPIGateway creates a new API gateway
func NewAPIGateway(discovery *discovery.ServiceDiscovery, logger *zap.Logger) *gateway.APIGateway {
	return gateway.NewAPIGateway(discovery, logger)
}

// registerGatewayHooks registers lifecycle hooks for the API gateway
func registerGatewayHooks(
	lc fx.Lifecycle,
	logger *zap.Logger,
	apiGateway *gateway.APIGateway,
) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info("Starting API gateway")
			
			// Add middleware
			apiGateway.Use(gin.Logger())
			apiGateway.Use(gin.Recovery())
			
			// Add correlation middleware for request tracing
			correlationMiddleware := common.NewCorrelationMiddleware(logger)
			apiGateway.Use(correlationMiddleware.Handler())
			
			// Add health check routes
			healthHandler := common.NewHealthHandler("api-gateway", "1.0.0", logger)
			healthHandler.RegisterRoutes(apiGateway.GetRouter())
			
			// Add routes
			addGatewayRoutes(apiGateway)
			
			// Start the API gateway in a goroutine
			go func() {
				if err := apiGateway.Run(":8080"); err != nil {
					logger.Error("Failed to start API gateway", zap.Error(err))
				}
			}()
			
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("Stopping API gateway")
			return nil
		},
	})
}

// addGatewayRoutes adds routes to the API gateway
func addGatewayRoutes(apiGateway *gateway.APIGateway) {
	// Add routes for the market data service
	apiGateway.AddRoute(gateway.Route{
		Method:      "GET",
		Path:        "/api/v1/market-data/:symbol",
		ServiceName: "market-data",
		ServicePath: "/market-data/:symbol",
		Middlewares: []gin.HandlerFunc{},
	})
	
	// Add routes for the orders service
	apiGateway.AddRoute(gateway.Route{
		Method:      "POST",
		Path:        "/api/v1/orders",
		ServiceName: "orders",
		ServicePath: "/orders",
		Middlewares: []gin.HandlerFunc{},
	})
	
	apiGateway.AddRoute(gateway.Route{
		Method:      "GET",
		Path:        "/api/v1/orders/:id",
		ServiceName: "orders",
		ServicePath: "/orders/:id",
		Middlewares: []gin.HandlerFunc{},
	})
	
	// Add routes for the risk service
	apiGateway.AddRoute(gateway.Route{
		Method:      "GET",
		Path:        "/api/v1/risk/:account",
		ServiceName: "risk",
		ServicePath: "/risk/:account",
		Middlewares: []gin.HandlerFunc{},
	})
}
