package marketdata

import (
	"context"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/db/repositories"
	"github.com/abdoElHodaky/tradSys/internal/marketdata/external"
	"github.com/patrickmn/go-cache"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// ServiceParams contains the parameters for creating a market data service
type ServiceParams struct {
	fx.In

	Logger     *zap.Logger
	Repository *repositories.MarketDataRepository `optional:"true"`
}

// Service provides market data operations
type Service struct {
	logger         *zap.Logger
	repository     *repositories.MarketDataRepository
	externalManager *external.Manager
	cache          *cache.Cache
}

// NewService creates a new market data service with fx dependency injection
func NewService(p ServiceParams) *Service {
	return &Service{
		logger:     p.Logger,
		repository: p.Repository,
		cache:      cache.New(5*time.Minute, 10*time.Minute),
	}
}

// GetMarketData returns current market data for a symbol
func (s *Service) GetMarketData(ctx context.Context, symbol, interval string) (float64, float64, int64, error) {
	// Check cache first
	cacheKey := symbol + ":" + interval
	if data, found := s.cache.Get(cacheKey); found {
		marketData := data.(map[string]interface{})
		return marketData["price"].(float64), marketData["volume"].(float64), marketData["timestamp"].(int64), nil
	}

	// If not in cache, fetch from external source or database
	price, volume, timestamp := 0.0, 0.0, time.Now().Unix()*1000
	var err error

	if s.externalManager != nil {
		price, volume, timestamp, err = s.externalManager.GetMarketData(symbol, interval)
		if err != nil {
			s.logger.Error("Failed to get market data from external source",
				zap.String("symbol", symbol),
				zap.String("interval", interval),
				zap.Error(err))
			return 0, 0, 0, err
		}
	} else if s.repository != nil {
		// Fetch from database
		// Implementation would go here
	} else {
		// Return placeholder data
		price = 50000.0
		volume = 100.0
	}

	// Cache the result
	s.cache.Set(cacheKey, map[string]interface{}{
		"price":     price,
		"volume":    volume,
		"timestamp": timestamp,
	}, cache.DefaultExpiration)

	return price, volume, timestamp, nil
}

// ServiceModule provides the market data service module for fx
var ServiceModule = fx.Options(
	fx.Provide(NewService),
	external.Module,
)

