package timeframe

import (
	"sync"
	"time"

	"go.uber.org/zap"
)

// TimeframeInterval represents a timeframe interval
type TimeframeInterval string

// Timeframe intervals
const (
	Interval1m  TimeframeInterval = "1m"
	Interval5m  TimeframeInterval = "5m"
	Interval15m TimeframeInterval = "15m"
	Interval30m TimeframeInterval = "30m"
	Interval1h  TimeframeInterval = "1h"
	Interval4h  TimeframeInterval = "4h"
	Interval1d  TimeframeInterval = "1d"
)

// OHLCV represents an OHLCV candle
type OHLCV struct {
	// Symbol is the trading symbol
	Symbol string

	// Interval is the timeframe interval
	Interval TimeframeInterval

	// Timestamp is the timestamp of the candle
	Timestamp time.Time

	// Open is the open price
	Open float64

	// High is the high price
	High float64

	// Low is the low price
	Low float64

	// Close is the close price
	Close float64

	// Volume is the volume
	Volume float64

	// TradeCount is the number of trades
	TradeCount int
}

// Trade represents a trade
type Trade struct {
	// Symbol is the trading symbol
	Symbol string

	// Price is the price of the trade
	Price float64

	// Volume is the volume of the trade
	Volume float64

	// Timestamp is the timestamp of the trade
	Timestamp time.Time
}

// TimeframeAggregator aggregates trades into OHLCV candles
type TimeframeAggregator struct {
	// Logger
	logger *zap.Logger

	// Current candles
	currentCandles map[string]map[TimeframeInterval]*OHLCV

	// Historical candles
	historicalCandles map[string]map[TimeframeInterval][]*OHLCV

	// Maximum number of historical candles to keep
	maxHistoricalCandles int

	// Mutex for thread safety
	mu sync.RWMutex

	// Subscribers
	subscribers map[string][]OHLCVCallback

	// Subscriber mutex
	subMu sync.RWMutex
}

// OHLCVCallback is a callback for OHLCV updates
type OHLCVCallback func(candle *OHLCV)

// NewTimeframeAggregator creates a new TimeframeAggregator
func NewTimeframeAggregator(logger *zap.Logger, maxHistoricalCandles int) *TimeframeAggregator {
	if maxHistoricalCandles <= 0 {
		maxHistoricalCandles = 1000
	}

	return &TimeframeAggregator{
		logger:              logger,
		currentCandles:      make(map[string]map[TimeframeInterval]*OHLCV),
		historicalCandles:   make(map[string]map[TimeframeInterval][]*OHLCV),
		maxHistoricalCandles: maxHistoricalCandles,
		subscribers:         make(map[string][]OHLCVCallback),
	}
}

// ProcessTrade processes a trade
func (a *TimeframeAggregator) ProcessTrade(trade *Trade) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Initialize maps if needed
	if _, exists := a.currentCandles[trade.Symbol]; !exists {
		a.currentCandles[trade.Symbol] = make(map[TimeframeInterval]*OHLCV)
	}

	if _, exists := a.historicalCandles[trade.Symbol]; !exists {
		a.historicalCandles[trade.Symbol] = make(map[TimeframeInterval][]*OHLCV)
	}

	// Process for each interval
	a.processTradeForInterval(trade, Interval1m)
	a.processTradeForInterval(trade, Interval5m)
	a.processTradeForInterval(trade, Interval15m)
	a.processTradeForInterval(trade, Interval30m)
	a.processTradeForInterval(trade, Interval1h)
	a.processTradeForInterval(trade, Interval4h)
	a.processTradeForInterval(trade, Interval1d)
}

// processTradeForInterval processes a trade for a specific interval
func (a *TimeframeAggregator) processTradeForInterval(trade *Trade, interval TimeframeInterval) {
	// Get the current candle
	candle, exists := a.currentCandles[trade.Symbol][interval]

	// Check if we need to create a new candle
	if !exists || !isInSameCandle(trade.Timestamp, candle.Timestamp, interval) {
		// If there was a previous candle, add it to historical candles
		if exists {
			if _, ok := a.historicalCandles[trade.Symbol][interval]; !ok {
				a.historicalCandles[trade.Symbol][interval] = make([]*OHLCV, 0, a.maxHistoricalCandles)
			}

			// Add to historical candles
			a.historicalCandles[trade.Symbol][interval] = append(
				a.historicalCandles[trade.Symbol][interval],
				candle,
			)

			// Trim historical candles if needed
			if len(a.historicalCandles[trade.Symbol][interval]) > a.maxHistoricalCandles {
				a.historicalCandles[trade.Symbol][interval] = a.historicalCandles[trade.Symbol][interval][1:]
			}

			// Notify subscribers
			a.notifySubscribers(candle)
		}

		// Create a new candle
		candle = &OHLCV{
			Symbol:     trade.Symbol,
			Interval:   interval,
			Timestamp:  normalizeTimestamp(trade.Timestamp, interval),
			Open:       trade.Price,
			High:       trade.Price,
			Low:        trade.Price,
			Close:      trade.Price,
			Volume:     trade.Volume,
			TradeCount: 1,
		}

		a.currentCandles[trade.Symbol][interval] = candle
	} else {
		// Update existing candle
		candle.High = max(candle.High, trade.Price)
		candle.Low = min(candle.Low, trade.Price)
		candle.Close = trade.Price
		candle.Volume += trade.Volume
		candle.TradeCount++
	}
}

// GetCurrentCandle gets the current candle for a symbol and interval
func (a *TimeframeAggregator) GetCurrentCandle(symbol string, interval TimeframeInterval) *OHLCV {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if _, exists := a.currentCandles[symbol]; !exists {
		return nil
	}

	return a.currentCandles[symbol][interval]
}

// GetHistoricalCandles gets historical candles for a symbol and interval
func (a *TimeframeAggregator) GetHistoricalCandles(
	symbol string,
	interval TimeframeInterval,
	limit int,
) []*OHLCV {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if _, exists := a.historicalCandles[symbol]; !exists {
		return nil
	}

	candles := a.historicalCandles[symbol][interval]
	if candles == nil {
		return nil
	}

	if limit <= 0 || limit > len(candles) {
		limit = len(candles)
	}

	result := make([]*OHLCV, limit)
	copy(result, candles[len(candles)-limit:])

	return result
}

// SubscribeOHLCV subscribes to OHLCV updates
func (a *TimeframeAggregator) SubscribeOHLCV(symbol string, callback OHLCVCallback) {
	a.subMu.Lock()
	defer a.subMu.Unlock()

	if _, exists := a.subscribers[symbol]; !exists {
		a.subscribers[symbol] = make([]OHLCVCallback, 0)
	}

	a.subscribers[symbol] = append(a.subscribers[symbol], callback)
}

// UnsubscribeOHLCV unsubscribes from OHLCV updates
func (a *TimeframeAggregator) UnsubscribeOHLCV(symbol string, callback OHLCVCallback) {
	a.subMu.Lock()
	defer a.subMu.Unlock()

	if _, exists := a.subscribers[symbol]; !exists {
		return
	}

	// Find and remove the callback
	for i, cb := range a.subscribers[symbol] {
		if &cb == &callback {
			a.subscribers[symbol] = append(
				a.subscribers[symbol][:i],
				a.subscribers[symbol][i+1:]...,
			)
			break
		}
	}
}

// notifySubscribers notifies subscribers of an OHLCV update
func (a *TimeframeAggregator) notifySubscribers(candle *OHLCV) {
	a.subMu.RLock()
	defer a.subMu.RUnlock()

	if callbacks, exists := a.subscribers[candle.Symbol]; exists {
		for _, callback := range callbacks {
			callback(candle)
		}
	}
}

// Helper functions

// isInSameCandle checks if two timestamps are in the same candle
func isInSameCandle(t1, t2 time.Time, interval TimeframeInterval) bool {
	switch interval {
	case Interval1m:
		return t1.Year() == t2.Year() && t1.Month() == t2.Month() && t1.Day() == t2.Day() &&
			t1.Hour() == t2.Hour() && t1.Minute() == t2.Minute()
	case Interval5m:
		return t1.Year() == t2.Year() && t1.Month() == t2.Month() && t1.Day() == t2.Day() &&
			t1.Hour() == t2.Hour() && t1.Minute()/5 == t2.Minute()/5
	case Interval15m:
		return t1.Year() == t2.Year() && t1.Month() == t2.Month() && t1.Day() == t2.Day() &&
			t1.Hour() == t2.Hour() && t1.Minute()/15 == t2.Minute()/15
	case Interval30m:
		return t1.Year() == t2.Year() && t1.Month() == t2.Month() && t1.Day() == t2.Day() &&
			t1.Hour() == t2.Hour() && t1.Minute()/30 == t2.Minute()/30
	case Interval1h:
		return t1.Year() == t2.Year() && t1.Month() == t2.Month() && t1.Day() == t2.Day() &&
			t1.Hour() == t2.Hour()
	case Interval4h:
		return t1.Year() == t2.Year() && t1.Month() == t2.Month() && t1.Day() == t2.Day() &&
			t1.Hour()/4 == t2.Hour()/4
	case Interval1d:
		return t1.Year() == t2.Year() && t1.Month() == t2.Month() && t1.Day() == t2.Day()
	default:
		return false
	}
}

// normalizeTimestamp normalizes a timestamp to the start of a candle
func normalizeTimestamp(t time.Time, interval TimeframeInterval) time.Time {
	switch interval {
	case Interval1m:
		return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), 0, 0, t.Location())
	case Interval5m:
		minute := t.Minute() / 5 * 5
		return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), minute, 0, 0, t.Location())
	case Interval15m:
		minute := t.Minute() / 15 * 15
		return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), minute, 0, 0, t.Location())
	case Interval30m:
		minute := t.Minute() / 30 * 30
		return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), minute, 0, 0, t.Location())
	case Interval1h:
		return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), 0, 0, 0, t.Location())
	case Interval4h:
		hour := t.Hour() / 4 * 4
		return time.Date(t.Year(), t.Month(), t.Day(), hour, 0, 0, 0, t.Location())
	case Interval1d:
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	default:
		return t
	}
}

// max returns the maximum of two float64 values
func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// min returns the minimum of two float64 values
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

