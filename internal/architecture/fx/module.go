package fx

import (
	"github.com/abdoElHodaky/tradSys/internal/core/matching"
	"github.com/abdoElHodaky/tradSys/internal/db"
	"github.com/abdoElHodaky/tradSys/internal/db/repositories"
	"github.com/abdoElHodaky/tradSys/internal/marketdata"
	"github.com/abdoElHodaky/tradSys/internal/marketdata/external"
	"github.com/abdoElHodaky/tradSys/internal/orders"
	"github.com/abdoElHodaky/tradSys/internal/risk"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Module provides the core architecture module for the fx application
var Module = fx.Options(
	// Provide core services
	fx.Provide(func(logger *zap.Logger) *order_matching.Engine {
		return order_matching.NewEngine(logger)
	}),

	// Include database module
	fx.Options(db.Module),

	// Include repositories module
	fx.Options(repositories.RepositoriesModule),
)

// NewOrdersModule creates a new orders module for the fx application
func NewOrdersModule() fx.Option {
	return fx.Options(
		// Provide order matching engine
		fx.Provide(func(logger *zap.Logger) *order_matching.Engine {
			return order_matching.NewEngine(logger)
		}),

		// Provide order management service
		fx.Provide(func(engine *order_matching.Engine, logger *zap.Logger) *orders.Service {
			return orders.NewService(engine, logger)
		}),
	)
}

// NewRiskModule creates a new risk management module for the fx application
func NewRiskModule() fx.Option {
	return fx.Options(
		// Provide risk management service
		fx.Provide(risk.NewFxService),
	)
}

// NewMarketDataModule creates a new market data module for the fx application
func NewMarketDataModule() fx.Option {
	return fx.Options(
		// Provide market data handler
		fx.Provide(func(engine *order_matching.Engine, logger *zap.Logger) *marketdata.Handler {
			return marketdata.NewHandler(engine, logger)
		}),

		// Include external market data module
		fx.Options(external.Module),
	)
}
