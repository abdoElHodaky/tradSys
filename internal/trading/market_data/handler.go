package market_data

import (
	"context"
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/trading/order_matching"
	"go.uber.org/zap"
)

// MarketDataType represents the type of market data
type MarketDataType string

const (
	// MarketDataTypeTrade represents a trade
	MarketDataTypeTrade MarketDataType = "trade"
	// MarketDataTypeOrderBook represents an order book
	MarketDataTypeOrderBook MarketDataType = "order_book"
	// MarketDataTypeTicker represents a ticker
	MarketDataTypeTicker MarketDataType = "ticker"
	// MarketDataTypeOHLCV represents OHLCV data
	MarketDataTypeOHLCV MarketDataType = "ohlcv"
)

// MarketData represents market data
type MarketData struct {
	// Type is the type of market data
	Type MarketDataType
	// Symbol is the trading symbol
	Symbol string
	// Timestamp is the time the market data was generated
	Timestamp time.Time
	// Data is the market data
	Data interface{}
}

// Trade represents a trade
type Trade struct {
	// ID is the unique identifier for the trade
	ID string
	// Symbol is the trading symbol
	Symbol string
	// Price is the price of the trade
	Price float64
	// Quantity is the quantity of the trade
	Quantity float64
	// Timestamp is the time the trade was executed
	Timestamp time.Time
	// Side is the side of the taker
	Side string
}

// OrderBook represents an order book
type OrderBook struct {
	// Symbol is the trading symbol
	Symbol string
	// Bids is the buy orders
	Bids [][]float64
	// Asks is the sell orders
	Asks [][]float64
	// Timestamp is the time the order book was generated
	Timestamp time.Time
}

// Ticker represents a ticker
type Ticker struct {
	// Symbol is the trading symbol
	Symbol string
	// LastPrice is the last traded price
	LastPrice float64
	// BidPrice is the highest bid price
	BidPrice float64
	// AskPrice is the lowest ask price
	AskPrice float64
	// Volume is the 24-hour volume
	Volume float64
	// High is the 24-hour high
	High float64
	// Low is the 24-hour low
	Low float64
	// Timestamp is the time the ticker was generated
	Timestamp time.Time
}

// OHLCV represents OHLCV data
type OHLCV struct {
	// Symbol is the trading symbol
	Symbol string
	// Timestamp is the time the OHLCV data was generated
	Timestamp time.Time
	// Open is the opening price
	Open float64
	// High is the highest price
	High float64
	// Low is the lowest price
	Low float64
	// Close is the closing price
	Close float64
	// Volume is the volume
	Volume float64
}

// Subscription represents a market data subscription
type Subscription struct {
	// ID is the unique identifier for the subscription
	ID string
	// Type is the type of market data
	Type MarketDataType
	// Symbol is the trading symbol
	Symbol string
	// Channel is the channel to send market data to
	Channel chan *MarketData
	// Interval is the interval for OHLCV data
	Interval string
}

// Handler represents a market data handler
type Handler struct {
	// Engine is the order matching engine
	Engine *order_matching.Engine
	// Subscriptions is a map of subscription ID to subscription
	Subscriptions map[string]*Subscription
	// SymbolSubscriptions is a map of symbol to subscriptions
	SymbolSubscriptions map[string]map[string]*Subscription
	// TypeSubscriptions is a map of market data type to subscriptions
	TypeSubscriptions map[MarketDataType]map[string]*Subscription
	// OHLCV is a map of symbol to OHLCV data
	OHLCV map[string]map[string]*OHLCV
	// Tickers is a map of symbol to ticker
	Tickers map[string]*Ticker
	// Mutex for thread safety
	mu sync.RWMutex
	// Logger
	logger *zap.Logger
	// Context
	ctx context.Context
	// Cancel function
	cancel context.CancelFunc
}

// NewHandler creates a new market data handler
func NewHandler(engine *order_matching.Engine, logger *zap.Logger) *Handler {
	ctx, cancel := context.WithCancel(context.Background())
	
	handler := &Handler{
		Engine:              engine,
		Subscriptions:       make(map[string]*Subscription),
		SymbolSubscriptions: make(map[string]map[string]*Subscription),
		TypeSubscriptions:   make(map[MarketDataType]map[string]*Subscription),
		OHLCV:               make(map[string]map[string]*OHLCV),
		Tickers:             make(map[string]*Ticker),
		logger:              logger,
		ctx:                 ctx,
		cancel:              cancel,
	}

	// Initialize market data types
	handler.TypeSubscriptions[MarketDataTypeTrade] = make(map[string]*Subscription)
	handler.TypeSubscriptions[MarketDataTypeOrderBook] = make(map[string]*Subscription)
	handler.TypeSubscriptions[MarketDataTypeTicker] = make(map[string]*Subscription)
	handler.TypeSubscriptions[MarketDataTypeOHLCV] = make(map[string]*Subscription)

	// Start processing trades
	go handler.processTrades()

	// Start generating order books
	go handler.generateOrderBooks()

	// Start generating tickers
	go handler.generateTickers()

	// Start generating OHLCV data
	go handler.generateOHLCV()

	return handler
}

// processTrades processes trades from the order matching engine
func (h *Handler) processTrades() {
	for {
		select {
		case <-h.ctx.Done():
			return
		case trade := <-h.Engine.TradeChannel:
			// Convert to market data trade
			mdTrade := &Trade{
				ID:        trade.ID,
				Symbol:    trade.Symbol,
				Price:     trade.Price,
				Quantity:  trade.Quantity,
				Timestamp: trade.Timestamp,
				Side:      string(trade.TakerSide),
			}

			// Create market data
			marketData := &MarketData{
				Type:      MarketDataTypeTrade,
				Symbol:    trade.Symbol,
				Timestamp: trade.Timestamp,
				Data:      mdTrade,
			}

			// Update ticker
			h.updateTicker(trade)

			// Update OHLCV
			h.updateOHLCV(trade)

			// Publish to subscribers
			h.publishMarketData(marketData)
		}
	}
}

// generateOrderBooks generates order books
func (h *Handler) generateOrderBooks() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-h.ctx.Done():
			return
		case <-ticker.C:
			h.mu.RLock()
			symbols := make([]string, 0, len(h.SymbolSubscriptions))
			for symbol := range h.SymbolSubscriptions {
				symbols = append(symbols, symbol)
			}
			h.mu.RUnlock()

			for _, symbol := range symbols {
				// Get order book
				bids, asks, _, err := h.Engine.GetMarketData(symbol, 10)
				if err != nil {
					h.logger.Error("Failed to get order book",
						zap.String("symbol", symbol),
						zap.Error(err))
					continue
				}

				// Create order book
				orderBook := &OrderBook{
					Symbol:    symbol,
					Bids:      bids,
					Asks:      asks,
					Timestamp: time.Now(),
				}

				// Create market data
				marketData := &MarketData{
					Type:      MarketDataTypeOrderBook,
					Symbol:    symbol,
					Timestamp: time.Now(),
					Data:      orderBook,
				}

				// Publish to subscribers
				h.publishMarketData(marketData)
			}
		}
	}
}

// generateTickers generates tickers
func (h *Handler) generateTickers() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-h.ctx.Done():
			return
		case <-ticker.C:
			h.mu.RLock()
			tickers := make(map[string]*Ticker)
			for symbol, ticker := range h.Tickers {
				tickers[symbol] = ticker
			}
			h.mu.RUnlock()

			for symbol, ticker := range tickers {
				// Create market data
				marketData := &MarketData{
					Type:      MarketDataTypeTicker,
					Symbol:    symbol,
					Timestamp: time.Now(),
					Data:      ticker,
				}

				// Publish to subscribers
				h.publishMarketData(marketData)
			}
		}
	}
}

// generateOHLCV generates OHLCV data
func (h *Handler) generateOHLCV() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-h.ctx.Done():
			return
		case <-ticker.C:
			h.mu.RLock()
			ohlcvData := make(map[string]map[string]*OHLCV)
			for symbol, intervals := range h.OHLCV {
				ohlcvData[symbol] = make(map[string]*OHLCV)
				for interval, ohlcv := range intervals {
					ohlcvData[symbol][interval] = ohlcv
				}
			}
			h.mu.RUnlock()

			for symbol, intervals := range ohlcvData {
				for interval, ohlcv := range intervals {
					// Check if interval has elapsed
					switch interval {
					case "1m":
						if time.Since(ohlcv.Timestamp) < 1*time.Minute {
							continue
						}
					case "5m":
						if time.Since(ohlcv.Timestamp) < 5*time.Minute {
							continue
						}
					case "15m":
						if time.Since(ohlcv.Timestamp) < 15*time.Minute {
							continue
						}
					case "30m":
						if time.Since(ohlcv.Timestamp) < 30*time.Minute {
							continue
						}
					case "1h":
						if time.Since(ohlcv.Timestamp) < 1*time.Hour {
							continue
						}
					case "4h":
						if time.Since(ohlcv.Timestamp) < 4*time.Hour {
							continue
						}
					case "1d":
						if time.Since(ohlcv.Timestamp) < 24*time.Hour {
							continue
						}
					default:
						continue
					}

					// Create new OHLCV
					newOHLCV := &OHLCV{
						Symbol:    symbol,
						Timestamp: time.Now(),
						Open:      ohlcv.Close,
						High:      ohlcv.Close,
						Low:       ohlcv.Close,
						Close:     ohlcv.Close,
						Volume:    0,
					}

					// Update OHLCV
					h.mu.Lock()
					h.OHLCV[symbol][interval] = newOHLCV
					h.mu.Unlock()

					// Create market data
					marketData := &MarketData{
						Type:      MarketDataTypeOHLCV,
						Symbol:    symbol,
						Timestamp: time.Now(),
						Data:      ohlcv,
					}

					// Publish to subscribers
					h.publishMarketData(marketData)
				}
			}
		}
	}
}

// updateTicker updates a ticker
func (h *Handler) updateTicker(trade *order_matching.Trade) {
	h.mu.Lock()
	defer h.mu.Unlock()

	ticker, exists := h.Tickers[trade.Symbol]
	if !exists {
		// Create new ticker
		ticker = &Ticker{
			Symbol:    trade.Symbol,
			LastPrice: trade.Price,
			BidPrice:  0,
			AskPrice:  0,
			Volume:    trade.Quantity,
			High:      trade.Price,
			Low:       trade.Price,
			Timestamp: trade.Timestamp,
		}
		h.Tickers[trade.Symbol] = ticker
	} else {
		// Update ticker
		ticker.LastPrice = trade.Price
		ticker.Volume += trade.Quantity
		ticker.Timestamp = trade.Timestamp

		// Update high and low
		if trade.Price > ticker.High {
			ticker.High = trade.Price
		}
		if trade.Price < ticker.Low {
			ticker.Low = trade.Price
		}
	}

	// Update bid and ask prices
	bids, asks, _, err := h.Engine.GetMarketData(trade.Symbol, 1)
	if err == nil {
		if len(bids) > 0 {
			ticker.BidPrice = bids[0][0]
		}
		if len(asks) > 0 {
			ticker.AskPrice = asks[0][0]
		}
	}
}

// updateOHLCV updates OHLCV data
func (h *Handler) updateOHLCV(trade *order_matching.Trade) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Get OHLCV data for symbol
	intervals, exists := h.OHLCV[trade.Symbol]
	if !exists {
		// Create new OHLCV data
		intervals = make(map[string]*OHLCV)
		h.OHLCV[trade.Symbol] = intervals
	}

	// Update OHLCV data for each interval
	for _, interval := range []string{"1m", "5m", "15m", "30m", "1h", "4h", "1d"} {
		ohlcv, exists := intervals[interval]
		if !exists {
			// Create new OHLCV
			ohlcv = &OHLCV{
				Symbol:    trade.Symbol,
				Timestamp: trade.Timestamp,
				Open:      trade.Price,
				High:      trade.Price,
				Low:       trade.Price,
				Close:     trade.Price,
				Volume:    trade.Quantity,
			}
			intervals[interval] = ohlcv
		} else {
			// Check if interval has elapsed
			var elapsed bool
			switch interval {
			case "1m":
				elapsed = time.Since(ohlcv.Timestamp) >= 1*time.Minute
			case "5m":
				elapsed = time.Since(ohlcv.Timestamp) >= 5*time.Minute
			case "15m":
				elapsed = time.Since(ohlcv.Timestamp) >= 15*time.Minute
			case "30m":
				elapsed = time.Since(ohlcv.Timestamp) >= 30*time.Minute
			case "1h":
				elapsed = time.Since(ohlcv.Timestamp) >= 1*time.Hour
			case "4h":
				elapsed = time.Since(ohlcv.Timestamp) >= 4*time.Hour
			case "1d":
				elapsed = time.Since(ohlcv.Timestamp) >= 24*time.Hour
			}

			if elapsed {
				// Create new OHLCV
				ohlcv = &OHLCV{
					Symbol:    trade.Symbol,
					Timestamp: trade.Timestamp,
					Open:      trade.Price,
					High:      trade.Price,
					Low:       trade.Price,
					Close:     trade.Price,
					Volume:    trade.Quantity,
				}
				intervals[interval] = ohlcv
			} else {
				// Update OHLCV
				ohlcv.Close = trade.Price
				ohlcv.Volume += trade.Quantity

				// Update high and low
				if trade.Price > ohlcv.High {
					ohlcv.High = trade.Price
				}
				if trade.Price < ohlcv.Low {
					ohlcv.Low = trade.Price
				}
			}
		}
	}
}

// publishMarketData publishes market data to subscribers
func (h *Handler) publishMarketData(marketData *MarketData) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// Get subscribers for symbol
	symbolSubs, exists := h.SymbolSubscriptions[marketData.Symbol]
	if exists {
		for _, sub := range symbolSubs {
			if sub.Type == marketData.Type || sub.Type == "" {
				select {
				case sub.Channel <- marketData:
				default:
					h.logger.Warn("Subscription channel full, dropping market data",
						zap.String("subscription_id", sub.ID),
						zap.String("symbol", marketData.Symbol),
						zap.String("type", string(marketData.Type)))
				}
			}
		}
	}

	// Get subscribers for type
	typeSubs, exists := h.TypeSubscriptions[marketData.Type]
	if exists {
		for _, sub := range typeSubs {
			if sub.Symbol == marketData.Symbol || sub.Symbol == "" {
				select {
				case sub.Channel <- marketData:
				default:
					h.logger.Warn("Subscription channel full, dropping market data",
						zap.String("subscription_id", sub.ID),
						zap.String("symbol", marketData.Symbol),
						zap.String("type", string(marketData.Type)))
				}
			}
		}
	}
}

// Subscribe subscribes to market data
func (h *Handler) Subscribe(subscription *Subscription) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Add to subscriptions
	h.Subscriptions[subscription.ID] = subscription

	// Add to symbol subscriptions
	if subscription.Symbol != "" {
		symbolSubs, exists := h.SymbolSubscriptions[subscription.Symbol]
		if !exists {
			symbolSubs = make(map[string]*Subscription)
			h.SymbolSubscriptions[subscription.Symbol] = symbolSubs
		}
		symbolSubs[subscription.ID] = subscription
	}

	// Add to type subscriptions
	if subscription.Type != "" {
		typeSubs, exists := h.TypeSubscriptions[subscription.Type]
		if !exists {
			typeSubs = make(map[string]*Subscription)
			h.TypeSubscriptions[subscription.Type] = typeSubs
		}
		typeSubs[subscription.ID] = subscription
	}

	return nil
}

// Unsubscribe unsubscribes from market data
func (h *Handler) Unsubscribe(subscriptionID string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Get subscription
	subscription, exists := h.Subscriptions[subscriptionID]
	if !exists {
		return ErrSubscriptionNotFound
	}

	// Remove from subscriptions
	delete(h.Subscriptions, subscriptionID)

	// Remove from symbol subscriptions
	if subscription.Symbol != "" {
		symbolSubs, exists := h.SymbolSubscriptions[subscription.Symbol]
		if exists {
			delete(symbolSubs, subscriptionID)
			if len(symbolSubs) == 0 {
				delete(h.SymbolSubscriptions, subscription.Symbol)
			}
		}
	}

	// Remove from type subscriptions
	if subscription.Type != "" {
		typeSubs, exists := h.TypeSubscriptions[subscription.Type]
		if exists {
			delete(typeSubs, subscriptionID)
			if len(typeSubs) == 0 {
				delete(h.TypeSubscriptions, subscription.Type)
			}
		}
	}

	// Close channel
	close(subscription.Channel)

	return nil
}

// GetTicker gets a ticker
func (h *Handler) GetTicker(symbol string) (*Ticker, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	ticker, exists := h.Tickers[symbol]
	if !exists {
		return nil, ErrTickerNotFound
	}

	return ticker, nil
}

// GetOHLCV gets OHLCV data
func (h *Handler) GetOHLCV(symbol, interval string) (*OHLCV, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	intervals, exists := h.OHLCV[symbol]
	if !exists {
		return nil, ErrOHLCVNotFound
	}

	ohlcv, exists := intervals[interval]
	if !exists {
		return nil, ErrOHLCVNotFound
	}

	return ohlcv, nil
}

// GetOrderBook gets an order book
func (h *Handler) GetOrderBook(symbol string, depth int) (*OrderBook, error) {
	bids, asks, _, err := h.Engine.GetMarketData(symbol, depth)
	if err != nil {
		return nil, err
	}

	return &OrderBook{
		Symbol:    symbol,
		Bids:      bids,
		Asks:      asks,
		Timestamp: time.Now(),
	}, nil
}

// Stop stops the handler
func (h *Handler) Stop() {
	h.cancel()
}

// Errors
var (
	ErrSubscriptionNotFound = NewError("subscription not found")
	ErrTickerNotFound       = NewError("ticker not found")
	ErrOHLCVNotFound        = NewError("OHLCV not found")
)

// Error represents an error
type Error struct {
	Message string
}

// NewError creates a new error
func NewError(message string) *Error {
	return &Error{
		Message: message,
	}
}

// Error returns the error message
func (e *Error) Error() string {
	return e.Message
}

