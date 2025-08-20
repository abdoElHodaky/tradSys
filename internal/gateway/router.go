package gateway

import (
	"github.com/abdoElHodaky/tradSys/internal/auth"
	"github.com/abdoElHodaky/tradSys/internal/config"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// RouterParams contains the parameters for creating a new router
type RouterParams struct {
	fx.In

	Logger      *zap.Logger
	Config      *config.Config
	Server      *Server
	AuthMiddleware *auth.Middleware
	// Add other handlers as needed
}

// Router represents the API Gateway router
type Router struct {
	logger *zap.Logger
	engine *gin.Engine
}

// NewRouter creates a new router with fx dependency injection
func NewRouter(p RouterParams) *Router {
	router := &Router{
		logger: p.Logger,
		engine: p.Server.Router(),
	}

	// Register routes
	router.registerHealthRoutes()
	router.registerAuthRoutes(p.AuthMiddleware)
	router.registerAPIRoutes(p.AuthMiddleware)

	return router
}

// registerHealthRoutes registers health check routes
func (r *Router) registerHealthRoutes() {
	r.engine.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "ok",
		})
	})
}

// registerAuthRoutes registers authentication routes
func (r *Router) registerAuthRoutes(authMiddleware *auth.Middleware) {
	auth := r.engine.Group("/auth")
	{
		auth.POST("/login", authMiddleware.LoginHandler)
		auth.POST("/refresh", authMiddleware.RefreshHandler)
	}
}

// registerAPIRoutes registers API routes with authentication
func (r *Router) registerAPIRoutes(authMiddleware *auth.Middleware) {
	// Create API group with authentication middleware
	api := r.engine.Group("/api")
	api.Use(authMiddleware.AuthRequired())
	
	// Market data routes
	marketData := api.Group("/market-data")
	{
		marketData.GET("/", forwardToService("marketdata", "/"))
		marketData.GET("/symbols", forwardToService("marketdata", "/symbols"))
		marketData.GET("/quotes/:symbol", forwardToService("marketdata", "/quotes/:symbol"))
		marketData.GET("/candles/:symbol", forwardToService("marketdata", "/candles/:symbol"))
	}

	// Order routes
	orders := api.Group("/orders")
	{
		orders.GET("/", forwardToService("orders", "/"))
		orders.POST("/", forwardToService("orders", "/"))
		orders.GET("/:id", forwardToService("orders", "/:id"))
		orders.PUT("/:id", forwardToService("orders", "/:id"))
		orders.DELETE("/:id", forwardToService("orders", "/:id"))
	}

	// Risk routes
	risk := api.Group("/risk")
	{
		risk.GET("/positions", forwardToService("risk", "/positions"))
		risk.GET("/limits", forwardToService("risk", "/limits"))
		risk.POST("/validate", forwardToService("risk", "/validate"))
	}

	// Pairs routes
	pairs := api.Group("/pairs")
	{
		pairs.GET("/", forwardToService("marketdata", "/pairs"))
		pairs.POST("/", forwardToService("marketdata", "/pairs"))
		pairs.GET("/:id", forwardToService("marketdata", "/pairs/:id"))
		pairs.PUT("/:id", forwardToService("marketdata", "/pairs/:id"))
		pairs.DELETE("/:id", forwardToService("marketdata", "/pairs/:id"))
		pairs.GET("/:id/stats", forwardToService("marketdata", "/pairs/:id/stats"))
	}

	// Strategy routes
	strategies := api.Group("/strategies")
	{
		strategies.GET("/", forwardToService("orders", "/strategies"))
		strategies.POST("/", forwardToService("orders", "/strategies"))
		strategies.GET("/:id", forwardToService("orders", "/strategies/:id"))
		strategies.PUT("/:id", forwardToService("orders", "/strategies/:id"))
		strategies.DELETE("/:id", forwardToService("orders", "/strategies/:id"))
		strategies.POST("/:id/start", forwardToService("orders", "/strategies/:id/start"))
		strategies.POST("/:id/stop", forwardToService("orders", "/strategies/:id/stop"))
	}

	// User routes
	users := api.Group("/users")
	{
		users.GET("/me", forwardToService("users", "/me"))
		users.PUT("/me", forwardToService("users", "/me"))
		
		// Admin-only routes
		admin := users.Group("/")
		admin.Use(authMiddleware.AdminRequired())
		{
			admin.GET("/", forwardToService("users", "/"))
			admin.POST("/", forwardToService("users", "/"))
			admin.GET("/:id", forwardToService("users", "/:id"))
			admin.PUT("/:id", forwardToService("users", "/:id"))
			admin.DELETE("/:id", forwardToService("users", "/:id"))
		}
	}
}

// forwardToService creates a handler that forwards requests to the appropriate microservice
func forwardToService(serviceName, path string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// This is a placeholder for the actual service forwarding logic
		// In a real implementation, this would use a service discovery mechanism
		// and forward the request to the appropriate service
		c.JSON(501, gin.H{
			"error": "Service forwarding not implemented",
			"service": serviceName,
			"path": path,
		})
	}
}

