package gateway

import (
	"context"
	"net/http"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/auth"
	"github.com/abdoElHodaky/tradSys/internal/config"
	"github.com/abdoElHodaky/tradSys/internal/monitoring"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// ServerParams contains the parameters for creating a new API Gateway server
type ServerParams struct {
	fx.In

	Lifecycle fx.Lifecycle
	Logger    *zap.Logger
	Config    *config.Config
	Metrics   *monitoring.MetricsCollector `optional:"true"`
}

// Server represents the API Gateway server
type Server struct {
	router *gin.Engine
	logger *zap.Logger
	config *config.Config
	server *http.Server
}

// NewServer creates a new API Gateway server with fx dependency injection
func NewServer(p ServerParams) *Server {
	// Set Gin mode based on configuration
	if p.Config.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Add middleware
	router.Use(gin.Recovery())
	router.Use(RequestLogger(p.Logger))
	
	// Configure CORS
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Add metrics middleware if available
	if p.Metrics != nil {
		router.Use(p.Metrics.GinMiddleware())
	}

	// Create server
	server := &Server{
		router: router,
		logger: p.Logger,
		config: p.Config,
		server: &http.Server{
			Addr:    p.Config.Gateway.Address,
			Handler: router,
		},
	}

	// Register lifecycle hooks
	p.Lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				p.Logger.Info("Starting API Gateway server", zap.String("address", p.Config.Gateway.Address))
				if err := server.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					p.Logger.Error("Failed to start API Gateway server", zap.Error(err))
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			p.Logger.Info("Stopping API Gateway server")
			return server.server.Shutdown(ctx)
		},
	})

	return server
}

// RequestLogger returns a gin middleware for logging HTTP requests
func RequestLogger(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()

		if raw != "" {
			path = path + "?" + raw
		}

		logger.Info("API request",
			zap.String("path", path),
			zap.String("method", method),
			zap.Int("status", statusCode),
			zap.String("ip", clientIP),
			zap.Duration("latency", latency),
		)
	}
}

// Router returns the Gin router
func (s *Server) Router() *gin.Engine {
	return s.router
}

// Module provides the API Gateway module for fx
var Module = fx.Options(
	fx.Provide(NewServer),
	fx.Provide(NewServiceProxy),
	fx.Provide(NewRouter),
)
