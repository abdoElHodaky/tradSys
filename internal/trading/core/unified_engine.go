package core

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/core/matching"
	"github.com/abdoElHodaky/tradSys/internal/core/settlement"
	"github.com/abdoElHodaky/tradSys/internal/risk"
	"github.com/abdoElHodaky/tradSys/internal/trading/types"
	"go.uber.org/zap"
)

// UnifiedTradingEngine provides a unified interface for all trading components
type UnifiedTradingEngine struct {
	config              *UnifiedEngineConfig
	logger              *zap.Logger
	orderMatchingEngine *order_matching.AdvancedOrderMatchingEngine
	riskEngine          *risk.RealTimeRiskEngine
	settlementProcessor *settlement.Processor
	eventBus            *EventBus
	metrics             *UnifiedMetrics
	isRunning           int32
	stopChannel         chan struct{}
	mu                  sync.RWMutex
}

// UnifiedEngineConfig contains configuration for the unified engine
type UnifiedEngineConfig struct {
	// Order Matching Configuration
	OrderMatching *order_matching.EngineConfig `json:"order_matching"`

	// Risk Management Configuration
	RiskManagement *risk.RiskEngineConfig `json:"risk_management"`

	// Settlement Configuration
	Settlement *SettlementConfig `json:"settlement"`

	// Performance Configuration
	MaxLatency         time.Duration `json:"max_latency"` // Target: <100μs end-to-end
	EnableMetrics      bool          `json:"enable_metrics"`
	EnableEventLogging bool          `json:"enable_event_logging"`

	// Integration Configuration
	EnableRiskIntegration       bool `json:"enable_risk_integration"`
	EnableSettlementIntegration bool `json:"enable_settlement_integration"`
	EnableCircuitBreaker        bool `json:"enable_circuit_breaker"`
}

// SettlementConfig contains settlement configuration
type SettlementConfig struct {
	EnableRealTimeSettlement bool          `json:"enable_real_time_settlement"`
	SettlementDelay          time.Duration `json:"settlement_delay"`
	EnableT0Settlement       bool          `json:"enable_t0_settlement"`
	MaxSettlementBatchSize   int           `json:"max_settlement_batch_size"`
}

// UnifiedMetrics tracks unified engine performance
type UnifiedMetrics struct {
	// Overall Performance
	TotalOrders     int64         `json:"total_orders"`
	TotalTrades     int64         `json:"total_trades"`
	TotalVolume     float64       `json:"total_volume"`
	AverageLatency  time.Duration `json:"average_latency"`
	MaxLatency      time.Duration `json:"max_latency"`
	OrdersPerSecond float64       `json:"orders_per_second"`
	TradesPerSecond float64       `json:"trades_per_second"`

	// Component Performance
	MatchingLatency   time.Duration `json:"matching_latency"`
	RiskCheckLatency  time.Duration `json:"risk_check_latency"`
	SettlementLatency time.Duration `json:"settlement_latency"`

	// Error Metrics
	RejectedOrders      int64 `json:"rejected_orders"`
	FailedSettlements   int64 `json:"failed_settlements"`
	CircuitBreakerTrips int64 `json:"circuit_breaker_trips"`

	// Timestamps
	LastUpdateTime time.Time `json:"last_update_time"`
	StartTime      time.Time `json:"start_time"`
}

// EventBus handles inter-component communication
type EventBus struct {
	subscribers map[EventType][]EventHandler
	eventQueue  chan *TradingEvent
	mu          sync.RWMutex
}

// EventType defines types of trading events
type EventType string

const (
	EventOrderReceived  EventType = "order_received"
	EventOrderValidated EventType = "order_validated"
	EventOrderRejected  EventType = "order_rejected"
	EventOrderMatched   EventType = "order_matched"
	EventTradeExecuted  EventType = "trade_executed"
	EventTradeSettled   EventType = "trade_settled"
	EventRiskBreach     EventType = "risk_breach"
	EventCircuitTripped EventType = "circuit_tripped"
	EventSystemError    EventType = "system_error"
)

// EventHandler defines the interface for event handlers
type EventHandler interface {
	HandleEvent(event *TradingEvent) error
}

// TradingEvent represents a trading system event
type TradingEvent struct {
	Type           EventType              `json:"type"`
	Symbol         string                 `json:"symbol"`
	Order          *types.Order           `json:"order,omitempty"`
	Trade          *Trade                 `json:"trade,omitempty"`
	Error          error                  `json:"error,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	Timestamp      time.Time              `json:"timestamp"`
	ProcessingTime time.Duration          `json:"processing_time"`
}

// Trade represents a completed trade
type Trade struct {
	ID           string          `json:"id"`
	Symbol       string          `json:"symbol"`
	Price        float64         `json:"price"`
	Quantity     float64         `json:"quantity"`
	BuyOrderID   string          `json:"buy_order_id"`
	SellOrderID  string          `json:"sell_order_id"`
	TakerSide    types.OrderSide `json:"taker_side"`
	MakerSide    types.OrderSide `json:"maker_side"`
	Timestamp    time.Time       `json:"timestamp"`
	SettlementID string          `json:"settlement_id,omitempty"`
	Status       TradeStatus     `json:"status"`
}

// TradeStatus represents the status of a trade
type TradeStatus string

const (
	TradeStatusPending   TradeStatus = "pending"
	TradeStatusSettled   TradeStatus = "settled"
	TradeStatusFailed    TradeStatus = "failed"
	TradeStatusCancelled TradeStatus = "cancelled"
)

// OrderRequest represents a complete order request with all validations
type OrderRequest struct {
	Order          *types.Order `json:"order"`
	SkipRiskChecks bool         `json:"skip_risk_checks"`
	SkipSettlement bool         `json:"skip_settlement"`
	Priority       int          `json:"priority"`
	RequestID      string       `json:"request_id"`
	ClientID       string       `json:"client_id"`
	RequestTime    time.Time    `json:"request_time"`
}

// OrderResponse represents the response to an order request
type OrderResponse struct {
	Order          *types.Order    `json:"order"`
	Trades         []*Trade        `json:"trades"`
	RiskCheck      *risk.RiskCheck `json:"risk_check,omitempty"`
	Success        bool            `json:"success"`
	Error          error           `json:"error,omitempty"`
	ProcessingTime time.Duration   `json:"processing_time"`
	ResponseTime   time.Time       `json:"response_time"`
}

// NewUnifiedTradingEngine creates a new unified trading engine
func NewUnifiedTradingEngine(config *UnifiedEngineConfig, logger *zap.Logger) (*UnifiedTradingEngine, error) {
	// Create order matching engine
	matchingEngine := order_matching.NewAdvancedOrderMatchingEngine(
		config.OrderMatching, logger.Named("matching"))

	// Create risk engine
	riskEngine := risk.NewRealTimeRiskEngine(
		config.RiskManagement, logger.Named("risk"))

	// Create settlement processor
	settlementProcessor := settlement.NewProcessor(logger.Named("settlement"))

	// Create event bus
	eventBus := &EventBus{
		subscribers: make(map[EventType][]EventHandler),
		eventQueue:  make(chan *TradingEvent, 10000),
	}

	engine := &UnifiedTradingEngine{
		config:              config,
		logger:              logger,
		orderMatchingEngine: matchingEngine,
		riskEngine:          riskEngine,
		settlementProcessor: settlementProcessor,
		eventBus:            eventBus,
		metrics: &UnifiedMetrics{
			StartTime:      time.Now(),
			LastUpdateTime: time.Now(),
		},
		stopChannel: make(chan struct{}),
	}

	return engine, nil
}

// Start starts the unified trading engine
func (e *UnifiedTradingEngine) Start(ctx context.Context) error {
	if !atomic.CompareAndSwapInt32(&e.isRunning, 0, 1) {
		return fmt.Errorf("unified trading engine is already running")
	}

	e.logger.Info("Starting unified trading engine",
		zap.Any("config", e.config))

	// Start components
	if err := e.orderMatchingEngine.Start(); err != nil {
		return fmt.Errorf("failed to start order matching engine: %w", err)
	}

	if err := e.riskEngine.Start(ctx); err != nil {
		return fmt.Errorf("failed to start risk engine: %w", err)
	}

	// Start event processing
	go e.processEvents(ctx)

	// Start metrics collection if enabled
	if e.config.EnableMetrics {
		go e.collectMetrics(ctx)
	}

	e.logger.Info("Unified trading engine started successfully")
	return nil
}

// Stop stops the unified trading engine
func (e *UnifiedTradingEngine) Stop() error {
	if !atomic.CompareAndSwapInt32(&e.isRunning, 1, 0) {
		return fmt.Errorf("unified trading engine is not running")
	}

	e.logger.Info("Stopping unified trading engine")

	// Stop components
	if err := e.orderMatchingEngine.Stop(); err != nil {
		e.logger.Error("Failed to stop order matching engine", zap.Error(err))
	}

	if err := e.riskEngine.Stop(); err != nil {
		e.logger.Error("Failed to stop risk engine", zap.Error(err))
	}

	close(e.stopChannel)
	e.logger.Info("Unified trading engine stopped")
	return nil
}

// ProcessOrder processes a complete order request through all components
func (e *UnifiedTradingEngine) ProcessOrder(ctx context.Context, request *OrderRequest) (*OrderResponse, error) {
	startTime := time.Now()

	response := &OrderResponse{
		Order:        request.Order,
		ResponseTime: time.Now(),
	}

	defer func() {
		response.ProcessingTime = time.Since(startTime)
		e.updateMetrics(response.ProcessingTime)

		// Log if latency exceeds target
		if response.ProcessingTime > e.config.MaxLatency {
			e.logger.Warn("Order processing exceeded latency target",
				zap.Duration("processing_time", response.ProcessingTime),
				zap.Duration("target", e.config.MaxLatency),
				zap.String("order_id", request.Order.ID))
		}
	}()

	// Publish order received event
	e.publishEvent(&TradingEvent{
		Type:      EventOrderReceived,
		Symbol:    request.Order.Symbol,
		Order:     request.Order,
		Timestamp: time.Now(),
	})

	// Step 1: Risk Management Pre-Trade Checks
	if e.config.EnableRiskIntegration && !request.SkipRiskChecks {
		riskCheckStart := time.Now()
		riskCheck, err := e.riskEngine.PreTradeCheck(request.Order)
		e.metrics.RiskCheckLatency = time.Since(riskCheckStart)

		response.RiskCheck = riskCheck

		if err != nil || !riskCheck.Passed {
			response.Success = false
			response.Error = err
			atomic.AddInt64(&e.metrics.RejectedOrders, 1)

			e.publishEvent(&TradingEvent{
				Type:      EventOrderRejected,
				Symbol:    request.Order.Symbol,
				Order:     request.Order,
				Error:     err,
				Timestamp: time.Now(),
			})

			return response, err
		}

		e.publishEvent(&TradingEvent{
			Type:      EventOrderValidated,
			Symbol:    request.Order.Symbol,
			Order:     request.Order,
			Timestamp: time.Now(),
		})
	}

	// Step 2: Order Matching
	matchingStart := time.Now()
	trades, err := e.orderMatchingEngine.AddOrder(request.Order)
	e.metrics.MatchingLatency = time.Since(matchingStart)

	if err != nil {
		response.Success = false
		response.Error = err
		atomic.AddInt64(&e.metrics.RejectedOrders, 1)

		e.publishEvent(&TradingEvent{
			Type:      EventSystemError,
			Symbol:    request.Order.Symbol,
			Order:     request.Order,
			Error:     err,
			Timestamp: time.Now(),
		})

		return response, err
	}

	// Convert trades to unified format
	unifiedTrades := make([]*Trade, len(trades))
	for i, trade := range trades {
		unifiedTrades[i] = &Trade{
			ID:          trade.ID,
			Symbol:      trade.Symbol,
			Price:       trade.Price,
			Quantity:    trade.Quantity,
			BuyOrderID:  trade.BuyOrderID,
			SellOrderID: trade.SellOrderID,
			TakerSide:   trade.TakerSide,
			MakerSide:   trade.MakerSide,
			Timestamp:   trade.Timestamp,
			Status:      TradeStatusPending,
		}
	}
	response.Trades = unifiedTrades

	// Publish trade events
	for _, trade := range unifiedTrades {
		e.publishEvent(&TradingEvent{
			Type:      EventTradeExecuted,
			Symbol:    trade.Symbol,
			Trade:     trade,
			Timestamp: time.Now(),
		})
	}

	// Step 3: Post-Trade Risk Checks
	if e.config.EnableRiskIntegration && len(trades) > 0 {
		riskTrades := make([]*risk.Trade, len(trades))
		for i, trade := range trades {
			riskTrades[i] = &risk.Trade{
				ID:        trade.ID,
				Symbol:    trade.Symbol,
				Price:     trade.Price,
				Quantity:  trade.Quantity,
				TakerSide: trade.TakerSide,
				Timestamp: trade.Timestamp,
			}
		}

		if err := e.riskEngine.PostTradeCheck(request.Order, riskTrades); err != nil {
			e.logger.Error("Post-trade risk check failed", zap.Error(err))
			// Note: We don't fail the order here as the trade has already been executed
		}
	}

	// Step 4: Settlement Processing
	if e.config.EnableSettlementIntegration && !request.SkipSettlement && len(unifiedTrades) > 0 {
		settlementStart := time.Now()
		for _, trade := range unifiedTrades {
			if err := e.processTradeSettlement(trade); err != nil {
				e.logger.Error("Trade settlement failed",
					zap.String("trade_id", trade.ID),
					zap.Error(err))
				atomic.AddInt64(&e.metrics.FailedSettlements, 1)
				trade.Status = TradeStatusFailed
			} else {
				trade.Status = TradeStatusSettled
				e.publishEvent(&TradingEvent{
					Type:      EventTradeSettled,
					Symbol:    trade.Symbol,
					Trade:     trade,
					Timestamp: time.Now(),
				})
			}
		}
		e.metrics.SettlementLatency = time.Since(settlementStart)
	}

	// Update metrics
	atomic.AddInt64(&e.metrics.TotalOrders, 1)
	atomic.AddInt64(&e.metrics.TotalTrades, int64(len(unifiedTrades)))
	for _, trade := range unifiedTrades {
		e.metrics.TotalVolume += trade.Quantity
	}

	response.Success = true
	return response, nil
}

// processTradeSettlement processes settlement for a trade
func (e *UnifiedTradingEngine) processTradeSettlement(trade *Trade) error {
	// This is a simplified settlement process
	// Real implementations would be much more complex

	if e.config.Settlement.EnableT0Settlement {
		// Immediate settlement
		return e.settlementProcessor.ProcessTrade(trade.ID, trade.Symbol,
			trade.Quantity, trade.Price)
	}

	// Delayed settlement
	if e.config.Settlement.SettlementDelay > 0 {
		time.Sleep(e.config.Settlement.SettlementDelay)
	}

	return e.settlementProcessor.ProcessTrade(trade.ID, trade.Symbol,
		trade.Quantity, trade.Price)
}

// Subscribe subscribes to trading events
func (e *UnifiedTradingEngine) Subscribe(eventType EventType, handler EventHandler) {
	e.eventBus.mu.Lock()
	defer e.eventBus.mu.Unlock()

	e.eventBus.subscribers[eventType] = append(e.eventBus.subscribers[eventType], handler)
}

// publishEvent publishes a trading event
func (e *UnifiedTradingEngine) publishEvent(event *TradingEvent) {
	select {
	case e.eventBus.eventQueue <- event:
	default:
		e.logger.Warn("Event queue full, dropping event",
			zap.String("event_type", string(event.Type)),
			zap.String("symbol", event.Symbol))
	}
}

// processEvents processes trading events
func (e *UnifiedTradingEngine) processEvents(ctx context.Context) {
	for {
		select {
		case event := <-e.eventBus.eventQueue:
			e.handleEvent(event)
		case <-ctx.Done():
			return
		case <-e.stopChannel:
			return
		}
	}
}

// handleEvent handles a trading event
func (e *UnifiedTradingEngine) handleEvent(event *TradingEvent) {
	e.eventBus.mu.RLock()
	handlers := e.eventBus.subscribers[event.Type]
	e.eventBus.mu.RUnlock()

	for _, handler := range handlers {
		if err := handler.HandleEvent(event); err != nil {
			e.logger.Error("Event handler failed",
				zap.String("event_type", string(event.Type)),
				zap.Error(err))
		}
	}

	// Log event if enabled
	if e.config.EnableEventLogging {
		e.logger.Info("Trading event",
			zap.String("type", string(event.Type)),
			zap.String("symbol", event.Symbol),
			zap.Duration("processing_time", event.ProcessingTime))
	}
}

// collectMetrics collects performance metrics
func (e *UnifiedTradingEngine) collectMetrics(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	var lastOrderCount, lastTradeCount int64
	lastTime := time.Now()

	for {
		select {
		case <-ticker.C:
			now := time.Now()
			duration := now.Sub(lastTime).Seconds()

			currentOrders := atomic.LoadInt64(&e.metrics.TotalOrders)
			currentTrades := atomic.LoadInt64(&e.metrics.TotalTrades)

			e.metrics.OrdersPerSecond = float64(currentOrders-lastOrderCount) / duration
			e.metrics.TradesPerSecond = float64(currentTrades-lastTradeCount) / duration

			lastOrderCount = currentOrders
			lastTradeCount = currentTrades
			lastTime = now
			e.metrics.LastUpdateTime = now

		case <-ctx.Done():
			return
		case <-e.stopChannel:
			return
		}
	}
}

// updateMetrics updates engine performance metrics
func (e *UnifiedTradingEngine) updateMetrics(latency time.Duration) {
	if latency > e.metrics.MaxLatency {
		e.metrics.MaxLatency = latency
	}

	// Simple moving average
	e.metrics.AverageLatency = (e.metrics.AverageLatency + latency) / 2
}

// GetMetrics returns current unified metrics
func (e *UnifiedTradingEngine) GetMetrics() *UnifiedMetrics {
	return e.metrics
}

// GetOrderBook returns the order book for a symbol
func (e *UnifiedTradingEngine) GetOrderBook(symbol string) *order_matching.AdvancedOrderBook {
	return e.orderMatchingEngine.GetOrderBook(symbol)
}

// GetPosition returns the current position for a symbol
func (e *UnifiedTradingEngine) GetPosition(symbol string) *risk.Position {
	return e.riskEngine.GetPosition(symbol)
}

// IsRunning returns true if the engine is running
func (e *UnifiedTradingEngine) IsRunning() bool {
	return atomic.LoadInt32(&e.isRunning) == 1
}
