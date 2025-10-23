package config

import (
	"database/sql"
	"fmt"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// HFTDatabaseConfig contains HFT-specific database configuration
type HFTDatabaseConfig struct {
	// SQLite settings
	WALMode         bool  `yaml:"wal_mode" default:"true"`
	CacheSize       int   `yaml:"cache_size" default:"10000"`    // 10MB cache
	MMapSize        int64 `yaml:"mmap_size" default:"268435456"` // 256MB memory mapping
	TempStoreMemory bool  `yaml:"temp_store_memory" default:"true"`

	// Connection settings
	MaxOpenConns    int           `yaml:"max_open_conns" default:"1"`     // SQLite is single-writer
	MaxIdleConns    int           `yaml:"max_idle_conns" default:"1"`     // Keep connection alive
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime" default:"1h"` // Long-lived connections

	// Performance settings
	Synchronous string `yaml:"synchronous" default:"NORMAL"` // NORMAL, FULL, OFF
	JournalMode string `yaml:"journal_mode" default:"WAL"`   // WAL, DELETE, TRUNCATE
	BusyTimeout int    `yaml:"busy_timeout" default:"30000"` // 30 seconds

	// HFT-specific settings
	PreparedStmts bool `yaml:"prepared_stmts" default:"true"`
	DisableFK     bool `yaml:"disable_fk" default:"true"`    // Skip FK checks for speed
	SilentLogger  bool `yaml:"silent_logger" default:"true"` // Disable query logging
}

// NewHFTDatabase creates a GORM database instance optimized for HFT workloads
func NewHFTDatabase(dbPath string, config *HFTDatabaseConfig) (*gorm.DB, error) {
	if config == nil {
		config = &HFTDatabaseConfig{
			WALMode:         true,
			CacheSize:       10000,
			MMapSize:        268435456,
			TempStoreMemory: true,
			MaxOpenConns:    1,
			MaxIdleConns:    1,
			ConnMaxLifetime: time.Hour,
			Synchronous:     "NORMAL",
			JournalMode:     "WAL",
			BusyTimeout:     30000,
			PreparedStmts:   true,
			DisableFK:       true,
			SilentLogger:    true,
		}
	}

	// Configure GORM
	gormConfig := &gorm.Config{
		PrepareStmt:                              config.PreparedStmts,
		DisableForeignKeyConstraintWhenMigrating: config.DisableFK,
	}

	// Set logger level
	if config.SilentLogger {
		gormConfig.Logger = logger.Default.LogMode(logger.Silent)
	}

	// Open database connection
	db, err := gorm.Open(sqlite.Open(dbPath), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Get underlying SQL DB for configuration
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying SQL DB: %w", err)
	}

	// Configure connection pool
	sqlDB.SetMaxOpenConns(config.MaxOpenConns)
	sqlDB.SetMaxIdleConns(config.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(config.ConnMaxLifetime)

	// Apply HFT-optimized SQLite pragmas
	if err := applyHFTPragmas(db, config); err != nil {
		return nil, fmt.Errorf("failed to apply HFT pragmas: %w", err)
	}

	return db, nil
}

// applyHFTPragmas applies HFT-optimized SQLite pragmas
func applyHFTPragmas(db *gorm.DB, config *HFTDatabaseConfig) error {
	pragmas := []struct {
		name  string
		value interface{}
	}{
		{"journal_mode", config.JournalMode},
		{"synchronous", config.Synchronous},
		{"cache_size", config.CacheSize},
		{"mmap_size", config.MMapSize},
		{"busy_timeout", config.BusyTimeout},
	}

	if config.TempStoreMemory {
		pragmas = append(pragmas, struct {
			name  string
			value interface{}
		}{"temp_store", "memory"})
	}

	// Additional HFT optimizations
	additionalPragmas := []struct {
		name  string
		value interface{}
	}{
		{"page_size", 4096},            // 4KB page size for better performance
		{"auto_vacuum", "INCREMENTAL"}, // Incremental vacuum for consistent performance
		{"secure_delete", "OFF"},       // Disable secure delete for speed
		{"count_changes", "OFF"},       // Disable change counting
		{"legacy_file_format", "OFF"},  // Use modern file format
		{"read_uncommitted", "ON"},     // Allow dirty reads for better performance
	}

	pragmas = append(pragmas, additionalPragmas...)

	// Apply all pragmas
	for _, pragma := range pragmas {
		sql := fmt.Sprintf("PRAGMA %s = %v", pragma.name, pragma.value)
		if err := db.Exec(sql).Error; err != nil {
			return fmt.Errorf("failed to execute pragma %s: %w", pragma.name, err)
		}
	}

	return nil
}

// NewHFTSQLDatabase creates a raw SQL database connection optimized for HFT
func NewHFTSQLDatabase(dbPath string, config *HFTDatabaseConfig) (*sql.DB, error) {
	if config == nil {
		config = &HFTDatabaseConfig{
			WALMode:         true,
			CacheSize:       10000,
			MMapSize:        268435456,
			TempStoreMemory: true,
			MaxOpenConns:    1,
			MaxIdleConns:    1,
			ConnMaxLifetime: time.Hour,
			Synchronous:     "NORMAL",
			JournalMode:     "WAL",
			BusyTimeout:     30000,
		}
	}

	// Open raw SQL connection
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open SQL database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(config.MaxOpenConns)
	db.SetMaxIdleConns(config.MaxIdleConns)
	db.SetConnMaxLifetime(config.ConnMaxLifetime)

	// Apply HFT pragmas using raw SQL
	if err := applyHFTPragmasSQL(db, config); err != nil {
		return nil, fmt.Errorf("failed to apply HFT pragmas: %w", err)
	}

	return db, nil
}

// applyHFTPragmasSQL applies HFT-optimized SQLite pragmas using raw SQL
func applyHFTPragmasSQL(db *sql.DB, config *HFTDatabaseConfig) error {
	pragmas := []string{
		fmt.Sprintf("PRAGMA journal_mode = %s", config.JournalMode),
		fmt.Sprintf("PRAGMA synchronous = %s", config.Synchronous),
		fmt.Sprintf("PRAGMA cache_size = %d", config.CacheSize),
		fmt.Sprintf("PRAGMA mmap_size = %d", config.MMapSize),
		fmt.Sprintf("PRAGMA busy_timeout = %d", config.BusyTimeout),
		"PRAGMA page_size = 4096",
		"PRAGMA auto_vacuum = INCREMENTAL",
		"PRAGMA secure_delete = OFF",
		"PRAGMA count_changes = OFF",
		"PRAGMA legacy_file_format = OFF",
		"PRAGMA read_uncommitted = ON",
	}

	if config.TempStoreMemory {
		pragmas = append(pragmas, "PRAGMA temp_store = memory")
	}

	// Execute all pragmas
	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			return fmt.Errorf("failed to execute pragma '%s': %w", pragma, err)
		}
	}

	return nil
}

// ValidateHFTDatabase validates that HFT optimizations are applied correctly
func ValidateHFTDatabase(db *gorm.DB) error {
	validations := []struct {
		pragma   string
		expected string
	}{
		{"journal_mode", "wal"},
		{"synchronous", "1"}, // NORMAL = 1
		{"temp_store", "2"},  // memory = 2
	}

	for _, validation := range validations {
		var result string
		sql := fmt.Sprintf("PRAGMA %s", validation.pragma)
		if err := db.Raw(sql).Scan(&result).Error; err != nil {
			return fmt.Errorf("failed to validate pragma %s: %w", validation.pragma, err)
		}

		if result != validation.expected {
			return fmt.Errorf("pragma %s validation failed: expected %s, got %s",
				validation.pragma, validation.expected, result)
		}
	}

	return nil
}

// GetDatabaseStats returns database performance statistics
func GetDatabaseStats(db *gorm.DB) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Get SQLite stats
	sqliteStats := []string{
		"cache_size", "page_count", "page_size", "freelist_count",
		"journal_mode", "synchronous", "temp_store", "mmap_size",
	}

	for _, stat := range sqliteStats {
		var value interface{}
		sql := fmt.Sprintf("PRAGMA %s", stat)
		if err := db.Raw(sql).Scan(&value).Error; err != nil {
			return nil, fmt.Errorf("failed to get stat %s: %w", stat, err)
		}
		stats[stat] = value
	}

	// Get connection pool stats
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying SQL DB: %w", err)
	}

	dbStats := sqlDB.Stats()
	stats["max_open_connections"] = dbStats.MaxOpenConnections
	stats["open_connections"] = dbStats.OpenConnections
	stats["in_use"] = dbStats.InUse
	stats["idle"] = dbStats.Idle
	stats["wait_count"] = dbStats.WaitCount
	stats["wait_duration"] = dbStats.WaitDuration
	stats["max_idle_closed"] = dbStats.MaxIdleClosed
	stats["max_idle_time_closed"] = dbStats.MaxIdleTimeClosed
	stats["max_lifetime_closed"] = dbStats.MaxLifetimeClosed

	return stats, nil
}
