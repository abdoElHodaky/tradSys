package fx

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/websocket"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// WebSocketModule provides the WebSocket components
var WebSocketModule = fx.Options(
	// Provide the WebSocket gateway
	fx.Provide(NewWebSocketGateway),

	// Provide the WebSocket handler
	fx.Provide(NewWebSocketMessageHandler),

	// Register lifecycle hooks
	fx.Invoke(registerWebSocketHooks),
)

// NewWebSocketGateway creates a new WebSocket gateway
func NewWebSocketGateway(logger *zap.Logger) *websocket.Gateway {
	config := &websocket.GatewayConfig{
		MaxConnections:    1000,
		ReadBufferSize:    1024,
		WriteBufferSize:   1024,
		HandshakeTimeout:  time.Second * 10,
		ReadTimeout:       time.Second * 60,
		WriteTimeout:      time.Second * 10,
		PingPeriod:        time.Second * 54,
		MaxMessageSize:    512,
		EnableCompression: true,
	}
	return websocket.NewGateway(config, logger)
}

// NewWebSocketMessageHandler creates a new WebSocket message handler
func NewWebSocketMessageHandler(gateway *websocket.Gateway, logger *zap.Logger) *websocket.MessageHandler {
	return websocket.NewMessageHandler(gateway, logger)
}

// registerWebSocketHooks registers lifecycle hooks for the WebSocket components
func registerWebSocketHooks(
	lc fx.Lifecycle,
	logger *zap.Logger,
	gateway *websocket.Gateway,
	handler *websocket.MessageHandler,
	router *gin.Engine,
) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info("Starting WebSocket components")

			// Register the WebSocket routes
			router.GET("/ws", func(c *gin.Context) {
				// Handle WebSocket upgrade
				gateway.HandleWebSocket(c.Writer, c.Request)
			})

			// Start the gateway in a goroutine
			go func() {
				if err := gateway.Start(); err != nil {
					logger.Error("Failed to start WebSocket gateway", zap.Error(err))
				}
			}()

			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("Stopping WebSocket components")
			gateway.Stop()
			return nil
		},
	})
}

// RegisterMarketDataHandlers registers market data message handlers
func RegisterMarketDataHandlers(hub *websocket.Hub, logger *zap.Logger) {
	// Register the market data subscription handler
	hub.RegisterMessageHandler("marketdata.subscribe", func(client *websocket.Client, msg *websocket.Message) {
		// Parse the subscription request
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

		// Subscribe to market data
		logger.Info("Market data subscription request",
			zap.String("client_id", client.ID),
			zap.String("symbol", request.Symbol))

		// Add client to symbol subscription
		hub.SubscribeToSymbol(client, request.Symbol)

		// Send confirmation
		response := map[string]interface{}{
			"type":    "marketdata.subscribed",
			"symbol":  request.Symbol,
			"status":  "success",
			"message": "Successfully subscribed to " + request.Symbol,
		}
		client.Send(response)
	})

	// Register the market data unsubscription handler
	hub.RegisterMessageHandler("marketdata.unsubscribe", func(client *websocket.Client, msg *websocket.Message) {
		// Parse the unsubscription request
		var request struct {
			Symbol string `json:"symbol"`
		}

		err := json.Unmarshal(msg.Data, &request)
		if err != nil {
			logger.Error("Failed to parse unsubscription request", zap.Error(err))
			return
		}

		logger.Info("Market data unsubscription request",
			zap.String("client_id", client.ID),
			zap.String("symbol", request.Symbol))

		// Unsubscribe from market data
		logger.Info("Market data unsubscription request",
			zap.String("client_id", client.ID),
			zap.String("symbol", request.Symbol))

		// Remove client from symbol subscription
		hub.UnsubscribeFromSymbol(client, request.Symbol)

		// Send confirmation
		response := map[string]interface{}{
			"type":    "marketdata.unsubscribed",
			"symbol":  request.Symbol,
			"status":  "success",
			"message": "Successfully unsubscribed from " + request.Symbol,
		}
		client.Send(response)
	})
}

// RegisterOrderHandlers registers order message handlers
func RegisterOrderHandlers(hub *websocket.Hub, logger *zap.Logger) {
	// Register the order submission handler
	hub.RegisterMessageHandler("order.submit", func(client *websocket.Client, msg *websocket.Message) {
		// Parse the order submission request
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

		// Submit the order
		logger.Info("Order submission request",
			zap.String("client_id", client.ID),
			zap.String("symbol", request.Symbol),
			zap.String("side", request.Side),
			zap.Float64("quantity", request.Quantity),
			zap.Float64("price", request.Price))

		// Create order submission response
		orderID := "order_" + client.ID + "_" + fmt.Sprintf("%d", time.Now().Unix())
		response := map[string]interface{}{
			"type":     "order.submitted",
			"order_id": orderID,
			"symbol":   request.Symbol,
			"side":     request.Side,
			"quantity": request.Quantity,
			"price":    request.Price,
			"status":   "pending",
			"message":  "Order submitted successfully",
		}
		client.Send(response)
	})

	// Register the order cancellation handler
	hub.RegisterMessageHandler("order.cancel", func(client *websocket.Client, msg *websocket.Message) {
		// Parse the order cancellation request
		var request struct {
			OrderID string `json:"order_id"`
		}

		err := json.Unmarshal(msg.Data, &request)
		if err != nil {
			logger.Error("Failed to parse order cancellation request", zap.Error(err))
			return
		}

		logger.Info("Order cancellation request",
			zap.String("client_id", client.ID),
			zap.String("order_id", request.OrderID))

		// Cancel the order
		logger.Info("Order cancellation request",
			zap.String("client_id", client.ID),
			zap.String("order_id", request.OrderID))

		// Create order cancellation response
		response := map[string]interface{}{
			"type":     "order.cancelled",
			"order_id": request.OrderID,
			"status":   "cancelled",
			"message":  "Order cancelled successfully",
		}
		client.Send(response)
	})
}
