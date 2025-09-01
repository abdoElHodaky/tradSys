package repositories

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// MarketDataRepositoryParams contains the parameters for creating a market data repository
type MarketDataRepositoryParams struct {
	fx.In

	Logger *zap.Logger
	DB     *sqlx.DB `optional:"true"`
}

// SQLXMarketDataRepository provides database operations for market data using sqlx
type SQLXMarketDataRepository struct {
	logger *zap.Logger
	db     *sqlx.DB
}

// MarketData represents market data in the database
type MarketData struct {
	ID        int64     `db:"id"`
	Symbol    string    `db:"symbol"`
	Interval  string    `db:"interval"`
	Price     float64   `db:"price"`
	Volume    float64   `db:"volume"`
	Timestamp time.Time `db:"timestamp"`
	CreatedAt time.Time `db:"created_at"`
}

// Symbol represents a trading symbol in the database
type Symbol struct {
	ID                int64     `db:"id"`
	Name              string    `db:"name"`
	BaseCurrency      string    `db:"base_currency"`
	QuoteCurrency     string    `db:"quote_currency"`
	PriceIncrement    float64   `db:"price_increment"`
	QuantityIncrement float64   `db:"quantity_increment"`
	MinOrderSize      float64   `db:"min_order_size"`
	MaxOrderSize      float64   `db:"max_order_size"`
	CreatedAt         time.Time `db:"created_at"`
	UpdatedAt         time.Time `db:"updated_at"`
}

// NewSQLXMarketDataRepository creates a new market data repository with fx dependency injection
func NewSQLXMarketDataRepository(p MarketDataRepositoryParams) *SQLXMarketDataRepository {
	return &SQLXMarketDataRepository{
		logger: p.Logger,
		db:     p.DB,
	}
}

// GetMarketData retrieves market data for a symbol and interval
func (r *SQLXMarketDataRepository) GetMarketData(ctx context.Context, symbol, interval string) (*MarketData, error) {
	if r.db == nil {
		r.logger.Warn("Database connection not available")
		return nil, nil
	}

	query := `
		SELECT id, symbol, interval, price, volume, timestamp, created_at
		FROM market_data
		WHERE symbol = $1 AND interval = $2
		ORDER BY timestamp DESC
		LIMIT 1
	`

	var data MarketData
	err := r.db.GetContext(ctx, &data, query, symbol, interval)
	if err != nil {
		r.logger.Error("Failed to get market data",
			zap.String("symbol", symbol),
			zap.String("interval", interval),
			zap.Error(err))
		return nil, err
	}

	return &data, nil
}

// GetHistoricalData retrieves historical market data for a symbol and interval
func (r *SQLXMarketDataRepository) GetHistoricalData(ctx context.Context, symbol, interval string, startTime, endTime time.Time, limit int) ([]*MarketData, error) {
	if r.db == nil {
		r.logger.Warn("Database connection not available")
		return nil, nil
	}

	query := `
		SELECT id, symbol, interval, price, volume, timestamp, created_at
		FROM market_data
		WHERE symbol = $1 AND interval = $2 AND timestamp BETWEEN $3 AND $4
		ORDER BY timestamp DESC
		LIMIT $5
	`

	var data []*MarketData
	err := r.db.SelectContext(ctx, &data, query, symbol, interval, startTime, endTime, limit)
	if err != nil {
		r.logger.Error("Failed to get historical market data",
			zap.String("symbol", symbol),
			zap.String("interval", interval),
			zap.Time("start_time", startTime),
			zap.Time("end_time", endTime),
			zap.Error(err))
		return nil, err
	}

	return data, nil
}

// GetSymbols retrieves available trading symbols
func (r *SQLXMarketDataRepository) GetSymbols(ctx context.Context, filter string) ([]*Symbol, error) {
	if r.db == nil {
		r.logger.Warn("Database connection not available")
		return nil, nil
	}

	query := `
		SELECT id, name, base_currency, quote_currency, price_increment, quantity_increment, min_order_size, max_order_size, created_at, updated_at
		FROM symbols
	`

	args := []interface{}{}
	if filter != "" {
		query += " WHERE name LIKE $1 OR base_currency LIKE $1 OR quote_currency LIKE $1"
		args = append(args, "%"+filter+"%")
	}

	var symbols []*Symbol
	err := r.db.SelectContext(ctx, &symbols, query, args...)
	if err != nil {
		r.logger.Error("Failed to get symbols",
			zap.String("filter", filter),
			zap.Error(err))
		return nil, err
	}

	return symbols, nil
}

// SaveMarketData saves market data to the database
func (r *SQLXMarketDataRepository) SaveMarketData(ctx context.Context, data *MarketData) error {
	if r.db == nil {
		r.logger.Warn("Database connection not available")
		return nil
	}

	query := `
		INSERT INTO market_data (symbol, interval, price, volume, timestamp, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`

	now := time.Now()
	data.CreatedAt = now

	row := r.db.QueryRowContext(ctx, query,
		data.Symbol,
		data.Interval,
		data.Price,
		data.Volume,
		data.Timestamp,
		data.CreatedAt,
	)

	err := row.Scan(&data.ID)
	if err != nil {
		r.logger.Error("Failed to save market data",
			zap.String("symbol", data.Symbol),
			zap.String("interval", data.Interval),
			zap.Error(err))
		return err
	}

	return nil
}

// SQLXMarketDataRepositoryModule provides the market data repository module for fx
var SQLXMarketDataRepositoryModule = fx.Options(
	fx.Provide(NewSQLXMarketDataRepository),
)

