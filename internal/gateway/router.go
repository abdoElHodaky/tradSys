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

	Logger         *zap.Logger
	Config         *config.Config
	Server         *Server
	AuthMiddleware *auth.Middleware
	ServiceProxy   *ServiceProxy
}

// Router represents the API Gateway router
type Router struct {
	logger *zap.Logger
	engine *gin.Engine
	proxy  *ServiceProxy
}

// NewRouter creates a new router with fx dependency injection
func NewRouter(p RouterParams) *Router {
	router := &Router{
		logger: p.Logger,
		engine: p.Server.Router(),
		proxy:  p.ServiceProxy,
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
		auth.POST("/login", authMiddleware.LoginHandler())
		auth.POST("/refresh", authMiddleware.RefreshHandler())
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
		marketData.GET("/", r.proxy.ForwardToService("marketdata", "/"))
		marketData.GET("/symbols", r.proxy.ForwardToService("marketdata", "/symbols"))
		marketData.GET("/quotes/:symbol", r.proxy.ForwardToService("marketdata", "/quotes/:symbol"))
		marketData.GET("/candles/:symbol", r.proxy.ForwardToService("marketdata", "/candles/:symbol"))
	}

	// Order routes
	orders := api.Group("/orders")
	{
		orders.GET("/", r.proxy.ForwardToService("orders", "/"))
		orders.POST("/", r.proxy.ForwardToService("orders", "/"))
		orders.GET("/:id", r.proxy.ForwardToService("orders", "/:id"))
		orders.PUT("/:id", r.proxy.ForwardToService("orders", "/:id"))
		orders.DELETE("/:id", r.proxy.ForwardToService("orders", "/:id"))
	}

	// Risk routes
	risk := api.Group("/risk")
	{
		risk.GET("/positions", r.proxy.ForwardToService("risk", "/positions"))
		risk.GET("/limits", r.proxy.ForwardToService("risk", "/limits"))
		risk.POST("/validate", r.proxy.ForwardToService("risk", "/validate"))
	}

	// Pairs routes
	pairs := api.Group("/pairs")
	{
		pairs.GET("/", r.proxy.ForwardToService("marketdata", "/pairs"))
		pairs.POST("/", r.proxy.ForwardToService("marketdata", "/pairs"))
		pairs.GET("/:id", r.proxy.ForwardToService("marketdata", "/pairs/:id"))
		pairs.PUT("/:id", r.proxy.ForwardToService("marketdata", "/pairs/:id"))
		pairs.DELETE("/:id", r.proxy.ForwardToService("marketdata", "/pairs/:id"))
		pairs.GET("/:id/stats", r.proxy.ForwardToService("marketdata", "/pairs/:id/stats"))
	}

	// Strategy routes
	strategies := api.Group("/strategies")
	{
		strategies.GET("/", r.proxy.ForwardToService("orders", "/strategies"))
		strategies.POST("/", r.proxy.ForwardToService("orders", "/strategies"))
		strategies.GET("/:id", r.proxy.ForwardToService("orders", "/strategies/:id"))
		strategies.PUT("/:id", r.proxy.ForwardToService("orders", "/strategies/:id"))
		strategies.DELETE("/:id", r.proxy.ForwardToService("orders", "/strategies/:id"))
		strategies.POST("/:id/start", r.proxy.ForwardToService("orders", "/strategies/:id/start"))
		strategies.POST("/:id/stop", r.proxy.ForwardToService("orders", "/strategies/:id/stop"))
	}

	// User routes
	users := api.Group("/users")
	{
		users.GET("/me", r.proxy.ForwardToService("users", "/me"))
		users.PUT("/me", r.proxy.ForwardToService("users", "/me"))
		
		// Admin-only routes
		admin := users.Group("/")
		admin.Use(authMiddleware.AdminRequired())
		{
			admin.GET("/", r.proxy.ForwardToService("users", "/"))
			admin.POST("/", r.proxy.ForwardToService("users", "/"))
			admin.GET("/:id", r.proxy.ForwardToService("users", "/:id"))
			admin.PUT("/:id", r.proxy.ForwardToService("users", "/:id"))
			admin.DELETE("/:id", r.proxy.ForwardToService("users", "/:id"))
		}
	}
}
