package handlers

import (
	"context"
	"net/http"
	"strconv"

	"github.com/abdoElHodaky/tradSys/internal/risk"
	"github.com/abdoElHodaky/tradSys/proto/risk"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// RiskHandlerParams contains the parameters for creating a risk handler
type RiskHandlerParams struct {
	fx.In

	Logger  *zap.Logger
	Service risk.RiskService
}

// RiskHandler handles risk management API requests
type RiskHandler struct {
	logger  *zap.Logger
	service risk.RiskService
}

// NewRiskHandler creates a new risk handler with fx dependency injection
func NewRiskHandler(p RiskHandlerParams) *RiskHandler {
	return &RiskHandler{
		logger:  p.Logger,
		service: p.Service,
	}
}

// RegisterRoutes registers the risk management routes
func (h *RiskHandler) RegisterRoutes(router *gin.RouterGroup) {
	riskGroup := router.Group("/risk")
	{
		riskGroup.GET("/positions", h.GetPositions)
		riskGroup.GET("/limits", h.GetLimits)
		riskGroup.POST("/limits", h.SetLimit)
		riskGroup.DELETE("/limits/:id", h.DeleteLimit)
		riskGroup.POST("/validate", h.ValidateOrder)
	}
}

// PositionResponse represents a position response
type PositionResponse struct {
	Symbol            string  `json:"symbol"`
	Quantity          float64 `json:"quantity"`
	AverageEntryPrice float64 `json:"average_entry_price"`
	UnrealizedPnL     float64 `json:"unrealized_pnl"`
	RealizedPnL       float64 `json:"realized_pnl"`
	LastUpdated       int64   `json:"last_updated"`
}

// RiskLimitResponse represents a risk limit response
type RiskLimitResponse struct {
	ID        string  `json:"id"`
	UserID    string  `json:"user_id"`
	Symbol    string  `json:"symbol"`
	Type      string  `json:"type"`
	Value     float64 `json:"value"`
	Enabled   bool    `json:"enabled"`
	CreatedAt int64   `json:"created_at"`
	UpdatedAt int64   `json:"updated_at"`
}

// SetLimitRequest represents a request to set a risk limit
type SetLimitRequest struct {
	Symbol  string  `json:"symbol"`
	Type    string  `json:"type"`
	Value   float64 `json:"value"`
	Enabled bool    `json:"enabled"`
}

// ValidateOrderRequest represents a request to validate an order
type ValidateOrderRequest struct {
	Symbol   string  `json:"symbol" binding:"required"`
	Side     string  `json:"side" binding:"required"`
	Type     string  `json:"type" binding:"required"`
	Quantity float64 `json:"quantity" binding:"required"`
	Price    float64 `json:"price" binding:"required"`
}

// ValidateOrderResponse represents a response to validate an order
type ValidateOrderResponse struct {
	Valid            bool     `json:"valid"`
	RejectionReasons []string `json:"rejection_reasons,omitempty"`
	Warnings         []string `json:"warnings,omitempty"`
}

// mapLimitTypeFromString maps a string to a limit type enum
func mapLimitTypeFromString(limitType string) risk.LimitType {
	switch limitType {
	case "max_position_size":
		return risk.LimitType_MAX_POSITION_SIZE
	case "max_order_size":
		return risk.LimitType_MAX_ORDER_SIZE
	case "max_daily_volume":
		return risk.LimitType_MAX_DAILY_VOLUME
	case "max_daily_loss":
		return risk.LimitType_MAX_DAILY_LOSS
	case "max_leverage":
		return risk.LimitType_MAX_LEVERAGE
	case "max_concentration":
		return risk.LimitType_MAX_CONCENTRATION
	default:
		return risk.LimitType_UNKNOWN_LIMIT
	}
}

// mapLimitTypeToString maps a limit type enum to a string
func mapLimitTypeToString(limitType risk.LimitType) string {
	switch limitType {
	case risk.LimitType_MAX_POSITION_SIZE:
		return "max_position_size"
	case risk.LimitType_MAX_ORDER_SIZE:
		return "max_order_size"
	case risk.LimitType_MAX_DAILY_VOLUME:
		return "max_daily_volume"
	case risk.LimitType_MAX_DAILY_LOSS:
		return "max_daily_loss"
	case risk.LimitType_MAX_LEVERAGE:
		return "max_leverage"
	case risk.LimitType_MAX_CONCENTRATION:
		return "max_concentration"
	default:
		return "unknown"
	}
}

// mapPositionResponse maps a proto position to a response position
func mapPositionResponse(position *risk.PositionResponse) PositionResponse {
	return PositionResponse{
		Symbol:            position.Symbol,
		Quantity:          position.Quantity,
		AverageEntryPrice: position.AverageEntryPrice,
		UnrealizedPnL:     position.UnrealizedPnl,
		RealizedPnL:       position.RealizedPnl,
		LastUpdated:       position.LastUpdated,
	}
}

// mapRiskLimitResponse maps a proto risk limit to a response risk limit
func mapRiskLimitResponse(limit *risk.RiskLimitResponse) RiskLimitResponse {
	return RiskLimitResponse{
		ID:        limit.Id,
		UserID:    limit.UserId,
		Symbol:    limit.Symbol,
		Type:      mapLimitTypeToString(limit.Type),
		Value:     limit.Value,
		Enabled:   limit.Enabled,
		CreatedAt: limit.CreatedAt,
		UpdatedAt: limit.UpdatedAt,
	}
}

// GetPositions gets positions for a user
func (h *RiskHandler) GetPositions(c *gin.Context) {
	// Get symbol from query
	symbol := c.Query("symbol")

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

	// Get positions
	response, err := h.service.GetPositions(ctx, userID.(string), symbol)
	if err != nil {
		h.logger.Error("Failed to get positions", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get positions"})
		return
	}

	// Map response
	var positions []PositionResponse
	for _, position := range response.Positions {
		positions = append(positions, mapPositionResponse(position))
	}

	// Return empty array instead of null if no positions found
	if positions == nil {
		positions = []PositionResponse{}
	}

	c.JSON(http.StatusOK, positions)
}

// GetLimits gets risk limits for a user
func (h *RiskHandler) GetLimits(c *gin.Context) {
	// Get symbol and type from query
	symbol := c.Query("symbol")
	limitTypeStr := c.Query("type")

	// Map limit type
	var limitType risk.LimitType
	if limitTypeStr != "" {
		limitType = mapLimitTypeFromString(limitTypeStr)
	} else {
		limitType = risk.LimitType_UNKNOWN_LIMIT
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

	// Get limits
	response, err := h.service.GetLimits(ctx, userID.(string), symbol, limitType)
	if err != nil {
		h.logger.Error("Failed to get limits", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get limits"})
		return
	}

	// Map response
	var limits []RiskLimitResponse
	for _, limit := range response.Limits {
		limits = append(limits, mapRiskLimitResponse(limit))
	}

	// Return empty array instead of null if no limits found
	if limits == nil {
		limits = []RiskLimitResponse{}
	}

	c.JSON(http.StatusOK, limits)
}

// SetLimit sets a risk limit for a user
func (h *RiskHandler) SetLimit(c *gin.Context) {
	var req SetLimitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate inputs
	if req.Type == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Limit type is required"})
		return
	}

	if req.Value <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Value must be greater than 0"})
		return
	}

	// Map limit type
	limitType := mapLimitTypeFromString(req.Type)
	if limitType == risk.LimitType_UNKNOWN_LIMIT {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid limit type"})
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

	// Set limit
	response, err := h.service.SetLimit(ctx, userID.(string), req.Symbol, limitType, req.Value, req.Enabled)
	if err != nil {
		h.logger.Error("Failed to set limit", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to set limit"})
		return
	}

	// Map response
	limit := mapRiskLimitResponse(response)

	c.JSON(http.StatusOK, limit)
}

// DeleteLimit deletes a risk limit
func (h *RiskHandler) DeleteLimit(c *gin.Context) {
	// Get limit ID from path
	limitID := c.Param("id")
	if limitID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Limit ID is required"})
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

	// Delete limit
	response, err := h.service.DeleteLimit(ctx, limitID, userID.(string))
	if err != nil {
		h.logger.Error("Failed to delete limit", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete limit"})
		return
	}

	if !response.Success {
		c.JSON(http.StatusBadRequest, gin.H{"error": response.Error})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// ValidateOrder validates an order against risk limits
func (h *RiskHandler) ValidateOrder(c *gin.Context) {
	var req ValidateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate inputs
	if req.Symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Symbol is required"})
		return
	}

	if req.Side != "buy" && req.Side != "sell" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Side must be 'buy' or 'sell'"})
		return
	}

	if req.Quantity <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Quantity must be greater than 0"})
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

	// Validate order
	response, err := h.service.ValidateOrder(ctx, userID.(string), req.Symbol, req.Side, req.Type, req.Quantity, req.Price)
	if err != nil {
		h.logger.Error("Failed to validate order", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate order"})
		return
	}

	// Map response
	validateResponse := ValidateOrderResponse{
		Valid:            response.Valid,
		RejectionReasons: response.RejectionReasons,
		Warnings:         response.Warnings,
	}

	c.JSON(http.StatusOK, validateResponse)
}

