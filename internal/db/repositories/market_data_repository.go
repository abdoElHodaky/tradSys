package repositories

import (
	"context"
	"time"

	"github.com/abdoElHodaky/tradSys/pkg/types"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// MarketDataRepository handles market data operations
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

// BatchCreate creates multiple market data records in a batch
func (r *MarketDataRepository) BatchCreate(ctx context.Context, marketData []*types.MarketData) error {
	if len(marketData) == 0 {
		return nil
	}

	result := r.db.WithContext(ctx).CreateInBatches(marketData, 100)
	if result.Error != nil {
		r.logger.Error("Failed to batch create market data", zap.Error(result.Error))
		return result.Error
	}

	r.logger.Debug("Batch created market data", zap.Int("count", len(marketData)))
	return nil
}

// GetOHLCVBySymbolAndTimeRange retrieves OHLCV data for a symbol within a time range
func (r *MarketDataRepository) GetOHLCVBySymbolAndTimeRange(ctx context.Context, symbol string, startTime, endTime time.Time) ([]*types.OHLCV, error) {
	var ohlcvData []*types.OHLCV

	result := r.db.WithContext(ctx).
		Where("symbol = ? AND timestamp BETWEEN ? AND ?", symbol, startTime, endTime).
		Order("timestamp ASC").
		Find(&ohlcvData)

	if result.Error != nil {
		r.logger.Error("Failed to get OHLCV data", 
			zap.Error(result.Error),
			zap.String("symbol", symbol),
			zap.Time("start_time", startTime),
			zap.Time("end_time", endTime))
		return nil, result.Error
	}

	r.logger.Debug("Retrieved OHLCV data", 
		zap.String("symbol", symbol),
		zap.Int("count", len(ohlcvData)))

	return ohlcvData, nil
}
