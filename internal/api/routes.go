package api

import (
	"github.com/abdoElHodaky/tradSys/internal/api/handlers"
	"github.com/abdoElHodaky/tradSys/internal/db/repositories"
	"github.com/abdoElHodaky/tradSys/internal/orders"
	"github.com/abdoElHodaky/tradSys/internal/strategy"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// RegisterRoutes registers all API routes
func RegisterRoutes(
	router *gin.Engine,
	logger *zap.Logger,
	orderService orders.OrderService,
	pairRepo *repositories.PairRepository,
	statsRepo *repositories.PairStatisticsRepository,
	positionRepo *repositories.PairPositionRepository,
	strategyManager *strategy.StrategyManager,
) {
	// API version group
	v1 := router.Group("/api/v1")
	
	// Register pairs handler
	pairsHandler := handlers.NewPairsHandler(
		pairRepo,
		statsRepo,
		positionRepo,
		strategyManager,
		logger,
	)
	pairsHandler.RegisterRoutes(v1)
	
	// Register other handlers here
}
