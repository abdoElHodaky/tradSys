package repositories

import (
	"context"
	"errors"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/db"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// MarketDataRepository represents a repository for market data
type MarketDataRepository struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewMarketDataRepository creates a new market data repository
func NewMarketDataRepository(db *gorm.DB, logger *zap.Logger) *MarketDataRepository {
	return &MarketDataRepository{
		db:     db,
		logger: logger,
	}
}

// Create creates a new market data entry
func (r *MarketDataRepository) Create(ctx context.Context, marketData *db.MarketData) error {
	result := r.db.WithContext(ctx).Create(marketData)
	if result.Error != nil {
		r.logger.Error("Failed to create market data",
			zap.Error(result.Error),
			zap.String("symbol", marketData.Symbol),
			zap.String("type", marketData.Type))
		return result.Error
	}
	return nil
}

// BatchCreate creates multiple market data entries
func (r *MarketDataRepository) BatchCreate(ctx context.Context, marketDataEntries []*db.MarketData) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, marketData := range marketDataEntries {
			if err := tx.Create(marketData).Error; err != nil {
				r.logger.Error("Failed to create market data in batch",
					zap.Error(err),
					zap.String("symbol", marketData.Symbol),
					zap.String("type", marketData.Type))
				return err
			}
		}
		return nil
	})
}

// GetLatestBySymbolAndType gets the latest market data by symbol and type
func (r *MarketDataRepository) GetLatestBySymbolAndType(ctx context.Context, symbol, dataType string) (*db.MarketData, error) {
	var marketData db.MarketData
	result := r.db.WithContext(ctx).
		Where("symbol = ? AND type = ?", symbol, dataType).
		Order("timestamp DESC").
		First(&marketData)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		r.logger.Error("Failed to get latest market data",
			zap.Error(result.Error),
			zap.String("symbol", symbol),
			zap.String("type", dataType))
		return nil, result.Error
	}
	return &marketData, nil
}

// GetBySymbolAndTimeRange gets market data by symbol and time range
func (r *MarketDataRepository) GetBySymbolAndTimeRange(ctx context.Context, symbol, dataType string, start, end time.Time) ([]*db.MarketData, error) {
	var marketDataEntries []*db.MarketData
	result := r.db.WithContext(ctx).
		Where("symbol = ? AND type = ? AND timestamp BETWEEN ? AND ?", symbol, dataType, start, end).
		Order("timestamp ASC").
		Find(&marketDataEntries)
	if result.Error != nil {
		r.logger.Error("Failed to get market data by time range",
			zap.Error(result.Error),
			zap.String("symbol", symbol),
			zap.String("type", dataType),
			zap.Time("start", start),
			zap.Time("end", end))
		return nil, result.Error
	}
	return marketDataEntries, nil
}

// GetOHLCVBySymbolAndTimeRange gets OHLCV data by symbol and time range
func (r *MarketDataRepository) GetOHLCVBySymbolAndTimeRange(ctx context.Context, symbol string, interval string, start, end time.Time) ([]*db.MarketData, error) {
	var marketDataEntries []*db.MarketData
	result := r.db.WithContext(ctx).
		Where("symbol = ? AND type = ? AND data LIKE ? AND timestamp BETWEEN ? AND ?",
			symbol, "ohlcv", "%\"interval\":\""+interval+"\"%", start, end).
		Order("timestamp ASC").
		Find(&marketDataEntries)
	if result.Error != nil {
		r.logger.Error("Failed to get OHLCV data",
			zap.Error(result.Error),
			zap.String("symbol", symbol),
			zap.String("interval", interval),
			zap.Time("start", start),
			zap.Time("end", end))
		return nil, result.Error
	}
	return marketDataEntries, nil
}

// DeleteOlderThan deletes market data older than a specified time
func (r *MarketDataRepository) DeleteOlderThan(ctx context.Context, dataType string, olderThan time.Time) error {
	result := r.db.WithContext(ctx).
		Where("type = ? AND timestamp < ?", dataType, olderThan).
		Delete(&db.MarketData{})
	if result.Error != nil {
		r.logger.Error("Failed to delete old market data",
			zap.Error(result.Error),
			zap.String("type", dataType),
			zap.Time("older_than", olderThan))
		return result.Error
	}
	return nil
}

// GetSymbols gets all symbols with market data
func (r *MarketDataRepository) GetSymbols(ctx context.Context) ([]string, error) {
	var symbols []string
	result := r.db.WithContext(ctx).Model(&db.MarketData{}).
		Distinct("symbol").
		Pluck("symbol", &symbols)
	if result.Error != nil {
		r.logger.Error("Failed to get symbols", zap.Error(result.Error))
		return nil, result.Error
	}
	return symbols, nil
}

// GetDataTypes gets all data types for a symbol
func (r *MarketDataRepository) GetDataTypes(ctx context.Context, symbol string) ([]string, error) {
	var dataTypes []string
	result := r.db.WithContext(ctx).Model(&db.MarketData{}).
		Where("symbol = ?", symbol).
		Distinct("type").
		Pluck("type", &dataTypes)
	if result.Error != nil {
		r.logger.Error("Failed to get data types",
			zap.Error(result.Error),
			zap.String("symbol", symbol))
		return nil, result.Error
	}
	return dataTypes, nil
}
