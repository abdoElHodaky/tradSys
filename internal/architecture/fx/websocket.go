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
	// Provide the WebSocket hub
	fx.Provide(NewWebSocketHub),

	// Provide the WebSocket handler
	fx.Provide(NewWebSocketHandler),

	// Register lifecycle hooks
	fx.Invoke(registerWebSocketHooks),
)

// NewWebSocketHub creates a new WebSocket hub
func NewWebSocketHub(logger *zap.Logger) *websocket.Hub {
	return websocket.NewHub(logger)
}

// NewWebSocketHandler creates a new WebSocket handler
func NewWebSocketHandler(hub *websocket.Hub, logger *zap.Logger) *websocket.WebSocketHandler {
	config := websocket.DefaultWebSocketHandlerConfig()
	return websocket.NewWebSocketHandler(hub, logger, config)
}

// registerWebSocketHooks registers lifecycle hooks for the WebSocket components
func registerWebSocketHooks(
	lc fx.Lifecycle,
	logger *zap.Logger,
	hub *websocket.Hub,
	handler *websocket.WebSocketHandler,
	router *gin.Engine,
) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info("Starting WebSocket components")

			// Register the WebSocket routes
			handler.RegisterRoutes(router)

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
