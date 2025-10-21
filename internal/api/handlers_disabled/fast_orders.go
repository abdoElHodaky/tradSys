package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/abdoElHodaky/tradSys/internal/db/models"
	"github.com/abdoElHodaky/tradSys/internal/trading/metrics"
	"github.com/abdoElHodaky/tradSys/internal/trading/pools"
)

// FastOrderHandler provides high-performance order handling with minimal allocations
type FastOrderHandler struct {
	// Dependencies would be injected here
	// orderService OrderService
	// userService  UserService
}

// NewFastOrderHandler creates a new fast order handler
func NewFastOrderHandler() *FastOrderHandler {
	return &FastOrderHandler{}
}

// FastCreateOrder handles order creation with zero-allocation JSON processing
func (h *FastOrderHandler) FastCreateOrder(c *gin.Context) {
	// Start latency tracking
	tracker := metrics.TrackOrderLatency()
	defer tracker.Finish()
	
	// Get pooled objects
	req := pools.GetOrderRequestFromPool()
	defer pools.PutOrderRequestToPool(req)
	
	resp := pools.GetOrderResponseFromPool()
	defer pools.PutOrderResponseToPool(resp)
	
	order := pools.GetOrderFromPool()
	defer pools.PutOrderToPool(order)
	
	// Bind JSON request with minimal allocations
	if err := c.ShouldBindJSON(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}
	
	// Validate request
	if err := h.validateOrderRequest(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "validation_failed",
			"message": err.Error(),
		})
		return
	}
	
	// Convert request to order model
	h.populateOrderFromRequest(order, req, c)
	
	// Process order (this would call actual order service)
	if err := h.processOrderFast(order); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "processing_failed",
			"message": err.Error(),
		})
		return
	}
	
	// Populate response from order
	resp.FromOrder(order)
	resp.ProcessingTime = time.Since(tracker.startTime).Nanoseconds()
	
	// Return response
	c.JSON(http.StatusCreated, resp)
}

// FastGetOrder retrieves an order with minimal allocations
func (h *FastOrderHandler) FastGetOrder(c *gin.Context) {
	tracker := metrics.TrackOrderLatency()
	defer tracker.Finish()
	
	orderID := c.Param("id")
	if orderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "missing_order_id",
			"message": "Order ID is required",
		})
		return
	}
	
	// Get pooled objects
	order := pools.GetOrderFromPool()
	defer pools.PutOrderToPool(order)
	
	resp := pools.GetOrderResponseFromPool()
	defer pools.PutOrderResponseToPool(resp)
	
	// Retrieve order (this would call actual order service)
	if err := h.getOrderFast(orderID, order); err != nil {
		if err.Error() == "order_not_found" {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "order_not_found",
				"message": "Order not found",
			})
			return
		}
		
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "retrieval_failed",
			"message": err.Error(),
		})
		return
	}
	
	// Populate response
	resp.FromOrder(order)
	resp.ProcessingTime = time.Since(tracker.startTime).Nanoseconds()
	
	c.JSON(http.StatusOK, resp)
}

// FastUpdateOrder updates an order with minimal allocations
func (h *FastOrderHandler) FastUpdateOrder(c *gin.Context) {
	tracker := metrics.TrackOrderLatency()
	defer tracker.Finish()
	
	orderID := c.Param("id")
	if orderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "missing_order_id",
			"message": "Order ID is required",
		})
		return
	}
	
	// Get pooled objects
	req := pools.GetOrderRequestFromPool()
	defer pools.PutOrderRequestToPool(req)
	
	order := pools.GetOrderFromPool()
	defer pools.PutOrderToPool(order)
	
	resp := pools.GetOrderResponseFromPool()
	defer pools.PutOrderResponseToPool(resp)
	
	// Bind JSON request
	if err := c.ShouldBindJSON(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}
	
	// Get existing order
	if err := h.getOrderFast(orderID, order); err != nil {
		if err.Error() == "order_not_found" {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "order_not_found",
				"message": "Order not found",
			})
			return
		}
		
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "retrieval_failed",
			"message": err.Error(),
		})
		return
	}
	
	// Update order fields
	h.updateOrderFromRequest(order, req)
	
	// Process update
	if err := h.updateOrderFast(order); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "update_failed",
			"message": err.Error(),
		})
		return
	}
	
	// Populate response
	resp.FromOrder(order)
	resp.ProcessingTime = time.Since(tracker.startTime).Nanoseconds()
	
	c.JSON(http.StatusOK, resp)
}

// FastCancelOrder cancels an order with minimal allocations
func (h *FastOrderHandler) FastCancelOrder(c *gin.Context) {
	tracker := metrics.TrackOrderLatency()
	defer tracker.Finish()
	
	orderID := c.Param("id")
	if orderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "missing_order_id",
			"message": "Order ID is required",
		})
		return
	}
	
	// Get pooled objects
	order := pools.GetOrderFromPool()
	defer pools.PutOrderToPool(order)
	
	resp := pools.GetOrderResponseFromPool()
	defer pools.PutOrderResponseToPool(resp)
	
	// Get existing order
	if err := h.getOrderFast(orderID, order); err != nil {
		if err.Error() == "order_not_found" {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "order_not_found",
				"message": "Order not found",
			})
			return
		}
		
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "retrieval_failed",
			"message": err.Error(),
		})
		return
	}
	
	// Cancel order
	if err := h.cancelOrderFast(order); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "cancellation_failed",
			"message": err.Error(),
		})
		return
	}
	
	// Populate response
	resp.FromOrder(order)
	resp.ProcessingTime = time.Since(tracker.startTime).Nanoseconds()
	
	c.JSON(http.StatusOK, resp)
}

// FastListOrders lists orders for a user with minimal allocations
func (h *FastOrderHandler) FastListOrders(c *gin.Context) {
	tracker := metrics.TrackOrderLatency()
	defer tracker.Finish()
	
	userID := c.GetString("user_id") // From auth middleware
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "User ID not found in context",
		})
		return
	}
	
	// Parse query parameters
	limitStr := c.DefaultQuery("limit", "50")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 1000 {
		limit = 50
	}
	
	// Get orders (this would call actual order service)
	orders, err := h.listOrdersFast(userID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "listing_failed",
			"message": err.Error(),
		})
		return
	}
	
	// Convert to response format using pools
	responses := make([]*pools.OrderResponse, 0, len(orders))
	for _, order := range orders {
		resp := pools.GetOrderResponseFromPool()
		resp.FromOrder(order)
		responses = append(responses, resp)
	}
	
	// Clean up pooled responses after sending
	defer func() {
		for _, resp := range responses {
			pools.PutOrderResponseToPool(resp)
		}
	}()
	
	processingTime := time.Since(tracker.startTime).Nanoseconds()
	
	c.JSON(http.StatusOK, gin.H{
		"orders":          responses,
		"count":           len(responses),
		"processing_time": processingTime,
	})
}

// Helper methods (these would integrate with actual services)

func (h *FastOrderHandler) validateOrderRequest(req *pools.OrderRequest) error {
	// Implement validation logic
	if req.Symbol == "" {
		return fmt.Errorf("symbol is required")
	}
	if req.Side != "buy" && req.Side != "sell" {
		return fmt.Errorf("side must be 'buy' or 'sell'")
	}
	if req.Type != "market" && req.Type != "limit" && req.Type != "stop" {
		return fmt.Errorf("type must be 'market', 'limit', or 'stop'")
	}
	if req.Quantity <= 0 {
		return fmt.Errorf("quantity must be greater than 0")
	}
	if req.Type == "limit" && req.Price <= 0 {
		return fmt.Errorf("price is required for limit orders")
	}
	if req.Type == "stop" && req.StopPrice <= 0 {
		return fmt.Errorf("stop_price is required for stop orders")
	}
	return nil
}

func (h *FastOrderHandler) populateOrderFromRequest(order *models.Order, req *pools.OrderRequest, c *gin.Context) {
	order.ID = uuid.New().String()
	order.UserID = c.GetString("user_id")
	order.Symbol = req.Symbol
	order.Side = req.Side
	order.Type = req.Type
	order.Quantity = req.Quantity
	order.Price = req.Price
	order.StopPrice = req.StopPrice
	order.Status = "pending"
	order.CreatedAt = time.Now()
	order.UpdatedAt = time.Now()
}

func (h *FastOrderHandler) updateOrderFromRequest(order *models.Order, req *pools.OrderRequest) {
	// Update only allowed fields
	if req.Quantity > 0 {
		order.Quantity = req.Quantity
	}
	if req.Price > 0 {
		order.Price = req.Price
	}
	if req.StopPrice > 0 {
		order.StopPrice = req.StopPrice
	}
	order.UpdatedAt = time.Now()
}

// Placeholder methods for actual service integration
func (h *FastOrderHandler) processOrderFast(order *models.Order) error {
	// This would integrate with the actual order processing service
	// For now, just simulate processing
	time.Sleep(time.Microsecond * 10) // Simulate 10μs processing time
	order.Status = "filled"
	order.FilledQuantity = order.Quantity
	order.AveragePrice = order.Price
	now := time.Now()
	order.ExecutedAt = &now
	return nil
}

func (h *FastOrderHandler) getOrderFast(orderID string, order *models.Order) error {
	// This would integrate with the actual order retrieval service
	// For now, just simulate retrieval
	if orderID == "not-found" {
		return fmt.Errorf("order_not_found")
	}
	
	// Populate with dummy data
	order.ID = orderID
	order.UserID = "user123"
	order.Symbol = "BTCUSD"
	order.Side = "buy"
	order.Type = "limit"
	order.Quantity = 1.0
	order.Price = 50000.0
	order.Status = "filled"
	order.FilledQuantity = 1.0
	order.AveragePrice = 50000.0
	order.CreatedAt = time.Now().Add(-time.Hour)
	order.UpdatedAt = time.Now()
	
	return nil
}

func (h *FastOrderHandler) updateOrderFast(order *models.Order) error {
	// This would integrate with the actual order update service
	order.UpdatedAt = time.Now()
	return nil
}

func (h *FastOrderHandler) cancelOrderFast(order *models.Order) error {
	// This would integrate with the actual order cancellation service
	order.Status = "cancelled"
	order.UpdatedAt = time.Now()
	return nil
}

func (h *FastOrderHandler) listOrdersFast(userID string, limit int) ([]*models.Order, error) {
	// This would integrate with the actual order listing service
	// For now, return dummy data
	orders := make([]*models.Order, 0, limit)
	
	for i := 0; i < min(limit, 10); i++ {
		order := &models.Order{
			ID:              fmt.Sprintf("order-%d", i),
			UserID:          userID,
			Symbol:          "BTCUSD",
			Side:            "buy",
			Type:            "limit",
			Quantity:        1.0,
			Price:           50000.0 + float64(i*100),
			Status:          "filled",
			FilledQuantity:  1.0,
			AveragePrice:    50000.0 + float64(i*100),
			CreatedAt:       time.Now().Add(-time.Duration(i) * time.Hour),
			UpdatedAt:       time.Now().Add(-time.Duration(i) * time.Hour),
		}
		orders = append(orders, order)
	}
	
	return orders, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
