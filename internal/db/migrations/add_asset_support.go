package migrations

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// AddAssetSupport adds tables and columns for multi-asset support
func AddAssetSupport(ctx context.Context, db *sqlx.DB, logger *zap.Logger) error {
	logger.Info("Running migration: AddAssetSupport")

	// Create asset_metadata table
	_, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS asset_metadata (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			symbol VARCHAR(50) UNIQUE NOT NULL,
			asset_type VARCHAR(20) NOT NULL,
			sector VARCHAR(100),
			industry VARCHAR(100),
			country VARCHAR(10),
			currency VARCHAR(10),
			exchange VARCHAR(50),
			attributes TEXT,
			is_active BOOLEAN DEFAULT TRUE,
			last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			deleted_at TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_asset_metadata_symbol ON asset_metadata(symbol);
		CREATE INDEX IF NOT EXISTS idx_asset_metadata_asset_type ON asset_metadata(asset_type);
		CREATE INDEX IF NOT EXISTS idx_asset_metadata_sector ON asset_metadata(sector);
		CREATE INDEX IF NOT EXISTS idx_asset_metadata_exchange ON asset_metadata(exchange);
		CREATE INDEX IF NOT EXISTS idx_asset_metadata_is_active ON asset_metadata(is_active);
	`)
	if err != nil {
		return fmt.Errorf("failed to create asset_metadata table: %w", err)
	}

	// Create asset_configurations table
	_, err = db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS asset_configurations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			asset_type VARCHAR(20) UNIQUE NOT NULL,
			trading_enabled BOOLEAN DEFAULT TRUE,
			min_order_size DECIMAL(20,8),
			max_order_size DECIMAL(20,8),
			price_increment DECIMAL(20,8),
			quantity_increment DECIMAL(20,8),
			trading_hours VARCHAR(100),
			settlement_days INTEGER DEFAULT 2,
			requires_approval BOOLEAN DEFAULT FALSE,
			risk_multiplier DECIMAL(10,4) DEFAULT 1.0,
			configuration TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			deleted_at TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_asset_configurations_asset_type ON asset_configurations(asset_type);
	`)
	if err != nil {
		return fmt.Errorf("failed to create asset_configurations table: %w", err)
	}

	// Create asset_pricing table
	_, err = db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS asset_pricing (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			symbol VARCHAR(50) NOT NULL,
			asset_type VARCHAR(20) NOT NULL,
			price DECIMAL(20,8) NOT NULL,
			bid_price DECIMAL(20,8),
			ask_price DECIMAL(20,8),
			volume DECIMAL(20,8),
			high_24h DECIMAL(20,8),
			low_24h DECIMAL(20,8),
			change_24h DECIMAL(20,8),
			change_percent_24h DECIMAL(10,4),
			market_cap DECIMAL(30,8),
			timestamp TIMESTAMP NOT NULL,
			source VARCHAR(50),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			deleted_at TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_asset_pricing_symbol ON asset_pricing(symbol);
		CREATE INDEX IF NOT EXISTS idx_asset_pricing_asset_type ON asset_pricing(asset_type);
		CREATE INDEX IF NOT EXISTS idx_asset_pricing_timestamp ON asset_pricing(timestamp);
	`)
	if err != nil {
		return fmt.Errorf("failed to create asset_pricing table: %w", err)
	}

	// Create asset_dividends table
	_, err = db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS asset_dividends (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			symbol VARCHAR(50) NOT NULL,
			asset_type VARCHAR(20) NOT NULL,
			ex_date TIMESTAMP NOT NULL,
			pay_date TIMESTAMP NOT NULL,
			record_date TIMESTAMP,
			amount DECIMAL(20,8) NOT NULL,
			currency VARCHAR(10),
			dividend_type VARCHAR(20),
			frequency VARCHAR(20),
			yield_percent DECIMAL(10,4),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			deleted_at TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_asset_dividends_symbol ON asset_dividends(symbol);
		CREATE INDEX IF NOT EXISTS idx_asset_dividends_asset_type ON asset_dividends(asset_type);
		CREATE INDEX IF NOT EXISTS idx_asset_dividends_ex_date ON asset_dividends(ex_date);
	`)
	if err != nil {
		return fmt.Errorf("failed to create asset_dividends table: %w", err)
	}

	// Add asset_type column to existing orders table if it doesn't exist
	_, err = db.ExecContext(ctx, `
		ALTER TABLE orders ADD COLUMN asset_type VARCHAR(20) DEFAULT 'STOCK';
	`)
	if err != nil {
		// Column might already exist, check if it's a "duplicate column" error
		logger.Warn("Could not add asset_type column to orders table (might already exist)", zap.Error(err))
	}

	// Add asset_metadata column to existing orders table if it doesn't exist
	_, err = db.ExecContext(ctx, `
		ALTER TABLE orders ADD COLUMN asset_metadata TEXT;
	`)
	if err != nil {
		// Column might already exist, check if it's a "duplicate column" error
		logger.Warn("Could not add asset_metadata column to orders table (might already exist)", zap.Error(err))
	}

	// Create index on asset_type column in orders table
	_, err = db.ExecContext(ctx, `
		CREATE INDEX IF NOT EXISTS idx_orders_asset_type ON orders(asset_type);
	`)
	if err != nil {
		logger.Warn("Could not create index on orders.asset_type", zap.Error(err))
	}

	// Insert default asset configurations
	_, err = db.ExecContext(ctx, `
		INSERT OR IGNORE INTO asset_configurations (
			asset_type, trading_enabled, min_order_size, max_order_size, 
			price_increment, quantity_increment, trading_hours, settlement_days,
			requires_approval, risk_multiplier
		) VALUES 
		('STOCK', TRUE, 1.0, 1000000.0, 0.01, 1.0, '09:30-16:00 EST', 2, FALSE, 1.0),
		('REIT', TRUE, 1.0, 1000000.0, 0.01, 1.0, '09:30-16:00 EST', 2, FALSE, 1.2),
		('MUTUAL_FUND', TRUE, 1.0, 1000000.0, 0.01, 0.001, '16:00 EST', 1, FALSE, 1.1),
		('ETF', TRUE, 1.0, 1000000.0, 0.01, 1.0, '09:30-16:00 EST', 2, FALSE, 1.0),
		('BOND', TRUE, 100.0, 10000000.0, 0.01, 100.0, '08:00-17:00 EST', 3, TRUE, 0.8),
		('CRYPTO', TRUE, 0.001, 1000000.0, 0.01, 0.00001, '24/7', 0, FALSE, 2.0),
		('FOREX', TRUE, 1000.0, 100000000.0, 0.00001, 1000.0, '24/5', 2, FALSE, 1.5),
		('COMMODITY', TRUE, 1.0, 1000000.0, 0.01, 1.0, 'Market dependent', 2, FALSE, 1.3);
	`)
	if err != nil {
		return fmt.Errorf("failed to insert default asset configurations: %w", err)
	}

	logger.Info("Migration AddAssetSupport completed successfully")
	return nil
}
