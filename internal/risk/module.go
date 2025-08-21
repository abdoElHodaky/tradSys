package risk

import (
	"github.com/abdoElHodaky/tradSys/internal/db/repositories"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Module provides the risk service module for fx
var Module = fx.Options(
	// Provide the risk repository
	fx.Provide(func(db *gorm.DB, logger *zap.Logger) *repositories.RiskRepository {
		return repositories.NewRiskRepository(db, logger)
	}),
	
	// Provide the risk service
	fx.Provide(NewService),
)

