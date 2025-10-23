package api

import (
	"github.com/abdoElHodaky/tradSys/internal/db/repositories"
	"github.com/abdoElHodaky/tradSys/internal/services"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// RegisterRoutes registers all API routes
func RegisterRoutes(
	router *gin.Engine,
	logger *zap.Logger,
	orderService services.OrderService,
	settlementService services.SettlementService,
	pairsService services.PairsService,
	riskService services.RiskService,
	strategyService services.StrategyService,
	pairRepo *repositories.PairRepository,
	statsRepo *repositories.PairStatisticsRepository,
	positionRepo *repositories.PairPositionRepository,
) {
	// API version group
	v1 := router.Group("/api/v1")

	// Create service handlers
	orderHandler := services.NewOrderHandler(orderService)
	pairsHandler := services.NewPairsHandler(pairsService)
	settlementHandler := services.NewSettlementHandler(settlementService)
	strategyHandler := services.NewStrategyHandler(strategyService)

	// Register order routes
	orderRoutes := v1.Group("/orders")
	{
		orderRoutes.POST("", orderHandler.CreateOrder)
		orderRoutes.GET("", orderHandler.ListOrders)
		orderRoutes.GET("/:id", orderHandler.GetOrder)
		orderRoutes.DELETE("/:id", orderHandler.CancelOrder)
	}

	// Register pairs routes
	pairRoutes := v1.Group("/pairs")
	{
		pairRoutes.GET("", pairsHandler.ListPairs)
		pairRoutes.GET("/:symbol", pairsHandler.GetPair)
		pairRoutes.GET("/:symbol/ticker", pairsHandler.GetTicker)
		pairRoutes.GET("/:symbol/orderbook", pairsHandler.GetOrderBook)
	}

	// Register settlement routes
	settlementRoutes := v1.Group("/settlements")
	{
		settlementRoutes.POST("", settlementHandler.ProcessSettlement)
		settlementRoutes.GET("", settlementHandler.ListSettlements)
		settlementRoutes.GET("/:id", settlementHandler.GetSettlement)
	}

	// Register strategy routes
	strategyRoutes := v1.Group("/strategies")
	{
		strategyRoutes.POST("", strategyHandler.CreateStrategy)
		strategyRoutes.GET("", strategyHandler.ListStrategies)
		strategyRoutes.GET("/:id", strategyHandler.GetStrategy)
		strategyRoutes.POST("/:id/start", strategyHandler.StartStrategy)
		strategyRoutes.POST("/:id/stop", strategyHandler.StopStrategy)
	}

	// Legacy handlers can be added here if needed for backward compatibility
	// For now, we use the new service layer exclusively
}
