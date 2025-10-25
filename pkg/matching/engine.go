package matching

import (
	"container/heap"
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/trading/types"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Use types from the shared types package
type OrderType = types.OrderType
type OrderSide = types.OrderSide
type OrderStatus = types.OrderStatus
type Order = types.Order

// Constants from types package
const (
	OrderTypeLimit             = types.OrderTypeLimit
	OrderTypeMarket            = types.OrderTypeMarket
	OrderTypeStop              = types.OrderTypeStop
	OrderTypeStopLimit         = types.OrderTypeStopLimit
	OrderTypeStopMarket        = types.OrderTypeStopMarket
	OrderSideBuy               = types.OrderSideBuy
	OrderSideSell              = types.OrderSideSell
	OrderStatusNew             = types.OrderStatusNew
	OrderStatusPartiallyFilled = types.OrderStatusPartiallyFilled
	OrderStatusFilled          = types.OrderStatusFilled
	OrderStatusCanceled        = types.OrderStatusCanceled
	OrderStatusCancelled       = types.OrderStatusCancelled
	OrderStatusRejected        = types.OrderStatusRejected
	OrderStatusExpired         = types.OrderStatusExpired
)

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
	// BuyOrderID is the buy order ID
	BuyOrderID string
	// SellOrderID is the sell order ID
	SellOrderID string
	// Timestamp is the time the trade was executed
	Timestamp time.Time
	// TakerSide is the side of the taker
	TakerSide OrderSide
	// MakerSide is the side of the maker
	MakerSide OrderSide
	// TakerFee is the fee for the taker
	TakerFee float64
	// MakerFee is the fee for the maker
	MakerFee float64
}

// OrderBook represents an order book for a symbol
type OrderBook struct {
	// Symbol is the trading symbol
	Symbol string
	// Bids is the buy orders
	Bids *OrderHeap
	// Asks is the sell orders
	Asks *OrderHeap
	// Orders is a map of order ID to order
	Orders map[string]*Order
	// StopBids is the stop buy orders
	StopBids *OrderHeap
	// StopAsks is the stop sell orders
	StopAsks *OrderHeap
	// LastPrice is the last traded price
	LastPrice float64
	// Mutex for thread safety
	mu sync.RWMutex
	// Logger
	logger *zap.Logger
}

// OrderHeap is a heap of orders
type OrderHeap struct {
	// Orders is the slice of orders
	Orders []*Order
	// Side is the side of the orders
	Side OrderSide
}

// Len returns the length of the heap
func (h OrderHeap) Len() int { return len(h.Orders) }

// Less returns whether the order at index i is less than the order at index j
func (h OrderHeap) Less(i, j int) bool {
	if h.Side == OrderSideBuy {
		// For buy orders, higher prices have higher priority
		if h.Orders[i].Price == h.Orders[j].Price {
			// If prices are equal, earlier orders have higher priority
			return h.Orders[i].CreatedAt.Before(h.Orders[j].CreatedAt)
		}
		return h.Orders[i].Price > h.Orders[j].Price
	} else {
		// For sell orders, lower prices have higher priority
		if h.Orders[i].Price == h.Orders[j].Price {
			// If prices are equal, earlier orders have higher priority
			return h.Orders[i].CreatedAt.Before(h.Orders[j].CreatedAt)
		}
		return h.Orders[i].Price < h.Orders[j].Price
	}
}

// Swap swaps the orders at indices i and j
func (h OrderHeap) Swap(i, j int) {
	h.Orders[i], h.Orders[j] = h.Orders[j], h.Orders[i]
}

// Push adds an order to the heap
func (h *OrderHeap) Push(x interface{}) {
	h.Orders = append(h.Orders, x.(*Order))
}

// Pop removes and returns the top order from the heap
func (h *OrderHeap) Pop() interface{} {
	old := h.Orders
	n := len(old)
	order := old[n-1]
	h.Orders = old[0 : n-1]
	return order
}

// Peek returns the top order from the heap without removing it
func (h *OrderHeap) Peek() *Order {
	if len(h.Orders) == 0 {
		return nil
	}
	return h.Orders[0]
}

// Remove removes an order from the heap
func (h *OrderHeap) Remove(orderID string) *Order {
	for i, order := range h.Orders {
		if order.ID == orderID {
			removedOrder := heap.Remove(h, i).(*Order)
			return removedOrder
		}
	}
	return nil
}

// NewOrderBook creates a new order book
func NewOrderBook(symbol string, logger *zap.Logger) *OrderBook {
	return &OrderBook{
		Symbol: symbol,
		Bids: &OrderHeap{
			Orders: make([]*Order, 0),
			Side:   OrderSideBuy,
		},
		Asks: &OrderHeap{
			Orders: make([]*Order, 0),
			Side:   OrderSideSell,
		},
		Orders: make(map[string]*Order),
		StopBids: &OrderHeap{
			Orders: make([]*Order, 0),
			Side:   OrderSideBuy,
		},
		StopAsks: &OrderHeap{
			Orders: make([]*Order, 0),
			Side:   OrderSideSell,
		},
		logger: logger,
	}
}

// AddOrder adds an order to the order book
func (ob *OrderBook) AddOrder(order *Order) []*Trade {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	ob.logger.Debug("Adding order to order book",
		zap.String("order_id", order.ID),
		zap.String("symbol", order.Symbol),
		zap.String("side", string(order.Side)),
		zap.String("type", string(order.Type)),
		zap.Float64("price", order.Price),
		zap.Float64("quantity", order.Quantity))

	// Store the order
	ob.Orders[order.ID] = order

	var trades []*Trade

	switch order.Type {
	case OrderTypeMarket:
		trades = ob.processMarketOrder(order)
	case OrderTypeLimit:
		trades = ob.processLimitOrder(order)
	case OrderTypeStop, OrderTypeStopLimit, OrderTypeStopMarket:
		trades = ob.processStopOrder(order)
	}

	return trades
}

// processMarketOrder processes a market order
func (ob *OrderBook) processMarketOrder(order *Order) []*Trade {
	var trades []*Trade
	remainingQuantity := order.Quantity

	if order.Side == OrderSideBuy {
		// Match against asks (sell orders)
		for ob.Asks.Len() > 0 && remainingQuantity > 0 {
			bestAsk := ob.Asks.Peek()
			if bestAsk == nil {
				break
			}

			trade := ob.executeTrade(order, bestAsk, &remainingQuantity)
			if trade != nil {
				trades = append(trades, trade)
			}

			if bestAsk.RemainingQuantity() <= 0 {
				heap.Pop(ob.Asks)
				bestAsk.Status = OrderStatusFilled
			}
		}
	} else {
		// Match against bids (buy orders)
		for ob.Bids.Len() > 0 && remainingQuantity > 0 {
			bestBid := ob.Bids.Peek()
			if bestBid == nil {
				break
			}

			trade := ob.executeTrade(order, bestBid, &remainingQuantity)
			if trade != nil {
				trades = append(trades, trade)
			}

			if bestBid.RemainingQuantity() <= 0 {
				heap.Pop(ob.Bids)
				bestBid.Status = OrderStatusFilled
			}
		}
	}

	// Update order status
	if remainingQuantity <= 0 {
		order.Status = OrderStatusFilled
	} else if remainingQuantity < order.Quantity {
		order.Status = OrderStatusPartiallyFilled
	} else {
		order.Status = OrderStatusRejected
	}

	return trades
}

// processLimitOrder processes a limit order
func (ob *OrderBook) processLimitOrder(order *Order) []*Trade {
	var trades []*Trade
	remainingQuantity := order.Quantity

	if order.Side == OrderSideBuy {
		// Try to match against asks
		for ob.Asks.Len() > 0 && remainingQuantity > 0 {
			bestAsk := ob.Asks.Peek()
			if bestAsk == nil || bestAsk.Price > order.Price {
				break
			}

			trade := ob.executeTrade(order, bestAsk, &remainingQuantity)
			if trade != nil {
				trades = append(trades, trade)
			}

			if bestAsk.RemainingQuantity() <= 0 {
				heap.Pop(ob.Asks)
				bestAsk.Status = OrderStatusFilled
			}
		}

		// If there's remaining quantity, add to bids
		if remainingQuantity > 0 {
			order.Quantity = remainingQuantity
			heap.Push(ob.Bids, order)
			order.Status = OrderStatusNew
		}
	} else {
		// Try to match against bids
		for ob.Bids.Len() > 0 && remainingQuantity > 0 {
			bestBid := ob.Bids.Peek()
			if bestBid == nil || bestBid.Price < order.Price {
				break
			}

			trade := ob.executeTrade(order, bestBid, &remainingQuantity)
			if trade != nil {
				trades = append(trades, trade)
			}

			if bestBid.RemainingQuantity() <= 0 {
				heap.Pop(ob.Bids)
				bestBid.Status = OrderStatusFilled
			}
		}

		// If there's remaining quantity, add to asks
		if remainingQuantity > 0 {
			order.Quantity = remainingQuantity
			heap.Push(ob.Asks, order)
			order.Status = OrderStatusNew
		}
	}

	// Update order status
	if remainingQuantity <= 0 {
		order.Status = OrderStatusFilled
	} else if remainingQuantity < order.Quantity {
		order.Status = OrderStatusPartiallyFilled
	}

	return trades
}

// processStopOrder processes a stop order
func (ob *OrderBook) processStopOrder(order *Order) []*Trade {
	// For now, just add to the appropriate stop order heap
	// In a full implementation, we would check if the stop price is triggered
	if order.Side == OrderSideBuy {
		heap.Push(ob.StopBids, order)
	} else {
		heap.Push(ob.StopAsks, order)
	}

	order.Status = OrderStatusNew
	return nil
}

// executeTrade executes a trade between two orders
func (ob *OrderBook) executeTrade(takerOrder, makerOrder *Order, remainingQuantity *float64) *Trade {
	tradeQuantity := min(*remainingQuantity, makerOrder.RemainingQuantity())
	if tradeQuantity <= 0 {
		return nil
	}

	trade := &Trade{
		ID:         uuid.New().String(),
		Symbol:     ob.Symbol,
		Price:      makerOrder.Price,
		Quantity:   tradeQuantity,
		Timestamp:  time.Now(),
		TakerSide:  takerOrder.Side,
		MakerSide:  makerOrder.Side,
		TakerFee:   0.001 * tradeQuantity * makerOrder.Price, // 0.1% fee
		MakerFee:   0.0005 * tradeQuantity * makerOrder.Price, // 0.05% fee
	}

	if takerOrder.Side == OrderSideBuy {
		trade.BuyOrderID = takerOrder.ID
		trade.SellOrderID = makerOrder.ID
	} else {
		trade.BuyOrderID = makerOrder.ID
		trade.SellOrderID = takerOrder.ID
	}

	// Update order quantities
	takerOrder.FilledQuantity += tradeQuantity
	makerOrder.FilledQuantity += tradeQuantity
	*remainingQuantity -= tradeQuantity

	// Update last price
	ob.LastPrice = makerOrder.Price

	ob.logger.Debug("Trade executed",
		zap.String("trade_id", trade.ID),
		zap.String("symbol", trade.Symbol),
		zap.Float64("price", trade.Price),
		zap.Float64("quantity", trade.Quantity),
		zap.String("taker_order_id", takerOrder.ID),
		zap.String("maker_order_id", makerOrder.ID))

	return trade
}

// CancelOrder cancels an order
func (ob *OrderBook) CancelOrder(orderID string) bool {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	order, exists := ob.Orders[orderID]
	if !exists {
		return false
	}

	// Remove from the appropriate heap
	var removed *Order
	switch order.Side {
	case OrderSideBuy:
		if order.Type == OrderTypeStop || order.Type == OrderTypeStopLimit || order.Type == OrderTypeStopMarket {
			removed = ob.StopBids.Remove(orderID)
		} else {
			removed = ob.Bids.Remove(orderID)
		}
	case OrderSideSell:
		if order.Type == OrderTypeStop || order.Type == OrderTypeStopLimit || order.Type == OrderTypeStopMarket {
			removed = ob.StopAsks.Remove(orderID)
		} else {
			removed = ob.Asks.Remove(orderID)
		}
	}

	if removed != nil {
		removed.Status = OrderStatusCancelled
		delete(ob.Orders, orderID)
		
		ob.logger.Debug("Order cancelled",
			zap.String("order_id", orderID),
			zap.String("symbol", ob.Symbol))
		
		return true
	}

	return false
}

// GetBestBid returns the best bid price
func (ob *OrderBook) GetBestBid() float64 {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	if ob.Bids.Len() > 0 {
		return ob.Bids.Peek().Price
	}
	return 0
}

// GetBestAsk returns the best ask price
func (ob *OrderBook) GetBestAsk() float64 {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	if ob.Asks.Len() > 0 {
		return ob.Asks.Peek().Price
	}
	return 0
}

// GetSpread returns the bid-ask spread
func (ob *OrderBook) GetSpread() float64 {
	bestBid := ob.GetBestBid()
	bestAsk := ob.GetBestAsk()

	if bestBid > 0 && bestAsk > 0 {
		return bestAsk - bestBid
	}
	return 0
}

// GetDepth returns the market depth
func (ob *OrderBook) GetDepth(levels int) ([]PriceLevel, []PriceLevel) {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	bids := make([]PriceLevel, 0, levels)
	asks := make([]PriceLevel, 0, levels)

	// Get bid levels
	bidMap := make(map[float64]float64)
	for _, order := range ob.Bids.Orders {
		bidMap[order.Price] += order.RemainingQuantity()
	}

	// Convert to sorted slice
	for price, quantity := range bidMap {
		bids = append(bids, PriceLevel{Price: price, Quantity: quantity})
		if len(bids) >= levels {
			break
		}
	}

	// Get ask levels
	askMap := make(map[float64]float64)
	for _, order := range ob.Asks.Orders {
		askMap[order.Price] += order.RemainingQuantity()
	}

	// Convert to sorted slice
	for price, quantity := range askMap {
		asks = append(asks, PriceLevel{Price: price, Quantity: quantity})
		if len(asks) >= levels {
			break
		}
	}

	return bids, asks
}

// PriceLevel represents a price level in the order book
type PriceLevel struct {
	Price    float64 `json:"price"`
	Quantity float64 `json:"quantity"`
}

// MatchingEngine represents the order matching engine
type MatchingEngine struct {
	// OrderBooks is a map of symbol to order book
	OrderBooks map[string]*OrderBook
	// TradeChannel is the channel for trades
	TradeChannel chan *Trade
	// Logger
	logger *zap.Logger
	// Mutex for thread safety
	mu sync.RWMutex
}

// NewMatchingEngine creates a new matching engine
func NewMatchingEngine(logger *zap.Logger) *MatchingEngine {
	return &MatchingEngine{
		OrderBooks:   make(map[string]*OrderBook),
		TradeChannel: make(chan *Trade, 1000),
		logger:       logger,
	}
}

// AddOrder adds an order to the matching engine
func (me *MatchingEngine) AddOrder(order *Order) []*Trade {
	me.mu.Lock()
	orderBook, exists := me.OrderBooks[order.Symbol]
	if !exists {
		orderBook = NewOrderBook(order.Symbol, me.logger)
		me.OrderBooks[order.Symbol] = orderBook
	}
	me.mu.Unlock()

	trades := orderBook.AddOrder(order)

	// Send trades to channel
	for _, trade := range trades {
		select {
		case me.TradeChannel <- trade:
		default:
			me.logger.Warn("Trade channel full, dropping trade",
				zap.String("trade_id", trade.ID))
		}
	}

	return trades
}

// CancelOrder cancels an order
func (me *MatchingEngine) CancelOrder(symbol, orderID string) bool {
	me.mu.RLock()
	orderBook, exists := me.OrderBooks[symbol]
	me.mu.RUnlock()

	if !exists {
		return false
	}

	return orderBook.CancelOrder(orderID)
}

// GetOrderBook returns the order book for a symbol
func (me *MatchingEngine) GetOrderBook(symbol string) *OrderBook {
	me.mu.RLock()
	defer me.mu.RUnlock()

	return me.OrderBooks[symbol]
}

// Helper function
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
