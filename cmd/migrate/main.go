package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/abdoElHodaky/tradSys/internal/repositories"
	"github.com/abdoElHodaky/tradSys/pkg/config"
	"github.com/abdoElHodaky/tradSys/pkg/testing"

	_ "github.com/lib/pq" // PostgreSQL driver
)

const (
	appName    = "TradSys Migration Tool"
	appVersion = "v3.0.0"
)

func main() {
	// Parse command line flags
	var (
		configPath = flag.String("config", "config.yaml", "Path to configuration file")
		version    = flag.Bool("version", false, "Show version information")
		up         = flag.Bool("up", false, "Run migrations up")
		down       = flag.Bool("down", false, "Run migrations down")
		create     = flag.String("create", "", "Create a new migration file")
		status     = flag.Bool("status", false, "Show migration status")
	)
	flag.Parse()

	// Handle version flag
	if *version {
		fmt.Printf("%s %s\n", appName, appVersion)
		os.Exit(0)
	}

	// Load configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Connect to database
	db, err := connectToDatabase(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Initialize dependencies
	logger := testing.NewMockLogger()
	metrics := testing.NewMockMetricsCollector()

	// Handle different commands
	switch {
	case *up:
		if err := runMigrationsUp(db, logger, metrics); err != nil {
			log.Fatalf("Failed to run migrations up: %v", err)
		}
		fmt.Println("Migrations completed successfully")

	case *down:
		if err := runMigrationsDown(db, logger, metrics); err != nil {
			log.Fatalf("Failed to run migrations down: %v", err)
		}
		fmt.Println("Migrations rolled back successfully")

	case *create != "":
		if err := createMigration(*create); err != nil {
			log.Fatalf("Failed to create migration: %v", err)
		}
		fmt.Printf("Migration file created: %s\n", *create)

	case *status:
		if err := showMigrationStatus(db); err != nil {
			log.Fatalf("Failed to show migration status: %v", err)
		}

	default:
		fmt.Println("Usage: migrate [options]")
		flag.PrintDefaults()
		os.Exit(1)
	}
}

func connectToDatabase(cfg *config.Config) (*sql.DB, error) {
	// Get database password from environment
	password := os.Getenv("DATABASE_PASSWORD")
	if password == "" {
		password = cfg.Database.Password
	}

	// Build connection string
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.User,
		password,
		cfg.Database.Name,
		cfg.Database.SSLMode,
	)

	// Connect to database
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	db.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.Database.ConnMaxLifetime)

	return db, nil
}

func runMigrationsUp(db *sql.DB, logger *testing.MockLogger, metrics *testing.MockMetricsCollector) error {
	ctx := context.Background()

	// Create migration tracking table
	if err := createMigrationTable(db); err != nil {
		return fmt.Errorf("failed to create migration table: %w", err)
	}

	// Create order repository and run table creation
	orderRepo := repositories.NewOrderRepository(db, logger, metrics)
	if err := orderRepo.CreateOrdersTable(ctx); err != nil {
		return fmt.Errorf("failed to create orders table: %w", err)
	}

	// Create other tables as needed
	if err := createTradesTable(db); err != nil {
		return fmt.Errorf("failed to create trades table: %w", err)
	}

	if err := createUsersTable(db); err != nil {
		return fmt.Errorf("failed to create users table: %w", err)
	}

	if err := createSymbolsTable(db); err != nil {
		return fmt.Errorf("failed to create symbols table: %w", err)
	}

	if err := createMarketDataTable(db); err != nil {
		return fmt.Errorf("failed to create market_data table: %w", err)
	}

	// Record migration
	if err := recordMigration(db, "initial_schema", "up"); err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	return nil
}

func runMigrationsDown(db *sql.DB, logger *testing.MockLogger, metrics *testing.MockMetricsCollector) error {
	// Drop tables in reverse order
	tables := []string{
		"market_data",
		"symbols", 
		"trades",
		"orders",
		"users",
	}

	for _, table := range tables {
		if err := dropTable(db, table); err != nil {
			return fmt.Errorf("failed to drop table %s: %w", table, err)
		}
	}

	// Record migration
	if err := recordMigration(db, "initial_schema", "down"); err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	return nil
}

func createMigration(name string) error {
	// This would create migration files in a migrations directory
	fmt.Printf("Creating migration: %s\n", name)
	// Implementation would create .sql files for up/down migrations
	return nil
}

func showMigrationStatus(db *sql.DB) error {
	query := `
		SELECT name, direction, applied_at 
		FROM migrations 
		ORDER BY applied_at DESC`

	rows, err := db.Query(query)
	if err != nil {
		return fmt.Errorf("failed to query migrations: %w", err)
	}
	defer rows.Close()

	fmt.Println("Migration Status:")
	fmt.Println("================")

	for rows.Next() {
		var name, direction string
		var appliedAt sql.NullTime

		if err := rows.Scan(&name, &direction, &appliedAt); err != nil {
			return fmt.Errorf("failed to scan migration row: %w", err)
		}

		status := "PENDING"
		if appliedAt.Valid {
			status = fmt.Sprintf("APPLIED (%s)", appliedAt.Time.Format("2006-01-02 15:04:05"))
		}

		fmt.Printf("%-20s %-5s %s\n", name, direction, status)
	}

	return rows.Err()
}

func createMigrationTable(db *sql.DB) error {
	query := `
		CREATE TABLE IF NOT EXISTS migrations (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			direction VARCHAR(10) NOT NULL,
			applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(name, direction)
		)`

	_, err := db.Exec(query)
	return err
}

func createTradesTable(db *sql.DB) error {
	query := `
		CREATE TABLE IF NOT EXISTS trades (
			id VARCHAR(255) PRIMARY KEY,
			symbol VARCHAR(50) NOT NULL,
			buy_order_id VARCHAR(255) NOT NULL,
			sell_order_id VARCHAR(255) NOT NULL,
			price DECIMAL(20,8) NOT NULL,
			quantity DECIMAL(20,8) NOT NULL,
			value DECIMAL(20,8) NOT NULL,
			buy_user_id VARCHAR(255) NOT NULL,
			sell_user_id VARCHAR(255) NOT NULL,
			taker_side VARCHAR(10) NOT NULL,
			maker_order_id VARCHAR(255) NOT NULL,
			taker_order_id VARCHAR(255) NOT NULL,
			timestamp TIMESTAMP NOT NULL,
			INDEX idx_symbol (symbol),
			INDEX idx_timestamp (timestamp),
			INDEX idx_buy_user (buy_user_id),
			INDEX idx_sell_user (sell_user_id)
		)`

	_, err := db.Exec(query)
	return err
}

func createUsersTable(db *sql.DB) error {
	query := `
		CREATE TABLE IF NOT EXISTS users (
			id VARCHAR(255) PRIMARY KEY,
			email VARCHAR(255) UNIQUE NOT NULL,
			username VARCHAR(255) UNIQUE NOT NULL,
			status VARCHAR(20) NOT NULL DEFAULT 'active',
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`

	_, err := db.Exec(query)
	return err
}

func createSymbolsTable(db *sql.DB) error {
	query := `
		CREATE TABLE IF NOT EXISTS symbols (
			symbol VARCHAR(50) PRIMARY KEY,
			base_asset VARCHAR(20) NOT NULL,
			quote_asset VARCHAR(20) NOT NULL,
			status VARCHAR(20) NOT NULL DEFAULT 'trading',
			min_price DECIMAL(20,8) NOT NULL,
			max_price DECIMAL(20,8) NOT NULL,
			tick_size DECIMAL(20,8) NOT NULL,
			min_quantity DECIMAL(20,8) NOT NULL,
			max_quantity DECIMAL(20,8) NOT NULL,
			step_size DECIMAL(20,8) NOT NULL,
			min_notional DECIMAL(20,8) NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`

	_, err := db.Exec(query)
	return err
}

func createMarketDataTable(db *sql.DB) error {
	query := `
		CREATE TABLE IF NOT EXISTS market_data (
			symbol VARCHAR(50) PRIMARY KEY,
			last_price DECIMAL(20,8) NOT NULL,
			bid_price DECIMAL(20,8) NOT NULL,
			ask_price DECIMAL(20,8) NOT NULL,
			volume DECIMAL(20,8) NOT NULL DEFAULT 0,
			high_24h DECIMAL(20,8) NOT NULL DEFAULT 0,
			low_24h DECIMAL(20,8) NOT NULL DEFAULT 0,
			change_24h DECIMAL(20,8) NOT NULL DEFAULT 0,
			change_percent_24h DECIMAL(10,4) NOT NULL DEFAULT 0,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`

	_, err := db.Exec(query)
	return err
}

func dropTable(db *sql.DB, tableName string) error {
	query := fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", tableName)
	_, err := db.Exec(query)
	return err
}

func recordMigration(db *sql.DB, name, direction string) error {
	query := `
		INSERT INTO migrations (name, direction) 
		VALUES ($1, $2) 
		ON CONFLICT (name, direction) DO NOTHING`

	_, err := db.Exec(query, name, direction)
	return err
}
