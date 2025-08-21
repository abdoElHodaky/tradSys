package api

import (
	"time"

	"github.com/abdoElHodaky/tradSys/internal/api/handlers"
	"github.com/abdoElHodaky/tradSys/internal/auth"
	"github.com/abdoElHodaky/tradSys/internal/db/repositories"
	"github.com/abdoElHodaky/tradSys/internal/orders"
	"github.com/abdoElHodaky/tradSys/internal/strategy"
	"github.com/abdoElHodaky/tradSys/internal/user"
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
	userRepo *repositories.UserRepository,
) {
	// Register Swagger documentation routes
	RegisterSwaggerRoutes(router)
	
	// Setup JWT authentication
	jwtConfig := auth.JWTConfig{
		SecretKey:     "your-secret-key", // In production, use environment variable
		TokenDuration: 24 * time.Hour,
		Issuer:        "tradsys-api",
	}
	jwtService := auth.NewJWTService(jwtConfig)
	authMiddleware := auth.NewMiddleware(jwtService, logger)
	
	// Setup user service and handler
	userService := user.NewService(logger, userRepo)
	userHandler := handlers.NewUserHandler(logger, userService)
	
	// Register user routes
	userHandler.RegisterRoutes(router, authMiddleware.JWTAuth())
	
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
	
	// Register order handler
	orderHandler := handlers.NewOrderHandler(orderService, logger)
	orderHandler.RegisterRoutes(v1, authMiddleware.JWTAuth())
}
