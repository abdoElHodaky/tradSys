package orders

import (
	"github.com/abdoElHodaky/tradSys/internal/db/repositories"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Module provides the order service module for fx
var Module = fx.Options(
	// Provide the order repository
	fx.Provide(func(db *repositories.DB, logger *zap.Logger) *repositories.OrderRepository {
		return repositories.NewOrderRepository(db.DB, logger)
	}),
	
	// Provide the order service
	fx.Provide(NewService),
)

