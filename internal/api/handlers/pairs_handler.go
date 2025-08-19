package handlers

import (
	"net/http"
	"time"
	
	"github.com/abdoElHodaky/tradSys/internal/db/models"
	"github.com/abdoElHodaky/tradSys/internal/db/repositories"
	"github.com/abdoElHodaky/tradSys/internal/statistics"
	"github.com/abdoElHodaky/tradSys/internal/strategy"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// PairsHandler handles API requests for pairs
type PairsHandler struct {
	pairRepo       *repositories.PairRepository
	statsRepo      *repositories.PairStatisticsRepository
	positionRepo   *repositories.PairPositionRepository
	strategyManager *strategy.StrategyManager
	logger         *zap.Logger
}

// NewPairsHandler creates a new pairs handler
func NewPairsHandler(
	pairRepo *repositories.PairRepository,
	statsRepo *repositories.PairStatisticsRepository,
	positionRepo *repositories.PairPositionRepository,
	strategyManager *strategy.StrategyManager,
	logger *zap.Logger,
) *PairsHandler {
	return &PairsHandler{
		pairRepo:       pairRepo,
		statsRepo:      statsRepo,
		positionRepo:   positionRepo,
		strategyManager: strategyManager,
		logger:         logger,
	}
}

// RegisterRoutes registers the pairs API routes
func (h *PairsHandler) RegisterRoutes(router *gin.RouterGroup) {
	pairs := router.Group("/pairs")
	{
		pairs.GET("", h.GetAllPairs)
		pairs.GET("/:id", h.GetPair)
		pairs.POST("", h.CreatePair)
		pairs.PUT("/:id", h.UpdatePair)
		pairs.DELETE("/:id", h.DeletePair)
		pairs.GET("/:id/statistics", h.GetPairStatistics)
		pairs.GET("/:id/positions", h.GetPairPositions)
		pairs.POST("/:id/analyze", h.AnalyzePair)
		pairs.POST("/:id/strategy", h.CreatePairStrategy)
		pairs.PUT("/:id/strategy/:strategy_id", h.UpdatePairStrategy)
		pairs.DELETE("/:id/strategy/:strategy_id", h.DeletePairStrategy)
	}
}

// PairRequest represents a request to create or update a pair
type PairRequest struct {
	Symbol1            string  `json:"symbol1" binding:"required"`
	Symbol2            string  `json:"symbol2" binding:"required"`
	Ratio              float64 `json:"ratio"`
	Status             string  `json:"status" binding:"required"`
	ZScoreThresholdEntry float64 `json:"z_score_threshold_entry"`
	ZScoreThresholdExit  float64 `json:"z_score_threshold_exit"`
	LookbackPeriod     int     `json:"lookback_period" binding:"required"`
	Notes              string  `json:"notes"`
}

// PairResponse represents a pair response
type PairResponse struct {
	ID                 uint      `json:"id"`
	PairID             string    `json:"pair_id"`
	Symbol1            string    `json:"symbol1"`
	Symbol2            string    `json:"symbol2"`
	Ratio              float64   `json:"ratio"`
	Status             string    `json:"status"`
	Correlation        float64   `json:"correlation"`
	Cointegration      float64   `json:"cointegration"`
	ZScoreThresholdEntry float64   `json:"z_score_threshold_entry"`
	ZScoreThresholdExit  float64   `json:"z_score_threshold_exit"`
	LookbackPeriod     int       `json:"lookback_period"`
	HalfLife           int       `json:"half_life"`
	CreatedBy          uint      `json:"created_by"`
	Notes              string    `json:"notes"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// GetAllPairs returns all pairs
func (h *PairsHandler) GetAllPairs(c *gin.Context) {
	pairs, err := h.pairRepo.GetAllPairs(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	var response []PairResponse
	for _, pair := range pairs {
		response = append(response, PairResponse{
			ID:                 pair.ID,
			PairID:             pair.PairID,
			Symbol1:            pair.Symbol1,
			Symbol2:            pair.Symbol2,
			Ratio:              pair.Ratio,
			Status:             string(pair.Status),
			Correlation:        pair.Correlation,
			Cointegration:      pair.Cointegration,
			ZScoreThresholdEntry: pair.ZScoreThresholdEntry,
			ZScoreThresholdExit:  pair.ZScoreThresholdExit,
			LookbackPeriod:     pair.LookbackPeriod,
			HalfLife:           pair.HalfLife,
			CreatedBy:          pair.CreatedBy,
			Notes:              pair.Notes,
			CreatedAt:          pair.CreatedAt,
			UpdatedAt:          pair.UpdatedAt,
		})
	}
	
	c.JSON(http.StatusOK, response)
}

// GetPair returns a pair by ID
func (h *PairsHandler) GetPair(c *gin.Context) {
	pairID := c.Param("id")
	
	pair, err := h.pairRepo.GetPair(c.Request.Context(), pairID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Pair not found"})
		return
	}
	
	response := PairResponse{
		ID:                 pair.ID,
		PairID:             pair.PairID,
		Symbol1:            pair.Symbol1,
		Symbol2:            pair.Symbol2,
		Ratio:              pair.Ratio,
		Status:             string(pair.Status),
		Correlation:        pair.Correlation,
		Cointegration:      pair.Cointegration,
		ZScoreThresholdEntry: pair.ZScoreThresholdEntry,
		ZScoreThresholdExit:  pair.ZScoreThresholdExit,
		LookbackPeriod:     pair.LookbackPeriod,
		HalfLife:           pair.HalfLife,
		CreatedBy:          pair.CreatedBy,
		Notes:              pair.Notes,
		CreatedAt:          pair.CreatedAt,
		UpdatedAt:          pair.UpdatedAt,
	}
	
	c.JSON(http.StatusOK, response)
}

// CreatePair creates a new pair
func (h *PairsHandler) CreatePair(c *gin.Context) {
	var request PairRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// Generate a unique pair ID
	pairID := uuid.New().String()
	
	// Create the pair
	pair := &models.Pair{
		PairID:               pairID,
		Symbol1:              request.Symbol1,
		Symbol2:              request.Symbol2,
		Ratio:                request.Ratio,
		Status:               models.PairStatus(request.Status),
		ZScoreThresholdEntry: request.ZScoreThresholdEntry,
		ZScoreThresholdExit:  request.ZScoreThresholdExit,
		LookbackPeriod:       request.LookbackPeriod,
		Notes:                request.Notes,
	}
	
	// Save the pair to the database
	if err := h.pairRepo.CreatePair(c.Request.Context(), pair); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	response := PairResponse{
		ID:                 pair.ID,
		PairID:             pair.PairID,
		Symbol1:            pair.Symbol1,
		Symbol2:            pair.Symbol2,
		Ratio:              pair.Ratio,
		Status:             string(pair.Status),
		Correlation:        pair.Correlation,
		Cointegration:      pair.Cointegration,
		ZScoreThresholdEntry: pair.ZScoreThresholdEntry,
		ZScoreThresholdExit:  pair.ZScoreThresholdExit,
		LookbackPeriod:     pair.LookbackPeriod,
		HalfLife:           pair.HalfLife,
		CreatedBy:          pair.CreatedBy,
		Notes:              pair.Notes,
		CreatedAt:          pair.CreatedAt,
		UpdatedAt:          pair.UpdatedAt,
	}
	
	c.JSON(http.StatusCreated, response)
}

// UpdatePair updates an existing pair
func (h *PairsHandler) UpdatePair(c *gin.Context) {
	pairID := c.Param("id")
	
	var request PairRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// Get the existing pair
	pair, err := h.pairRepo.GetPair(c.Request.Context(), pairID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Pair not found"})
		return
	}
	
	// Update the pair
	pair.Symbol1 = request.Symbol1
	pair.Symbol2 = request.Symbol2
	pair.Ratio = request.Ratio
	pair.Status = models.PairStatus(request.Status)
	pair.ZScoreThresholdEntry = request.ZScoreThresholdEntry
	pair.ZScoreThresholdExit = request.ZScoreThresholdExit
	pair.LookbackPeriod = request.LookbackPeriod
	pair.Notes = request.Notes
	
	// Save the updated pair to the database
	if err := h.pairRepo.UpdatePair(c.Request.Context(), pair); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	response := PairResponse{
		ID:                 pair.ID,
		PairID:             pair.PairID,
		Symbol1:            pair.Symbol1,
		Symbol2:            pair.Symbol2,
		Ratio:              pair.Ratio,
		Status:             string(pair.Status),
		Correlation:        pair.Correlation,
		Cointegration:      pair.Cointegration,
		ZScoreThresholdEntry: pair.ZScoreThresholdEntry,
		ZScoreThresholdExit:  pair.ZScoreThresholdExit,
		LookbackPeriod:     pair.LookbackPeriod,
		HalfLife:           pair.HalfLife,
		CreatedBy:          pair.CreatedBy,
		Notes:              pair.Notes,
		CreatedAt:          pair.CreatedAt,
		UpdatedAt:          pair.UpdatedAt,
	}
	
	c.JSON(http.StatusOK, response)
}

// DeletePair deletes a pair
func (h *PairsHandler) DeletePair(c *gin.Context) {
	pairID := c.Param("id")
	
	// Delete the pair
	if err := h.pairRepo.DeletePair(c.Request.Context(), pairID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "Pair deleted successfully"})
}

// PairStatisticsResponse represents a pair statistics response
type PairStatisticsResponse struct {
	ID            uint      `json:"id"`
	PairID        string    `json:"pair_id"`
	Timestamp     time.Time `json:"timestamp"`
	Correlation   float64   `json:"correlation"`
	Cointegration float64   `json:"cointegration"`
	SpreadMean    float64   `json:"spread_mean"`
	SpreadStdDev  float64   `json:"spread_std_dev"`
	CurrentZScore float64   `json:"current_z_score"`
	SpreadValue   float64   `json:"spread_value"`
}

// GetPairStatistics returns statistics for a pair
func (h *PairsHandler) GetPairStatistics(c *gin.Context) {
	pairID := c.Param("id")
	
	// Get the latest statistics
	stats, err := h.statsRepo.GetLatestStatistics(c.Request.Context(), pairID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Pair statistics not found"})
		return
	}
	
	response := PairStatisticsResponse{
		ID:            stats.ID,
		PairID:        stats.PairID,
		Timestamp:     stats.Timestamp,
		Correlation:   stats.Correlation,
		Cointegration: stats.Cointegration,
		SpreadMean:    stats.SpreadMean,
		SpreadStdDev:  stats.SpreadStdDev,
		CurrentZScore: stats.CurrentZScore,
		SpreadValue:   stats.SpreadValue,
	}
	
	c.JSON(http.StatusOK, response)
}

// PairPositionResponse represents a pair position response
type PairPositionResponse struct {
	ID             uint      `json:"id"`
	PairID         string    `json:"pair_id"`
	EntryTimestamp time.Time `json:"entry_timestamp"`
	Symbol1        string    `json:"symbol1"`
	Symbol2        string    `json:"symbol2"`
	Quantity1      float64   `json:"quantity1"`
	Quantity2      float64   `json:"quantity2"`
	EntryPrice1    float64   `json:"entry_price1"`
	EntryPrice2    float64   `json:"entry_price2"`
	CurrentPrice1  float64   `json:"current_price1"`
	CurrentPrice2  float64   `json:"current_price2"`
	EntrySpread    float64   `json:"entry_spread"`
	CurrentSpread  float64   `json:"current_spread"`
	EntryZScore    float64   `json:"entry_z_score"`
	CurrentZScore  float64   `json:"current_z_score"`
	PnL            float64   `json:"pnl"`
	Status         string    `json:"status"`
	ExitTimestamp  time.Time `json:"exit_timestamp,omitempty"`
}

// GetPairPositions returns positions for a pair
func (h *PairsHandler) GetPairPositions(c *gin.Context) {
	pairID := c.Param("id")
	
	// Get the positions
	positions, err := h.positionRepo.GetPositionHistory(c.Request.Context(), pairID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	var response []PairPositionResponse
	for _, pos := range positions {
		response = append(response, PairPositionResponse{
			ID:             pos.ID,
			PairID:         pos.PairID,
			EntryTimestamp: pos.EntryTimestamp,
			Symbol1:        pos.Symbol1,
			Symbol2:        pos.Symbol2,
			Quantity1:      pos.Quantity1,
			Quantity2:      pos.Quantity2,
			EntryPrice1:    pos.EntryPrice1,
			EntryPrice2:    pos.EntryPrice2,
			CurrentPrice1:  pos.CurrentPrice1,
			CurrentPrice2:  pos.CurrentPrice2,
			EntrySpread:    pos.EntrySpread,
			CurrentSpread:  pos.CurrentSpread,
			EntryZScore:    pos.EntryZScore,
			CurrentZScore:  pos.CurrentZScore,
			PnL:            pos.PnL,
			Status:         pos.Status,
			ExitTimestamp:  pos.ExitTimestamp,
		})
	}
	
	c.JSON(http.StatusOK, response)
}

// AnalyzeRequest represents a request to analyze a pair
type AnalyzeRequest struct {
	Prices1 []float64 `json:"prices1" binding:"required"`
	Prices2 []float64 `json:"prices2" binding:"required"`
}

// AnalyzeResponse represents the response from analyzing a pair
type AnalyzeResponse struct {
	Correlation   float64   `json:"correlation"`
	Cointegration float64   `json:"cointegration"`
	IsCointegrated bool     `json:"is_cointegrated"`
	OptimalRatio  float64   `json:"optimal_ratio"`
	SpreadMean    float64   `json:"spread_mean"`
	SpreadStdDev  float64   `json:"spread_std_dev"`
	CurrentZScore float64   `json:"current_z_score"`
	HalfLife      int       `json:"half_life"`
	Spread        []float64 `json:"spread"`
}

// AnalyzePair analyzes a pair using historical price data
func (h *PairsHandler) AnalyzePair(c *gin.Context) {
	pairID := c.Param("id")
	
	var request AnalyzeRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// Get the pair
	pair, err := h.pairRepo.GetPair(c.Request.Context(), pairID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Pair not found"})
		return
	}
	
	// Calculate correlation
	correlation, err := statistics.CalculateCorrelation(request.Prices1, request.Prices2)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// Calculate optimal hedge ratio
	optimalRatio, err := statistics.CalculateOptimalHedgeRatio(request.Prices1, request.Prices2)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// Calculate cointegration
	cointegration, isCointegrated, err := statistics.EngleGrangerTest(request.Prices1, request.Prices2)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// Calculate spread
	spread, err := statistics.CalculateSpread(request.Prices1, request.Prices2, optimalRatio)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// Calculate spread statistics
	spreadMean, err := statistics.CalculateMean(spread)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	spreadStdDev, err := statistics.CalculateStdDev(spread, spreadMean)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// Calculate current z-score
	currentSpread := request.Prices1[len(request.Prices1)-1] - (optimalRatio * request.Prices2[len(request.Prices2)-1])
	currentZScore := statistics.CalculateZScore(currentSpread, spreadMean, spreadStdDev)
	
	// Estimate half-life
	halfLife, err := statistics.EstimateHalfLife(spread)
	if err != nil {
		halfLife = 0 // Set to 0 if estimation fails
	}
	
	// Update pair with analysis results
	pair.Correlation = correlation
	pair.Cointegration = cointegration
	pair.Ratio = optimalRatio
	pair.HalfLife = halfLife
	
	if err := h.pairRepo.UpdatePair(c.Request.Context(), pair); err != nil {
		h.logger.Error("Failed to update pair with analysis results",
			zap.Error(err),
			zap.String("pair_id", pairID))
	}
	
	// Create a statistics record
	stats := &models.PairStatistics{
		PairID:        pairID,
		Timestamp:     time.Now(),
		Correlation:   correlation,
		Cointegration: cointegration,
		SpreadMean:    spreadMean,
		SpreadStdDev:  spreadStdDev,
		CurrentZScore: currentZScore,
		SpreadValue:   currentSpread,
	}
	
	if err := h.statsRepo.Create(c.Request.Context(), stats); err != nil {
		h.logger.Error("Failed to create pair statistics",
			zap.Error(err),
			zap.String("pair_id", pairID))
	}
	
	response := AnalyzeResponse{
		Correlation:   correlation,
		Cointegration: cointegration,
		IsCointegrated: isCointegrated,
		OptimalRatio:  optimalRatio,
		SpreadMean:    spreadMean,
		SpreadStdDev:  spreadStdDev,
		CurrentZScore: currentZScore,
		HalfLife:      halfLife,
		Spread:        spread,
	}
	
	c.JSON(http.StatusOK, response)
}

// CreateStrategyRequest represents a request to create a strategy for a pair
type CreateStrategyRequest struct {
	Name           string  `json:"name" binding:"required"`
	ZScoreEntry    float64 `json:"z_score_entry" binding:"required"`
	ZScoreExit     float64 `json:"z_score_exit" binding:"required"`
	PositionSize   float64 `json:"position_size" binding:"required"`
	MaxPositions   int     `json:"max_positions" binding:"required"`
	UpdateInterval string  `json:"update_interval" binding:"required"`
}

// CreatePairStrategy creates a strategy for a pair
func (h *PairsHandler) CreatePairStrategy(c *gin.Context) {
	pairID := c.Param("id")
	
	var request CreateStrategyRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// Get the pair
	pair, err := h.pairRepo.GetPair(c.Request.Context(), pairID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Pair not found"})
		return
	}
	
	// Parse update interval
	updateInterval, err := time.ParseDuration(request.UpdateInterval)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid update interval format"})
		return
	}
	
	// Create strategy parameters
	params := strategy.StatisticalArbitrageParams{
		Name:           request.Name,
		PairID:         pairID,
		Symbol1:        pair.Symbol1,
		Symbol2:        pair.Symbol2,
		Ratio:          pair.Ratio,
		ZScoreEntry:    request.ZScoreEntry,
		ZScoreExit:     request.ZScoreExit,
		PositionSize:   request.PositionSize,
		MaxPositions:   request.MaxPositions,
		LookbackPeriod: pair.LookbackPeriod,
		UpdateInterval: updateInterval,
	}
	
	// Create the strategy
	strat, err := h.strategyManager.CreatePairsStrategy(c.Request.Context(), params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	// Start the strategy
	if err := h.strategyManager.StartStrategy(c.Request.Context(), strat.GetName()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusCreated, gin.H{
		"message": "Strategy created and started successfully",
		"strategy_name": strat.GetName(),
	})
}

// UpdateStrategyRequest represents a request to update a strategy
type UpdateStrategyRequest struct {
	ZScoreEntry    float64 `json:"z_score_entry"`
	ZScoreExit     float64 `json:"z_score_exit"`
	PositionSize   float64 `json:"position_size"`
	MaxPositions   int     `json:"max_positions"`
	UpdateInterval string  `json:"update_interval"`
}

// UpdatePairStrategy updates a strategy for a pair
func (h *PairsHandler) UpdatePairStrategy(c *gin.Context) {
	strategyID := c.Param("strategy_id")
	
	var request UpdateStrategyRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// Get the strategy
	strat, err := h.strategyManager.GetStrategy(strategyID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Strategy not found"})
		return
	}
	
	// Update strategy parameters
	params := strat.GetParameters()
	
	if request.ZScoreEntry != 0 {
		params["z_score_entry"] = request.ZScoreEntry
	}
	
	if request.ZScoreExit != 0 {
		params["z_score_exit"] = request.ZScoreExit
	}
	
	if request.PositionSize != 0 {
		params["position_size"] = request.PositionSize
	}
	
	if request.MaxPositions != 0 {
		params["max_positions"] = request.MaxPositions
	}
	
	if request.UpdateInterval != "" {
		params["update_interval"] = request.UpdateInterval
	}
	
	// Set the updated parameters
	if err := strat.SetParameters(params); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"message": "Strategy updated successfully",
		"strategy_name": strat.GetName(),
	})
}

// DeletePairStrategy deletes a strategy for a pair
func (h *PairsHandler) DeletePairStrategy(c *gin.Context) {
	strategyID := c.Param("strategy_id")
	
	// Stop the strategy
	if err := h.strategyManager.StopStrategy(c.Request.Context(), strategyID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	// Unregister the strategy
	if err := h.strategyManager.UnregisterStrategy(strategyID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"message": "Strategy deleted successfully",
	})
}
