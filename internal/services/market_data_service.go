package services

import (
	"context"
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/pkg/errors"
	"github.com/abdoElHodaky/tradSys/pkg/interfaces"
	"github.com/abdoElHodaky/tradSys/pkg/types"
)

// MarketDataService implements the MarketDataService interface
type MarketDataService struct {
	// Data storage
	marketData     map[string]*types.MarketData
	ohlcvData      map[string]map[string][]*types.OHLCV // symbol -> interval -> data
	symbols        map[string]*types.Symbol
	mu             sync.RWMutex

	// Dependencies
	publisher interfaces.EventPublisher
	logger    interfaces.Logger
	metrics   interfaces.MetricsCollector

	// Subscriptions
	marketDataSubscribers map[string][]func(*types.MarketData)
	ohlcvSubscribers      map[string]map[string][]func(*types.OHLCV) // symbol -> interval -> callbacks
	subscribersMu         sync.RWMutex

	// Configuration
	maxOHLCVHistory int
	updateInterval  time.Duration

	// Lifecycle
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewMarketDataService creates a new market data service
func NewMarketDataService(
	publisher interfaces.EventPublisher,
	logger interfaces.Logger,
	metrics interfaces.MetricsCollector,
) *MarketDataService {
	ctx, cancel := context.WithCancel(context.Background())

	return &MarketDataService{
		marketData:            make(map[string]*types.MarketData),
		ohlcvData:             make(map[string]map[string][]*types.OHLCV),
		symbols:               make(map[string]*types.Symbol),
		publisher:             publisher,
		logger:                logger,
		metrics:               metrics,
		marketDataSubscribers: make(map[string][]func(*types.MarketData)),
		ohlcvSubscribers:      make(map[string]map[string][]func(*types.OHLCV)),
		maxOHLCVHistory:       1000, // Keep last 1000 candles per interval
		updateInterval:        time.Second,
		ctx:                   ctx,
		cancel:                cancel,
	}
}

// Start starts the market data service
func (s *MarketDataService) Start() error {
	s.logger.Info("Starting market data service")

	// Start background updater
	s.wg.Add(1)
	go s.backgroundUpdater()

	return nil
}

// Stop stops the market data service
func (s *MarketDataService) Stop() error {
	s.logger.Info("Stopping market data service")

	s.cancel()
	s.wg.Wait()

	return nil
}

// GetMarketData gets current market data for a symbol
func (s *MarketDataService) GetMarketData(ctx context.Context, symbol string) (*types.MarketData, error) {
	start := time.Now()
	defer func() {
		s.metrics.RecordTimer("market_data_service.get_market_data.duration", time.Since(start), map[string]string{
			"symbol": symbol,
		})
	}()

	if symbol == "" {
		return nil, errors.New(errors.ErrInvalidInput, "symbol cannot be empty")
	}

	s.mu.RLock()
	data, exists := s.marketData[symbol]
	s.mu.RUnlock()

	if !exists {
		s.metrics.IncrementCounter("market_data_service.get_failed", map[string]string{
			"symbol": symbol,
			"error":  "not_found",
		})
		return nil, errors.New(errors.ErrSymbolNotFound, "market data not found for symbol")
	}

	s.metrics.IncrementCounter("market_data_service.get_success", map[string]string{
		"symbol": symbol,
	})

	// Return a copy to prevent external modification
	return &types.MarketData{
		Symbol:           data.Symbol,
		LastPrice:        data.LastPrice,
		BidPrice:         data.BidPrice,
		AskPrice:         data.AskPrice,
		Volume:           data.Volume,
		High24h:          data.High24h,
		Low24h:           data.Low24h,
		Change24h:        data.Change24h,
		ChangePercent24h: data.ChangePercent24h,
		Timestamp:        data.Timestamp,
	}, nil
}

// GetOHLCV gets OHLCV data for a symbol
func (s *MarketDataService) GetOHLCV(ctx context.Context, symbol string, interval string, limit int) ([]*types.OHLCV, error) {
	start := time.Now()
	defer func() {
		s.metrics.RecordTimer("market_data_service.get_ohlcv.duration", time.Since(start), map[string]string{
			"symbol":   symbol,
			"interval": interval,
		})
	}()

	if symbol == "" {
		return nil, errors.New(errors.ErrInvalidInput, "symbol cannot be empty")
	}

	if interval == "" {
		return nil, errors.New(errors.ErrInvalidInput, "interval cannot be empty")
	}

	if limit <= 0 {
		limit = 100 // Default limit
	}
	if limit > 1000 {
		limit = 1000 // Maximum limit
	}

	s.mu.RLock()
	symbolData, exists := s.ohlcvData[symbol]
	if !exists {
		s.mu.RUnlock()
		s.metrics.IncrementCounter("market_data_service.get_ohlcv_failed", map[string]string{
			"symbol":   symbol,
			"interval": interval,
			"error":    "symbol_not_found",
		})
		return nil, errors.New(errors.ErrSymbolNotFound, "OHLCV data not found for symbol")
	}

	intervalData, exists := symbolData[interval]
	if !exists {
		s.mu.RUnlock()
		s.metrics.IncrementCounter("market_data_service.get_ohlcv_failed", map[string]string{
			"symbol":   symbol,
			"interval": interval,
			"error":    "interval_not_found",
		})
		return nil, errors.New(errors.ErrInvalidInput, "OHLCV data not found for interval")
	}

	// Get the most recent data up to the limit
	dataLen := len(intervalData)
	startIdx := 0
	if dataLen > limit {
		startIdx = dataLen - limit
	}

	result := make([]*types.OHLCV, 0, limit)
	for i := startIdx; i < dataLen; i++ {
		// Return copies to prevent external modification
		ohlcv := intervalData[i]
		result = append(result, &types.OHLCV{
			Symbol:    ohlcv.Symbol,
			Open:      ohlcv.Open,
			High:      ohlcv.High,
			Low:       ohlcv.Low,
			Close:     ohlcv.Close,
			Volume:    ohlcv.Volume,
			Timestamp: ohlcv.Timestamp,
			Interval:  ohlcv.Interval,
		})
	}
	s.mu.RUnlock()

	s.metrics.IncrementCounter("market_data_service.get_ohlcv_success", map[string]string{
		"symbol":   symbol,
		"interval": interval,
	})
	s.metrics.RecordGauge("market_data_service.ohlcv_returned", float64(len(result)), map[string]string{
		"symbol":   symbol,
		"interval": interval,
	})

	return result, nil
}

// SubscribeMarketData subscribes to market data updates
func (s *MarketDataService) SubscribeMarketData(symbol string, callback func(*types.MarketData)) error {
	if symbol == "" {
		return errors.New(errors.ErrInvalidInput, "symbol cannot be empty")
	}

	if callback == nil {
		return errors.New(errors.ErrInvalidInput, "callback cannot be nil")
	}

	s.subscribersMu.Lock()
	defer s.subscribersMu.Unlock()

	s.marketDataSubscribers[symbol] = append(s.marketDataSubscribers[symbol], callback)

	s.metrics.IncrementCounter("market_data_service.subscriptions", map[string]string{
		"symbol": symbol,
		"type":   "market_data",
	})

	s.logger.Debug("Market data subscription added", "symbol", symbol)
	return nil
}

// SubscribeOHLCV subscribes to OHLCV updates
func (s *MarketDataService) SubscribeOHLCV(symbol string, interval string, callback func(*types.OHLCV)) error {
	if symbol == "" {
		return errors.New(errors.ErrInvalidInput, "symbol cannot be empty")
	}

	if interval == "" {
		return errors.New(errors.ErrInvalidInput, "interval cannot be empty")
	}

	if callback == nil {
		return errors.New(errors.ErrInvalidInput, "callback cannot be nil")
	}

	s.subscribersMu.Lock()
	defer s.subscribersMu.Unlock()

	if s.ohlcvSubscribers[symbol] == nil {
		s.ohlcvSubscribers[symbol] = make(map[string][]func(*types.OHLCV))
	}

	s.ohlcvSubscribers[symbol][interval] = append(s.ohlcvSubscribers[symbol][interval], callback)

	s.metrics.IncrementCounter("market_data_service.subscriptions", map[string]string{
		"symbol":   symbol,
		"interval": interval,
		"type":     "ohlcv",
	})

	s.logger.Debug("OHLCV subscription added", "symbol", symbol, "interval", interval)
	return nil
}

// GetSymbols gets all available symbols
func (s *MarketDataService) GetSymbols(ctx context.Context) ([]*types.Symbol, error) {
	start := time.Now()
	defer func() {
		s.metrics.RecordTimer("market_data_service.get_symbols.duration", time.Since(start), nil)
	}()

	s.mu.RLock()
	defer s.mu.RUnlock()

	symbols := make([]*types.Symbol, 0, len(s.symbols))
	for _, symbol := range s.symbols {
		// Return copies to prevent external modification
		symbols = append(symbols, &types.Symbol{
			Symbol:      symbol.Symbol,
			BaseAsset:   symbol.BaseAsset,
			QuoteAsset:  symbol.QuoteAsset,
			Status:      symbol.Status,
			MinPrice:    symbol.MinPrice,
			MaxPrice:    symbol.MaxPrice,
			TickSize:    symbol.TickSize,
			MinQuantity: symbol.MinQuantity,
			MaxQuantity: symbol.MaxQuantity,
			StepSize:    symbol.StepSize,
			MinNotional: symbol.MinNotional,
		})
	}

	s.metrics.IncrementCounter("market_data_service.get_symbols_success", nil)
	s.metrics.RecordGauge("market_data_service.symbols_returned", float64(len(symbols)), nil)

	return symbols, nil
}

// UpdateMarketData updates market data for a symbol
func (s *MarketDataService) UpdateMarketData(symbol string, data *types.MarketData) error {
	if symbol == "" {
		return errors.New(errors.ErrInvalidInput, "symbol cannot be empty")
	}

	if data == nil {
		return errors.New(errors.ErrInvalidInput, "market data cannot be nil")
	}

	s.mu.Lock()
	data.Timestamp = time.Now()
	s.marketData[symbol] = data
	s.mu.Unlock()

	// Notify subscribers
	s.notifyMarketDataSubscribers(symbol, data)

	// Publish event
	if s.publisher != nil {
		event := &interfaces.MarketDataEvent{
			Type:       interfaces.MarketDataEventTick,
			Symbol:     symbol,
			MarketData: data,
			Timestamp:  time.Now(),
		}
		if err := s.publisher.PublishMarketDataEvent(context.Background(), event); err != nil {
			s.logger.Error("Failed to publish market data event", "error", err, "symbol", symbol)
		}
	}

	s.metrics.IncrementCounter("market_data_service.updated", map[string]string{
		"symbol": symbol,
		"type":   "market_data",
	})

	return nil
}

// UpdateOHLCV updates OHLCV data for a symbol
func (s *MarketDataService) UpdateOHLCV(symbol string, interval string, ohlcv *types.OHLCV) error {
	if symbol == "" {
		return errors.New(errors.ErrInvalidInput, "symbol cannot be empty")
	}

	if interval == "" {
		return errors.New(errors.ErrInvalidInput, "interval cannot be empty")
	}

	if ohlcv == nil {
		return errors.New(errors.ErrInvalidInput, "OHLCV data cannot be nil")
	}

	s.mu.Lock()
	if s.ohlcvData[symbol] == nil {
		s.ohlcvData[symbol] = make(map[string][]*types.OHLCV)
	}

	ohlcv.Timestamp = time.Now()
	ohlcv.Symbol = symbol
	ohlcv.Interval = interval

	// Add to the data array
	s.ohlcvData[symbol][interval] = append(s.ohlcvData[symbol][interval], ohlcv)

	// Trim history if it exceeds the maximum
	if len(s.ohlcvData[symbol][interval]) > s.maxOHLCVHistory {
		s.ohlcvData[symbol][interval] = s.ohlcvData[symbol][interval][1:]
	}
	s.mu.Unlock()

	// Notify subscribers
	s.notifyOHLCVSubscribers(symbol, interval, ohlcv)

	// Publish event
	if s.publisher != nil {
		event := &interfaces.MarketDataEvent{
			Type:      interfaces.MarketDataEventOHLCV,
			Symbol:    symbol,
			OHLCV:     ohlcv,
			Timestamp: time.Now(),
		}
		if err := s.publisher.PublishMarketDataEvent(context.Background(), event); err != nil {
			s.logger.Error("Failed to publish OHLCV event", "error", err, "symbol", symbol, "interval", interval)
		}
	}

	s.metrics.IncrementCounter("market_data_service.updated", map[string]string{
		"symbol":   symbol,
		"interval": interval,
		"type":     "ohlcv",
	})

	return nil
}

// AddSymbol adds a new symbol
func (s *MarketDataService) AddSymbol(symbol *types.Symbol) error {
	if symbol == nil {
		return errors.New(errors.ErrInvalidInput, "symbol cannot be nil")
	}

	if symbol.Symbol == "" {
		return errors.New(errors.ErrMissingField, "symbol name is required")
	}

	s.mu.Lock()
	s.symbols[symbol.Symbol] = symbol
	s.mu.Unlock()

	s.metrics.IncrementCounter("market_data_service.symbol_added", map[string]string{
		"symbol": symbol.Symbol,
	})

	s.logger.Info("Symbol added", "symbol", symbol.Symbol, "base_asset", symbol.BaseAsset, "quote_asset", symbol.QuoteAsset)
	return nil
}

// GetMarketDataStatistics returns statistics about market data
func (s *MarketDataService) GetMarketDataStatistics() *MarketDataStatistics {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := &MarketDataStatistics{
		TotalSymbols:     len(s.symbols),
		ActiveSymbols:    len(s.marketData),
		SymbolStats:      make(map[string]*SymbolMarketDataStats),
		IntervalStats:    make(map[string]int),
	}

	// Count OHLCV data by interval
	for symbol, intervals := range s.ohlcvData {
		symbolStats := &SymbolMarketDataStats{
			Symbol:        symbol,
			IntervalCount: len(intervals),
		}

		for interval, data := range intervals {
			stats.IntervalStats[interval] += len(data)
			symbolStats.TotalCandles += len(data)
		}

		stats.SymbolStats[symbol] = symbolStats
	}

	// Count subscribers
	s.subscribersMu.RLock()
	stats.MarketDataSubscribers = len(s.marketDataSubscribers)
	for _, intervals := range s.ohlcvSubscribers {
		for _, callbacks := range intervals {
			stats.OHLCVSubscribers += len(callbacks)
		}
	}
	s.subscribersMu.RUnlock()

	return stats
}

// backgroundUpdater runs background tasks
func (s *MarketDataService) backgroundUpdater() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.updateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.performBackgroundTasks()
		}
	}
}

// performBackgroundTasks performs periodic background tasks
func (s *MarketDataService) performBackgroundTasks() {
	// Update metrics
	stats := s.GetMarketDataStatistics()
	s.metrics.RecordGauge("market_data_service.total_symbols", float64(stats.TotalSymbols), nil)
	s.metrics.RecordGauge("market_data_service.active_symbols", float64(stats.ActiveSymbols), nil)
	s.metrics.RecordGauge("market_data_service.market_data_subscribers", float64(stats.MarketDataSubscribers), nil)
	s.metrics.RecordGauge("market_data_service.ohlcv_subscribers", float64(stats.OHLCVSubscribers), nil)

	// Clean up old data if needed
	s.cleanupOldData()
}

// cleanupOldData removes old OHLCV data beyond the maximum history
func (s *MarketDataService) cleanupOldData() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for symbol, intervals := range s.ohlcvData {
		for interval, data := range intervals {
			if len(data) > s.maxOHLCVHistory {
				// Keep only the most recent data
				s.ohlcvData[symbol][interval] = data[len(data)-s.maxOHLCVHistory:]
			}
		}
	}
}

// notifyMarketDataSubscribers notifies all market data subscribers
func (s *MarketDataService) notifyMarketDataSubscribers(symbol string, data *types.MarketData) {
	s.subscribersMu.RLock()
	callbacks, exists := s.marketDataSubscribers[symbol]
	s.subscribersMu.RUnlock()

	if !exists || len(callbacks) == 0 {
		return
	}

	// Notify all subscribers asynchronously
	for _, callback := range callbacks {
		go func(cb func(*types.MarketData)) {
			defer func() {
				if r := recover(); r != nil {
					s.logger.Error("Market data callback panicked", "error", r, "symbol", symbol)
				}
			}()
			cb(data)
		}(callback)
	}
}

// notifyOHLCVSubscribers notifies all OHLCV subscribers
func (s *MarketDataService) notifyOHLCVSubscribers(symbol string, interval string, ohlcv *types.OHLCV) {
	s.subscribersMu.RLock()
	symbolCallbacks, exists := s.ohlcvSubscribers[symbol]
	if !exists {
		s.subscribersMu.RUnlock()
		return
	}

	callbacks, exists := symbolCallbacks[interval]
	s.subscribersMu.RUnlock()

	if !exists || len(callbacks) == 0 {
		return
	}

	// Notify all subscribers asynchronously
	for _, callback := range callbacks {
		go func(cb func(*types.OHLCV)) {
			defer func() {
				if r := recover(); r != nil {
					s.logger.Error("OHLCV callback panicked", "error", r, "symbol", symbol, "interval", interval)
				}
			}()
			cb(ohlcv)
		}(callback)
	}
}

// MarketDataStatistics contains statistics about market data
type MarketDataStatistics struct {
	TotalSymbols            int                              `json:"total_symbols"`
	ActiveSymbols           int                              `json:"active_symbols"`
	MarketDataSubscribers   int                              `json:"market_data_subscribers"`
	OHLCVSubscribers        int                              `json:"ohlcv_subscribers"`
	SymbolStats             map[string]*SymbolMarketDataStats `json:"symbol_stats"`
	IntervalStats           map[string]int                   `json:"interval_stats"`
}

// SymbolMarketDataStats contains statistics for a specific symbol
type SymbolMarketDataStats struct {
	Symbol        string `json:"symbol"`
	IntervalCount int    `json:"interval_count"`
	TotalCandles  int    `json:"total_candles"`
}
