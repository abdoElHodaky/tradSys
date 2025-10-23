package services

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Handlers struct contains all service handlers
type Handlers struct {
	OrderService      OrderService
	SettlementService SettlementService
	PairsService      PairsService
	RiskService       RiskService
	StrategyService   StrategyService
}

// NewHandlers creates a new handlers instance
func NewHandlers(
	orderService OrderService,
	settlementService SettlementService,
	pairsService PairsService,
	riskService RiskService,
	strategyService StrategyService,
) *Handlers {
	return &Handlers{
		OrderService:      orderService,
		SettlementService: settlementService,
		PairsService:      pairsService,
		RiskService:       riskService,
		StrategyService:   strategyService,
	}
}

// OrderHandler handles order-related HTTP requests
type OrderHandler struct {
	service OrderService
}

// NewOrderHandler creates a new order handler
func NewOrderHandler(service OrderService) *OrderHandler {
	return &OrderHandler{service: service}
}

// CreateOrder handles POST /orders
func (h *OrderHandler) CreateOrder(c *gin.Context) {
	var order Order
	if err := c.ShouldBindJSON(&order); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.service.CreateOrder(c.Request.Context(), &order)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, result)
}

// GetOrder handles GET /orders/:id
func (h *OrderHandler) GetOrder(c *gin.Context) {
	id := c.Param("id")

	order, err := h.service.GetOrder(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, order)
}

// ListOrders handles GET /orders
func (h *OrderHandler) ListOrders(c *gin.Context) {
	filter := &OrderFilter{}

	// Parse query parameters
	if accountID := c.Query("account_id"); accountID != "" {
		filter.AccountID = &accountID
	}
	if symbol := c.Query("symbol"); symbol != "" {
		filter.Symbol = &symbol
	}
	if side := c.Query("side"); side != "" {
		filter.Side = &side
	}
	if status := c.Query("status"); status != "" {
		filter.Status = &status
	}
	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			filter.Limit = limit
		}
	}
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil {
			filter.Offset = offset
		}
	}

	orders, err := h.service.ListOrders(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, orders)
}

// CancelOrder handles DELETE /orders/:id
func (h *OrderHandler) CancelOrder(c *gin.Context) {
	id := c.Param("id")

	err := h.service.CancelOrder(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Order cancelled successfully"})
}

// PairsHandler handles pairs-related HTTP requests
type PairsHandler struct {
	service PairsService
}

// NewPairsHandler creates a new pairs handler
func NewPairsHandler(service PairsService) *PairsHandler {
	return &PairsHandler{service: service}
}

// GetPair handles GET /pairs/:symbol
func (h *PairsHandler) GetPair(c *gin.Context) {
	symbol := c.Param("symbol")

	pair, err := h.service.GetPair(c.Request.Context(), symbol)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, pair)
}

// ListPairs handles GET /pairs
func (h *PairsHandler) ListPairs(c *gin.Context) {
	filter := &PairFilter{}

	// Parse query parameters
	if baseAsset := c.Query("base_asset"); baseAsset != "" {
		filter.BaseAsset = &baseAsset
	}
	if quoteAsset := c.Query("quote_asset"); quoteAsset != "" {
		filter.QuoteAsset = &quoteAsset
	}
	if status := c.Query("status"); status != "" {
		filter.Status = &status
	}
	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			filter.Limit = limit
		}
	}
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil {
			filter.Offset = offset
		}
	}

	pairs, err := h.service.ListPairs(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, pairs)
}

// GetTicker handles GET /pairs/:symbol/ticker
func (h *PairsHandler) GetTicker(c *gin.Context) {
	symbol := c.Param("symbol")

	ticker, err := h.service.GetTicker(c.Request.Context(), symbol)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, ticker)
}

// GetOrderBook handles GET /pairs/:symbol/orderbook
func (h *PairsHandler) GetOrderBook(c *gin.Context) {
	symbol := c.Param("symbol")
	depth := 10 // default depth

	if depthStr := c.Query("depth"); depthStr != "" {
		if d, err := strconv.Atoi(depthStr); err == nil && d > 0 {
			depth = d
		}
	}

	orderBook, err := h.service.GetOrderBook(c.Request.Context(), symbol, depth)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, orderBook)
}

// SettlementHandler handles settlement-related HTTP requests
type SettlementHandler struct {
	service SettlementService
}

// NewSettlementHandler creates a new settlement handler
func NewSettlementHandler(service SettlementService) *SettlementHandler {
	return &SettlementHandler{service: service}
}

// GetSettlement handles GET /settlements/:id
func (h *SettlementHandler) GetSettlement(c *gin.Context) {
	id := c.Param("id")

	settlement, err := h.service.GetSettlement(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, settlement)
}

// ListSettlements handles GET /settlements
func (h *SettlementHandler) ListSettlements(c *gin.Context) {
	filter := &SettlementFilter{}

	// Parse query parameters
	if tradeID := c.Query("trade_id"); tradeID != "" {
		filter.TradeID = &tradeID
	}
	if status := c.Query("status"); status != "" {
		filter.Status = &status
	}
	if currency := c.Query("currency"); currency != "" {
		filter.Currency = &currency
	}
	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			filter.Limit = limit
		}
	}
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil {
			filter.Offset = offset
		}
	}

	settlements, err := h.service.ListSettlements(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, settlements)
}

// ProcessSettlement handles POST /settlements
func (h *SettlementHandler) ProcessSettlement(c *gin.Context) {
	var trade Trade
	if err := c.ShouldBindJSON(&trade); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	settlement, err := h.service.ProcessSettlement(c.Request.Context(), &trade)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, settlement)
}

// StrategyHandler handles strategy-related HTTP requests
type StrategyHandler struct {
	service StrategyService
}

// NewStrategyHandler creates a new strategy handler
func NewStrategyHandler(service StrategyService) *StrategyHandler {
	return &StrategyHandler{service: service}
}

// CreateStrategy handles POST /strategies
func (h *StrategyHandler) CreateStrategy(c *gin.Context) {
	var strategy Strategy
	if err := c.ShouldBindJSON(&strategy); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.service.CreateStrategy(c.Request.Context(), &strategy)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, result)
}

// GetStrategy handles GET /strategies/:id
func (h *StrategyHandler) GetStrategy(c *gin.Context) {
	id := c.Param("id")

	strategy, err := h.service.GetStrategy(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, strategy)
}

// ListStrategies handles GET /strategies
func (h *StrategyHandler) ListStrategies(c *gin.Context) {
	filter := &StrategyFilter{}

	// Parse query parameters
	if strategyType := c.Query("type"); strategyType != "" {
		filter.Type = &strategyType
	}
	if status := c.Query("status"); status != "" {
		filter.Status = &status
	}
	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			filter.Limit = limit
		}
	}
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil {
			filter.Offset = offset
		}
	}

	strategies, err := h.service.ListStrategies(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, strategies)
}

// StartStrategy handles POST /strategies/:id/start
func (h *StrategyHandler) StartStrategy(c *gin.Context) {
	id := c.Param("id")

	err := h.service.StartStrategy(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Strategy started successfully"})
}

// StopStrategy handles POST /strategies/:id/stop
func (h *StrategyHandler) StopStrategy(c *gin.Context) {
	id := c.Param("id")

	err := h.service.StopStrategy(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Strategy stopped successfully"})
}
