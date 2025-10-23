package db

import (
	"fmt"
	"time"

	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DBConfig represents the database configuration
// Renamed to avoid conflict with UnifiedConfig
type DBConfig struct {
	// Host is the database host
	Host string
	// Port is the database port
	Port int
	// Username is the database username
	Username string
	// Password is the database password
	Password string
	// Database is the database name
	Database string
	// SSLMode is the SSL mode
	SSLMode string
	// MaxOpenConns is the maximum number of open connections
	MaxOpenConns int
	// MaxIdleConns is the maximum number of idle connections
	MaxIdleConns int
	// ConnMaxLifetime is the maximum lifetime of a connection
	ConnMaxLifetime time.Duration
}

// DefaultConfig returns the default database configuration
func DefaultConfig() *DBConfig {
	return &DBConfig{
		Host:            "localhost",
		Port:            5432,
		Username:        "postgres",
		Password:        "postgres",
		Database:        "tradsys",
		SSLMode:         "disable",
		MaxOpenConns:    25,
		MaxIdleConns:    10,
		ConnMaxLifetime: time.Hour,
	}
}

// DSN returns the database connection string
func (c *DBConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.Username, c.Password, c.Database, c.SSLMode,
	)
}

// Connect connects to the database
func Connect(config *DBConfig, zapLogger *zap.Logger) (*gorm.DB, error) {
	// Create a GORM logger that uses zap
	gormLogger := logger.New(
		&zapGormWriter{zapLogger: zapLogger},
		logger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  logger.Warn,
			IgnoreRecordNotFoundError: true,
			Colorful:                  false,
		},
	)

	// Connect to the database
	db, err := gorm.Open(postgres.Open(config.DSN()), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		return nil, err
	}

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	sqlDB.SetMaxOpenConns(config.MaxOpenConns)
	sqlDB.SetMaxIdleConns(config.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(config.ConnMaxLifetime)

	return db, nil
}

// zapGormWriter is a logger.Writer implementation that uses zap
type zapGormWriter struct {
	zapLogger *zap.Logger
}

// Printf implements the logger.Writer interface
func (w *zapGormWriter) Printf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	w.zapLogger.Debug("gorm", zap.String("msg", msg))
}

// Removed duplicate InitializeDatabase function - using the comprehensive one in init.go

// runMigrations runs database migrations
func runMigrations(db *gorm.DB, logger *zap.Logger) error {
	logger.Info("Running database migrations")

	// Auto-migrate models
	if err := db.AutoMigrate(
		&Order{},
		&Trade{},
		&Position{},
		&RiskLimit{},
		&MarketData{},
	); err != nil {
		return err
	}

	// Create indexes
	if err := createIndexes(db, logger); err != nil {
		return err
	}

	return nil
}

// createIndexes creates database indexes
func createIndexes(db *gorm.DB, logger *zap.Logger) error {
	logger.Info("Creating database indexes")

	// Create indexes for orders
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_orders_user_id ON orders(user_id)").Error; err != nil {
		return err
	}
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_orders_symbol ON orders(symbol)").Error; err != nil {
		return err
	}
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status)").Error; err != nil {
		return err
	}
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_orders_created_at ON orders(created_at)").Error; err != nil {
		return err
	}
	if err := db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_orders_client_order_id ON orders(client_order_id) WHERE client_order_id != ''").Error; err != nil {
		return err
	}

	// Create indexes for trades
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_trades_order_id ON trades(order_id)").Error; err != nil {
		return err
	}
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_trades_symbol ON trades(symbol)").Error; err != nil {
		return err
	}
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_trades_executed_at ON trades(executed_at)").Error; err != nil {
		return err
	}

	// Create indexes for positions
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_positions_user_id ON positions(user_id)").Error; err != nil {
		return err
	}
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_positions_symbol ON positions(symbol)").Error; err != nil {
		return err
	}
	if err := db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_positions_user_id_symbol ON positions(user_id, symbol)").Error; err != nil {
		return err
	}

	// Create indexes for risk limits
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_risk_limits_user_id ON risk_limits(user_id)").Error; err != nil {
		return err
	}
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_risk_limits_type ON risk_limits(type)").Error; err != nil {
		return err
	}

	// Create indexes for market data
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_market_data_symbol ON market_data(symbol)").Error; err != nil {
		return err
	}
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_market_data_type ON market_data(type)").Error; err != nil {
		return err
	}
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_market_data_timestamp ON market_data(timestamp)").Error; err != nil {
		return err
	}

	return nil
}
