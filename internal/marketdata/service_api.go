package marketdata

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/db"
	"github.com/abdoElHodaky/tradSys/internal/marketdata/external"
	"github.com/patrickmn/go-cache"
	"go.uber.org/zap"
)

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
	callback := func(data interface{}) error {
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
		return nil
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
	callback := func(data interface{}) error {
		// Cache the data
		s.Cache.Set(
			"trades:"+symbol,
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
		return nil
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
	callback := func(data interface{}) error {
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
		return nil
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
	callback := func(data interface{}) error {
		// Cache the data
		s.Cache.Set(
			fmt.Sprintf("ohlcv:%s:%s", symbol, interval),
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
		return nil
	}

	// Subscribe to external provider
	if err := provider.SubscribeOHLCV(ctx, symbol, interval, callback); err != nil {
		return nil, err
	}

	return subscription, nil
}

// Unsubscribe unsubscribes from a subscription
func (s *Service) Unsubscribe(ctx context.Context, subscriptionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	subscription, exists := s.Subscriptions[subscriptionID]
	if !exists {
		return fmt.Errorf("subscription not found: %s", subscriptionID)
	}

	// Remove from subscriptions
	delete(s.Subscriptions, subscriptionID)

	// Remove from symbol subscriptions
	if symbolSubs, exists := s.SymbolSubscriptions[subscription.Symbol]; exists {
		delete(symbolSubs, subscriptionID)
		if len(symbolSubs) == 0 {
			delete(s.SymbolSubscriptions, subscription.Symbol)
		}
	}

	// Close the channel
	close(subscription.Channel)

	// Unsubscribe from external provider
	provider, err := s.ExternalManager.GetDefaultProvider()
	if err != nil {
		return err
	}

	switch subscription.Type {
	case external.MarketDataTypeOrderBook:
		return provider.UnsubscribeOrderBook(ctx, subscription.Symbol)
	case external.MarketDataTypeTrade:
		return provider.UnsubscribeTrades(ctx, subscription.Symbol)
	case external.MarketDataTypeTicker:
		return provider.UnsubscribeTicker(ctx, subscription.Symbol)
	case external.MarketDataTypeOHLCV:
		return provider.UnsubscribeOHLCV(ctx, subscription.Symbol, subscription.Interval)
	}

	return nil
}

// GetOrderBook gets the current order book for a symbol
func (s *Service) GetOrderBook(ctx context.Context, symbol string) (*external.OrderBookData, error) {
	// Try to get from cache first
	if cached, found := s.Cache.Get("orderbook:" + symbol); found {
		if orderBook, ok := cached.(*external.OrderBookData); ok {
			return orderBook, nil
		}
	}

	// Get from external provider
	provider, err := s.ExternalManager.GetDefaultProvider()
	if err != nil {
		return nil, err
	}

	return provider.GetOrderBook(ctx, symbol)
}

// GetTrades gets recent trades for a symbol
func (s *Service) GetTrades(ctx context.Context, symbol string, limit int) ([]external.TradeData, error) {
	// Get from external provider
	provider, err := s.ExternalManager.GetDefaultProvider()
	if err != nil {
		return nil, err
	}

	return provider.GetTrades(ctx, symbol, limit)
}

// GetTicker gets the current ticker for a symbol
func (s *Service) GetTicker(ctx context.Context, symbol string) (*external.TickerData, error) {
	// Try to get from cache first
	if cached, found := s.Cache.Get("ticker:" + symbol); found {
		if ticker, ok := cached.(*external.TickerData); ok {
			return ticker, nil
		}
	}

	// Get from external provider
	provider, err := s.ExternalManager.GetDefaultProvider()
	if err != nil {
		return nil, err
	}

	return provider.GetTicker(ctx, symbol)
}

// GetOHLCV gets OHLCV data for a symbol
func (s *Service) GetOHLCV(ctx context.Context, symbol, interval string, limit int) ([]external.OHLCVData, error) {
	// Get from external provider
	provider, err := s.ExternalManager.GetDefaultProvider()
	if err != nil {
		return nil, err
	}

	return provider.GetOHLCV(ctx, symbol, interval, limit)
}

// GetHistoricalOHLCV gets historical OHLCV data from the database
func (s *Service) GetHistoricalOHLCV(ctx context.Context, symbol, interval string, start, end time.Time) ([]*db.MarketData, error) {
	return s.MarketDataRepository.GetBySymbolAndTimeRange(ctx, symbol, interval, start, end)
}

// AddMarketDataSource adds a new market data source
func (s *Service) AddMarketDataSource(name string, config map[string]interface{}) error {
	return s.ExternalManager.AddSource(name, config)
}

// GetMarketData gets market data from cache or external provider
func (s *Service) GetMarketData(ctx context.Context, symbol string, dataType external.MarketDataType) (interface{}, error) {
	cacheKey := fmt.Sprintf("%s:%s", dataType, symbol)

	// Try cache first
	if cached, found := s.Cache.Get(cacheKey); found {
		return cached, nil
	}

	// Get from external provider based on type
	provider, err := s.ExternalManager.GetDefaultProvider()
	if err != nil {
		return nil, err
	}

	switch dataType {
	case external.MarketDataTypeOrderBook:
		return provider.GetOrderBook(ctx, symbol)
	case external.MarketDataTypeTicker:
		return provider.GetTicker(ctx, symbol)
	case external.MarketDataTypeTrade:
		return provider.GetTrades(ctx, symbol, 100)
	default:
		return nil, fmt.Errorf("unsupported market data type: %s", dataType)
	}
}

// generateID generates a random ID for subscriptions
func generateID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// randomString generates a random string of specified length
func randomString(length int) string {
	bytes := make([]byte, length/2)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)[:length]
}
