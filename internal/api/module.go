package api

import (
	"github.com/abdoElHodaky/tradSys/internal/api/handlers"
	"github.com/abdoElHodaky/tradSys/internal/auth"
	"github.com/abdoElHodaky/tradSys/internal/orders"
	"github.com/abdoElHodaky/tradSys/internal/risk"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

// Module provides the API module for fx
var Module = fx.Options(
	// Provide handlers
	fx.Provide(handlers.NewOrderHandler),
	fx.Provide(handlers.NewRiskHandler),
	
	// Register routes
	fx.Invoke(func(
		router *gin.Engine,
		authMiddleware *auth.Middleware,
		orderHandler *handlers.OrderHandler,
		riskHandler *handlers.RiskHandler,
	) {
		// Create API group with authentication middleware
		api := router.Group("/api")
		api.Use(authMiddleware.AuthRequired())
		
		// Register routes
		orderHandler.RegisterRoutes(api)
		riskHandler.RegisterRoutes(api)
	}),
	
	// Include service modules
	orders.ServiceModule,
	risk.Module,
)

