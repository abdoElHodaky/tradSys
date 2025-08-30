package dss

import (
	"context"
	
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Module provides the DSS components
var Module = fx.Options(
	// Provide the DSS service
	fx.Provide(NewService),
	
	// Provide the DSS API
	fx.Provide(NewAPI),
	
	// Provide the WebSocket manager
	fx.Provide(NewWebSocketManager),
	
	// Register lifecycle hooks
	fx.Invoke(registerDSSHooks),
)

// registerDSSHooks registers lifecycle hooks for the DSS components
func registerDSSHooks(
	lc fx.Lifecycle,
	logger *zap.Logger,
	api *API,
	router *gin.Engine,
) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info("Starting DSS components")
			
			// Register the DSS API routes
			api.RegisterRoutes(router)
			
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("Stopping DSS components")
			return nil
		},
	})
}

