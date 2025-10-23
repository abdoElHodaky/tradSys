package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/services"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ETFHandlers handles ETF-specific API endpoints
type ETFHandlers struct {
	etfService *services.ETFService
	logger     *zap.Logger
}

// NewETFHandlers creates a new ETF handlers instance
func NewETFHandlers(etfService *services.ETFService, logger *zap.Logger) *ETFHandlers {
	return &ETFHandlers{
		etfService: etfService,
		logger:     logger,
	}
}

// CreateETFRequest represents the request body for creating an ETF
type CreateETFRequest struct {
	Symbol           string  `json:"symbol" binding:"required"`
	BenchmarkIndex   string  `json:"benchmark_index" binding:"required"`
	CreationUnitSize int     `json:"creation_unit_size" binding:"required"`
	ExpenseRatio     float64 `json:"expense_ratio" binding:"required"`
}

// UpdateETFMetricsRequest represents the request body for updating ETF metrics
type UpdateETFMetricsRequest struct {
	NAV           float64 `json:"nav" binding:"required"`
	TrackingError float64 `json:"tracking_error"`
	AUM           float64 `json:"aum"`
	DividendYield float64 `json:"dividend_yield"`
}

// CreationRedemptionRequest represents the request body for creation/redemption operations
type CreationRedemptionRequest struct {
	OperationType         string  `json:"operation_type" binding:"required"`
	Units                 int     `json:"units" binding:"required"`
	SharesPerUnit         int     `json:"shares_per_unit" binding:"required"`
	NAVPerShare           float64 `json:"nav_per_share" binding:"required"`
	AuthorizedParticipant string  `json:"authorized_participant" binding:"required"`
}

// CreateETF creates a new ETF
// @Summary Create ETF
// @Description Create a new ETF with initial metadata
// @Tags ETF
// @Accept json
// @Produce json
// @Param request body CreateETFRequest true "ETF creation request"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/etfs [post]
func (h *ETFHandlers) CreateETF(c *gin.Context) {
	var req CreateETFRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid ETF creation request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.etfService.CreateETF(req.Symbol, req.BenchmarkIndex, req.CreationUnitSize, req.ExpenseRatio)
	if err != nil {
		h.logger.Error("Failed to create ETF", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.logger.Info("ETF created successfully", zap.String("symbol", req.Symbol))
	c.JSON(http.StatusCreated, gin.H{
		"message": "ETF created successfully",
		"symbol":  req.Symbol,
	})
}

// GetETFMetrics retrieves comprehensive ETF metrics
// @Summary Get ETF metrics
// @Description Get comprehensive metrics for an ETF
// @Tags ETF
// @Produce json
// @Param symbol path string true "ETF symbol"
// @Success 200 {object} services.ETFMetrics
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/etfs/{symbol}/metrics [get]
func (h *ETFHandlers) GetETFMetrics(c *gin.Context) {
	symbol := c.Param("symbol")
	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Symbol is required"})
		return
	}

	metrics, err := h.etfService.GetETFMetrics(symbol)
	if err != nil {
		h.logger.Error("Failed to get ETF metrics", zap.String("symbol", symbol), zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, metrics)
}

// UpdateETFMetrics updates ETF-specific metrics
// @Summary Update ETF metrics
// @Description Update ETF-specific performance metrics
// @Tags ETF
// @Accept json
// @Produce json
// @Param symbol path string true "ETF symbol"
// @Param request body UpdateETFMetricsRequest true "ETF metrics update request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/etfs/{symbol}/metrics [post]
func (h *ETFHandlers) UpdateETFMetrics(c *gin.Context) {
	symbol := c.Param("symbol")
	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Symbol is required"})
		return
	}

	var req UpdateETFMetricsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid ETF metrics update request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.etfService.UpdateETFMetrics(symbol, req.NAV, req.TrackingError, req.AUM, req.DividendYield)
	if err != nil {
		h.logger.Error("Failed to update ETF metrics", zap.String("symbol", symbol), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.logger.Info("ETF metrics updated successfully", zap.String("symbol", symbol))
	c.JSON(http.StatusOK, gin.H{
		"message": "ETF metrics updated successfully",
		"symbol":  symbol,
	})
}

// GetTrackingError calculates and returns the tracking error for an ETF
// @Summary Get tracking error
// @Description Calculate tracking error for an ETF over specified period
// @Tags ETF
// @Produce json
// @Param symbol path string true "ETF symbol"
// @Param days query int false "Number of days for calculation" default(30)
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/etfs/{symbol}/tracking-error [get]
func (h *ETFHandlers) GetTrackingError(c *gin.Context) {
	symbol := c.Param("symbol")
	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Symbol is required"})
		return
	}

	daysStr := c.DefaultQuery("days", "30")
	days, err := strconv.Atoi(daysStr)
	if err != nil || days <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid days parameter"})
		return
	}

	trackingError, err := h.etfService.GetTrackingError(symbol, days)
	if err != nil {
		h.logger.Error("Failed to calculate tracking error", zap.String("symbol", symbol), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"symbol":         symbol,
		"tracking_error": trackingError,
		"period_days":    days,
		"calculated_at":  time.Now(),
	})
}

// ProcessCreationRedemption handles ETF creation/redemption operations
// @Summary Process creation/redemption
// @Description Process ETF creation or redemption operation
// @Tags ETF
// @Accept json
// @Produce json
// @Param symbol path string true "ETF symbol"
// @Param request body CreationRedemptionRequest true "Creation/redemption request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/etfs/{symbol}/creation-redemption [post]
func (h *ETFHandlers) ProcessCreationRedemption(c *gin.Context) {
	symbol := c.Param("symbol")
	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Symbol is required"})
		return
	}

	var req CreationRedemptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid creation/redemption request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate operation type
	if req.OperationType != "creation" && req.OperationType != "redemption" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Operation type must be 'creation' or 'redemption'"})
		return
	}

	// Create operation object
	operation := &services.CreationRedemptionOperation{
		Symbol:                symbol,
		OperationType:         req.OperationType,
		Units:                 req.Units,
		SharesPerUnit:         req.SharesPerUnit,
		TotalShares:           req.Units * req.SharesPerUnit,
		NAVPerShare:           req.NAVPerShare,
		TotalValue:            float64(req.Units*req.SharesPerUnit) * req.NAVPerShare,
		AuthorizedParticipant: req.AuthorizedParticipant,
		Timestamp:             time.Now(),
		Status:                "processed",
	}

	err := h.etfService.ProcessCreationRedemption(operation)
	if err != nil {
		h.logger.Error("Failed to process creation/redemption", 
			zap.String("symbol", symbol), 
			zap.String("operation", req.OperationType), 
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.logger.Info("Creation/redemption processed successfully", 
		zap.String("symbol", symbol), 
		zap.String("operation", req.OperationType),
		zap.Int("units", req.Units))

	c.JSON(http.StatusOK, gin.H{
		"message":       "Operation processed successfully",
		"symbol":        symbol,
		"operation":     req.OperationType,
		"units":         req.Units,
		"total_shares":  operation.TotalShares,
		"total_value":   operation.TotalValue,
		"processed_at":  operation.Timestamp,
	})
}

// GetETFHoldings retrieves the current holdings composition of an ETF
// @Summary Get ETF holdings
// @Description Get current holdings composition of an ETF
// @Tags ETF
// @Produce json
// @Param symbol path string true "ETF symbol"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/etfs/{symbol}/holdings [get]
func (h *ETFHandlers) GetETFHoldings(c *gin.Context) {
	symbol := c.Param("symbol")
	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Symbol is required"})
		return
	}

	holdings, err := h.etfService.GetETFHoldings(symbol)
	if err != nil {
		h.logger.Error("Failed to get ETF holdings", zap.String("symbol", symbol), zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	// Calculate total weight
	var totalWeight float64
	for _, holding := range holdings {
		totalWeight += holding.Weight
	}

	c.JSON(http.StatusOK, gin.H{
		"symbol":        symbol,
		"holdings":      holdings,
		"total_weight":  totalWeight,
		"holdings_count": len(holdings),
		"retrieved_at":  time.Now(),
	})
}

// ValidateETFOrder validates an ETF order
// @Summary Validate ETF order
// @Description Validate an ETF order against ETF-specific rules
// @Tags ETF
// @Produce json
// @Param symbol query string true "ETF symbol"
// @Param quantity query number true "Order quantity"
// @Param price query number true "Order price"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/etfs/validate-order [get]
func (h *ETFHandlers) ValidateETFOrder(c *gin.Context) {
	symbol := c.Query("symbol")
	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Symbol is required"})
		return
	}

	quantityStr := c.Query("quantity")
	if quantityStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Quantity is required"})
		return
	}

	quantity, err := strconv.ParseFloat(quantityStr, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid quantity"})
		return
	}

	priceStr := c.Query("price")
	if priceStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Price is required"})
		return
	}

	price, err := strconv.ParseFloat(priceStr, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid price"})
		return
	}

	err = h.etfService.ValidateETFOrder(symbol, quantity, price)
	if err != nil {
		h.logger.Warn("ETF order validation failed", 
			zap.String("symbol", symbol),
			zap.Float64("quantity", quantity),
			zap.Float64("price", price),
			zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"valid":  false,
			"error":  err.Error(),
			"symbol": symbol,
		})
		return
	}

	h.logger.Debug("ETF order validation passed", 
		zap.String("symbol", symbol),
		zap.Float64("quantity", quantity),
		zap.Float64("price", price))

	c.JSON(http.StatusOK, gin.H{
		"valid":    true,
		"message":  "Order validation passed",
		"symbol":   symbol,
		"quantity": quantity,
		"price":    price,
	})
}

// GetETFLiquidity retrieves liquidity metrics for an ETF
// @Summary Get ETF liquidity
// @Description Get liquidity metrics and analysis for an ETF
// @Tags ETF
// @Produce json
// @Param symbol path string true "ETF symbol"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/etfs/{symbol}/liquidity [get]
func (h *ETFHandlers) GetETFLiquidity(c *gin.Context) {
	symbol := c.Param("symbol")
	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Symbol is required"})
		return
	}

	metrics, err := h.etfService.GetETFMetrics(symbol)
	if err != nil {
		h.logger.Error("Failed to get ETF metrics for liquidity", zap.String("symbol", symbol), zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"symbol":     symbol,
		"liquidity":  metrics.Liquidity,
		"retrieved_at": time.Now(),
	})
}

// TriggerRebalance triggers ETF rebalancing
// @Summary Trigger rebalance
// @Description Trigger ETF portfolio rebalancing
// @Tags ETF
// @Accept json
// @Produce json
// @Param symbol path string true "ETF symbol"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/etfs/{symbol}/rebalance [post]
func (h *ETFHandlers) TriggerRebalance(c *gin.Context) {
	symbol := c.Param("symbol")
	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Symbol is required"})
		return
	}

	// In a real implementation, this would trigger actual rebalancing
	h.logger.Info("ETF rebalancing triggered", zap.String("symbol", symbol))

	c.JSON(http.StatusOK, gin.H{
		"message":      "Rebalancing triggered successfully",
		"symbol":       symbol,
		"triggered_at": time.Now(),
		"status":       "initiated",
	})
}
