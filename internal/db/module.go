package db

import (
	"context"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Module provides the database module for the fx application
var Module = fx.Options(
	fx.Provide(NewDatabase),
)

// NewDatabase creates a new database connection for the fx application
func NewDatabase(lifecycle fx.Lifecycle, logger *zap.Logger) (*gorm.DB, error) {
	config := DefaultConfig()
	
	// Connect to the database
	db, err := Connect(config, logger)
	if err != nil {
		return nil, err
	}
	
	// Initialize the database
	if err := InitializeDatabase(db, logger); err != nil {
		return nil, err
	}
	
	// Register lifecycle hooks
	lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info("Database connection established")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("Closing database connection")
			sqlDB, err := db.DB()
			if err != nil {
				return err
			}
			return sqlDB.Close()
		},
	})
	
	return db, nil
}
