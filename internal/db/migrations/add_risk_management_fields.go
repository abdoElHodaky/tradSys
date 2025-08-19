package migrations

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// AddRiskManagementFields adds stop loss and take profit fields to the orders table
func AddRiskManagementFields(ctx context.Context, db *sqlx.DB, logger *zap.Logger) error {
	logger.Info("Running migration: AddRiskManagementFields")

	// Check if columns already exist
	var count int
	err := db.GetContext(ctx, &count, `
		SELECT COUNT(*) 
		FROM information_schema.columns 
		WHERE table_name = 'orders' 
		AND column_name IN ('stop_loss', 'take_profit', 'strategy', 'timestamp')
	`)
	if err != nil {
		return fmt.Errorf("failed to check if columns exist: %w", err)
	}

	// If all columns already exist, skip migration
	if count == 4 {
		logger.Info("Migration AddRiskManagementFields already applied")
		return nil
	}

	// Add columns if they don't exist
	_, err = db.ExecContext(ctx, `
		ALTER TABLE orders 
		ADD COLUMN IF NOT EXISTS stop_loss FLOAT DEFAULT 0,
		ADD COLUMN IF NOT EXISTS take_profit FLOAT DEFAULT 0,
		ADD COLUMN IF NOT EXISTS strategy VARCHAR(255),
		ADD COLUMN IF NOT EXISTS timestamp TIMESTAMP;
	`)
	if err != nil {
		return fmt.Errorf("failed to add risk management fields: %w", err)
	}

	// Add index on strategy column
	_, err = db.ExecContext(ctx, `
		CREATE INDEX IF NOT EXISTS idx_orders_strategy ON orders(strategy);
	`)
	if err != nil {
		return fmt.Errorf("failed to create index on strategy column: %w", err)
	}

	// Add index on timestamp column
	_, err = db.ExecContext(ctx, `
		CREATE INDEX IF NOT EXISTS idx_orders_timestamp ON orders(timestamp);
	`)
	if err != nil {
		return fmt.Errorf("failed to create index on timestamp column: %w", err)
	}

	logger.Info("Migration AddRiskManagementFields completed successfully")
	return nil
}
