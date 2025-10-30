package marketdata

import (
	"context"
	"fmt"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/config"
	"github.com/abdoElHodaky/tradSys/internal/db"
	"github.com/abdoElHodaky/tradSys/internal/db/repositories"
	"github.com/abdoElHodaky/tradSys/internal/marketdata/external"
	"github.com/patrickmn/go-cache"
	"go.uber.org/zap"
)

// ServiceParams contains parameters for creating a new service
type ServiceParams struct {
	Repository *repositories.MarketDataRepository
	Logger     *zap.Logger
	Config     *config.Config
}

// NewService creates a new market data service with fx dependency injection
func NewService(p ServiceParams) *Service {
	ctx, cancel := context.WithCancel(context.Background())

	// Initialize external manager with default configuration
	externalManager := external.NewManager(p.Logger)

	service := &Service{
		MarketDataRepository: p.Repository,
		ExternalManager:      externalManager,
		Cache:                cache.New(5*time.Minute, 10*time.Minute),
		Subscriptions:        make(map[string]*Subscription),
		SymbolSubscriptions:  make(map[string]map[string]*Subscription),
		logger:               p.Logger,
		config:               p.Config,
		ctx:                  ctx,
		cancel:               cancel,
	}

	return service
}

// Start starts the market data service
func (s *Service) Start(ctx context.Context) error {
	s.logger.Info("Starting market data service")

	// Start data persistence task
	go s.persistMarketData()

	// Initialize external data sources
	if err := s.initializeDataSources(); err != nil {
		return fmt.Errorf("failed to initialize data sources: %w", err)
	}

	s.logger.Info("Market data service started successfully")
	return nil
}

// Stop stops the market data service
func (s *Service) Stop(ctx context.Context) error {
	s.logger.Info("Stopping market data service")

	// Cancel context to stop all goroutines
	s.cancel()

	// Close all subscriptions
	s.mu.Lock()
	for _, subscription := range s.Subscriptions {
		close(subscription.Channel)
	}
	s.Subscriptions = make(map[string]*Subscription)
	s.SymbolSubscriptions = make(map[string]map[string]*Subscription)
	s.mu.Unlock()

	s.logger.Info("Market data service stopped successfully")
	return nil
}

// initializeDataSources initializes external market data sources
func (s *Service) initializeDataSources() error {
	s.logger.Info("Initializing market data sources")

	// Add default data sources based on configuration
	// This is a placeholder - in production, you'd configure actual providers
	if err := s.ExternalManager.AddSource("binance", map[string]interface{}{
		"api_key":    "",
		"secret_key": "",
		"testnet":    true,
	}); err != nil {
		s.logger.Warn("Failed to add Binance data source", zap.Error(err))
	}

	return nil
}

// persistMarketData periodically persists market data to the database
func (s *Service) persistMarketData() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.persistCachedMarketData()
		}
	}
}

// persistCachedMarketData persists cached market data to the database
func (s *Service) persistCachedMarketData() {
	// Get all items from cache
	items := s.Cache.Items()

	// Create a batch of market data entries
	marketDataEntries := make([]*db.MarketData, 0, len(items))

	for _, item := range items {
		// Skip expired items
		if item.Expired() {
			continue
		}

		// Convert cached data to market data entry
		switch data := item.Object.(type) {
		case *external.OrderBookData:
			marketDataEntries = append(marketDataEntries, &db.MarketData{
				Symbol:    data.Symbol,
				Type:      string(external.MarketDataTypeOrderBook),
				Timestamp: data.Timestamp,
				Data:      s.serializeOrderBookData(data),
			})
		case *external.TradeData:
			marketDataEntries = append(marketDataEntries, &db.MarketData{
				Symbol:    data.Symbol,
				Type:      string(external.MarketDataTypeTrade),
				Price:     data.Price,
				Volume:    data.Quantity,
				Timestamp: data.Timestamp,
				Data:      s.serializeTradeData(data),
			})
		case *external.TickerData:
			marketDataEntries = append(marketDataEntries, &db.MarketData{
				Symbol:    data.Symbol,
				Type:      string(external.MarketDataTypeTicker),
				Price:     data.Price,
				Volume:    data.Volume,
				Timestamp: data.Timestamp,
				Data:      s.serializeTickerData(data),
			})
		case *external.OHLCVData:
			marketDataEntries = append(marketDataEntries, &db.MarketData{
				Symbol:    data.Symbol,
				Type:      string(external.MarketDataTypeOHLCV),
				Open:      data.Open,
				High:      data.High,
				Low:       data.Low,
				Close:     data.Close,
				Volume:    data.Volume,
				Timestamp: data.Timestamp,
				Data:      s.serializeOHLCVData(data),
			})
		}
	}

	// Persist market data entries
	if len(marketDataEntries) > 0 {
		if err := s.MarketDataRepository.BatchCreate(context.Background(), marketDataEntries); err != nil {
			s.logger.Error("Failed to persist market data", zap.Error(err))
		}
	}
}

// serializeOrderBookData serializes order book data to JSON
func (s *Service) serializeOrderBookData(data *external.OrderBookData) string {
	// In a real implementation, this would serialize the data to JSON
	return "{}"
}

// serializeTradeData serializes trade data to JSON
func (s *Service) serializeTradeData(data *external.TradeData) string {
	// In a real implementation, this would serialize the data to JSON
	return "{}"
}

// serializeTickerData serializes ticker data to JSON
func (s *Service) serializeTickerData(data *external.TickerData) string {
	// In a real implementation, this would serialize the data to JSON
	return "{}"
}

// serializeOHLCVData serializes OHLCV data to JSON
func (s *Service) serializeOHLCVData(data *external.OHLCVData) string {
	// In a real implementation, this would serialize the data to JSON
	return "{}"
}
