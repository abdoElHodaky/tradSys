package risk

import (
	"context"
	"fmt"
	"math"
	"sync/atomic"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/common/pool"
	"github.com/abdoElHodaky/tradSys/internal/trading/types"
	"go.uber.org/zap"
)

// NewRealTimeRiskEngine creates a new real-time risk engine
func NewRealTimeRiskEngine(config *RiskEngineConfig, logger *zap.Logger) *RealTimeRiskEngine {
	engine := &RealTimeRiskEngine{
		config:          config,
		logger:          logger,
		metrics:         &RiskMetrics{LastUpdateTime: time.Now()},
		stopChannel:     make(chan struct{}),
		eventChannel:    make(chan *RiskEvent, 10000),
		positionManager: &PositionManager{},
		limitManager: &LimitManager{
			positionLimits: make(map[string]float64),
			orderLimits:    make(map[string]float64),
			dailyLossLimit: config.MaxDailyLoss,
		},
		varCalculator: &VaRCalculator{
			enabled:           config.EnableVaRCalculation,
			confidenceLevel:   config.VaRConfidenceLevel,
			timeHorizon:       config.VaRTimeHorizon,
			historicalReturns: make(map[string][]float64),
			correlationMatrix: make(map[string]map[string]float64),
		},
		circuitBreaker: &CircuitBreaker{
			enabled:   config.EnableCircuitBreaker,
			threshold: 10,
			timeout:   time.Minute * 5,
		},
	}

	// Initialize object pools for performance
	engine.eventPool = pool.NewObjectPool(func() interface{} {
		return &RiskEvent{}
	}, 1000)

	engine.checkPool = pool.NewObjectPool(func() interface{} {
		return &RiskCheck{}
	}, 1000)

	return engine
}

// Start starts the risk engine
func (e *RealTimeRiskEngine) Start(ctx context.Context) error {
	if !atomic.CompareAndSwapInt32(&e.isRunning, 0, 1) {
		return fmt.Errorf("risk engine is already running")
	}

	e.logger.Info("Starting real-time risk engine",
		zap.Any("config", e.config))

	// Start event processing goroutine
	go e.processEvents(ctx)

	// Start VaR calculation goroutine if enabled
	if e.config.EnableVaRCalculation {
		go e.calculateVaRPeriodically(ctx)
	}

	// Start circuit breaker monitoring if enabled
	if e.config.EnableCircuitBreaker {
		go e.monitorCircuitBreaker(ctx)
	}

	return nil
}

// Stop stops the risk engine
func (e *RealTimeRiskEngine) Stop() error {
	if !atomic.CompareAndSwapInt32(&e.isRunning, 1, 0) {
		return fmt.Errorf("risk engine is not running")
	}

	close(e.stopChannel)
	e.logger.Info("Real-time risk engine stopped")
	return nil
}

// PreTradeCheck performs pre-trade risk checks with HFT latency requirements
func (e *RealTimeRiskEngine) PreTradeCheck(order *types.Order) (*RiskCheck, error) {
	startTime := time.Now()
	defer func() {
		latency := time.Since(startTime)
		e.updateMetrics(latency)

		// Log if latency exceeds target
		if latency > e.config.MaxLatency {
			e.logger.Warn("Pre-trade check exceeded latency target",
				zap.Duration("latency", latency),
				zap.Duration("target", e.config.MaxLatency),
				zap.String("order_id", order.ID))
		}
	}()

	if !e.config.EnablePreTradeChecks {
		return &RiskCheck{
			CheckType: "pre_trade",
			Passed:    true,
			Message:   "Pre-trade checks disabled",
			Latency:   time.Since(startTime),
			Timestamp: time.Now(),
		}, nil
	}

	// Check circuit breaker first (fastest check)
	if e.config.EnableCircuitBreaker && e.isCircuitBreakerOpen() {
		return &RiskCheck{
			CheckType: "circuit_breaker",
			Passed:    false,
			Message:   "Circuit breaker is open",
			Latency:   time.Since(startTime),
			Timestamp: time.Now(),
		}, fmt.Errorf("circuit breaker is open")
	}

	// Check order size limits
	if err := e.checkOrderSizeLimits(order); err != nil {
		return &RiskCheck{
			CheckType:    "order_size",
			Passed:       false,
			CurrentValue: order.Quantity,
			LimitValue:   e.getOrderSizeLimit(order.Symbol),
			Message:      err.Error(),
			Latency:      time.Since(startTime),
			Timestamp:    time.Now(),
		}, err
	}

	// Check position limits
	if err := e.checkPositionLimits(order); err != nil {
		return &RiskCheck{
			CheckType:    "position_limit",
			Passed:       false,
			CurrentValue: e.getProjectedPosition(order),
			LimitValue:   e.getPositionLimit(order.Symbol),
			Message:      err.Error(),
			Latency:      time.Since(startTime),
			Timestamp:    time.Now(),
		}, err
	}

	// Check daily loss limits
	if err := e.checkDailyLossLimits(order); err != nil {
		return &RiskCheck{
			CheckType:    "daily_loss",
			Passed:       false,
			CurrentValue: e.limitManager.currentDailyLoss,
			LimitValue:   e.limitManager.dailyLossLimit,
			Message:      err.Error(),
			Latency:      time.Since(startTime),
			Timestamp:    time.Now(),
		}, err
	}

	// All checks passed
	atomic.AddInt64(&e.metrics.TotalChecks, 1)
	return &RiskCheck{
		CheckType: "pre_trade",
		Passed:    true,
		Message:   "All pre-trade checks passed",
		Latency:   time.Since(startTime),
		Timestamp: time.Now(),
	}, nil
}

// PostTradeCheck performs post-trade risk checks
func (e *RealTimeRiskEngine) PostTradeCheck(order *types.Order, trades []*Trade) error {
	if !e.config.EnablePostTradeChecks {
		return nil
	}

	startTime := time.Now()
	defer func() {
		latency := time.Since(startTime)
		e.updateMetrics(latency)
	}()

	// Update positions
	for _, trade := range trades {
		e.updatePosition(trade)
	}

	// Check if any limits were breached after the trade
	if err := e.checkPostTradeLimits(order); err != nil {
		e.publishEvent(&RiskEvent{
			Type:      EventLimitBreach,
			Symbol:    order.Symbol,
			Order:     order,
			Severity:  SeverityError,
			Message:   err.Error(),
			Timestamp: time.Now(),
		})
		return err
	}

	return nil
}

// checkOrderSizeLimits checks order size limits
func (e *RealTimeRiskEngine) checkOrderSizeLimits(order *types.Order) error {
	e.limitManager.mu.RLock()
	limit, exists := e.limitManager.orderLimits[order.Symbol]
	e.limitManager.mu.RUnlock()

	if !exists {
		limit = e.config.MaxOrderSize
	}

	if order.Quantity > limit {
		atomic.AddInt64(&e.metrics.RejectedOrders, 1)
		return fmt.Errorf("order size %f exceeds limit %f for symbol %s",
			order.Quantity, limit, order.Symbol)
	}

	return nil
}

// checkPositionLimits checks position limits
func (e *RealTimeRiskEngine) checkPositionLimits(order *types.Order) error {
	// Get current position
	position := e.getPosition(order.Symbol)

	// Calculate new position after order
	var newQuantity float64
	if order.Side == types.OrderSideBuy {
		newQuantity = position.Quantity + order.Quantity
	} else {
		newQuantity = position.Quantity - order.Quantity
	}

	// Check against position limits
	e.limitManager.mu.RLock()
	limit, exists := e.limitManager.positionLimits[order.Symbol]
	e.limitManager.mu.RUnlock()

	if !exists {
		limit = e.config.MaxPositionSize
	}

	if math.Abs(newQuantity) > limit {
		atomic.AddInt64(&e.metrics.RejectedOrders, 1)
		return fmt.Errorf("position size %f would exceed limit %f for symbol %s",
			math.Abs(newQuantity), limit, order.Symbol)
	}

	return nil
}

// checkDailyLossLimits checks daily loss limits
func (e *RealTimeRiskEngine) checkDailyLossLimits(order *types.Order) error {
	e.limitManager.mu.RLock()
	currentLoss := e.limitManager.currentDailyLoss
	limit := e.limitManager.dailyLossLimit
	e.limitManager.mu.RUnlock()

	if currentLoss > limit {
		atomic.AddInt64(&e.metrics.RejectedOrders, 1)
		return fmt.Errorf("daily loss %f exceeds limit %f", currentLoss, limit)
	}

	return nil
}

// checkPostTradeLimits checks limits after trade execution
func (e *RealTimeRiskEngine) checkPostTradeLimits(order *types.Order) error {
	// Check if position limits are still within bounds
	position := e.getPosition(order.Symbol)
	limit := e.getPositionLimit(order.Symbol)

	if math.Abs(position.Quantity) > limit {
		return fmt.Errorf("position %f exceeds limit %f after trade", 
			math.Abs(position.Quantity), limit)
	}

	return nil
}

// updatePosition updates position after a trade
func (e *RealTimeRiskEngine) updatePosition(trade *Trade) {
	positionInterface, _ := e.positionManager.positions.LoadOrStore(trade.Symbol, &RealtimePosition{
		Symbol: trade.Symbol,
	})
	
	position := positionInterface.(*RealtimePosition)
	
	// Update position based on trade
	if trade.Side == "buy" {
		// Calculate new average price
		totalValue := position.Quantity*position.AveragePrice + trade.Quantity*trade.Price
		position.Quantity += trade.Quantity
		if position.Quantity > 0 {
			position.AveragePrice = totalValue / position.Quantity
		}
	} else {
		position.Quantity -= trade.Quantity
		// Realize P&L on sale
		if position.AveragePrice > 0 {
			pnl := (trade.Price - position.AveragePrice) * trade.Quantity
			position.RealizedPnL += pnl
		}
	}
	
	position.LastUpdateTime = time.Now()
	e.positionManager.positions.Store(trade.Symbol, position)
}

// getPosition gets current position for a symbol
func (e *RealTimeRiskEngine) getPosition(symbol string) *RealtimePosition {
	if positionInterface, exists := e.positionManager.positions.Load(symbol); exists {
		return positionInterface.(*RealtimePosition)
	}
	
	return &RealtimePosition{Symbol: symbol}
}

// getOrderSizeLimit gets order size limit for a symbol
func (e *RealTimeRiskEngine) getOrderSizeLimit(symbol string) float64 {
	e.limitManager.mu.RLock()
	defer e.limitManager.mu.RUnlock()
	
	if limit, exists := e.limitManager.orderLimits[symbol]; exists {
		return limit
	}
	return e.config.MaxOrderSize
}

// getPositionLimit gets position limit for a symbol
func (e *RealTimeRiskEngine) getPositionLimit(symbol string) float64 {
	e.limitManager.mu.RLock()
	defer e.limitManager.mu.RUnlock()
	
	if limit, exists := e.limitManager.positionLimits[symbol]; exists {
		return limit
	}
	return e.config.MaxPositionSize
}

// getProjectedPosition calculates projected position after order
func (e *RealTimeRiskEngine) getProjectedPosition(order *types.Order) float64 {
	position := e.getPosition(order.Symbol)
	
	if order.Side == types.OrderSideBuy {
		return math.Abs(position.Quantity + order.Quantity)
	}
	return math.Abs(position.Quantity - order.Quantity)
}

// isCircuitBreakerOpen checks if circuit breaker is open
func (e *RealTimeRiskEngine) isCircuitBreakerOpen() bool {
	e.circuitBreaker.mu.RLock()
	defer e.circuitBreaker.mu.RUnlock()
	
	return e.circuitBreaker.state == CircuitBreakerOpen
}

// publishEvent publishes a risk event
func (e *RealTimeRiskEngine) publishEvent(event *RiskEvent) {
	select {
	case e.eventChannel <- event:
	default:
		e.logger.Warn("Event channel full, dropping event",
			zap.String("event_type", string(event.Type)),
			zap.String("symbol", event.Symbol))
	}
}

// processEvents processes risk events
func (e *RealTimeRiskEngine) processEvents(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-e.stopChannel:
			return
		case event := <-e.eventChannel:
			e.handleEvent(event)
		}
	}
}

// handleEvent handles a risk event
func (e *RealTimeRiskEngine) handleEvent(event *RiskEvent) {
	e.logger.Info("Processing risk event",
		zap.String("type", string(event.Type)),
		zap.String("symbol", event.Symbol),
		zap.String("severity", string(event.Severity)),
		zap.String("message", event.Message))
	
	// In a real implementation, would:
	// - Send alerts to risk managers
	// - Update dashboards
	// - Trigger automated responses
	// - Log to audit trail
}

// calculateVaRPeriodically calculates VaR periodically
func (e *RealTimeRiskEngine) calculateVaRPeriodically(ctx context.Context) {
	ticker := time.NewTicker(time.Hour) // Calculate VaR every hour
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-e.stopChannel:
			return
		case <-ticker.C:
			e.calculatePortfolioVaR()
		}
	}
}

// calculatePortfolioVaR calculates portfolio VaR
func (e *RealTimeRiskEngine) calculatePortfolioVaR() {
	// Simplified VaR calculation
	// In production, would use more sophisticated models
	
	e.logger.Debug("Calculating portfolio VaR")
	
	// This is a placeholder - real implementation would:
	// 1. Collect historical returns for all positions
	// 2. Calculate correlation matrix
	// 3. Apply Monte Carlo or parametric VaR calculation
	// 4. Generate VaR reports
}

// monitorCircuitBreaker monitors circuit breaker conditions
func (e *RealTimeRiskEngine) monitorCircuitBreaker(ctx context.Context) {
	ticker := time.NewTicker(time.Second * 10) // Check every 10 seconds
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-e.stopChannel:
			return
		case <-ticker.C:
			e.checkCircuitBreakerConditions()
		}
	}
}

// checkCircuitBreakerConditions checks if circuit breaker should be triggered
func (e *RealTimeRiskEngine) checkCircuitBreakerConditions() {
	// Simplified circuit breaker logic
	// In production, would check:
	// - Market volatility
	// - Price movements
	// - Volume spikes
	// - System health metrics
	
	e.circuitBreaker.mu.Lock()
	defer e.circuitBreaker.mu.Unlock()
	
	// Check if we should transition from open to half-open
	if e.circuitBreaker.state == CircuitBreakerOpen {
		if time.Since(e.circuitBreaker.lastFailureTime) > e.circuitBreaker.timeout {
			e.circuitBreaker.state = CircuitBreakerHalfOpen
			e.logger.Info("Circuit breaker transitioned to half-open")
		}
	}
}

// updateMetrics updates risk engine metrics
func (e *RealTimeRiskEngine) updateMetrics(latency time.Duration) {
	// Update average latency
	if e.metrics.AverageLatency == 0 {
		e.metrics.AverageLatency = latency
	} else {
		e.metrics.AverageLatency = (e.metrics.AverageLatency + latency) / 2
	}
	
	// Update max latency
	if latency > e.metrics.MaxLatency {
		e.metrics.MaxLatency = latency
	}
	
	e.metrics.LastUpdateTime = time.Now()
}
