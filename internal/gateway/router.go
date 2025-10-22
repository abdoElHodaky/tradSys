package gateway

import (
	"io"
	"net/http"
	"time"

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
	config *config.Config
	engine *gin.Engine
}

// NewRouter creates a new router with fx dependency injection
func NewRouter(p RouterParams) *Router {
	router := &Router{
		logger: p.Logger,
		config: p.Config,
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
	// Create auth service and handlers
	authService := auth.NewService(auth.ServiceParams{
		Logger: r.logger,
		Config: r.config,
	})
	authHandlers := auth.NewHandlers(authService, r.logger)
	
	authGroup := r.engine.Group("/auth")
	{
		// Public routes (no authentication required)
		authGroup.POST("/login", authHandlers.Login)
		authGroup.POST("/refresh", authHandlers.RefreshToken)
		
		// Protected routes (authentication required)
		protected := authGroup.Group("/")
		protected.Use(authHandlers.ValidateToken)
		{
			protected.POST("/logout", authHandlers.Logout)
			protected.GET("/profile", authHandlers.Profile)
			protected.POST("/change-password", authHandlers.ChangePassword)
		}
	}
}

// registerAPIRoutes registers API routes with authentication
func (r *Router) registerAPIRoutes(authMiddleware *auth.Middleware) {
	// Create API group with authentication middleware
	api := r.engine.Group("/api")
	api.Use(authMiddleware.JWTAuth())
	
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
		admin.Use(authMiddleware.RoleAuth("admin"))
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
		// Service discovery and forwarding logic
		serviceURL := getServiceURL(serviceName)
		if serviceURL == "" {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "Service unavailable",
				"service": serviceName,
				"message": "Service not found in registry",
			})
			return
		}

		// Forward request to service
		targetURL := serviceURL + path
		
		// Create proxy request
		req, err := http.NewRequest(c.Request.Method, targetURL, c.Request.Body)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to create proxy request",
				"details": err.Error(),
			})
			return
		}

		// Copy headers
		for key, values := range c.Request.Header {
			for _, value := range values {
				req.Header.Add(key, value)
			}
		}

		// Execute request
		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{
				"error": "Service request failed",
				"service": serviceName,
				"details": err.Error(),
			})
			return
		}
		defer resp.Body.Close()

		// Copy response
		for key, values := range resp.Header {
			for _, value := range values {
				c.Header(key, value)
			}
		}
		
		c.Status(resp.StatusCode)
		io.Copy(c.Writer, resp.Body)
	}
}

// getServiceURL returns the URL for a given service name
func getServiceURL(serviceName string) string {
	// Service registry mapping
	services := map[string]string{
		"orders":     "http://localhost:8081",
		"risk":       "http://localhost:8082", 
		"marketdata": "http://localhost:8083",
		"users":      "http://localhost:8084",
		"auth":       "http://localhost:8085",
	}
	
	return services[serviceName]
}
