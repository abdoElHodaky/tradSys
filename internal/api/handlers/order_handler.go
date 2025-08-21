package handlers

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/abdoElHodaky/tradSys/internal/orders"
	"github.com/abdoElHodaky/tradSys/proto/orders"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// OrderHandler handles order-related API endpoints
type OrderHandler struct {
	service orders.OrderService
	logger  *zap.Logger
}

// NewOrderHandler creates a new order handler
func NewOrderHandler(service orders.OrderService, logger *zap.Logger) *OrderHandler {
	return &OrderHandler{
		service: service,
		logger:  logger,
	}
}

// RegisterRoutes registers the order API routes
func (h *OrderHandler) RegisterRoutes(router *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	ordersGroup := router.Group("/orders")
	ordersGroup.Use(authMiddleware) // Require authentication for all order endpoints
	{
		ordersGroup.GET("", h.GetOrders)
		ordersGroup.POST("", h.CreateOrder)
		ordersGroup.GET("/:id", h.GetOrder)
		ordersGroup.DELETE("/:id", h.CancelOrder)
	}
}

// CreateOrderRequest represents a request to create an order
type CreateOrderRequest struct {
	Symbol        string  `json:"symbol" binding:"required"`
	Type          string  `json:"type" binding:"required,oneof=market limit stop stop_limit"`
	Side          string  `json:"side" binding:"required,oneof=buy sell"`
	Quantity      float64 `json:"quantity" binding:"required,gt=0"`
	Price         float64 `json:"price"`
	StopPrice     float64 `json:"stop_price"`
	ClientOrderID string  `json:"client_order_id"`
}

// OrderResponse represents an order response
type OrderResponse struct {
	OrderID        string  `json:"order_id"`
	Symbol         string  `json:"symbol"`
	Type           string  `json:"type"`
	Side           string  `json:"side"`
	Status         string  `json:"status"`
	Quantity       float64 `json:"quantity"`
	FilledQuantity float64 `json:"filled_quantity"`
	Price          float64 `json:"price"`
	StopPrice      float64 `json:"stop_price"`
	CreatedAt      int64   `json:"created_at"`
	UpdatedAt      int64   `json:"updated_at"`
	ClientOrderID  string  `json:"client_order_id"`
}

// CreateOrder creates a new order
func (h *OrderHandler) CreateOrder(c *gin.Context) {
	var req CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Map request to order types
	var orderType orders.OrderType
	switch req.Type {
	case "market":
		orderType = orders.OrderType_MARKET
	case "limit":
		orderType = orders.OrderType_LIMIT
	case "stop":
		orderType = orders.OrderType_STOP
	case "stop_limit":
		orderType = orders.OrderType_STOP_LIMIT
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order type"})
		return
	}

	var orderSide orders.OrderSide
	switch req.Side {
	case "buy":
		orderSide = orders.OrderSide_BUY
	case "sell":
		orderSide = orders.OrderSide_SELL
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order side"})
		return
	}

	// Validate price for limit orders
	if orderType == orders.OrderType_LIMIT && req.Price <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Price must be greater than 0 for limit orders"})
		return
	}

	// Validate stop price for stop orders
	if (orderType == orders.OrderType_STOP || orderType == orders.OrderType_STOP_LIMIT) && req.StopPrice <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Stop price must be greater than 0 for stop orders"})
		return
	}

	// Create a context with user ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	
	// Add user ID to context
	ctx := context.WithValue(c.Request.Context(), "user_id", userID.(string))
	
	// Add client IP to context for audit purposes
	ctx = context.WithValue(ctx, "client_ip", c.ClientIP())

	// Create order
	order, err := h.service.CreateOrder(
		ctx,
		req.Symbol,
		orderType,
		orderSide,
		req.Quantity,
		req.Price,
		req.StopPrice,
		req.ClientOrderID,
	)
	if err != nil {
		h.logger.Error("Failed to create order", zap.Error(err))
		
		// Return appropriate error message based on the error
		if err.Error() == "client order ID already exists" {
			c.JSON(http.StatusConflict, gin.H{"error": "Client order ID already exists"})
		} else if err.Error() == "symbol is required" || 
			err.Error() == "quantity must be greater than 0" ||
			err.Error() == "price must be greater than 0 for limit orders" ||
			err.Error() == "stop price must be greater than 0 for stop orders" {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create order"})
		}
		return
	}

	// Map response
	response := mapOrderResponse(order)
	c.JSON(http.StatusCreated, response)
}

// GetOrder gets an order by ID
func (h *OrderHandler) GetOrder(c *gin.Context) {
	orderID := c.Param("id")
	if orderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order ID is required"})
		return
	}

	// Create a context with user ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	
	// Add user ID to context
	ctx := context.WithValue(c.Request.Context(), "user_id", userID.(string))
	
	// Add client IP to context for audit purposes
	ctx = context.WithValue(ctx, "client_ip", c.ClientIP())

	order, err := h.service.GetOrder(ctx, orderID)
	if err != nil {
		h.logger.Error("Failed to get order", zap.Error(err), zap.String("order_id", orderID))
		
		// Don't expose internal errors to the client
		if err.Error() == "order not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get order"})
		}
		return
	}

	response := mapOrderResponse(order)
	c.JSON(http.StatusOK, response)
}

// CancelOrder cancels an order
func (h *OrderHandler) CancelOrder(c *gin.Context) {
	orderID := c.Param("id")
	if orderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order ID is required"})
		return
	}

	// Create a context with user ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	
	// Add user ID to context
	ctx := context.WithValue(c.Request.Context(), "user_id", userID.(string))
	
	// Add client IP to context for audit purposes
	ctx = context.WithValue(ctx, "client_ip", c.ClientIP())

	order, err := h.service.CancelOrder(ctx, orderID)
	if err != nil {
		h.logger.Error("Failed to cancel order", zap.Error(err), zap.String("order_id", orderID))
		
		// Return appropriate error message based on the error
		if err.Error() == "order not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		} else if strings.HasPrefix(err.Error(), "order cannot be canceled: status is") {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to cancel order"})
		}
		return
	}

	response := mapOrderResponse(order)
	c.JSON(http.StatusOK, response)
}

// GetOrders gets all orders with filtering
func (h *OrderHandler) GetOrders(c *gin.Context) {
	// Parse query parameters
	symbol := c.Query("symbol")
	
	var status orders.OrderStatus
	statusStr := c.Query("status")
	switch statusStr {
	case "pending":
		status = orders.OrderStatus_PENDING
	case "open", "new", "partially_filled":
		status = orders.OrderStatus_OPEN
	case "filled":
		status = orders.OrderStatus_FILLED
	case "canceled":
		status = orders.OrderStatus_CANCELLED
	case "rejected":
		status = orders.OrderStatus_REJECTED
	default:
		status = orders.OrderStatus_UNKNOWN
	}

	startTime, _ := strconv.ParseInt(c.Query("start_time"), 10, 64)
	endTime, _ := strconv.ParseInt(c.Query("end_time"), 10, 64)
	limit, _ := strconv.ParseInt(c.Query("limit"), 10, 32)
	if limit <= 0 {
		limit = 100
	}

	// Create a context with user ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	
	// Add user ID to context
	ctx := context.WithValue(c.Request.Context(), "user_id", userID.(string))
	
	// Add client IP to context for audit purposes
	ctx = context.WithValue(ctx, "client_ip", c.ClientIP())

	orderList, err := h.service.GetOrders(
		ctx,
		symbol,
		status,
		startTime,
		endTime,
		int32(limit),
	)
	if err != nil {
		h.logger.Error("Failed to get orders", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get orders"})
		return
	}

	// Map response
	var response []OrderResponse
	for _, order := range orderList {
		response = append(response, mapOrderResponse(order))
	}

	// Return empty array instead of null if no orders found
	if response == nil {
		response = []OrderResponse{}
	}

	c.JSON(http.StatusOK, response)
}

// mapOrderResponse maps a proto order to an API response
func mapOrderResponse(order *orders.OrderResponse) OrderResponse {
	var orderType string
	switch order.Type {
	case orders.OrderType_MARKET:
		orderType = "market"
	case orders.OrderType_LIMIT:
		orderType = "limit"
	case orders.OrderType_STOP:
		orderType = "stop"
	case orders.OrderType_STOP_LIMIT:
		orderType = "stop_limit"
	}

	var orderSide string
	switch order.Side {
	case orders.OrderSide_BUY:
		orderSide = "buy"
	case orders.OrderSide_SELL:
		orderSide = "sell"
	}

	var orderStatus string
	switch order.Status {
	case orders.OrderStatus_PENDING:
		orderStatus = "pending"
	case orders.OrderStatus_OPEN:
		orderStatus = "open"
	case orders.OrderStatus_FILLED:
		orderStatus = "filled"
	case orders.OrderStatus_CANCELED:
		orderStatus = "canceled"
	case orders.OrderStatus_REJECTED:
		orderStatus = "rejected"
	}

	return OrderResponse{
		OrderID:        order.OrderId,
		Symbol:         order.Symbol,
		Type:           orderType,
		Side:           orderSide,
		Status:         orderStatus,
		Quantity:       order.Quantity,
		FilledQuantity: order.FilledQuantity,
		Price:          order.Price,
		StopPrice:      order.StopPrice,
		CreatedAt:      order.CreatedAt,
		UpdatedAt:      order.UpdatedAt,
		ClientOrderID:  order.ClientOrderId,
	}
}
