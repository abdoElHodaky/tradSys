package marketdata

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/config"
	"github.com/abdoElHodaky/tradSys/internal/db"
	"github.com/abdoElHodaky/tradSys/internal/db/repositories"
	"github.com/abdoElHodaky/tradSys/internal/marketdata/external"
	"github.com/patrickmn/go-cache"
	"go.uber.org/fx"
	"go.uber.org/zap"
)



// Service represents a market data service
type Service struct {
	// MarketDataRepository is the market data repository
	MarketDataRepository *repositories.MarketDataRepository
	// ExternalManager is the external market data provider manager
	ExternalManager *external.Manager
	// Cache is a cache for market data
	Cache *cache.Cache
	// Subscriptions is a map of subscription ID to subscription
	Subscriptions map[string]*Subscription
	// SymbolSubscriptions is a map of symbol to subscriptions
	SymbolSubscriptions map[string]map[string]*Subscription
	// Logger
	logger *zap.Logger
	// Config
	config *config.Config
	// Mutex for thread safety
	mu sync.RWMutex
	// Context
	ctx context.Context
	// Cancel function
	cancel context.CancelFunc
}

// Subscription represents a market data subscription
type Subscription struct {
	// ID is the unique identifier for the subscription
	ID string
	// Symbol is the trading symbol
	Symbol string
	// Type is the type of market data
	Type external.MarketDataType
	// Interval is the interval for OHLCV data
	Interval string
	// Channel is the channel for sending market data
	Channel chan interface{}
	// CreatedAt is the time the subscription was created
	CreatedAt time.Time
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

// SubscribeOrderBook subscribes to order book updates
func (s *Service) SubscribeOrderBook(ctx context.Context, symbol string) (*Subscription, error) {
	// Create subscription
	subscription := &Subscription{
		ID:        generateID(),
		Symbol:    symbol,
		Type:      external.MarketDataTypeOrderBook,
		Channel:   make(chan interface{}, 100),
		CreatedAt: time.Now(),
	}
	
	// Add to subscriptions
	s.mu.Lock()
	s.Subscriptions[subscription.ID] = subscription
	
	// Add to symbol subscriptions
	if _, exists := s.SymbolSubscriptions[symbol]; !exists {
		s.SymbolSubscriptions[symbol] = make(map[string]*Subscription)
	}
	s.SymbolSubscriptions[symbol][subscription.ID] = subscription
	s.mu.Unlock()
	
	// Subscribe to external provider
	provider, err := s.ExternalManager.GetDefaultProvider()
	if err != nil {
		return nil, err
	}
	
	// Create callback function
	callback := func(data interface{}) {
		// Cache the data
		s.Cache.Set(
			"orderbook:"+symbol, 
			data, 
			cache.DefaultExpiration,
		)
		
		// Send to subscriber
		select {
		case subscription.Channel <- data:
		default:
			s.logger.Warn("Order book channel full, dropping update",
				zap.String("subscription_id", subscription.ID),
				zap.String("symbol", symbol))
		}
	}
	
	// Subscribe to external provider
	if err := provider.SubscribeOrderBook(ctx, symbol, callback); err != nil {
		return nil, err
	}
	
	return subscription, nil
}

// SubscribeTrades subscribes to trade updates
func (s *Service) SubscribeTrades(ctx context.Context, symbol string) (*Subscription, error) {
	// Create subscription
	subscription := &Subscription{
		ID:        generateID(),
		Symbol:    symbol,
		Type:      external.MarketDataTypeTrade,
		Channel:   make(chan interface{}, 100),
		CreatedAt: time.Now(),
	}
	
	// Add to subscriptions
	s.mu.Lock()
	s.Subscriptions[subscription.ID] = subscription
	
	// Add to symbol subscriptions
	if _, exists := s.SymbolSubscriptions[symbol]; !exists {
		s.SymbolSubscriptions[symbol] = make(map[string]*Subscription)
	}
	s.SymbolSubscriptions[symbol][subscription.ID] = subscription
	s.mu.Unlock()
	
	// Subscribe to external provider
	provider, err := s.ExternalManager.GetDefaultProvider()
	if err != nil {
		return nil, err
	}
	
	// Create callback function
	callback := func(data interface{}) {
		// Cache the data
		s.Cache.Set(
			"trade:"+symbol, 
			data, 
			cache.DefaultExpiration,
		)
		
		// Send to subscriber
		select {
		case subscription.Channel <- data:
		default:
			s.logger.Warn("Trade channel full, dropping update",
				zap.String("subscription_id", subscription.ID),
				zap.String("symbol", symbol))
		}
	}
	
	// Subscribe to external provider
	if err := provider.SubscribeTrades(ctx, symbol, callback); err != nil {
		return nil, err
	}
	
	return subscription, nil
}

// SubscribeTicker subscribes to ticker updates
func (s *Service) SubscribeTicker(ctx context.Context, symbol string) (*Subscription, error) {
	// Create subscription
	subscription := &Subscription{
		ID:        generateID(),
		Symbol:    symbol,
		Type:      external.MarketDataTypeTicker,
		Channel:   make(chan interface{}, 100),
		CreatedAt: time.Now(),
	}
	
	// Add to subscriptions
	s.mu.Lock()
	s.Subscriptions[subscription.ID] = subscription
	
	// Add to symbol subscriptions
	if _, exists := s.SymbolSubscriptions[symbol]; !exists {
		s.SymbolSubscriptions[symbol] = make(map[string]*Subscription)
	}
	s.SymbolSubscriptions[symbol][subscription.ID] = subscription
	s.mu.Unlock()
	
	// Subscribe to external provider
	provider, err := s.ExternalManager.GetDefaultProvider()
	if err != nil {
		return nil, err
	}
	
	// Create callback function
	callback := func(data interface{}) {
		// Cache the data
		s.Cache.Set(
			"ticker:"+symbol, 
			data, 
			cache.DefaultExpiration,
		)
		
		// Send to subscriber
		select {
		case subscription.Channel <- data:
		default:
			s.logger.Warn("Ticker channel full, dropping update",
				zap.String("subscription_id", subscription.ID),
				zap.String("symbol", symbol))
		}
	}
	
	// Subscribe to external provider
	if err := provider.SubscribeTicker(ctx, symbol, callback); err != nil {
		return nil, err
	}
	
	return subscription, nil
}

// SubscribeOHLCV subscribes to OHLCV updates
func (s *Service) SubscribeOHLCV(ctx context.Context, symbol, interval string) (*Subscription, error) {
	// Create subscription
	subscription := &Subscription{
		ID:        generateID(),
		Symbol:    symbol,
		Type:      external.MarketDataTypeOHLCV,
		Interval:  interval,
		Channel:   make(chan interface{}, 100),
		CreatedAt: time.Now(),
	}
	
	// Add to subscriptions
	s.mu.Lock()
	s.Subscriptions[subscription.ID] = subscription
	
	// Add to symbol subscriptions
	if _, exists := s.SymbolSubscriptions[symbol]; !exists {
		s.SymbolSubscriptions[symbol] = make(map[string]*Subscription)
	}
	s.SymbolSubscriptions[symbol][subscription.ID] = subscription
	s.mu.Unlock()
	
	// Subscribe to external provider
	provider, err := s.ExternalManager.GetDefaultProvider()
	if err != nil {
		return nil, err
	}
	
	// Create callback function
	callback := func(data interface{}) {
		// Cache the data
		s.Cache.Set(
			"ohlcv:"+symbol+":"+interval, 
			data, 
			cache.DefaultExpiration,
		)
		
		// Send to subscriber
		select {
		case subscription.Channel <- data:
		default:
			s.logger.Warn("OHLCV channel full, dropping update",
				zap.String("subscription_id", subscription.ID),
				zap.String("symbol", symbol),
				zap.String("interval", interval))
		}
	}
	
	// Subscribe to external provider
	if err := provider.SubscribeOHLCV(ctx, symbol, interval, callback); err != nil {
		return nil, err
	}
	
	return subscription, nil
}

// Unsubscribe unsubscribes from market data
func (s *Service) Unsubscribe(ctx context.Context, subscriptionID string) error {
	s.mu.Lock()
	subscription, exists := s.Subscriptions[subscriptionID]
	if !exists {
		s.mu.Unlock()
		return nil
	}
	
	// Remove from subscriptions
	delete(s.Subscriptions, subscriptionID)
	
	// Remove from symbol subscriptions
	if symbolSubs, exists := s.SymbolSubscriptions[subscription.Symbol]; exists {
		delete(symbolSubs, subscriptionID)
	}
	
	// Get subscription details before unlocking
	symbol := subscription.Symbol
	dataType := subscription.Type
	interval := subscription.Interval
	
	s.mu.Unlock()
	
	// Unsubscribe from external provider
	provider, err := s.ExternalManager.GetDefaultProvider()
	if err != nil {
		return err
	}
	
	// Unsubscribe based on data type
	switch dataType {
	case external.MarketDataTypeOrderBook:
		return provider.UnsubscribeOrderBook(ctx, symbol)
	case external.MarketDataTypeTrade:
		return provider.UnsubscribeTrades(ctx, symbol)
	case external.MarketDataTypeTicker:
		return provider.UnsubscribeTicker(ctx, symbol)
	case external.MarketDataTypeOHLCV:
		return provider.UnsubscribeOHLCV(ctx, symbol, interval)
	}
	
	return nil
}

// GetOrderBook gets the order book
func (s *Service) GetOrderBook(ctx context.Context, symbol string) (*external.OrderBookData, error) {
	// Check cache first
	if cachedData, found := s.Cache.Get("orderbook:" + symbol); found {
		return cachedData.(*external.OrderBookData), nil
	}
	
	// Get from external provider
	provider, err := s.ExternalManager.GetDefaultProvider()
	if err != nil {
		return nil, err
	}
	
	orderBook, err := provider.GetOrderBook(ctx, symbol)
	if err != nil {
		return nil, err
	}
	
	// Cache the data
	s.Cache.Set("orderbook:"+symbol, orderBook, cache.DefaultExpiration)
	
	return orderBook, nil
}

// GetTrades gets trades
func (s *Service) GetTrades(ctx context.Context, symbol string, limit int) ([]external.TradeData, error) {
	// Get from external provider
	provider, err := s.ExternalManager.GetDefaultProvider()
	if err != nil {
		return nil, err
	}
	
	trades, err := provider.GetTrades(ctx, symbol, limit)
	if err != nil {
		return nil, err
	}
	
	return trades, nil
}

// GetTicker gets the ticker
func (s *Service) GetTicker(ctx context.Context, symbol string) (*external.TickerData, error) {
	// Check cache first
	if cachedData, found := s.Cache.Get("ticker:" + symbol); found {
		return cachedData.(*external.TickerData), nil
	}
	
	// Get from external provider
	provider, err := s.ExternalManager.GetDefaultProvider()
	if err != nil {
		return nil, err
	}
	
	ticker, err := provider.GetTicker(ctx, symbol)
	if err != nil {
		return nil, err
	}
	
	// Cache the data
	s.Cache.Set("ticker:"+symbol, ticker, cache.DefaultExpiration)
	
	return ticker, nil
}

// GetOHLCV gets OHLCV data
func (s *Service) GetOHLCV(ctx context.Context, symbol, interval string, limit int) ([]external.OHLCVData, error) {
	// Get from external provider
	provider, err := s.ExternalManager.GetDefaultProvider()
	if err != nil {
		return nil, err
	}
	
	ohlcv, err := provider.GetOHLCV(ctx, symbol, interval, limit)
	if err != nil {
		return nil, err
	}
	
	return ohlcv, nil
}

// GetHistoricalOHLCV gets historical OHLCV data from the database
func (s *Service) GetHistoricalOHLCV(ctx context.Context, symbol, interval string, start, end time.Time) ([]*db.MarketData, error) {
	return s.MarketDataRepository.GetOHLCVBySymbolAndTimeRange(ctx, symbol, interval, start, end)
}

// AddMarketDataSource adds a new market data source
func (s *Service) AddMarketDataSource(ctx context.Context, source string, config interface{}) error {
	s.logger.Info("Adding market data source", zap.String("source", source))
	
	// Validate source configuration
	if source == "" {
		return fmt.Errorf("source name cannot be empty")
	}
	
	// Check if source already exists
	if s.externalManager != nil {
		// Add the source to the external manager
		if err := s.externalManager.AddProvider(source, config); err != nil {
			s.logger.Error("Failed to add market data source", 
				zap.String("source", source), 
				zap.Error(err))
			return fmt.Errorf("failed to add source %s: %w", source, err)
		}
		
		// Start subscriptions for the new source
		s.logger.Info("Successfully added market data source", zap.String("source", source))
	}
	
	return nil
}

// GetMarketData retrieves market data for a symbol within a time range
func (s *Service) GetMarketData(ctx context.Context, symbol string, timeRange interface{}) (interface{}, error) {
	s.logger.Info("Getting market data", zap.String("symbol", symbol))
	
	// Validate input parameters
	if symbol == "" {
		return nil, fmt.Errorf("symbol cannot be empty")
	}
	
	// Check cache first for performance
	if s.cache != nil {
		cacheKey := fmt.Sprintf("market_data:%s", symbol)
		if cachedData, found := s.cache.Get(cacheKey); found {
			s.logger.Debug("Retrieved market data from cache", zap.String("symbol", symbol))
			return cachedData, nil
		}
	}
	
	// Try to get real-time ticker data first
	tickerData, err := s.GetTicker(ctx, symbol)
	if err != nil {
		s.logger.Warn("Failed to get ticker data", zap.String("symbol", symbol), zap.Error(err))
	}
	
	// If external manager is available, try to get historical data
	var historicalData interface{}
	if s.externalManager != nil {
		if data, err := s.externalManager.GetHistoricalData(symbol, timeRange); err == nil {
			historicalData = data
		}
	}
	
	// Combine ticker and historical data
	result := map[string]interface{}{
		"symbol":     symbol,
		"ticker":     tickerData,
		"historical": historicalData,
		"timestamp":  time.Now().Unix(),
	}
	
	// Cache the result for future requests
	if s.cache != nil {
		cacheKey := fmt.Sprintf("market_data:%s", symbol)
		s.cache.Set(cacheKey, result, 30*time.Second) // Cache for 30 seconds
	}
	
	return result, nil
}

// Stop stops the service
func (s *Service) Stop() {
	s.cancel()
	
	// Close all subscription channels
	s.mu.Lock()
	defer s.mu.Unlock()
	
	for _, subscription := range s.Subscriptions {
		close(subscription.Channel)
	}
}

// Helper function to generate a unique ID
func generateID() string {
	return "sub_" + time.Now().Format("20060102150405") + "_" + randomString(8)
}

// Helper function to generate a random string
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[time.Now().UnixNano()%int64(len(charset))]
		time.Sleep(1 * time.Nanosecond)
	}
	return string(result)
}
