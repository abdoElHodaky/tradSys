package handlers

import (
	"net/http"
	"strconv"

	"github.com/abdoElHodaky/tradSys/internal/services"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// REITHandlers provides HTTP handlers for REIT-specific operations
type REITHandlers struct {
	reitService *services.REITService
	logger      *zap.Logger
}

// NewREITHandlers creates new REIT handlers
func NewREITHandlers(reitService *services.REITService, logger *zap.Logger) *REITHandlers {
	return &REITHandlers{
		reitService: reitService,
		logger:      logger,
	}
}

// REITMetricsRequest represents a request to update REIT metrics
type REITMetricsRequest struct {
	Symbol            string  `json:"symbol" binding:"required"`
	FFO               float64 `json:"ffo"`
	AFFO              float64 `json:"affo"`
	NAVPerShare       float64 `json:"nav_per_share"`
	DividendYield     float64 `json:"dividend_yield"`
	PayoutRatio       float64 `json:"payout_ratio"`
	DebtToEquity      float64 `json:"debt_to_equity"`
	OccupancyRate     float64 `json:"occupancy_rate"`
	PriceToFFO        float64 `json:"price_to_ffo"`
	PriceToNAV        float64 `json:"price_to_nav"`
	TotalReturn       float64 `json:"total_return"`
}

// CreateREITRequest represents a request to create REIT metadata
type CreateREITRequest struct {
	Symbol         string                 `json:"symbol" binding:"required"`
	REITType       string                 `json:"reit_type" binding:"required"`
	PropertySector string                 `json:"property_sector" binding:"required"`
	Attributes     map[string]interface{} `json:"attributes,omitempty"`
}

// GetREITMetrics retrieves REIT performance metrics
func (h *REITHandlers) GetREITMetrics(c *gin.Context) {
	symbol := c.Param("symbol")
	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Symbol is required"})
		return
	}

	metrics, err := h.reitService.GetREITMetrics(c.Request.Context(), symbol)
	if err != nil {
		h.logger.Error("Failed to get REIT metrics", zap.Error(err), zap.String("symbol", symbol))
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, metrics)
}

// UpdateREITMetrics updates REIT performance metrics
func (h *REITHandlers) UpdateREITMetrics(c *gin.Context) {
	var req REITMetricsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	metrics := &services.REITMetrics{
		Symbol:        req.Symbol,
		FFO:           req.FFO,
		AFFO:          req.AFFO,
		NAVPerShare:   req.NAVPerShare,
		DividendYield: req.DividendYield,
		PayoutRatio:   req.PayoutRatio,
		DebtToEquity:  req.DebtToEquity,
		OccupancyRate: req.OccupancyRate,
		PriceToFFO:    req.PriceToFFO,
		PriceToNAV:    req.PriceToNAV,
		TotalReturn:   req.TotalReturn,
	}

	if err := h.reitService.UpdateREITMetrics(c.Request.Context(), metrics); err != nil {
		h.logger.Error("Failed to update REIT metrics", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "REIT metrics updated successfully"})
}

// CreateREIT creates REIT-specific metadata
func (h *REITHandlers) CreateREIT(c *gin.Context) {
	var req CreateREITRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Convert string types to proper enums
	reitType := services.REITType(req.REITType)
	propertySector := services.PropertySector(req.PropertySector)

	err := h.reitService.CreateREITMetadata(c.Request.Context(), req.Symbol, reitType, propertySector, req.Attributes)
	if err != nil {
		h.logger.Error("Failed to create REIT metadata", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "REIT created successfully", "symbol": req.Symbol})
}

// GetREITsByPropertySector retrieves REITs by property sector
func (h *REITHandlers) GetREITsByPropertySector(c *gin.Context) {
	sectorStr := c.Param("sector")
	if sectorStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Property sector is required"})
		return
	}

	sector := services.PropertySector(sectorStr)
	reits, err := h.reitService.GetREITsByPropertySector(c.Request.Context(), sector)
	if err != nil {
		h.logger.Error("Failed to get REITs by property sector", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"property_sector": sector,
		"reits":           reits,
		"count":           len(reits),
	})
}

// CalculateDividendYield calculates the current dividend yield for a REIT
func (h *REITHandlers) CalculateDividendYield(c *gin.Context) {
	symbol := c.Param("symbol")
	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Symbol is required"})
		return
	}

	yield, err := h.reitService.CalculateDividendYield(c.Request.Context(), symbol)
	if err != nil {
		h.logger.Error("Failed to calculate dividend yield", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"symbol":         symbol,
		"dividend_yield": yield,
		"calculated_at":  "now",
	})
}

// GetREITDividendSchedule returns the dividend payment schedule for a REIT
func (h *REITHandlers) GetREITDividendSchedule(c *gin.Context) {
	symbol := c.Param("symbol")
	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Symbol is required"})
		return
	}

	schedule, err := h.reitService.GetREITDividendSchedule(c.Request.Context(), symbol)
	if err != nil {
		h.logger.Error("Failed to get REIT dividend schedule", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"symbol":            symbol,
		"dividend_schedule": schedule,
		"count":             len(schedule),
	})
}

// AnalyzeREITPerformance provides comprehensive REIT performance analysis
func (h *REITHandlers) AnalyzeREITPerformance(c *gin.Context) {
	symbol := c.Param("symbol")
	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Symbol is required"})
		return
	}

	analysis, err := h.reitService.AnalyzeREITPerformance(c.Request.Context(), symbol)
	if err != nil {
		h.logger.Error("Failed to analyze REIT performance", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, analysis)
}

// ValidateREITOrder validates a REIT order
func (h *REITHandlers) ValidateREITOrder(c *gin.Context) {
	symbol := c.Query("symbol")
	quantityStr := c.Query("quantity")
	priceStr := c.Query("price")

	if symbol == "" || quantityStr == "" || priceStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Symbol, quantity, and price are required"})
		return
	}

	quantity, err := strconv.ParseFloat(quantityStr, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid quantity"})
		return
	}

	price, err := strconv.ParseFloat(priceStr, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid price"})
		return
	}

	err = h.reitService.ValidateREITOrder(c.Request.Context(), symbol, quantity, price)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"valid": false,
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"valid":    true,
		"message":  "REIT order validation passed",
		"symbol":   symbol,
		"quantity": quantity,
		"price":    price,
	})
}

// GetPropertySectors returns all available property sectors
func (h *REITHandlers) GetPropertySectors(c *gin.Context) {
	sectors := []gin.H{
		{"sector": "residential", "name": "Residential"},
		{"sector": "commercial", "name": "Commercial"},
		{"sector": "industrial", "name": "Industrial"},
		{"sector": "retail", "name": "Retail"},
		{"sector": "office", "name": "Office"},
		{"sector": "healthcare", "name": "Healthcare"},
		{"sector": "hospitality", "name": "Hospitality"},
		{"sector": "data_center", "name": "Data Center"},
		{"sector": "self_storage", "name": "Self Storage"},
		{"sector": "timberland", "name": "Timberland"},
	}

	c.JSON(http.StatusOK, gin.H{
		"property_sectors": sectors,
		"count":            len(sectors),
	})
}

// GetREITTypes returns all available REIT types
func (h *REITHandlers) GetREITTypes(c *gin.Context) {
	types := []gin.H{
		{"type": "equity", "name": "Equity REIT", "description": "Owns and operates income-generating real estate"},
		{"type": "mortgage", "name": "Mortgage REIT", "description": "Provides financing for income-generating real estate"},
		{"type": "hybrid", "name": "Hybrid REIT", "description": "Combines equity and mortgage REIT strategies"},
	}

	c.JSON(http.StatusOK, gin.H{
		"reit_types": types,
		"count":      len(types),
	})
}
