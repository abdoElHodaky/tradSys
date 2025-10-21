package strategies

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/db/models"
	"github.com/abdoElHodaky/tradSys/proto/marketdata"
	"github.com/abdoElHodaky/tradSys/proto/orders"
	"go.uber.org/zap"
)

// BacktestEngine is responsible for backtesting strategies
type BacktestEngine struct {
	logger     *zap.Logger
	strategies map[string]Strategy
	marketData []*marketdata.MarketDataResponse
	trades     []models.Trade
	mu         sync.RWMutex
}

// NewBacktestEngine creates a new backtest engine
func NewBacktestEngine(logger *zap.Logger) *BacktestEngine {
	return &BacktestEngine{
		logger:     logger,
		strategies: make(map[string]Strategy),
		marketData: make([]*marketdata.MarketDataResponse, 0),
		trades:     make([]models.Trade, 0),
	}
}

// RegisterStrategy registers a strategy for backtesting
func (e *BacktestEngine) RegisterStrategy(strategy Strategy) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	name := strategy.GetName()
	if _, exists := e.strategies[name]; exists {
		return ErrStrategyAlreadyRegistered
	}
	
	e.strategies[name] = strategy
	
	e.logger.Info("Strategy registered for backtesting", zap.String("name", name))
	
	return nil
}

// LoadMarketData loads market data for backtesting
func (e *BacktestEngine) LoadMarketData(data []*marketdata.MarketDataResponse) {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	e.marketData = data
	
	e.logger.Info("Market data loaded for backtesting",
		zap.Int("data_points", len(data)),
		zap.Time("start_time", time.Unix(0, data[0].Timestamp*int64(time.Millisecond))),
		zap.Time("end_time", time.Unix(0, data[len(data)-1].Timestamp*int64(time.Millisecond))))
}

// RunBacktest runs a backtest for a strategy
func (e *BacktestEngine) RunBacktest(ctx context.Context, strategyName string, initialCapital float64) (*BacktestResult, error) {
	e.mu.Lock()
	strategy, exists := e.strategies[strategyName]
	if !exists {
		e.mu.Unlock()
		return nil, ErrStrategyNotFound
	}
	
	marketData := make([]*marketdata.MarketDataResponse, len(e.marketData))
	copy(marketData, e.marketData)
	e.mu.Unlock()
	
	if len(marketData) == 0 {
		return nil, fmt.Errorf("no market data available for backtesting")
	}
	
	// Initialize backtest state
	capital := initialCapital
	positions := make(map[string]float64)
	trades := make([]models.Trade, 0)
	metrics := make(map[string]float64)
	
	// Initialize strategy
	if err := strategy.Initialize(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize strategy: %w", err)
	}
	
	// Start strategy
	if err := strategy.Start(ctx); err != nil {
		return nil, fmt.Errorf("failed to start strategy: %w", err)
	}
	
	// Create a mock order service for backtesting
	mockOrderService := newMockOrderService(e.logger, &capital, positions, &trades)
	
	// Process market data
	startTime := time.Unix(0, marketData[0].Timestamp*int64(time.Millisecond))
	endTime := time.Unix(0, marketData[len(marketData)-1].Timestamp*int64(time.Millisecond))
	
	e.logger.Info("Starting backtest",
		zap.String("strategy", strategyName),
		zap.Time("start_time", startTime),
		zap.Time("end_time", endTime),
		zap.Float64("initial_capital", initialCapital))
	
	// Process each market data point
	for _, data := range marketData {
		// Update context with timestamp
		timestamp := time.Unix(0, data.Timestamp*int64(time.Millisecond))
		ctx = context.WithValue(ctx, "timestamp", timestamp)
		
		// Process market data
		if err := strategy.OnMarketData(ctx, data); err != nil {
			e.logger.Error("Failed to process market data",
				zap.Error(err),
				zap.String("strategy", strategyName),
				zap.Time("timestamp", timestamp))
		}
		
		// Process any orders that were created
		mockOrderService.ProcessOrders(ctx, data)
	}
	
	// Stop strategy
	if err := strategy.Stop(ctx); err != nil {
		e.logger.Error("Failed to stop strategy",
			zap.Error(err),
			zap.String("strategy", strategyName))
	}
	
	// Calculate final positions value
	positionsValue := 0.0
	symbols := make([]string, 0)
	for symbol, quantity := range positions {
		// Find the last price for this symbol
		lastPrice := 0.0
		for i := len(marketData) - 1; i >= 0; i-- {
			if marketData[i].Symbol == symbol {
				lastPrice = marketData[i].LastPrice
				break
			}
		}
		
		positionsValue += quantity * lastPrice
		symbols = append(symbols, symbol)
	}
	
	// Calculate final capital
	finalCapital := capital + positionsValue
	
	// Calculate PnL
	pnl := finalCapital - initialCapital
	
	// Calculate metrics
	metrics["pnl"] = pnl
	metrics["return"] = pnl / initialCapital * 100
	metrics["trade_count"] = float64(len(trades))
	
	if len(trades) > 0 {
		winCount := 0
		lossCount := 0
		totalProfit := 0.0
		totalLoss := 0.0
		
		for _, trade := range trades {
			if trade.Price > 0 {
				winCount++
				totalProfit += trade.Price
			} else {
				lossCount++
				totalLoss += trade.Price
			}
		}
		
		metrics["win_rate"] = float64(winCount) / float64(len(trades)) * 100
		metrics["profit_factor"] = totalProfit / math.Abs(totalLoss)
	}
	
	// Create backtest result
	result := &BacktestResult{
		Strategy:       strategyName,
		StartTime:      startTime,
		EndTime:        endTime,
		Symbols:        symbols,
		InitialCapital: initialCapital,
		FinalCapital:   finalCapital,
		PnL:            pnl,
		Trades:         trades,
		Metrics:        metrics,
	}
	
	e.logger.Info("Backtest completed",
		zap.String("strategy", strategyName),
		zap.Float64("initial_capital", initialCapital),
		zap.Float64("final_capital", finalCapital),
		zap.Float64("pnl", pnl),
		zap.Int("trade_count", len(trades)))
	
	return result, nil
}

// mockOrderService is a mock implementation of the OrderServiceClient for backtesting
type mockOrderService struct {
	logger     *zap.Logger
	capital    *float64
	positions  map[string]float64
	trades     *[]models.Trade
	orders     map[string]*orders.OrderResponse
	nextOrderID int64
	mu         sync.RWMutex
}

// newMockOrderService creates a new mock order service
func newMockOrderService(logger *zap.Logger, capital *float64, positions map[string]float64, trades *[]models.Trade) *mockOrderService {
	return &mockOrderService{
		logger:     logger,
		capital:    capital,
		positions:  positions,
		trades:     trades,
		orders:     make(map[string]*orders.OrderResponse),
		nextOrderID: 1,
	}
}

// CreateOrder creates a new order
func (s *mockOrderService) CreateOrder(ctx context.Context, req *orders.CreateOrderRequest) (*orders.OrderResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Generate order ID
	orderID := fmt.Sprintf("BACKTEST-ORDER-%d", s.nextOrderID)
	s.nextOrderID++
	
	// Create order response
	order := &orders.OrderResponse{
		OrderId:       orderID,
		ClientOrderId: req.ClientOrderId,
		Symbol:        req.Symbol,
		Side:          req.Side,
		Type:          req.Type,
		TimeInForce:   req.TimeInForce,
		Quantity:      req.Quantity,
		Price:         req.Price,
		Status:        "NEW",
		FilledQty:     0,
		AvgPrice:      0,
		Timestamp:     time.Now().UnixMilli(),
	}
	
	// Store order
	s.orders[orderID] = order
	
	return order, nil
}

// CancelOrder cancels an order
func (s *mockOrderService) CancelOrder(ctx context.Context, req *orders.CancelOrderRequest) (*orders.OrderResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Find order
	order, exists := s.orders[req.OrderId]
	if !exists {
		return nil, fmt.Errorf("order not found: %s", req.OrderId)
	}
	
	// Cancel order
	order.Status = "CANCELLED"
	
	return order, nil
}

// ProcessOrders processes orders based on market data
func (s *mockOrderService) ProcessOrders(ctx context.Context, data *marketdata.MarketDataResponse) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Process each order
	for orderID, order := range s.orders {
		// Skip orders that are not for this symbol
		if order.Symbol != data.Symbol {
			continue
		}
		
		// Skip orders that are already filled or cancelled
		if order.Status == "FILLED" || order.Status == "CANCELLED" || order.Status == "REJECTED" {
			continue
		}
		
		// Process order based on type
		switch order.Type {
		case "MARKET":
			// Market orders are filled immediately at the current price
			s.fillOrder(orderID, order, data.LastPrice)
			
		case "LIMIT":
			// Limit orders are filled if the price is favorable
			if order.Side == "BUY" && data.LastPrice <= order.Price {
				s.fillOrder(orderID, order, data.LastPrice)
			} else if order.Side == "SELL" && data.LastPrice >= order.Price {
				s.fillOrder(orderID, order, data.LastPrice)
			}
		}
	}
}

// fillOrder fills an order
func (s *mockOrderService) fillOrder(orderID string, order *orders.OrderResponse, price float64) {
	// Update order
	order.Status = "FILLED"
	order.FilledQty = order.Quantity
	order.AvgPrice = price
	
	// Update capital and positions
	cost := order.Quantity * price
	if order.Side == "BUY" {
		*s.capital -= cost
		s.positions[order.Symbol] += order.Quantity
	} else if order.Side == "SELL" {
		*s.capital += cost
		s.positions[order.Symbol] -= order.Quantity
	}
	
	// Create trade
	trade := models.Trade{
		TradeID:   fmt.Sprintf("BACKTEST-TRADE-%s", orderID),
		OrderID:   orderID,
		Symbol:    order.Symbol,
		Side:      models.OrderSide(order.Side),
		Quantity:  order.Quantity,
		Price:     price,
		Timestamp: time.Unix(0, order.Timestamp*int64(time.Millisecond)),
	}
	
	// Add trade to trades
	*s.trades = append(*s.trades, trade)
	
	s.logger.Debug("Order filled",
		zap.String("order_id", orderID),
		zap.String("symbol", order.Symbol),
		zap.String("side", order.Side),
		zap.Float64("quantity", order.Quantity),
		zap.Float64("price", price),
		zap.Float64("cost", cost))
}
