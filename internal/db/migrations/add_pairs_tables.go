package migrations

import (
	"context"
	"fmt"
	
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// AddPairsTables adds tables for pairs trading
func AddPairsTables(ctx context.Context, db *sqlx.DB, logger *zap.Logger) error {
	logger.Info("Running migration: AddPairsTables")
	
	// Create pairs table
	_, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS pairs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			pair_id VARCHAR(255) UNIQUE NOT NULL,
			symbol1 VARCHAR(255) NOT NULL,
			symbol2 VARCHAR(255) NOT NULL,
			ratio FLOAT NOT NULL,
			status VARCHAR(50) NOT NULL,
			correlation FLOAT,
			cointegration FLOAT,
			z_score_threshold_entry FLOAT,
			z_score_threshold_exit FLOAT,
			lookback_period INTEGER NOT NULL,
			half_life INTEGER,
			created_by INTEGER,
			notes TEXT,
			created_at TIMESTAMP,
			updated_at TIMESTAMP,
			deleted_at TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_pairs_symbol1 ON pairs(symbol1);
		CREATE INDEX IF NOT EXISTS idx_pairs_symbol2 ON pairs(symbol2);
		CREATE INDEX IF NOT EXISTS idx_pairs_status ON pairs(status);
	`)
	if err != nil {
		return fmt.Errorf("failed to create pairs table: %w", err)
	}
	
	// Create pair_statistics table
	_, err = db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS pair_statistics (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			pair_id VARCHAR(255) NOT NULL,
			timestamp TIMESTAMP NOT NULL,
			correlation FLOAT,
			cointegration FLOAT,
			spread_mean FLOAT,
			spread_std_dev FLOAT,
			current_z_score FLOAT,
			spread_value FLOAT,
			created_at TIMESTAMP,
			updated_at TIMESTAMP,
			deleted_at TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_pair_statistics_pair_id ON pair_statistics(pair_id);
		CREATE INDEX IF NOT EXISTS idx_pair_statistics_timestamp ON pair_statistics(timestamp);
	`)
	if err != nil {
		return fmt.Errorf("failed to create pair_statistics table: %w", err)
	}
	
	// Create pair_positions table
	_, err = db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS pair_positions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			pair_id VARCHAR(255) NOT NULL,
			entry_timestamp TIMESTAMP NOT NULL,
			symbol1 VARCHAR(255) NOT NULL,
			symbol2 VARCHAR(255) NOT NULL,
			quantity1 FLOAT NOT NULL,
			quantity2 FLOAT NOT NULL,
			entry_price1 FLOAT NOT NULL,
			entry_price2 FLOAT NOT NULL,
			current_price1 FLOAT,
			current_price2 FLOAT,
			entry_spread FLOAT NOT NULL,
			current_spread FLOAT,
			entry_z_score FLOAT NOT NULL,
			current_z_score FLOAT,
			pnl FLOAT,
			status VARCHAR(50) NOT NULL,
			exit_timestamp TIMESTAMP,
			created_at TIMESTAMP,
			updated_at TIMESTAMP,
			deleted_at TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_pair_positions_pair_id ON pair_positions(pair_id);
		CREATE INDEX IF NOT EXISTS idx_pair_positions_status ON pair_positions(status);
	`)
	if err != nil {
		return fmt.Errorf("failed to create pair_positions table: %w", err)
	}
	
	logger.Info("Migration AddPairsTables completed successfully")
	return nil
}
