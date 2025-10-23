package handlers

import (
	"net/http"
	"strconv"

	"github.com/abdoElHodaky/tradSys/internal/db/models"
	"github.com/abdoElHodaky/tradSys/internal/services"
	"github.com/abdoElHodaky/tradSys/internal/trading/types"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// AssetHandlers provides HTTP handlers for asset-related operations
type AssetHandlers struct {
	assetService *services.AssetService
	logger       *zap.Logger
}

// NewAssetHandlers creates new asset handlers
func NewAssetHandlers(assetService *services.AssetService, logger *zap.Logger) *AssetHandlers {
	return &AssetHandlers{
		assetService: assetService,
		logger:       logger,
	}
}

// AssetMetadataRequest represents a request to create/update asset metadata
type AssetMetadataRequest struct {
	Symbol     string                 `json:"symbol" binding:"required"`
	AssetType  string                 `json:"asset_type" binding:"required"`
	Sector     string                 `json:"sector,omitempty"`
	Industry   string                 `json:"industry,omitempty"`
	Country    string                 `json:"country,omitempty"`
	Currency   string                 `json:"currency,omitempty"`
	Exchange   string                 `json:"exchange,omitempty"`
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}

// AssetPricingRequest represents a request to update asset pricing
type AssetPricingRequest struct {
	Symbol           string  `json:"symbol" binding:"required"`
	AssetType        string  `json:"asset_type" binding:"required"`
	Price            float64 `json:"price" binding:"required"`
	BidPrice         float64 `json:"bid_price,omitempty"`
	AskPrice         float64 `json:"ask_price,omitempty"`
	Volume           float64 `json:"volume,omitempty"`
	High24h          float64 `json:"high_24h,omitempty"`
	Low24h           float64 `json:"low_24h,omitempty"`
	Change24h        float64 `json:"change_24h,omitempty"`
	ChangePercent24h float64 `json:"change_percent_24h,omitempty"`
	MarketCap        float64 `json:"market_cap,omitempty"`
	Source           string  `json:"source,omitempty"`
}

// AssetDividendRequest represents a request to create asset dividend
type AssetDividendRequest struct {
	Symbol       string  `json:"symbol" binding:"required"`
	AssetType    string  `json:"asset_type" binding:"required"`
	ExDate       string  `json:"ex_date" binding:"required"`
	PayDate      string  `json:"pay_date" binding:"required"`
	RecordDate   string  `json:"record_date,omitempty"`
	Amount       float64 `json:"amount" binding:"required"`
	Currency     string  `json:"currency,omitempty"`
	DividendType string  `json:"dividend_type,omitempty"`
	Frequency    string  `json:"frequency,omitempty"`
	YieldPercent float64 `json:"yield_percent,omitempty"`
}

// GetAssetMetadata retrieves metadata for a specific asset
func (h *AssetHandlers) GetAssetMetadata(c *gin.Context) {
	symbol := c.Param("symbol")
	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Symbol is required"})
		return
	}

	metadata, err := h.assetService.GetAssetMetadata(c.Request.Context(), symbol)
	if err != nil {
		h.logger.Error("Failed to get asset metadata", zap.Error(err), zap.String("symbol", symbol))
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, metadata)
}

// CreateAssetMetadata creates new asset metadata
func (h *AssetHandlers) CreateAssetMetadata(c *gin.Context) {
	var req AssetMetadataRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	assetType, err := types.AssetTypeFromString(req.AssetType)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	metadata := &models.AssetMetadata{
		Symbol:     req.Symbol,
		AssetType:  assetType,
		Sector:     req.Sector,
		Industry:   req.Industry,
		Country:    req.Country,
		Currency:   req.Currency,
		Exchange:   req.Exchange,
		Attributes: models.AssetAttributes(req.Attributes),
		IsActive:   true,
	}

	if err := h.assetService.CreateAssetMetadata(c.Request.Context(), metadata); err != nil {
		h.logger.Error("Failed to create asset metadata", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, metadata)
}

// UpdateAssetMetadata updates existing asset metadata
func (h *AssetHandlers) UpdateAssetMetadata(c *gin.Context) {
	symbol := c.Param("symbol")
	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Symbol is required"})
		return
	}

	var req AssetMetadataRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	assetType, err := types.AssetTypeFromString(req.AssetType)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := &models.AssetMetadata{
		Symbol:     req.Symbol,
		AssetType:  assetType,
		Sector:     req.Sector,
		Industry:   req.Industry,
		Country:    req.Country,
		Currency:   req.Currency,
		Exchange:   req.Exchange,
		Attributes: models.AssetAttributes(req.Attributes),
	}

	if err := h.assetService.UpdateAssetMetadata(c.Request.Context(), symbol, updates); err != nil {
		h.logger.Error("Failed to update asset metadata", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Asset metadata updated successfully"})
}

// ListAssets returns a paginated list of assets
func (h *AssetHandlers) ListAssets(c *gin.Context) {
	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 50
	}
	offset := (page - 1) * limit

	// Parse asset type filter
	var assetTypeFilter *types.AssetType
	if assetTypeStr := c.Query("asset_type"); assetTypeStr != "" {
		if assetType, err := types.AssetTypeFromString(assetTypeStr); err == nil {
			assetTypeFilter = &assetType
		}
	}

	assets, total, err := h.assetService.ListAssets(c.Request.Context(), offset, limit, assetTypeFilter)
	if err != nil {
		h.logger.Error("Failed to list assets", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"assets": assets,
		"pagination": gin.H{
			"page":       page,
			"limit":      limit,
			"total":      total,
			"total_pages": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

// GetAssetsByType retrieves all assets of a specific type
func (h *AssetHandlers) GetAssetsByType(c *gin.Context) {
	assetTypeStr := c.Param("type")
	assetType, err := types.AssetTypeFromString(assetTypeStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	assets, err := h.assetService.GetAssetsByType(c.Request.Context(), assetType)
	if err != nil {
		h.logger.Error("Failed to get assets by type", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"asset_type": assetType,
		"assets":     assets,
		"count":      len(assets),
	})
}

// GetAssetPricing retrieves latest pricing for an asset
func (h *AssetHandlers) GetAssetPricing(c *gin.Context) {
	symbol := c.Param("symbol")
	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Symbol is required"})
		return
	}

	pricing, err := h.assetService.GetAssetPricing(c.Request.Context(), symbol)
	if err != nil {
		h.logger.Error("Failed to get asset pricing", zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, pricing)
}

// UpdateAssetPricing updates pricing information for an asset
func (h *AssetHandlers) UpdateAssetPricing(c *gin.Context) {
	var req AssetPricingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	assetType, err := types.AssetTypeFromString(req.AssetType)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	pricing := &models.AssetPricing{
		Symbol:           req.Symbol,
		AssetType:        assetType,
		Price:            req.Price,
		BidPrice:         req.BidPrice,
		AskPrice:         req.AskPrice,
		Volume:           req.Volume,
		High24h:          req.High24h,
		Low24h:           req.Low24h,
		Change24h:        req.Change24h,
		ChangePercent24h: req.ChangePercent24h,
		MarketCap:        req.MarketCap,
		Source:           req.Source,
	}

	if err := h.assetService.UpdateAssetPricing(c.Request.Context(), pricing); err != nil {
		h.logger.Error("Failed to update asset pricing", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Asset pricing updated successfully"})
}

// GetAssetDividends retrieves dividend information for an asset
func (h *AssetHandlers) GetAssetDividends(c *gin.Context) {
	symbol := c.Param("symbol")
	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Symbol is required"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if limit < 1 || limit > 100 {
		limit = 10
	}

	dividends, err := h.assetService.GetAssetDividends(c.Request.Context(), symbol, limit)
	if err != nil {
		h.logger.Error("Failed to get asset dividends", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"symbol":    symbol,
		"dividends": dividends,
		"count":     len(dividends),
	})
}

// GetAssetConfiguration retrieves configuration for an asset type
func (h *AssetHandlers) GetAssetConfiguration(c *gin.Context) {
	assetTypeStr := c.Param("type")
	assetType, err := types.AssetTypeFromString(assetTypeStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	config, err := h.assetService.GetAssetConfiguration(c.Request.Context(), assetType)
	if err != nil {
		h.logger.Error("Failed to get asset configuration", zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, config)
}

// GetAssetInfo returns comprehensive information about an asset
func (h *AssetHandlers) GetAssetInfo(c *gin.Context) {
	symbol := c.Param("symbol")
	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Symbol is required"})
		return
	}

	info, err := h.assetService.GetAssetInfo(c.Request.Context(), symbol)
	if err != nil {
		h.logger.Error("Failed to get asset info", zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, info)
}

// GetSupportedAssetTypes returns all supported asset types
func (h *AssetHandlers) GetSupportedAssetTypes(c *gin.Context) {
	assetTypes := types.GetAllAssetTypes()
	
	response := make([]gin.H, len(assetTypes))
	for i, assetType := range assetTypes {
		response[i] = gin.H{
			"type":                    assetType,
			"name":                    string(assetType),
			"requires_special_handling": assetType.RequiresSpecialHandling(),
			"trading_hours":           assetType.GetTradingHours(),
			"relevant_attributes":     assetType.GetRelevantAttributes(),
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"supported_asset_types": response,
		"count":                 len(assetTypes),
	})
}
