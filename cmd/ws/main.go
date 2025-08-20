package main

import (
	"net/http"

	"github.com/abdoElHodaky/tradSys/internal/config"
	"github.com/abdoElHodaky/tradSys/internal/micro"
	"github.com/abdoElHodaky/tradSys/internal/ws"
	"github.com/abdoElHodaky/tradSys/proto/ws"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	app := fx.New(
		fx.Supply(logger),
		config.Module,
		micro.Module,
		ws.ServerModule,
		ws.Module,
		fx.Provide(newGinEngine),
		fx.Invoke(registerWsHandler),
		fx.Invoke(setupRoutes),
	)

	app.Run()
}

func newGinEngine() *gin.Engine {
	r := gin.Default()
	return r
}

func registerWsHandler(
	lc fx.Lifecycle,
	logger *zap.Logger,
	service *micro.Service,
	handler *ws.Handler,
) {
	// Register the handler with the service
	if err := ws.RegisterWebSocketServiceHandler(service.Server(), handler); err != nil {
		logger.Fatal("Failed to register handler", zap.Error(err))
	}

	logger.Info("WebSocket service registered")
}

func setupRoutes(
	lc fx.Lifecycle,
	logger *zap.Logger,
	router *gin.Engine,
	wsServer *ws.Server,
) {
	// WebSocket endpoint
	router.GET("/ws", func(c *gin.Context) {
		wsServer.HandleWebSocket(c.Writer, c.Request)
	})

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	})

	// Start HTTP server
	lc.Append(fx.Hook{
		OnStart: func(ctx fx.Context) error {
			go func() {
				if err := router.Run(":8080"); err != nil {
					logger.Error("Failed to start HTTP server", zap.Error(err))
				}
			}()
			return nil
		},
		OnStop: func(ctx fx.Context) error {
			logger.Info("Stopping HTTP server")
			return nil
		},
	})
}

