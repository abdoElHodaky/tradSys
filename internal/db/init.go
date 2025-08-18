package db

import (
	"github.com/abdoElHodaky/tradSys/internal/db/query"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// InitializeDatabase sets up the database with optimizations
func InitializeDatabase(db *gorm.DB, logger *zap.Logger) error {
	// Create optimizer
	optimizer := query.NewOptimizer(db, logger)
	
	// Enable SQLite optimizations
	if err := optimizer.EnableQueryOptimizations(); err != nil {
		logger.Error("Failed to enable database optimizations", zap.Error(err))
		return err
	}
	
	// Run migrations
	if err := MigrateSchema(db, logger); err != nil {
		logger.Error("Failed to migrate database schema", zap.Error(err))
		return err
	}
	
	// Optimize tables after migration
	tables := []string{"orders", "trades", "positions", "quotes", "ohlcv", "market_depths", 
		"risk_limits", "circuit_breakers", "risk_checks", "strategies", "strategy_executions", "signals"}
	
	for _, table := range tables {
		if err := optimizer.OptimizeTable(table); err != nil {
			logger.Warn("Failed to optimize table", zap.String("table", table), zap.Error(err))
			// Continue with other tables even if one fails
		}
	}
	
	// Create indexes for common queries
	createCommonIndexes(db, optimizer, logger)
	
	logger.Info("Database initialized with optimizations")
	return nil
}

// createCommonIndexes creates indexes for common query patterns
func createCommonIndexes(db *gorm.DB, optimizer *query.Optimizer, logger *zap.Logger) {
	// Order indexes
	orderIndexes := []struct {
		name    string
		columns []string
		unique  bool
	}{
		{"idx_orders_symbol_status", []string{"symbol", "status"}, false},
		{"idx_orders_client_id", []string{"client_id"}, false},
		{"idx_orders_created_at", []string{"created_at"}, false},
	}
	
	for _, idx := range orderIndexes {
		if err := optimizer.CreateIndex("orders", idx.name, idx.columns, idx.unique); err != nil {
			logger.Warn("Failed to create index", zap.String("index", idx.name), zap.Error(err))
		}
	}
	
	// Trade indexes
	tradeIndexes := []struct {
		name    string
		columns []string
		unique  bool
	}{
		{"idx_trades_order_id", []string{"order_id"}, false},
		{"idx_trades_symbol_timestamp", []string{"symbol", "timestamp"}, false},
	}
	
	for _, idx := range tradeIndexes {
		if err := optimizer.CreateIndex("trades", idx.name, idx.columns, idx.unique); err != nil {
			logger.Warn("Failed to create index", zap.String("index", idx.name), zap.Error(err))
		}
	}
	
	// Quote indexes
	quoteIndexes := []struct {
		name    string
		columns []string
		unique  bool
	}{
		{"idx_quotes_timestamp", []string{"timestamp"}, false},
	}
	
	for _, idx := range quoteIndexes {
		if err := optimizer.CreateIndex("quotes", idx.name, idx.columns, idx.unique); err != nil {
			logger.Warn("Failed to create index", zap.String("index", idx.name), zap.Error(err))
		}
	}
	
	// Other indexes for remaining tables would be added here
}

