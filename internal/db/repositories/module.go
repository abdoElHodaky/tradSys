package repositories

import (
	"database/sql"

	"github.com/abdoElHodaky/tradSys/internal/repositories"
	"github.com/abdoElHodaky/tradSys/pkg/interfaces"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// RepositoriesModule provides the repositories module for the fx application
var RepositoriesModule = fx.Options(
	fx.Provide(NewOrderRepositoryWrapper),
	// TODO: Implement missing repositories
	// fx.Provide(NewTradeRepository),
	// fx.Provide(NewPositionRepository),
	// fx.Provide(NewRiskRepository),
	// fx.Provide(NewMarketDataRepository),
)

// Individual repository modules for specific services
var OrderRepositoryModule = fx.Options(
	fx.Provide(NewOrderRepositoryWrapper),
)

// TODO: Implement missing repository modules
// var RiskRepositoryModule = fx.Options(
// 	fx.Provide(NewRiskRepository),
// )

// var MarketDataRepositoryModule = fx.Options(
// 	fx.Provide(NewMarketDataRepository),
// )

// Repositories contains all repositories
type Repositories struct {
	OrderRepository *repositories.OrderRepository
	// TODO: Add other repositories when implemented
	// TradeRepository      *TradeRepository
	// PositionRepository   *PositionRepository
	// RiskRepository       *RiskRepository
	// MarketDataRepository *MarketDataRepository
}

// NewOrderRepositoryWrapper creates an order repository with the expected signature
func NewOrderRepositoryWrapper(
	db *sql.DB,
	logger interfaces.Logger,
	metrics interfaces.MetricsCollector,
) *repositories.OrderRepository {
	return repositories.NewOrderRepository(db, logger, metrics)
}

// NewRepositories creates all repositories
func NewRepositories(
	orderRepo *repositories.OrderRepository,
) *Repositories {
	return &Repositories{
		OrderRepository: orderRepo,
		// TODO: Add other repositories when implemented
	}
}
