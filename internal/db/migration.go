package db

import (
	"github.com/abdoElHodaky/tradSys/internal/db/models"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// MigrateSchema performs database schema migration
func MigrateSchema(db *gorm.DB, logger *zap.Logger) error {
	logger.Info("Running database migrations")

	// Define all models to migrate
	models := []interface{}{
		&models.Order{},
		&models.Trade{},
		&models.Position{},
		&models.Quote{},
		&models.OHLCV{},
		&models.MarketDepth{},
		&models.RiskLimit{},
		&models.CircuitBreaker{},
		&models.RiskCheck{},
		&models.Strategy{},
		&models.StrategyExecution{},
		&models.Signal{},
		&models.Pair{},
		&models.PairStatistics{},
		&models.PairPosition{},
	}

	// Run migrations
	err := db.AutoMigrate(models...)
	if err != nil {
		logger.Error("Database migration failed", zap.Error(err))
		return err
	}

	logger.Info("Database migration completed successfully")
	return nil
}
