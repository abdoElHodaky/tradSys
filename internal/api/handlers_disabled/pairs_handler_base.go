package handlers

import (
	"time"

	"github.com/abdoElHodaky/tradSys/internal/db/repositories"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// PairsHandler handles API requests for pairs
type PairsHandler struct {
	pairRepo        *repositories.PairRepository
	statsRepo       *repositories.PairStatisticsRepository
	positionRepo    *repositories.PairPositionRepository
	strategyManager *strategy.StrategyManager
	logger          *zap.Logger
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
		pairRepo:        pairRepo,
		statsRepo:       statsRepo,
		positionRepo:    positionRepo,
		strategyManager: strategyManager,
		logger:          logger,
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
	Symbol1              string  `json:"symbol1" binding:"required"`
	Symbol2              string  `json:"symbol2" binding:"required"`
	Ratio                float64 `json:"ratio"`
	Status               string  `json:"status" binding:"required"`
	ZScoreThresholdEntry float64 `json:"z_score_threshold_entry"`
	ZScoreThresholdExit  float64 `json:"z_score_threshold_exit"`
	LookbackPeriod       int     `json:"lookback_period" binding:"required"`
	Notes                string  `json:"notes"`
}

// PairResponse represents a pair response
type PairResponse struct {
	ID                   uint      `json:"id"`
	PairID               string    `json:"pair_id"`
	Symbol1              string    `json:"symbol1"`
	Symbol2              string    `json:"symbol2"`
	Ratio                float64   `json:"ratio"`
	Status               string    `json:"status"`
	Correlation          float64   `json:"correlation"`
	Cointegration        float64   `json:"cointegration"`
	ZScoreThresholdEntry float64   `json:"z_score_threshold_entry"`
	ZScoreThresholdExit  float64   `json:"z_score_threshold_exit"`
	LookbackPeriod       int       `json:"lookback_period"`
	HalfLife             int       `json:"half_life"`
	CreatedBy            uint      `json:"created_by"`
	Notes                string    `json:"notes"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}
