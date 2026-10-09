package fx

import (
	"context"
	"encoding/json"

	"github.com/abdoElHodaky/tradSys/internal/ws"
	transport "github.com/abdoElHodaky/tradSys/internal/ws/transport"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// WebSocketModule provides the WebSocket components
var WebSocketModule = fx.Options(
	// Provide the WebSocket hub
	fx.Provide(NewWebSocketHub),

	// Provide the WebSocket handler
	fx.Provide(NewWebSocketHandler),

	// Register lifecycle hooks
	fx.Invoke(registerWebSocketHooks),
)

// NewWebSocketHub creates a new WebSocket hub
func NewWebSocketHub(logger *zap.Logger) *transport.Hub {
	return transport.NewHub(logger)
}

// NewWebSocketHandler creates a new WebSocket handler
func NewWebSocketHandler(hub *transport.Hub, logger *zap.Logger) *ws.Server {
	// Create a simple server using the ws package
	return ws.NewServer(ws.ServerParams{
		Logger: logger,
	})
}

// registerWebSocketHooks registers lifecycle hooks for the WebSocket components
func registerWebSocketHooks(
	lc fx.Lifecycle,
	logger *zap.Logger,
	hub *transport.Hub,
	server *ws.Server,
	router *gin.Engine,
) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info("Starting WebSocket components")

			// Start the hub in a goroutine
			go hub.Run()

			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("Stopping WebSocket components")
			return nil
		},
	})
}

// RegisterMarketDataHandlers registers market data message handlers
func RegisterMarketDataHandlers(hub *transport.Hub, logger *zap.Logger) {
	// Register the market data subscription handler
	hub.RegisterMessageHandler("marketdata.subscribe", func(client *transport.Client, msg *transport.Message) {
		// Parse the subscription request from the message data
		var request struct {
			Symbol string `json:"symbol"`
		}

		err := json.Unmarshal(msg.Data, &request)
		if err != nil {
			logger.Error("Failed to parse subscription request", zap.Error(err))
			return
		}

		logger.Info("Market data subscription request",
			zap.String("client_id", client.ID),
			zap.String("symbol", request.Symbol))
	})
}

// RegisterOrderHandlers registers order message handlers
func RegisterOrderHandlers(hub *transport.Hub, logger *zap.Logger) {
	// Register the order submission handler
	hub.RegisterMessageHandler("order.submit", func(client *transport.Client, msg *transport.Message) {
		// Parse the order submission request from the message data
		var request struct {
			Symbol string  `json:"symbol"`
			Side   string  `json:"side"`
			Price  float64 `json:"price"`
			Size   float64 `json:"size"`
		}

		err := json.Unmarshal(msg.Data, &request)
		if err != nil {
			logger.Error("Failed to parse order submission request", zap.Error(err))
			return
		}

		logger.Info("Order submission request",
			zap.String("client_id", client.ID),
			zap.String("symbol", request.Symbol),
			zap.String("side", request.Side),
			zap.Float64("price", request.Price),
			zap.Float64("size", request.Size))
	})
}
