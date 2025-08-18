package repositories

import (
	"context"
	"errors"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/db/models"
	"github.com/abdoElHodaky/tradSys/internal/db/query"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// MarketDataRepository handles database operations for market data
type MarketDataRepository struct {
	db        *gorm.DB
	logger    *zap.Logger
	optimizer *query.Optimizer
}

// NewMarketDataRepository creates a new market data repository
func NewMarketDataRepository(db *gorm.DB, logger *zap.Logger) *MarketDataRepository {
	repo := &MarketDataRepository{
		db:        db,
		logger:    logger,
		optimizer: query.NewOptimizer(db, logger),
	}
	
	return repo
}

// SaveQuote inserts or updates a quote
func (r *MarketDataRepository) SaveQuote(ctx context.Context, quote *models.Quote) error {
	// Use a transaction for better performance when doing upsert
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return tx.Error
	}
	
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	
	// Try to update existing quote
	result := tx.Model(&models.Quote{}).
		Where("symbol = ? AND exchange = ?", quote.Symbol, quote.Exchange).
		Updates(map[string]interface{}{
			"bid":       quote.Bid,
			"ask":       quote.Ask,
			"bid_size":  quote.BidSize,
			"ask_size":  quote.AskSize,
			"timestamp": quote.Timestamp,
			"updated_at": time.Now(),
		})
	
	// If no record was updated, create a new one
	if result.RowsAffected == 0 {
		if err := tx.Create(quote).Error; err != nil {
			tx.Rollback()
			r.logger.Error("Failed to create quote", 
				zap.Error(err),
				zap.String("symbol", quote.Symbol))
			return err
		}
	} else if result.Error != nil {
		tx.Rollback()
		r.logger.Error("Failed to update quote", 
			zap.Error(result.Error),
			zap.String("symbol", quote.Symbol))
		return result.Error
	}
	
	return tx.Commit().Error
}

// GetLatestQuote retrieves the latest quote for a symbol
func (r *MarketDataRepository) GetLatestQuote(ctx context.Context, symbol, exchange string) (*models.Quote, error) {
	var quote models.Quote
	
	builder := query.NewBuilder(r.db, r.logger).
		Table("quotes").
		UseIndex("idx_quotes_symbol_exchange_timestamp").
		Where("symbol = ?", symbol).
		Where("exchange = ?", exchange).
		OrderBy("timestamp DESC").
		Limit(1)
	
	err := builder.First(&quote)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		r.logger.Error("Failed to get latest quote",
			zap.Error(err),
			zap.String("symbol", symbol),
			zap.String("exchange", exchange))
		return nil, err
	}
	
	return &quote, nil
}

// GetQuoteHistory retrieves historical quotes for a symbol
func (r *MarketDataRepository) GetQuoteHistory(ctx context.Context, symbol, exchange string, start, end time.Time, limit int) ([]*models.Quote, error) {
	var quotes []*models.Quote
	
	builder := query.NewBuilder(r.db, r.logger).
		Table("quotes").
		UseIndex("idx_quotes_symbol_exchange_timestamp").
		Where("symbol = ?", symbol).
		Where("exchange = ?", exchange).
		Where("timestamp BETWEEN ? AND ?", start, end).
		OrderBy("timestamp ASC")
	
	if limit > 0 {
		builder.Limit(limit)
	}
	
	err := builder.Execute(&quotes)
	if err != nil {
		r.logger.Error("Failed to get quote history",
			zap.Error(err),
			zap.String("symbol", symbol),
			zap.String("exchange", exchange))
		return nil, err
	}
	
	return quotes, nil
}

// SaveOHLCV inserts OHLCV data
func (r *MarketDataRepository) SaveOHLCV(ctx context.Context, ohlcv *models.OHLCV) error {
	result := r.db.WithContext(ctx).Create(ohlcv)
	if result.Error != nil {
		r.logger.Error("Failed to save OHLCV data", 
			zap.Error(result.Error),
			zap.String("symbol", ohlcv.Symbol))
		return result.Error
	}
	return nil
}

// GetOHLCV retrieves OHLCV data for a symbol and timeframe
func (r *MarketDataRepository) GetOHLCV(ctx context.Context, symbol, timeframe string, start, end time.Time) ([]*models.OHLCV, error) {
	var data []*models.OHLCV
	
	builder := query.NewBuilder(r.db, r.logger).
		Table("ohlcv").
		UseIndex("idx_ohlcv_symbol_timeframe_timestamp").
		Where("symbol = ?", symbol).
		Where("timeframe = ?", timeframe).
		Where("timestamp BETWEEN ? AND ?", start, end).
		OrderBy("timestamp ASC")
	
	// Analyze the query plan
	query, args := builder.Build()
	plan, err := r.optimizer.AnalyzeQuery(query, args...)
	if err == nil {
		r.logger.Debug("OHLCV query execution plan", zap.String("plan", plan))
	}
	
	err = builder.Execute(&data)
	if err != nil {
		r.logger.Error("Failed to get OHLCV data",
			zap.Error(err),
			zap.String("symbol", symbol),
			zap.String("timeframe", timeframe))
		return nil, err
	}
	
	return data, nil
}

// SaveMarketDepth saves market depth data
func (r *MarketDataRepository) SaveMarketDepth(ctx context.Context, depth *models.MarketDepth) error {
	result := r.db.WithContext(ctx).Create(depth)
	if result.Error != nil {
		r.logger.Error("Failed to save market depth", 
			zap.Error(result.Error),
			zap.String("symbol", depth.Symbol))
		return result.Error
	}
	return nil
}

// GetMarketDepth retrieves market depth for a symbol
func (r *MarketDataRepository) GetMarketDepth(ctx context.Context, symbol, exchange string) ([]*models.MarketDepth, error) {
	var depths []*models.MarketDepth
	
	builder := query.NewBuilder(r.db, r.logger).
		Table("market_depths").
		Where("symbol = ?", symbol).
		Where("exchange = ?", exchange).
		OrderBy("level ASC")
	
	err := builder.Execute(&depths)
	if err != nil {
		r.logger.Error("Failed to get market depth",
			zap.Error(err),
			zap.String("symbol", symbol),
			zap.String("exchange", exchange))
		return nil, err
	}
	
	return depths, nil
}

