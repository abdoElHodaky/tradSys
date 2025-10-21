package market_data

import (
	"context"
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/core/matching"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// MarketDataType represents the type of market data
type MarketDataType string

const (
	// MarketDataTypeOrderBook represents order book data
	MarketDataTypeOrderBook MarketDataType = "order_book"
	// MarketDataTypeTrade represents trade data
	MarketDataTypeTrade MarketDataType = "trade"
	// MarketDataTypeTicker represents ticker data
	MarketDataTypeTicker MarketDataType = "ticker"
	// MarketDataTypeOHLCV represents OHLCV data
	MarketDataTypeOHLCV MarketDataType = "ohlcv"
)

// Subscription represents a market data subscription
type Subscription struct {
	// ID is the unique identifier for the subscription
	ID string
	// UserID is the user ID
	UserID string
	// Symbol is the trading symbol
	Symbol string
	// Type is the type of market data
	Type MarketDataType
	// Channel is the channel for sending market data
	Channel chan interface{}
	// CreatedAt is the time the subscription was created
	CreatedAt time.Time
}

// OrderBookUpdate represents an order book update
type OrderBookUpdate struct {
	// Symbol is the trading symbol
	Symbol string
	// Bids is the bids
	Bids [][]float64
	// Asks is the asks
	Asks [][]float64
	// Timestamp is the time of the update
	Timestamp time.Time
}

// TradeUpdate represents a trade update
type TradeUpdate struct {
	// Symbol is the trading symbol
	Symbol string
	// Price is the price of the trade
	Price float64
	// Quantity is the quantity of the trade
	Quantity float64
	// Side is the side of the trade
	Side string
	// Timestamp is the time of the trade
	Timestamp time.Time
}

// TickerUpdate represents a ticker update
type TickerUpdate struct {
	// Symbol is the trading symbol
	Symbol string
	// Price is the current price
	Price float64
	// Volume is the 24-hour volume
	Volume float64
	// Change is the 24-hour price change
	Change float64
	// ChangePercent is the 24-hour price change percentage
	ChangePercent float64
	// High is the 24-hour high price
	High float64
	// Low is the 24-hour low price
	Low float64
	// Timestamp is the time of the update
	Timestamp time.Time
}

// OHLCVUpdate represents an OHLCV update
type OHLCVUpdate struct {
	// Symbol is the trading symbol
	Symbol string
	// Interval is the interval
	Interval string
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
	// Timestamp is the time of the update
	Timestamp time.Time
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
	// Mutex for thread safety
	mu sync.RWMutex
	// Logger
	logger *zap.Logger
	// Context
	ctx context.Context
	// Cancel function
	cancel context.CancelFunc
	// Ticker data
	tickers map[string]*TickerUpdate
	// OHLCV data
	ohlcv map[string]map[string]*OHLCVUpdate
}

// NewHandler creates a new market data handler
func NewHandler(engine *order_matching.Engine, logger *zap.Logger) *Handler {
	ctx, cancel := context.WithCancel(context.Background())
	
	handler := &Handler{
		Engine:              engine,
		Subscriptions:       make(map[string]*Subscription),
		SymbolSubscriptions: make(map[string]map[string]*Subscription),
		TypeSubscriptions:   make(map[MarketDataType]map[string]*Subscription),
		logger:              logger,
		ctx:                 ctx,
		cancel:              cancel,
		tickers:             make(map[string]*TickerUpdate),
		ohlcv:               make(map[string]map[string]*OHLCVUpdate),
	}
	
	// Initialize type subscriptions
	handler.TypeSubscriptions[MarketDataTypeOrderBook] = make(map[string]*Subscription)
	handler.TypeSubscriptions[MarketDataTypeTrade] = make(map[string]*Subscription)
	handler.TypeSubscriptions[MarketDataTypeTicker] = make(map[string]*Subscription)
	handler.TypeSubscriptions[MarketDataTypeOHLCV] = make(map[string]*Subscription)
	
	// Start trade processor
	go handler.processTrades()
	
	// Start order book processor
	go handler.processOrderBooks()
	
	// Start ticker processor
	go handler.processTickers()
	
	// Start OHLCV processor
	go handler.processOHLCV()
	
	return handler
}

// Subscribe subscribes to market data
func (h *Handler) Subscribe(userID, symbol string, dataType MarketDataType) (*Subscription, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	
	// Create subscription
	subscription := &Subscription{
		ID:        uuid.New().String(),
		UserID:    userID,
		Symbol:    symbol,
		Type:      dataType,
		Channel:   make(chan interface{}, 100),
		CreatedAt: time.Now(),
	}
	
	// Add to subscriptions
	h.Subscriptions[subscription.ID] = subscription
	
	// Add to symbol subscriptions
	if _, exists := h.SymbolSubscriptions[symbol]; !exists {
		h.SymbolSubscriptions[symbol] = make(map[string]*Subscription)
	}
	h.SymbolSubscriptions[symbol][subscription.ID] = subscription
	
	// Add to type subscriptions
	h.TypeSubscriptions[dataType][subscription.ID] = subscription
	
	return subscription, nil
}

// Unsubscribe unsubscribes from market data
func (h *Handler) Unsubscribe(subscriptionID string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	
	subscription, exists := h.Subscriptions[subscriptionID]
	if !exists {
		return nil
	}
	
	// Remove from subscriptions
	delete(h.Subscriptions, subscriptionID)
	
	// Remove from symbol subscriptions
	if symbolSubs, exists := h.SymbolSubscriptions[subscription.Symbol]; exists {
		delete(symbolSubs, subscriptionID)
	}
	
	// Remove from type subscriptions
	delete(h.TypeSubscriptions[subscription.Type], subscriptionID)
	
	// Close channel
	close(subscription.Channel)
	
	return nil
}

// processTrades processes trades from the order matching engine
func (h *Handler) processTrades() {
	for {
		select {
		case <-h.ctx.Done():
			return
		case trade := <-h.Engine.TradeChannel:
			// Create trade update
			update := &TradeUpdate{
				Symbol:    trade.Symbol,
				Price:     trade.Price,
				Quantity:  trade.Quantity,
				Side:      string(trade.TakerSide),
				Timestamp: trade.Timestamp,
			}
			
			// Update ticker
			h.updateTicker(trade.Symbol, trade.Price, trade.Quantity)
			
			// Update OHLCV
			h.updateOHLCV(trade.Symbol, trade.Price, trade.Quantity, trade.Timestamp)
			
			// Send to subscribers
			h.mu.RLock()
			
			// Send to symbol subscribers
			if symbolSubs, exists := h.SymbolSubscriptions[trade.Symbol]; exists {
				for _, sub := range symbolSubs {
					if sub.Type == MarketDataTypeTrade {
						select {
						case sub.Channel <- update:
						default:
							h.logger.Warn("Trade channel full, dropping update",
								zap.String("subscription_id", sub.ID),
								zap.String("symbol", trade.Symbol))
						}
					}
				}
			}
			
			h.mu.RUnlock()
		}
	}
}

// processOrderBooks processes order books
func (h *Handler) processOrderBooks() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-h.ctx.Done():
			return
		case <-ticker.C:
			h.mu.RLock()
			
			// Process each symbol
			for symbol := range h.SymbolSubscriptions {
				// Get order book
				bids, asks, _, err := h.Engine.GetMarketData(symbol, 10)
				if err != nil {
					continue
				}
				
				// Create order book update
				update := &OrderBookUpdate{
					Symbol:    symbol,
					Bids:      bids,
					Asks:      asks,
					Timestamp: time.Now(),
				}
				
				// Send to subscribers
				if symbolSubs, exists := h.SymbolSubscriptions[symbol]; exists {
					for _, sub := range symbolSubs {
						if sub.Type == MarketDataTypeOrderBook {
							select {
							case sub.Channel <- update:
							default:
								h.logger.Warn("Order book channel full, dropping update",
									zap.String("subscription_id", sub.ID),
									zap.String("symbol", symbol))
							}
						}
					}
				}
			}
			
			h.mu.RUnlock()
		}
	}
}

// processTickers processes tickers
func (h *Handler) processTickers() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-h.ctx.Done():
			return
		case <-ticker.C:
			h.mu.RLock()
			
			// Process each ticker
			for symbol, tickerData := range h.tickers {
				// Send to subscribers
				if symbolSubs, exists := h.SymbolSubscriptions[symbol]; exists {
					for _, sub := range symbolSubs {
						if sub.Type == MarketDataTypeTicker {
							select {
							case sub.Channel <- tickerData:
							default:
								h.logger.Warn("Ticker channel full, dropping update",
									zap.String("subscription_id", sub.ID),
									zap.String("symbol", symbol))
							}
						}
					}
				}
			}
			
			h.mu.RUnlock()
		}
	}
}

// processOHLCV processes OHLCV data
func (h *Handler) processOHLCV() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-h.ctx.Done():
			return
		case <-ticker.C:
			h.mu.RLock()
			
			// Process each symbol
			for symbol, intervals := range h.ohlcv {
				// Process each interval
				for interval, ohlcvData := range intervals {
					// Send to subscribers
					if symbolSubs, exists := h.SymbolSubscriptions[symbol]; exists {
						for _, sub := range symbolSubs {
							if sub.Type == MarketDataTypeOHLCV {
								select {
								case sub.Channel <- ohlcvData:
								default:
									h.logger.Warn("OHLCV channel full, dropping update",
										zap.String("subscription_id", sub.ID),
										zap.String("symbol", symbol),
										zap.String("interval", interval))
								}
							}
						}
					}
				}
			}
			
			h.mu.RUnlock()
		}
	}
}

// updateTicker updates a ticker
func (h *Handler) updateTicker(symbol string, price, quantity float64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	
	ticker, exists := h.tickers[symbol]
	if !exists {
		ticker = &TickerUpdate{
			Symbol:        symbol,
			Price:         price,
			Volume:        0,
			Change:        0,
			ChangePercent: 0,
			High:          price,
			Low:           price,
			Timestamp:     time.Now(),
		}
		h.tickers[symbol] = ticker
	}
	
	// Update ticker
	ticker.Price = price
	ticker.Volume += quantity
	
	// Update high and low
	if price > ticker.High {
		ticker.High = price
	}
	if price < ticker.Low {
		ticker.Low = price
	}
	
	// Update timestamp
	ticker.Timestamp = time.Now()
}

// updateOHLCV updates OHLCV data
func (h *Handler) updateOHLCV(symbol string, price, quantity float64, timestamp time.Time) {
	h.mu.Lock()
	defer h.mu.Unlock()
	
	// Initialize symbol if not exists
	if _, exists := h.ohlcv[symbol]; !exists {
		h.ohlcv[symbol] = make(map[string]*OHLCVUpdate)
	}
	
	// Update 1-minute OHLCV
	interval := "1m"
	ohlcv, exists := h.ohlcv[symbol][interval]
	
	// Check if we need to create a new candle
	if !exists || timestamp.Minute() != ohlcv.Timestamp.Minute() {
		ohlcv = &OHLCVUpdate{
			Symbol:    symbol,
			Interval:  interval,
			Open:      price,
			High:      price,
			Low:       price,
			Close:     price,
			Volume:    quantity,
			Timestamp: timestamp,
		}
		h.ohlcv[symbol][interval] = ohlcv
	} else {
		// Update existing candle
		ohlcv.Close = price
		ohlcv.Volume += quantity
		
		// Update high and low
		if price > ohlcv.High {
			ohlcv.High = price
		}
		if price < ohlcv.Low {
			ohlcv.Low = price
		}
	}
}

// Stop stops the handler
func (h *Handler) Stop() {
	h.cancel()
	
	// Close all subscription channels
	h.mu.Lock()
	defer h.mu.Unlock()
	
	for _, subscription := range h.Subscriptions {
		close(subscription.Channel)
	}
}

