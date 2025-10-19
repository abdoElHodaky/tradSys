package repositories

import (
	"go.uber.org/fx"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RepositoriesModule provides the repositories module for the fx application
var RepositoriesModule = fx.Options(
	fx.Provide(NewOrderRepository),
	fx.Provide(NewTradeRepository),
	fx.Provide(NewPositionRepository),
	fx.Provide(NewRiskRepository),
	fx.Provide(NewMarketDataRepository),
)

// Repositories contains all repositories
type Repositories struct {
	OrderRepository     *OrderRepository
	TradeRepository     *TradeRepository
	PositionRepository  *PositionRepository
	RiskRepository      *RiskRepository
	MarketDataRepository *MarketDataRepository
}

// NewRepositories creates all repositories
func NewRepositories(
	db *gorm.DB,
	logger *zap.Logger,
) *Repositories {
	return &Repositories{
		OrderRepository:     NewOrderRepository(db, logger),
		TradeRepository:     NewTradeRepository(db, logger),
		PositionRepository:  NewPositionRepository(db, logger),
		RiskRepository:      NewRiskRepository(db, logger),
		MarketDataRepository: NewMarketDataRepository(db, logger),
	}
}
