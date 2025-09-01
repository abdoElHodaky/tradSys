package fx

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/abdoElHodaky/tradSys/internal/transport/websocket"
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

// MarketDataSubscription represents a market data subscription
type MarketDataSubscription struct {
	ClientID string
	Symbol   string
}

// MarketDataManager manages market data subscriptions
type MarketDataManager struct {
	subscriptions map[string]map[string]bool // symbol -> clientID -> bool
	clients       map[string]map[string]bool // clientID -> symbol -> bool
	mu            sync.RWMutex
	logger        *zap.Logger
}

// NewMarketDataManager creates a new market data manager
func NewMarketDataManager(logger *zap.Logger) *MarketDataManager {
	return &MarketDataManager{
		subscriptions: make(map[string]map[string]bool),
		clients:       make(map[string]map[string]bool),
		logger:        logger,
	}
}

// Subscribe subscribes a client to market data for a symbol
func (m *MarketDataManager) Subscribe(clientID, symbol string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Initialize maps if they don't exist
	if _, ok := m.subscriptions[symbol]; !ok {
		m.subscriptions[symbol] = make(map[string]bool)
	}
	if _, ok := m.clients[clientID]; !ok {
		m.clients[clientID] = make(map[string]bool)
	}

	// Add the subscription
	m.subscriptions[symbol][clientID] = true
	m.clients[clientID][symbol] = true

	m.logger.Debug("Client subscribed to market data",
		zap.String("client_id", clientID),
		zap.String("symbol", symbol),
	)
}

// Unsubscribe unsubscribes a client from market data for a symbol
func (m *MarketDataManager) Unsubscribe(clientID, symbol string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Remove the subscription
	if clients, ok := m.subscriptions[symbol]; ok {
		delete(clients, clientID)
		// Clean up empty maps
		if len(clients) == 0 {
			delete(m.subscriptions, symbol)
		}
	}

	// Remove from client map
	if symbols, ok := m.clients[clientID]; ok {
		delete(symbols, symbol)
		// Clean up empty maps
		if len(symbols) == 0 {
			delete(m.clients, clientID)
		}
	}

	m.logger.Debug("Client unsubscribed from market data",
		zap.String("client_id", clientID),
		zap.String("symbol", symbol),
	)
}

// UnsubscribeAll unsubscribes a client from all market data
func (m *MarketDataManager) UnsubscribeAll(clientID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Get all symbols the client is subscribed to
	symbols, ok := m.clients[clientID]
	if !ok {
		return
	}

	// Remove the client from all symbol subscriptions
	for symbol := range symbols {
		if clients, ok := m.subscriptions[symbol]; ok {
			delete(clients, clientID)
			// Clean up empty maps
			if len(clients) == 0 {
				delete(m.subscriptions, symbol)
			}
		}
	}

	// Remove the client from the clients map
	delete(m.clients, clientID)

	m.logger.Debug("Client unsubscribed from all market data",
		zap.String("client_id", clientID),
	)
}

// GetSubscribedClients gets all clients subscribed to a symbol
func (m *MarketDataManager) GetSubscribedClients(symbol string) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	clients, ok := m.subscriptions[symbol]
	if !ok {
		return nil
	}

	// Convert map keys to slice
	result := make([]string, 0, len(clients))
	for clientID := range clients {
		result = append(result, clientID)
	}

	return result
}

// RegisterMarketDataHandlers registers market data message handlers
func RegisterMarketDataHandlers(hub *websocket.Hub, logger *zap.Logger) {
	// Create a market data manager
	manager := NewMarketDataManager(logger)

	// Register client disconnect handler to clean up subscriptions
	hub.OnClientDisconnect(func(client *websocket.Client) {
		manager.UnsubscribeAll(client.ID)
	})

	// Register the market data subscription handler
	hub.RegisterMessageHandler("marketdata.subscribe", func(client *websocket.Client, msg *websocket.Message) {
		// Parse the subscription request
		var request struct {
			Symbol string `json:"symbol"`
		}
		
		err := json.Unmarshal(msg.Data, &request)
		if err != nil {
			logger.Error("Failed to parse subscription request", zap.Error(err))
			sendErrorResponse(client, "Invalid subscription request", err)
			return
		}
		
		logger.Info("Market data subscription request",
			zap.String("client_id", client.ID),
			zap.String("symbol", request.Symbol))
		
		// Subscribe the client to the symbol
		manager.Subscribe(client.ID, request.Symbol)
		
		// Send confirmation
		sendSuccessResponse(client, "Subscribed to market data", map[string]string{
			"symbol": request.Symbol,
		})
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
			sendErrorResponse(client, "Invalid unsubscription request", err)
			return
		}
		
		logger.Info("Market data unsubscription request",
			zap.String("client_id", client.ID),
			zap.String("symbol", request.Symbol))
		
		// Unsubscribe the client from the symbol
		manager.Unsubscribe(client.ID, request.Symbol)
		
		// Send confirmation
		sendSuccessResponse(client, "Unsubscribed from market data", map[string]string{
			"symbol": request.Symbol,
		})
	})
}

// OrderManager manages order operations
type OrderManager struct {
	orders  map[string]map[string]*Order // clientID -> orderID -> Order
	mu      sync.RWMutex
	logger  *zap.Logger
	nextID  uint64
	idMutex sync.Mutex
}

// Order represents a trading order
type Order struct {
	ID        string  `json:"id"`
	ClientID  string  `json:"client_id"`
	Symbol    string  `json:"symbol"`
	Side      string  `json:"side"`
	Price     float64 `json:"price"`
	Size      float64 `json:"size"`
	Status    string  `json:"status"`
	Timestamp int64   `json:"timestamp"`
}

// NewOrderManager creates a new order manager
func NewOrderManager(logger *zap.Logger) *OrderManager {
	return &OrderManager{
		orders: make(map[string]map[string]*Order),
		logger: logger,
	}
}

// SubmitOrder submits a new order
func (m *OrderManager) SubmitOrder(clientID, symbol, side string, price, size float64) (*Order, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Generate a new order ID
	orderID := fmt.Sprintf("order-%d", m.getNextID())

	// Create the order
	order := &Order{
		ID:        orderID,
		ClientID:  clientID,
		Symbol:    symbol,
		Side:      side,
		Price:     price,
		Size:      size,
		Status:    "pending",
		Timestamp: time.Now().UnixNano(),
	}

	// Initialize client orders map if it doesn't exist
	if _, ok := m.orders[clientID]; !ok {
		m.orders[clientID] = make(map[string]*Order)
	}

	// Store the order
	m.orders[clientID][orderID] = order

	m.logger.Info("Order submitted",
		zap.String("client_id", clientID),
		zap.String("order_id", orderID),
		zap.String("symbol", symbol),
		zap.String("side", side),
		zap.Float64("price", price),
		zap.Float64("size", size),
	)

	return order, nil
}

// CancelOrder cancels an order
func (m *OrderManager) CancelOrder(clientID, orderID string) (*Order, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if client exists
	clientOrders, ok := m.orders[clientID]
	if !ok {
		return nil, fmt.Errorf("client not found: %s", clientID)
	}

	// Check if order exists
	order, ok := clientOrders[orderID]
	if !ok {
		return nil, fmt.Errorf("order not found: %s", orderID)
	}

	// Check if order can be cancelled
	if order.Status != "pending" && order.Status != "partial" {
		return nil, fmt.Errorf("order cannot be cancelled: %s", order.Status)
	}

	// Update order status
	order.Status = "cancelled"

	m.logger.Info("Order cancelled",
		zap.String("client_id", clientID),
		zap.String("order_id", orderID),
	)

	return order, nil
}

// GetOrder gets an order by ID
func (m *OrderManager) GetOrder(clientID, orderID string) (*Order, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Check if client exists
	clientOrders, ok := m.orders[clientID]
	if !ok {
		return nil, fmt.Errorf("client not found: %s", clientID)
	}

	// Check if order exists
	order, ok := clientOrders[orderID]
	if !ok {
		return nil, fmt.Errorf("order not found: %s", orderID)
	}

	return order, nil
}

// GetClientOrders gets all orders for a client
func (m *OrderManager) GetClientOrders(clientID string) []*Order {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Check if client exists
	clientOrders, ok := m.orders[clientID]
	if !ok {
		return nil
	}

	// Convert map values to slice
	result := make([]*Order, 0, len(clientOrders))
	for _, order := range clientOrders {
		result = append(result, order)
	}

	return result
}

// getNextID gets the next order ID
func (m *OrderManager) getNextID() uint64 {
	m.idMutex.Lock()
	defer m.idMutex.Unlock()
	m.nextID++
	return m.nextID
}

// RegisterOrderHandlers registers order message handlers
func RegisterOrderHandlers(hub *websocket.Hub, logger *zap.Logger) {
	// Create an order manager
	manager := NewOrderManager(logger)

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
			sendErrorResponse(client, "Invalid order submission request", err)
			return
		}
		
		logger.Info("Order submission request",
			zap.String("client_id", client.ID),
			zap.String("symbol", request.Symbol),
			zap.String("side", request.Side),
			zap.Float64("price", request.Price),
			zap.Float64("size", request.Size))
		
		// Validate the request
		if request.Symbol == "" {
			sendErrorResponse(client, "Symbol is required", nil)
			return
		}
		if request.Side != "buy" && request.Side != "sell" {
			sendErrorResponse(client, "Side must be 'buy' or 'sell'", nil)
			return
		}
		if request.Price <= 0 {
			sendErrorResponse(client, "Price must be greater than zero", nil)
			return
		}
		if request.Size <= 0 {
			sendErrorResponse(client, "Size must be greater than zero", nil)
			return
		}
		
		// Submit the order
		order, err := manager.SubmitOrder(client.ID, request.Symbol, request.Side, request.Price, request.Size)
		if err != nil {
			logger.Error("Failed to submit order", zap.Error(err))
			sendErrorResponse(client, "Failed to submit order", err)
			return
		}
		
		// Send confirmation
		sendSuccessResponse(client, "Order submitted", order)
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
			sendErrorResponse(client, "Invalid order cancellation request", err)
			return
		}
		
		logger.Info("Order cancellation request",
			zap.String("client_id", client.ID),
			zap.String("order_id", request.OrderID))
		
		// Validate the request
		if request.OrderID == "" {
			sendErrorResponse(client, "Order ID is required", nil)
			return
		}
		
		// Cancel the order
		order, err := manager.CancelOrder(client.ID, request.OrderID)
		if err != nil {
			logger.Error("Failed to cancel order", zap.Error(err))
			sendErrorResponse(client, "Failed to cancel order", err)
			return
		}
		
		// Send confirmation
		sendSuccessResponse(client, "Order cancelled", order)
	})
	
	// Register the order status handler
	hub.RegisterMessageHandler("order.status", func(client *websocket.Client, msg *websocket.Message) {
		// Parse the order status request
		var request struct {
			OrderID string `json:"order_id"`
		}
		
		err := json.Unmarshal(msg.Data, &request)
		if err != nil {
			logger.Error("Failed to parse order status request", zap.Error(err))
			sendErrorResponse(client, "Invalid order status request", err)
			return
		}
		
		logger.Debug("Order status request",
			zap.String("client_id", client.ID),
			zap.String("order_id", request.OrderID))
		
		// Get the order
		order, err := manager.GetOrder(client.ID, request.OrderID)
		if err != nil {
			logger.Error("Failed to get order", zap.Error(err))
			sendErrorResponse(client, "Failed to get order", err)
			return
		}
		
		// Send the order status
		sendSuccessResponse(client, "Order status", order)
	})
}

// Helper functions for sending responses

// WebSocketResponse represents a WebSocket response
type WebSocketResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// sendSuccessResponse sends a success response to the client
func sendSuccessResponse(client *websocket.Client, message string, data interface{}) {
	response := WebSocketResponse{
		Status:  "success",
		Message: message,
		Data:    data,
	}
	
	// Marshal the response
	responseBytes, err := json.Marshal(response)
	if err != nil {
		// This should never happen
		return
	}
	
	// Send the response
	client.Send(responseBytes)
}

// sendErrorResponse sends an error response to the client
func sendErrorResponse(client *websocket.Client, message string, err error) {
	response := WebSocketResponse{
		Status:  "error",
		Message: message,
	}
	
	if err != nil {
		response.Error = err.Error()
	}
	
	// Marshal the response
	responseBytes, err := json.Marshal(response)
	if err != nil {
		// This should never happen
		return
	}
	
	// Send the response
	client.Send(responseBytes)
}

