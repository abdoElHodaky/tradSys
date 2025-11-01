// 🎯 **Standard Engine Processors**
// Generated using TradSys Code Splitting Standards
//
// This file contains the order matching algorithms and processing logic
// for the Standard Order Matching Engine. These functions handle order processing,
// trade execution, and heap-based order book management.
//
// Performance Requirements: Standard latency, heap-based priority queues
// File size limit: 410 lines

package order_matching

import (
	"container/heap"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Heap interface implementation for OrderHeap

// Len returns the length of the heap
func (h OrderHeap) Len() int { return len(h.Orders) }

// Less returns whether the order at index i is less than the order at index j
func (h OrderHeap) Less(i, j int) bool {
	if h.Side == OrderSideBuy {
		// For buy orders, higher prices have higher priority
		if h.Orders[i].Price == h.Orders[j].Price {
			// If prices are equal, older orders have higher priority
			return h.Orders[i].CreatedAt.Before(h.Orders[j].CreatedAt)
		}
		return h.Orders[i].Price > h.Orders[j].Price
	}
	// For sell orders, lower prices have higher priority
	if h.Orders[i].Price == h.Orders[j].Price {
		// If prices are equal, older orders have higher priority
		return h.Orders[i].CreatedAt.Before(h.Orders[j].CreatedAt)
	}
	return h.Orders[i].Price < h.Orders[j].Price
}

// Swap swaps the orders at indices i and j
func (h OrderHeap) Swap(i, j int) {
	h.Orders[i], h.Orders[j] = h.Orders[j], h.Orders[i]
	h.Orders[i].Index = i
	h.Orders[j].Index = j
}

// Push adds an order to the heap
func (h *OrderHeap) Push(x interface{}) {
	n := len(h.Orders)
	order := x.(*Order)
	order.Index = n
	h.Orders = append(h.Orders, order)
}

// Pop removes and returns the top order from the heap
func (h *OrderHeap) Pop() interface{} {
	old := h.Orders
	n := len(old)
	order := old[n-1]
	old[n-1] = nil   // avoid memory leak
	order.Index = -1 // for safety
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

// processOrder processes an order and returns any trades that were executed
func (ob *OrderBook) processOrder(order *Order) ([]*Trade, error) {
	trades := make([]*Trade, 0)

	// Handle market orders
	if order.Type == OrderTypeMarket {
		if order.Side == OrderSideBuy {
			// Process market buy order
			for ob.Asks.Len() > 0 && order.Quantity > order.FilledQuantity {
				bestAsk := ob.Asks.Peek()
				trade := ob.matchOrders(order, bestAsk)
				trades = append(trades, trade)

				// Update order status
				if bestAsk.Status == OrderStatusFilled {
					heap.Pop(ob.Asks)
				}
			}
		} else {
			// Process market sell order
			for ob.Bids.Len() > 0 && order.Quantity > order.FilledQuantity {
				bestBid := ob.Bids.Peek()
				trade := ob.matchOrders(order, bestBid)
				trades = append(trades, trade)

				// Update order status
				if bestBid.Status == OrderStatusFilled {
					heap.Pop(ob.Bids)
				}
			}
		}

		// If market order is not fully filled, cancel the remaining quantity
		if order.Quantity > order.FilledQuantity {
			order.Status = OrderStatusPartiallyFilled
			ob.logger.Warn("Market order not fully filled",
				zap.String("order_id", order.ID),
				zap.Float64("quantity", order.Quantity),
				zap.Float64("filled_quantity", order.FilledQuantity))
		} else {
			order.Status = OrderStatusFilled
		}
	} else if order.Type == OrderTypeLimit {
		// Handle limit orders
		if order.Side == OrderSideBuy {
			// Process limit buy order
			for ob.Asks.Len() > 0 && order.Quantity > order.FilledQuantity {
				bestAsk := ob.Asks.Peek()
				// Check if the best ask price is less than or equal to the buy price
				if bestAsk.Price <= order.Price {
					trade := ob.matchOrders(order, bestAsk)
					trades = append(trades, trade)

					// Update order status
					if bestAsk.Status == OrderStatusFilled {
						heap.Pop(ob.Asks)
					}
				} else {
					break
				}
			}

			// If limit order is not fully filled, add it to the order book
			if order.Quantity > order.FilledQuantity {
				if order.FilledQuantity > 0 {
					order.Status = OrderStatusPartiallyFilled
				}
				heap.Push(ob.Bids, order)
			} else {
				order.Status = OrderStatusFilled
			}
		} else {
			// Process limit sell order
			for ob.Bids.Len() > 0 && order.Quantity > order.FilledQuantity {
				bestBid := ob.Bids.Peek()
				// Check if the best bid price is greater than or equal to the sell price
				if bestBid.Price >= order.Price {
					trade := ob.matchOrders(order, bestBid)
					trades = append(trades, trade)

					// Update order status
					if bestBid.Status == OrderStatusFilled {
						heap.Pop(ob.Bids)
					}
				} else {
					break
				}
			}

			// If limit order is not fully filled, add it to the order book
			if order.Quantity > order.FilledQuantity {
				if order.FilledQuantity > 0 {
					order.Status = OrderStatusPartiallyFilled
				}
				heap.Push(ob.Asks, order)
			} else {
				order.Status = OrderStatusFilled
			}
		}
	}

	// Update last price if trades were executed
	if len(trades) > 0 {
		ob.LastPrice = trades[len(trades)-1].Price
		// Check stop orders
		ob.checkStopOrders()
	}

	return trades, nil
}

// matchOrders matches two orders and returns a trade
func (ob *OrderBook) matchOrders(takerOrder, makerOrder *Order) *Trade {
	// Calculate trade quantity (minimum of remaining quantities)
	takerRemaining := takerOrder.Quantity - takerOrder.FilledQuantity
	makerRemaining := makerOrder.Quantity - makerOrder.FilledQuantity
	tradeQuantity := takerRemaining
	if makerRemaining < tradeQuantity {
		tradeQuantity = makerRemaining
	}

	// Use maker's price for trade execution
	tradePrice := makerOrder.Price

	// Create trade
	trade := &Trade{
		ID:           uuid.New().String(),
		Symbol:       ob.Symbol,
		Price:        tradePrice,
		Quantity:     tradeQuantity,
		TakerOrderID: takerOrder.ID,
		MakerOrderID: makerOrder.ID,
		Timestamp:    time.Now(),
		TakerSide:    takerOrder.Side,
	}

	// Set buy/sell order IDs based on sides
	if takerOrder.Side == OrderSideBuy {
		trade.BuyOrderID = takerOrder.ID
		trade.SellOrderID = makerOrder.ID
		trade.MakerSide = OrderSideSell
	} else {
		trade.BuyOrderID = makerOrder.ID
		trade.SellOrderID = takerOrder.ID
		trade.MakerSide = OrderSideBuy
	}

	// Update order quantities
	takerOrder.FilledQuantity += tradeQuantity
	makerOrder.FilledQuantity += tradeQuantity

	// Update order statuses
	if takerOrder.FilledQuantity >= takerOrder.Quantity {
		takerOrder.Status = OrderStatusFilled
	} else {
		takerOrder.Status = OrderStatusPartiallyFilled
	}

	if makerOrder.FilledQuantity >= makerOrder.Quantity {
		makerOrder.Status = OrderStatusFilled
	} else {
		makerOrder.Status = OrderStatusPartiallyFilled
	}

	// Update timestamps
	takerOrder.UpdatedAt = time.Now()
	makerOrder.UpdatedAt = time.Now()

	ob.logger.Debug("Trade executed",
		zap.String("trade_id", trade.ID),
		zap.String("symbol", trade.Symbol),
		zap.Float64("price", trade.Price),
		zap.Float64("quantity", trade.Quantity),
		zap.String("taker_order_id", trade.TakerOrderID),
		zap.String("maker_order_id", trade.MakerOrderID))

	return trade
}

// checkStopOrders checks and triggers stop orders based on the last price
func (ob *OrderBook) checkStopOrders() {
	// Check stop buy orders
	for ob.StopBids.Len() > 0 {
		stopOrder := ob.StopBids.Peek()
		if ob.LastPrice >= stopOrder.StopPrice {
			// Trigger stop order
			heap.Pop(ob.StopBids)

			// Convert to market or limit order
			if stopOrder.Type == OrderTypeStopMarket {
				stopOrder.Type = OrderTypeMarket
			} else {
				stopOrder.Type = OrderTypeLimit
			}

			// Process the triggered order
			trades, err := ob.processOrder(stopOrder)
			if err != nil {
				ob.logger.Error("Error processing triggered stop order",
					zap.String("order_id", stopOrder.ID),
					zap.Error(err))
			} else if len(trades) > 0 {
				ob.logger.Info("Stop order triggered and executed",
					zap.String("order_id", stopOrder.ID),
					zap.Float64("stop_price", stopOrder.StopPrice),
					zap.Float64("trigger_price", ob.LastPrice),
					zap.Int("trades_count", len(trades)))
			}
		} else {
			break
		}
	}

	// Check stop sell orders
	for ob.StopAsks.Len() > 0 {
		stopOrder := ob.StopAsks.Peek()
		if ob.LastPrice <= stopOrder.StopPrice {
			// Trigger stop order
			heap.Pop(ob.StopAsks)

			// Convert to market or limit order
			if stopOrder.Type == OrderTypeStopMarket {
				stopOrder.Type = OrderTypeMarket
			} else {
				stopOrder.Type = OrderTypeLimit
			}

			// Process the triggered order
			trades, err := ob.processOrder(stopOrder)
			if err != nil {
				ob.logger.Error("Error processing triggered stop order",
					zap.String("order_id", stopOrder.ID),
					zap.Error(err))
			} else if len(trades) > 0 {
				ob.logger.Info("Stop order triggered and executed",
					zap.String("order_id", stopOrder.ID),
					zap.Float64("stop_price", stopOrder.StopPrice),
					zap.Float64("trigger_price", ob.LastPrice),
					zap.Int("trades_count", len(trades)))
			}
		} else {
			break
		}
	}
}

// validateOrder validates an order before processing
func (ob *OrderBook) validateOrder(order *Order) error {
	if order == nil {
		return ErrInvalidOrder
	}

	if order.Quantity <= 0 {
		return ErrInvalidQuantity
	}

	if order.Type == OrderTypeLimit && order.Price <= 0 {
		return ErrInvalidPrice
	}

	if (order.Type == OrderTypeStopLimit || order.Type == OrderTypeStopMarket) && order.StopPrice <= 0 {
		return ErrInvalidPrice
	}

	return nil
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

// GetOrderCount returns the total number of orders in the book
func (ob *OrderBook) GetOrderCount() int {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	return len(ob.Orders)
}

// GetActiveOrderCount returns the number of active orders (bids + asks)
func (ob *OrderBook) GetActiveOrderCount() int {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	return ob.Bids.Len() + ob.Asks.Len()
}

// GetStopOrderCount returns the number of stop orders
func (ob *OrderBook) GetStopOrderCount() int {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	return ob.StopBids.Len() + ob.StopAsks.Len()
}
