package db

import (
	"fmt"
	"time"

	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Config holds database configuration
type Config struct {
	Path            string
	MaxIdleConns    int
	MaxOpenConns    int
	ConnMaxLifetime time.Duration
}

// DefaultConfig returns a default SQLite configuration
func DefaultConfig() *Config {
	return &Config{
		Path:            "tradesys.db",
		MaxIdleConns:    10,
		MaxOpenConns:    50,
		ConnMaxLifetime: time.Hour,
	}
}

// Connect establishes a connection to the SQLite database
func Connect(config *Config, zapLogger *zap.Logger) (*gorm.DB, error) {
	// Configure GORM logger
	gormLogger := logger.New(
		&zapAdapter{zapLogger},
		logger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  logger.Warn,
			IgnoreRecordNotFoundError: true,
			Colorful:                  false,
		},
	)

	// Connect to SQLite
	db, err := gorm.Open(sqlite.Open(config.Path), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database: %w", err)
	}

	sqlDB.SetMaxIdleConns(config.MaxIdleConns)
	sqlDB.SetMaxOpenConns(config.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(config.ConnMaxLifetime)

	// Enable WAL mode for better performance
	if _, err := sqlDB.Exec("PRAGMA journal_mode=WAL;"); err != nil {
		zapLogger.Warn("Failed to enable WAL mode", zap.Error(err))
	}

	// Use memory-mapped I/O for better performance
	if _, err := sqlDB.Exec("PRAGMA mmap_size=1073741824;"); err != nil { // 1GB
		zapLogger.Warn("Failed to set mmap_size", zap.Error(err))
	}

	// Other performance optimizations
	pragmas := []string{
		"PRAGMA synchronous=NORMAL",         // Sync less often for better performance
		"PRAGMA cache_size=-102400",         // Use 100MB of memory for DB cache
		"PRAGMA temp_store=MEMORY",          // Store temp tables in memory
		"PRAGMA foreign_keys=ON",            // Enable foreign key constraints
		"PRAGMA auto_vacuum=INCREMENTAL",    // Incremental vacuum
	}

	for _, pragma := range pragmas {
		if _, err := sqlDB.Exec(pragma); err != nil {
			zapLogger.Warn("Failed to set pragma", zap.String("pragma", pragma), zap.Error(err))
		}
	}

	return db, nil
}

// zapAdapter adapts zap.Logger to GORM's logger interface
type zapAdapter struct {
	*zap.Logger
}

func (a *zapAdapter) Printf(format string, args ...interface{}) {
	a.Sugar().Debugf(format, args...)
}

