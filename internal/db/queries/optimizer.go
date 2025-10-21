package queries

import (
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Optimizer provides query optimization utilities
type Optimizer struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewOptimizer creates a new query optimizer
func NewOptimizer(db *gorm.DB, logger *zap.Logger) *Optimizer {
	return &Optimizer{
		db:     db,
		logger: logger,
	}
}

// AnalyzeQuery analyzes a query and returns execution plan
func (o *Optimizer) AnalyzeQuery(query string, args ...interface{}) (string, error) {
	rows, err := o.db.Raw(fmt.Sprintf("EXPLAIN QUERY PLAN %s", query), args...).Rows()
	if err != nil {
		return "", err
	}
	defer rows.Close()

	var planBuilder strings.Builder
	for rows.Next() {
		var id, parent, notused int
		var detail string
		if err := rows.Scan(&id, &parent, &notused, &detail); err != nil {
			return "", err
		}
		planBuilder.WriteString(fmt.Sprintf("ID: %d, Parent: %d, Detail: %s\n", id, parent, detail))
	}

	return planBuilder.String(), nil
}

// OptimizeTable analyzes and optimizes a table
func (o *Optimizer) OptimizeTable(table string) error {
	// For SQLite, run ANALYZE to update statistics
	result := o.db.Exec(fmt.Sprintf("ANALYZE %s", table))
	if result.Error != nil {
		o.logger.Error("Failed to optimize table",
			zap.String("table", table),
			zap.Error(result.Error))
		return result.Error
	}
	
	o.logger.Info("Table optimized",
		zap.String("table", table))
	return nil
}

// CreateIndex creates an index if it doesn't exist
func (o *Optimizer) CreateIndex(table, indexName string, columns []string, unique bool) error {
	uniqueStr := ""
	if unique {
		uniqueStr = "UNIQUE"
	}
	
	query := fmt.Sprintf("CREATE %s INDEX IF NOT EXISTS %s ON %s (%s)",
		uniqueStr, indexName, table, strings.Join(columns, ", "))
	
	result := o.db.Exec(query)
	if result.Error != nil {
		o.logger.Error("Failed to create index",
			zap.String("table", table),
			zap.String("index", indexName),
			zap.Error(result.Error))
		return result.Error
	}
	
	o.logger.Info("Index created or already exists",
		zap.String("table", table),
		zap.String("index", indexName))
	return nil
}

// GetSlowQueries returns recent slow queries
func (o *Optimizer) GetSlowQueries(threshold time.Duration) ([]map[string]interface{}, error) {
	// This requires SQLite query logging to be enabled
	// For a production system, you would implement a custom query logger
	var results []map[string]interface{}
	
	// This is a placeholder - in a real system you would query your query log table
	// For demonstration purposes only
	return results, nil
}

// EnableQueryOptimizations enables SQLite optimizations
func (o *Optimizer) EnableQueryOptimizations() error {
	// Get raw SQL connection
	sqlDB, err := o.db.DB()
	if err != nil {
		return err
	}
	
	// Set pragmas for optimization
	pragmas := []string{
		"PRAGMA journal_mode=WAL",           // Use Write-Ahead Logging
		"PRAGMA synchronous=NORMAL",         // Sync less often for better performance
		"PRAGMA cache_size=-102400",         // Use 100MB of memory for DB cache
		"PRAGMA mmap_size=1073741824",       // Memory map up to 1GB
		"PRAGMA temp_store=MEMORY",          // Store temp tables in memory
		"PRAGMA foreign_keys=ON",            // Enable foreign key constraints
		"PRAGMA auto_vacuum=INCREMENTAL",    // Incremental vacuum
	}
	
	for _, pragma := range pragmas {
		if _, err := sqlDB.Exec(pragma); err != nil {
			o.logger.Error("Failed to set pragma",
				zap.String("pragma", pragma),
				zap.Error(err))
			return err
		}
	}
	
	o.logger.Info("SQLite optimizations enabled")
	return nil
}
