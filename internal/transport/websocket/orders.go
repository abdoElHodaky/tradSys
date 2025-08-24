package websocket

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/trading/orders"
	"go.uber.org/zap"
)

// OrderHandler handles WebSocket connections for orders
type OrderHandler struct {
	hub             *Hub
	logger          *zap.Logger
	orderService    *orders.OrderService
	subscriptions   map[string]bool // Map of client ID to subscription status
	subscriptionsMu sync.RWMutex
}

// OrderMessage represents an order message
type OrderMessage struct {
	Action  string          `json:"action"`
	OrderID string          `json:"order_id,omitempty"`
	Order   json.RawMessage `json:"order,omitempty"`
}

// OrderUpdate represents an order update
type OrderUpdate struct {
	OrderID   string    `json:"order_id"`
	Symbol    string    `json:"symbol"`
	Side      string    `json:"side"`
	Type      string    `json:"type"`
	Price     float64   `json:"price,omitempty"`
	Size      float64   `json:"size"`
	Status    string    `json:"status"`
	FilledQty float64   `json:"filled_qty"`
	AvgPrice  float64   `json:"avg_price,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// NewOrderHandler creates a new order handler
func NewOrderHandler(
	hub *Hub,
	logger *zap.Logger,
	orderService *orders.OrderService,
) *OrderHandler {
	handler := &OrderHandler{
		hub:           hub,
		logger:        logger,
		orderService:  orderService,
		subscriptions: make(map[string]bool),
	}

	// Register message handlers
	hub.RegisterMessageHandler("order.subscribe", handler.handleSubscribe)
	hub.RegisterMessageHandler("order.unsubscribe", handler.handleUnsubscribe)
	hub.RegisterMessageHandler("order.submit", handler.handleSubmit)
	hub.RegisterMessageHandler("order.cancel", handler.handleCancel)
	hub.RegisterMessageHandler("order.get", handler.handleGet)
	hub.RegisterMessageHandler("order.list", handler.handleList)

	// Start the order update goroutine
	go handler.processOrderUpdates()

	return handler
}

// handleSubscribe handles an order subscription request
func (h *OrderHandler) handleSubscribe(client *Client, msg *Message) {
	h.logger.Debug("Order subscription request", zap.String("client_id", client.ID))

	// Subscribe the client
	h.subscriptionsMu.Lock()
	h.subscriptions[client.ID] = true
	h.subscriptionsMu.Unlock()

	// Send confirmation
	response := Message{
		Type: "order.subscribed",
		Data: json.RawMessage(`{"status":"success"}`),
	}
	client.SendMessage(&response)
}

// handleUnsubscribe handles an order unsubscription request
func (h *OrderHandler) handleUnsubscribe(client *Client, msg *Message) {
	h.logger.Debug("Order unsubscription request", zap.String("client_id", client.ID))

	// Unsubscribe the client
	h.subscriptionsMu.Lock()
	delete(h.subscriptions, client.ID)
	h.subscriptionsMu.Unlock()

	// Send confirmation
	response := Message{
		Type: "order.unsubscribed",
		Data: json.RawMessage(`{"status":"success"}`),
	}
	client.SendMessage(&response)
}

// handleSubmit handles an order submission request
func (h *OrderHandler) handleSubmit(client *Client, msg *Message) {
	// Parse the order submission request
	var request OrderMessage
	if err := json.Unmarshal(msg.Data, &request); err != nil {
		h.logger.Error("Failed to parse order submission request",
			zap.Error(err),
			zap.String("client_id", client.ID))
		return
	}

	// Parse the order
	var orderRequest orders.OrderRequest
	if err := json.Unmarshal(request.Order, &orderRequest); err != nil {
		h.logger.Error("Failed to parse order",
			zap.Error(err),
			zap.String("client_id", client.ID))
		return
	}

	h.logger.Debug("Order submission request",
		zap.String("client_id", client.ID),
		zap.String("symbol", orderRequest.Symbol),
		zap.String("side", orderRequest.Side),
		zap.String("type", orderRequest.Type),
		zap.Float64("price", orderRequest.Price),
		zap.Float64("size", orderRequest.Size))

	// Set the client ID
	orderRequest.ClientID = client.ID

	// Submit the order
	orderResponse, err := h.orderService.SubmitOrder(orderRequest)
	if err != nil {
		h.logger.Error("Failed to submit order",
			zap.Error(err),
			zap.String("client_id", client.ID))

		// Send error response
		errorResponse := Message{
			Type: "order.error",
			Data: json.RawMessage(`{"error":"` + err.Error() + `"}`),
		}
		client.SendMessage(&errorResponse)
		return
	}

	// Create the response
	response := Message{
		Type: "order.submitted",
		Data: h.serializeOrderResponse(orderResponse),
	}
	client.SendMessage(&response)
}

// handleCancel handles an order cancellation request
func (h *OrderHandler) handleCancel(client *Client, msg *Message) {
	// Parse the order cancellation request
	var request OrderMessage
	if err := json.Unmarshal(msg.Data, &request); err != nil {
		h.logger.Error("Failed to parse order cancellation request",
			zap.Error(err),
			zap.String("client_id", client.ID))
		return
	}

	h.logger.Debug("Order cancellation request",
		zap.String("client_id", client.ID),
		zap.String("order_id", request.OrderID))

	// Cancel the order
	err := h.orderService.CancelOrder(request.OrderID, client.ID)
	if err != nil {
		h.logger.Error("Failed to cancel order",
			zap.Error(err),
			zap.String("client_id", client.ID),
			zap.String("order_id", request.OrderID))

		// Send error response
		errorResponse := Message{
			Type: "order.error",
			Data: json.RawMessage(`{"error":"` + err.Error() + `"}`),
		}
		client.SendMessage(&errorResponse)
		return
	}

	// Send confirmation
	response := Message{
		Type: "order.cancelled",
		Data: json.RawMessage(`{"order_id":"` + request.OrderID + `","status":"cancelled"}`),
	}
	client.SendMessage(&response)
}

// handleGet handles an order get request
func (h *OrderHandler) handleGet(client *Client, msg *Message) {
	// Parse the order get request
	var request OrderMessage
	if err := json.Unmarshal(msg.Data, &request); err != nil {
		h.logger.Error("Failed to parse order get request",
			zap.Error(err),
			zap.String("client_id", client.ID))
		return
	}

	h.logger.Debug("Order get request",
		zap.String("client_id", client.ID),
		zap.String("order_id", request.OrderID))

	// Get the order
	orderResponse, err := h.orderService.GetOrder(request.OrderID, client.ID)
	if err != nil {
		h.logger.Error("Failed to get order",
			zap.Error(err),
			zap.String("client_id", client.ID),
			zap.String("order_id", request.OrderID))

		// Send error response
		errorResponse := Message{
			Type: "order.error",
			Data: json.RawMessage(`{"error":"` + err.Error() + `"}`),
		}
		client.SendMessage(&errorResponse)
		return
	}

	// Create the response
	response := Message{
		Type: "order.details",
		Data: h.serializeOrderResponse(orderResponse),
	}
	client.SendMessage(&response)
}

// handleList handles an order list request
func (h *OrderHandler) handleList(client *Client, msg *Message) {
	h.logger.Debug("Order list request", zap.String("client_id", client.ID))

	// Get the orders
	orderResponses, err := h.orderService.GetOrders(client.ID)
	if err != nil {
		h.logger.Error("Failed to get orders",
			zap.Error(err),
			zap.String("client_id", client.ID))

		// Send error response
		errorResponse := Message{
			Type: "order.error",
			Data: json.RawMessage(`{"error":"` + err.Error() + `"}`),
		}
		client.SendMessage(&errorResponse)
		return
	}

	// Create the response
	response := Message{
		Type: "order.list",
		Data: h.serializeOrderResponses(orderResponses),
	}
	client.SendMessage(&response)
}

// processOrderUpdates processes order updates
func (h *OrderHandler) processOrderUpdates() {
	// Subscribe to order updates
	updates := h.orderService.GetUpdateChannel()

	for update := range updates {
		// Get the client ID from the order
		clientID := update.ClientID

		// Check if the client is subscribed
		if h.isClientSubscribed(clientID) {
			// Create the update
			orderUpdate := OrderUpdate{
				OrderID:   update.OrderID,
				Symbol:    update.Symbol,
				Side:      update.Side,
				Type:      update.Type,
				Price:     update.Price,
				Size:      update.Size,
				Status:    update.Status,
				FilledQty: update.FilledQty,
				AvgPrice:  update.AvgPrice,
				Timestamp: update.Timestamp,
			}

			// Send the update to the client
			message := Message{
				Type: "order.update",
				Data: h.serializeOrderUpdate(orderUpdate),
			}
			h.hub.SendToClient(clientID, &message)
		}
	}
}

// isClientSubscribed checks if a client is subscribed
func (h *OrderHandler) isClientSubscribed(clientID string) bool {
	h.subscriptionsMu.RLock()
	defer h.subscriptionsMu.RUnlock()
	return h.subscriptions[clientID]
}

// serializeOrderResponse serializes an order response
func (h *OrderHandler) serializeOrderResponse(order *orders.OrderResponse) json.RawMessage {
	data, err := json.Marshal(order)
	if err != nil {
		h.logger.Error("Failed to serialize order response", zap.Error(err))
		return json.RawMessage("{}")
	}

	return data
}

// serializeOrderResponses serializes multiple order responses
func (h *OrderHandler) serializeOrderResponses(orders []*orders.OrderResponse) json.RawMessage {
	data, err := json.Marshal(orders)
	if err != nil {
		h.logger.Error("Failed to serialize order responses", zap.Error(err))
		return json.RawMessage("[]")
	}

	return data
}

// serializeOrderUpdate serializes an order update
func (h *OrderHandler) serializeOrderUpdate(update OrderUpdate) json.RawMessage {
	data, err := json.Marshal(update)
	if err != nil {
		h.logger.Error("Failed to serialize order update", zap.Error(err))
		return json.RawMessage("{}")
	}

	return data
}

// GetSubscribedClients gets the subscribed clients
func (h *OrderHandler) GetSubscribedClients() []string {
	h.subscriptionsMu.RLock()
	defer h.subscriptionsMu.RUnlock()

	clients := make([]string, 0, len(h.subscriptions))
	for clientID := range h.subscriptions {
		clients = append(clients, clientID)
	}

	return clients
}

