package risk_management

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/common/pool"
	"github.com/abdoElHodaky/tradSys/internal/trading/types"
	"go.uber.org/zap"
)

// EventProcessor handles real-time risk event processing with HFT performance
type EventProcessor struct {
	config       *RiskEngineConfig
	logger       *zap.Logger
	eventPool    *pool.ObjectPool
	checkPool    *pool.ObjectPool
	eventChannel chan *RiskEvent
	stopChannel  chan struct{}
	isRunning    int32
	mu           sync.RWMutex
	
	// Event processing metrics
	eventsProcessed int64
	processingTime  int64 // nanoseconds
	errorCount      int64
}

// RiskEvent represents a risk event that needs processing
type RiskEvent struct {
	Type        RiskEventType   `json:"type"`
	OrderID     string          `json:"order_id"`
	UserID      string          `json:"user_id"`
	Symbol      string          `json:"symbol"`
	Side        types.OrderSide `json:"side"`
	Quantity    float64         `json:"quantity"`
	Price       float64         `json:"price"`
	Timestamp   time.Time       `json:"timestamp"`
	Metadata    map[string]interface{} `json:"metadata"`
	ResultChan  chan *RiskCheckResult  `json:"-"`
}

// RiskEventType defines the type of risk event
type RiskEventType int

const (
	RiskEventPreTrade RiskEventType = iota
	RiskEventPostTrade
	RiskEventPositionUpdate
	RiskEventMarketData
	RiskEventLimitBreach
	RiskEventCircuitBreaker
)

// String returns the string representation of the risk event type
func (ret RiskEventType) String() string {
	switch ret {
	case RiskEventPreTrade:
		return "pre_trade"
	case RiskEventPostTrade:
		return "post_trade"
	case RiskEventPositionUpdate:
		return "position_update"
	case RiskEventMarketData:
		return "market_data"
	case RiskEventLimitBreach:
		return "limit_breach"
	case RiskEventCircuitBreaker:
		return "circuit_breaker"
	default:
		return "unknown"
	}
}

// RiskCheckResult contains the result of a risk check
type RiskCheckResult struct {
	Approved     bool              `json:"approved"`
	Reason       string            `json:"reason"`
	RiskScore    float64           `json:"risk_score"`
	Violations   []string          `json:"violations"`
	MaxOrderSize float64           `json:"max_order_size"`
	Latency      time.Duration     `json:"latency"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// NewEventProcessor creates a new event processor
func NewEventProcessor(config *RiskEngineConfig, logger *zap.Logger) *EventProcessor {
	return &EventProcessor{
		config:       config,
		logger:       logger,
		eventPool:    pool.NewObjectPool(1000, func() interface{} { return &RiskEvent{} }),
		checkPool:    pool.NewObjectPool(1000, func() interface{} { return &RiskCheckResult{} }),
		eventChannel: make(chan *RiskEvent, 10000), // High-capacity buffer
		stopChannel:  make(chan struct{}),
	}
}

// Start starts the event processor
func (ep *EventProcessor) Start(ctx context.Context) error {
	if !atomic.CompareAndSwapInt32(&ep.isRunning, 0, 1) {
		return fmt.Errorf("event processor already running")
	}

	ep.logger.Info("Starting risk event processor")

	// Start event processing goroutines
	numWorkers := 4 // Configurable based on CPU cores
	for i := 0; i < numWorkers; i++ {
		go ep.processEvents(ctx, i)
	}

	// Start metrics collection
	go ep.collectMetrics(ctx)

	return nil
}

// Stop stops the event processor
func (ep *EventProcessor) Stop(ctx context.Context) error {
	if !atomic.CompareAndSwapInt32(&ep.isRunning, 1, 0) {
		return fmt.Errorf("event processor not running")
	}

	ep.logger.Info("Stopping risk event processor")
	close(ep.stopChannel)

	// Wait for graceful shutdown with timeout
	select {
	case <-time.After(5 * time.Second):
		ep.logger.Warn("Event processor shutdown timeout")
	case <-ctx.Done():
		ep.logger.Info("Event processor stopped")
	}

	return nil
}

// SubmitEvent submits a risk event for processing
func (ep *EventProcessor) SubmitEvent(event *RiskEvent) error {
	if atomic.LoadInt32(&ep.isRunning) == 0 {
		return fmt.Errorf("event processor not running")
	}

	select {
	case ep.eventChannel <- event:
		return nil
	default:
		atomic.AddInt64(&ep.errorCount, 1)
		return fmt.Errorf("event channel full, dropping event")
	}
}

// processEvents processes risk events in a worker goroutine
func (ep *EventProcessor) processEvents(ctx context.Context, workerID int) {
	ep.logger.Debug("Starting event processor worker", zap.Int("workerID", workerID))

	for {
		select {
		case <-ep.stopChannel:
			ep.logger.Debug("Event processor worker stopping", zap.Int("workerID", workerID))
			return
		case <-ctx.Done():
			ep.logger.Debug("Event processor worker context cancelled", zap.Int("workerID", workerID))
			return
		case event := <-ep.eventChannel:
			ep.processEvent(event, workerID)
		}
	}
}

// processEvent processes a single risk event
func (ep *EventProcessor) processEvent(event *RiskEvent, workerID int) {
	startTime := time.Now()
	
	// Get result object from pool
	result := ep.checkPool.Get().(*RiskCheckResult)
	defer ep.checkPool.Put(result)

	// Reset result
	*result = RiskCheckResult{
		Approved:  true,
		Metadata:  make(map[string]interface{}),
	}

	// Process based on event type
	switch event.Type {
	case RiskEventPreTrade:
		ep.processPreTradeEvent(event, result)
	case RiskEventPostTrade:
		ep.processPostTradeEvent(event, result)
	case RiskEventPositionUpdate:
		ep.processPositionUpdateEvent(event, result)
	case RiskEventMarketData:
		ep.processMarketDataEvent(event, result)
	case RiskEventLimitBreach:
		ep.processLimitBreachEvent(event, result)
	case RiskEventCircuitBreaker:
		ep.processCircuitBreakerEvent(event, result)
	default:
		result.Approved = false
		result.Reason = "Unknown event type"
	}

	// Calculate processing latency
	processingTime := time.Since(startTime)
	result.Latency = processingTime
	
	// Update metrics
	atomic.AddInt64(&ep.eventsProcessed, 1)
	atomic.AddInt64(&ep.processingTime, processingTime.Nanoseconds())

	// Send result back
	if event.ResultChan != nil {
		select {
		case event.ResultChan <- result:
		default:
			ep.logger.Warn("Result channel full, dropping result")
		}
	}

	// Log slow processing
	if processingTime > ep.config.MaxLatency {
		ep.logger.Warn("Slow event processing detected",
			zap.String("eventType", event.Type.String()),
			zap.Duration("latency", processingTime),
			zap.Duration("maxLatency", ep.config.MaxLatency),
			zap.Int("workerID", workerID),
		)
	}
}

// processPreTradeEvent processes pre-trade risk checks
func (ep *EventProcessor) processPreTradeEvent(event *RiskEvent, result *RiskCheckResult) {
	// Pre-trade checks: position limits, order size, etc.
	if event.Quantity > ep.config.MaxOrderSize {
		result.Approved = false
		result.Reason = "Order size exceeds maximum allowed"
		result.Violations = append(result.Violations, "max_order_size")
		return
	}

	// Additional pre-trade checks would go here
	result.RiskScore = ep.calculateRiskScore(event)
}

// processPostTradeEvent processes post-trade risk checks
func (ep *EventProcessor) processPostTradeEvent(event *RiskEvent, result *RiskCheckResult) {
	// Post-trade checks: position updates, P&L calculations, etc.
	result.RiskScore = ep.calculateRiskScore(event)
}

// processPositionUpdateEvent processes position update events
func (ep *EventProcessor) processPositionUpdateEvent(event *RiskEvent, result *RiskCheckResult) {
	// Position update processing
	result.Metadata["position_updated"] = true
}

// processMarketDataEvent processes market data events
func (ep *EventProcessor) processMarketDataEvent(event *RiskEvent, result *RiskCheckResult) {
	// Market data processing for risk calculations
	result.Metadata["market_data_processed"] = true
}

// processLimitBreachEvent processes limit breach events
func (ep *EventProcessor) processLimitBreachEvent(event *RiskEvent, result *RiskCheckResult) {
	// Limit breach handling
	result.Approved = false
	result.Reason = "Risk limit breached"
	result.Violations = append(result.Violations, "limit_breach")
}

// processCircuitBreakerEvent processes circuit breaker events
func (ep *EventProcessor) processCircuitBreakerEvent(event *RiskEvent, result *RiskCheckResult) {
	// Circuit breaker handling
	result.Approved = false
	result.Reason = "Circuit breaker triggered"
	result.Violations = append(result.Violations, "circuit_breaker")
}

// calculateRiskScore calculates a risk score for an event
func (ep *EventProcessor) calculateRiskScore(event *RiskEvent) float64 {
	// Simple risk scoring - would be more sophisticated in production
	score := 0.0
	
	// Size-based risk
	if event.Quantity > ep.config.MaxOrderSize*0.8 {
		score += 0.3
	}
	
	// Price-based risk (simplified)
	if event.Price > 0 {
		score += 0.1
	}
	
	return score
}

// collectMetrics collects and logs performance metrics
func (ep *EventProcessor) collectMetrics(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	var lastProcessed int64
	var lastProcessingTime int64

	for {
		select {
		case <-ep.stopChannel:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			processed := atomic.LoadInt64(&ep.eventsProcessed)
			processingTime := atomic.LoadInt64(&ep.processingTime)
			errors := atomic.LoadInt64(&ep.errorCount)

			deltaProcessed := processed - lastProcessed
			deltaTime := processingTime - lastProcessingTime

			var avgLatency time.Duration
			if deltaProcessed > 0 {
				avgLatency = time.Duration(deltaTime / deltaProcessed)
			}

			ep.logger.Info("Event processor metrics",
				zap.Int64("eventsProcessed", processed),
				zap.Int64("deltaProcessed", deltaProcessed),
				zap.Duration("avgLatency", avgLatency),
				zap.Int64("errors", errors),
				zap.Float64("eventsPerSecond", float64(deltaProcessed)/10.0),
			)

			lastProcessed = processed
			lastProcessingTime = processingTime
		}
	}
}

// GetMetrics returns current processing metrics
func (ep *EventProcessor) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"events_processed":   atomic.LoadInt64(&ep.eventsProcessed),
		"processing_time_ns": atomic.LoadInt64(&ep.processingTime),
		"error_count":        atomic.LoadInt64(&ep.errorCount),
		"is_running":         atomic.LoadInt32(&ep.isRunning) == 1,
	}
}
