package strategies

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/proto/marketdata"
	"github.com/abdoElHodaky/tradSys/proto/orders"
	"go.uber.org/zap"
)

// MarketMakingStrategy implements a basic market making strategy
type MarketMakingStrategy struct {
	*BaseStrategy
	
	// Strategy parameters
	symbol        string
	spreadBps     float64 // Spread in basis points
	quantity      float64
	maxPosition   float64
	refreshPeriod time.Duration
	
	// Strategy state
	position      float64
	activeOrders  map[string]*orders.OrderResponse
	lastUpdate    time.Time
	lastMidPrice  float64
	
	// Services
	orderService  orders.OrderServiceClient
	
	// Mutex for thread safety
	mu            sync.RWMutex
}

// NewMarketMakingStrategy creates a new market making strategy
func NewMarketMakingStrategy(
	name string,
	logger *zap.Logger,
	symbol string,
	spreadBps float64,
	quantity float64,
	maxPosition float64,
	refreshPeriod time.Duration,
	orderService orders.OrderServiceClient,
) *MarketMakingStrategy {
	return &MarketMakingStrategy{
		BaseStrategy:  NewBaseStrategy(name, logger),
		symbol:        symbol,
		spreadBps:     spreadBps,
		quantity:      quantity,
		maxPosition:   maxPosition,
		refreshPeriod: refreshPeriod,
		position:      0,
		activeOrders:  make(map[string]*orders.OrderResponse),
		lastUpdate:    time.Time{},
		lastMidPrice:  0,
		orderService:  orderService,
	}
}

// Initialize initializes the strategy
func (s *MarketMakingStrategy) Initialize(ctx context.Context) error {
	if err := s.BaseStrategy.Initialize(ctx); err != nil {
		return err
	}
	
	// Initialize strategy-specific state
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.position = 0
	s.activeOrders = make(map[string]*orders.OrderResponse)
	s.lastUpdate = time.Time{}
	s.lastMidPrice = 0
	
	s.logger.Info("Market making strategy initialized",
		zap.String("symbol", s.symbol),
		zap.Float64("spread_bps", s.spreadBps),
		zap.Float64("quantity", s.quantity),
		zap.Float64("max_position", s.maxPosition),
		zap.Duration("refresh_period", s.refreshPeriod))
	
	return nil
}

// Start starts the strategy
func (s *MarketMakingStrategy) Start(ctx context.Context) error {
	if err := s.BaseStrategy.Start(ctx); err != nil {
		return err
	}
	
	// Start strategy-specific processes
	go s.refreshQuotes(ctx)
	
	s.logger.Info("Market making strategy started", zap.String("symbol", s.symbol))
	
	return nil
}

// Stop stops the strategy
func (s *MarketMakingStrategy) Stop(ctx context.Context) error {
	if err := s.BaseStrategy.Stop(ctx); err != nil {
		return err
	}
	
	// Cancel all active orders
	if err := s.cancelAllOrders(ctx); err != nil {
		s.logger.Error("Failed to cancel all orders", zap.Error(err))
	}
	
	s.logger.Info("Market making strategy stopped", zap.String("symbol", s.symbol))
	
	return nil
}

// OnMarketData processes market data updates
func (s *MarketMakingStrategy) OnMarketData(ctx context.Context, data *marketdata.MarketDataResponse) error {
	// Check if the market data is for our symbol
	if data.Symbol != s.symbol {
		return nil
	}
	
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Update last mid price
	midPrice := (data.BidPrice + data.AskPrice) / 2
	s.lastMidPrice = midPrice
	s.lastUpdate = time.Now()
	
	// Check if we need to refresh quotes
	if time.Since(s.lastUpdate) > s.refreshPeriod {
		go s.refreshQuotes(ctx)
	}
	
	return nil
}

// OnOrderUpdate processes order updates
func (s *MarketMakingStrategy) OnOrderUpdate(ctx context.Context, order *orders.OrderResponse) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Check if this is one of our orders
	if _, exists := s.activeOrders[order.OrderId]; !exists {
		return nil
	}
	
	// Update order in our active orders map
	s.activeOrders[order.OrderId] = order
	
	// Update position if the order is filled
	if order.Status == "FILLED" || order.Status == "PARTIAL" {
		if order.Side == "BUY" {
			s.position += order.FilledQty
		} else if order.Side == "SELL" {
			s.position -= order.FilledQty
		}
		
		s.logger.Info("Order filled",
			zap.String("order_id", order.OrderId),
			zap.String("side", order.Side),
			zap.Float64("filled_qty", order.FilledQty),
			zap.Float64("position", s.position))
	}
	
	// Remove order from active orders if it's no longer active
	if order.Status == "FILLED" || order.Status == "CANCELLED" || order.Status == "REJECTED" {
		delete(s.activeOrders, order.OrderId)
	}
	
	return nil
}

// refreshQuotes refreshes the quotes
func (s *MarketMakingStrategy) refreshQuotes(ctx context.Context) {
	s.mu.Lock()
	
	// Check if the strategy is running
	if !s.running {
		s.mu.Unlock()
		return
	}
	
	// Check if we have a valid mid price
	if s.lastMidPrice <= 0 {
		s.mu.Unlock()
		return
	}
	
	// Calculate bid and ask prices
	spreadAmount := s.lastMidPrice * s.spreadBps / 10000 // Convert basis points to decimal
	bidPrice := math.Floor((s.lastMidPrice - spreadAmount/2) * 100) / 100 // Round down to 2 decimal places
	askPrice := math.Ceil((s.lastMidPrice + spreadAmount/2) * 100) / 100  // Round up to 2 decimal places
	
	// Calculate bid and ask quantities based on position
	bidQty := s.quantity
	askQty := s.quantity
	
	// Adjust quantities based on position
	if s.position > 0 {
		// If we have a long position, reduce bid quantity and increase ask quantity
		positionRatio := math.Min(1, math.Abs(s.position)/s.maxPosition)
		bidQty = s.quantity * (1 - positionRatio)
		askQty = s.quantity * (1 + positionRatio)
	} else if s.position < 0 {
		// If we have a short position, increase bid quantity and reduce ask quantity
		positionRatio := math.Min(1, math.Abs(s.position)/s.maxPosition)
		bidQty = s.quantity * (1 + positionRatio)
		askQty = s.quantity * (1 - positionRatio)
	}
	
	// Round quantities to appropriate precision
	bidQty = math.Floor(bidQty*1000) / 1000 // Round down to 3 decimal places
	askQty = math.Floor(askQty*1000) / 1000 // Round down to 3 decimal places
	
	// Cancel existing orders
	activeOrdersCopy := make(map[string]*orders.OrderResponse)
	for id, order := range s.activeOrders {
		activeOrdersCopy[id] = order
	}
	s.mu.Unlock()
	
	for _, order := range activeOrdersCopy {
		cancelRequest := &orders.CancelOrderRequest{
			OrderId: order.OrderId,
			Symbol:  s.symbol,
		}
		
		_, err := s.orderService.CancelOrder(ctx, cancelRequest)
		if err != nil {
			s.logger.Error("Failed to cancel order",
				zap.Error(err),
				zap.String("order_id", order.OrderId))
		}
	}
	
	// Place new orders
	if bidQty > 0 {
		bidRequest := &orders.CreateOrderRequest{
			ClientOrderId: fmt.Sprintf("%s-BID-%d", s.name, time.Now().UnixNano()),
			Symbol:        s.symbol,
			Side:          "BUY",
			Type:          "LIMIT",
			TimeInForce:   "GTC",
			Quantity:      bidQty,
			Price:         bidPrice,
		}
		
		bidResponse, err := s.orderService.CreateOrder(ctx, bidRequest)
		if err != nil {
			s.logger.Error("Failed to place bid order",
				zap.Error(err),
				zap.Float64("price", bidPrice),
				zap.Float64("quantity", bidQty))
		} else {
			s.mu.Lock()
			s.activeOrders[bidResponse.OrderId] = bidResponse
			s.mu.Unlock()
			
			s.logger.Info("Placed bid order",
				zap.String("order_id", bidResponse.OrderId),
				zap.Float64("price", bidPrice),
				zap.Float64("quantity", bidQty))
		}
	}
	
	if askQty > 0 {
		askRequest := &orders.CreateOrderRequest{
			ClientOrderId: fmt.Sprintf("%s-ASK-%d", s.name, time.Now().UnixNano()),
			Symbol:        s.symbol,
			Side:          "SELL",
			Type:          "LIMIT",
			TimeInForce:   "GTC",
			Quantity:      askQty,
			Price:         askPrice,
		}
		
		askResponse, err := s.orderService.CreateOrder(ctx, askRequest)
		if err != nil {
			s.logger.Error("Failed to place ask order",
				zap.Error(err),
				zap.Float64("price", askPrice),
				zap.Float64("quantity", askQty))
		} else {
			s.mu.Lock()
			s.activeOrders[askResponse.OrderId] = askResponse
			s.mu.Unlock()
			
			s.logger.Info("Placed ask order",
				zap.String("order_id", askResponse.OrderId),
				zap.Float64("price", askPrice),
				zap.Float64("quantity", askQty))
		}
	}
	
	s.mu.Lock()
	s.lastUpdate = time.Now()
	s.mu.Unlock()
}

// cancelAllOrders cancels all active orders
func (s *MarketMakingStrategy) cancelAllOrders(ctx context.Context) error {
	s.mu.Lock()
	activeOrdersCopy := make(map[string]*orders.OrderResponse)
	for id, order := range s.activeOrders {
		activeOrdersCopy[id] = order
	}
	s.mu.Unlock()
	
	for _, order := range activeOrdersCopy {
		cancelRequest := &orders.CancelOrderRequest{
			OrderId: order.OrderId,
			Symbol:  s.symbol,
		}
		
		_, err := s.orderService.CancelOrder(ctx, cancelRequest)
		if err != nil {
			s.logger.Error("Failed to cancel order",
				zap.Error(err),
				zap.String("order_id", order.OrderId))
		} else {
			s.mu.Lock()
			delete(s.activeOrders, order.OrderId)
			s.mu.Unlock()
			
			s.logger.Info("Cancelled order", zap.String("order_id", order.OrderId))
		}
	}
	
	return nil
}

// GetParameters returns the strategy parameters
func (s *MarketMakingStrategy) GetParameters() map[string]interface{} {
	params := s.BaseStrategy.GetParameters()
	
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	params["symbol"] = s.symbol
	params["spread_bps"] = s.spreadBps
	params["quantity"] = s.quantity
	params["max_position"] = s.maxPosition
	params["refresh_period"] = s.refreshPeriod.String()
	params["position"] = s.position
	params["active_orders"] = len(s.activeOrders)
	params["last_mid_price"] = s.lastMidPrice
	
	return params
}

// SetParameters sets the strategy parameters
func (s *MarketMakingStrategy) SetParameters(params map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Update parameters
	if symbol, ok := params["symbol"].(string); ok {
		s.symbol = symbol
	}
	
	if spreadBps, ok := params["spread_bps"].(float64); ok {
		s.spreadBps = spreadBps
	}
	
	if quantity, ok := params["quantity"].(float64); ok {
		s.quantity = quantity
	}
	
	if maxPosition, ok := params["max_position"].(float64); ok {
		s.maxPosition = maxPosition
	}
	
	if refreshPeriodStr, ok := params["refresh_period"].(string); ok {
		if refreshPeriod, err := time.ParseDuration(refreshPeriodStr); err == nil {
			s.refreshPeriod = refreshPeriod
		}
	}
	
	return nil
}
