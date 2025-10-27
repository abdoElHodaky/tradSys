package testing

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/pkg/interfaces"
	"github.com/abdoElHodaky/tradSys/pkg/types"
)

// MockLogger provides a mock implementation of the Logger interface
type MockLogger struct {
	logs []LogEntry
	mu   sync.RWMutex
}

// LogEntry represents a log entry
type LogEntry struct {
	Level   string
	Message string
	Fields  []interface{}
	Time    time.Time
}

// NewMockLogger creates a new mock logger
func NewMockLogger() *MockLogger {
	return &MockLogger{
		logs: make([]LogEntry, 0),
	}
}

// Debug logs a debug message
func (m *MockLogger) Debug(msg string, fields ...interface{}) {
	m.addLog("DEBUG", msg, fields)
}

// Info logs an info message
func (m *MockLogger) Info(msg string, fields ...interface{}) {
	m.addLog("INFO", msg, fields)
}

// Warn logs a warning message
func (m *MockLogger) Warn(msg string, fields ...interface{}) {
	m.addLog("WARN", msg, fields)
}

// Error logs an error message
func (m *MockLogger) Error(msg string, fields ...interface{}) {
	m.addLog("ERROR", msg, fields)
}

// Fatal logs a fatal message
func (m *MockLogger) Fatal(msg string, fields ...interface{}) {
	m.addLog("FATAL", msg, fields)
}

// addLog adds a log entry
func (m *MockLogger) addLog(level, msg string, fields []interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.logs = append(m.logs, LogEntry{
		Level:   level,
		Message: msg,
		Fields:  fields,
		Time:    time.Now(),
	})
}

// GetLogs returns all log entries
func (m *MockLogger) GetLogs() []LogEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()

	logs := make([]LogEntry, len(m.logs))
	copy(logs, m.logs)
	return logs
}

// GetLogsByLevel returns log entries filtered by level
func (m *MockLogger) GetLogsByLevel(level string) []LogEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var filtered []LogEntry
	for _, log := range m.logs {
		if log.Level == level {
			filtered = append(filtered, log)
		}
	}
	return filtered
}

// Clear clears all log entries
func (m *MockLogger) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logs = m.logs[:0]
}

// MockMetricsCollector provides a mock implementation of the MetricsCollector interface
type MockMetricsCollector struct {
	counters   map[string]float64
	gauges     map[string]float64
	histograms map[string][]float64
	timers     map[string][]time.Duration
	mu         sync.RWMutex
}

// NewMockMetricsCollector creates a new mock metrics collector
func NewMockMetricsCollector() *MockMetricsCollector {
	return &MockMetricsCollector{
		counters:   make(map[string]float64),
		gauges:     make(map[string]float64),
		histograms: make(map[string][]float64),
		timers:     make(map[string][]time.Duration),
	}
}

// IncrementCounter increments a counter metric
func (m *MockMetricsCollector) IncrementCounter(name string, tags map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := m.buildKey(name, tags)
	m.counters[key]++
}

// RecordGauge records a gauge metric
func (m *MockMetricsCollector) RecordGauge(name string, value float64, tags map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := m.buildKey(name, tags)
	m.gauges[key] = value
}

// RecordHistogram records a histogram metric
func (m *MockMetricsCollector) RecordHistogram(name string, value float64, tags map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := m.buildKey(name, tags)
	m.histograms[key] = append(m.histograms[key], value)
}

// RecordTimer records a timer metric
func (m *MockMetricsCollector) RecordTimer(name string, duration time.Duration, tags map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := m.buildKey(name, tags)
	m.timers[key] = append(m.timers[key], duration)
}

// Counter increments a counter metric (interface compatibility)
func (m *MockMetricsCollector) Counter(name string, value float64, tags map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := m.buildKey(name, tags)
	m.counters[key] += value
}

// Gauge sets a gauge metric (interface compatibility)
func (m *MockMetricsCollector) Gauge(name string, value float64, tags map[string]string) {
	m.RecordGauge(name, value, tags)
}

// Histogram records a histogram metric (interface compatibility)
func (m *MockMetricsCollector) Histogram(name string, value float64, tags map[string]string) {
	m.RecordHistogram(name, value, tags)
}

// Timer records a timing metric (interface compatibility)
func (m *MockMetricsCollector) Timer(name string, duration time.Duration, tags map[string]string) {
	m.RecordTimer(name, duration, tags)
}

// buildKey builds a key from name and tags
func (m *MockMetricsCollector) buildKey(name string, tags map[string]string) string {
	if len(tags) == 0 {
		return name
	}

	key := name
	for k, v := range tags {
		key += fmt.Sprintf(",%s=%s", k, v)
	}
	return key
}

// GetCounter returns the value of a counter
func (m *MockMetricsCollector) GetCounter(name string, tags map[string]string) float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := m.buildKey(name, tags)
	return m.counters[key]
}

// GetGauge returns the value of a gauge
func (m *MockMetricsCollector) GetGauge(name string, tags map[string]string) float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := m.buildKey(name, tags)
	return m.gauges[key]
}

// GetHistogram returns the values of a histogram
func (m *MockMetricsCollector) GetHistogram(name string, tags map[string]string) []float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := m.buildKey(name, tags)
	values := m.histograms[key]
	result := make([]float64, len(values))
	copy(result, values)
	return result
}

// GetTimer returns the values of a timer
func (m *MockMetricsCollector) GetTimer(name string, tags map[string]string) []time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := m.buildKey(name, tags)
	values := m.timers[key]
	result := make([]time.Duration, len(values))
	copy(result, values)
	return result
}

// Clear clears all metrics
func (m *MockMetricsCollector) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.counters = make(map[string]float64)
	m.gauges = make(map[string]float64)
	m.histograms = make(map[string][]float64)
	m.timers = make(map[string][]time.Duration)
}

// MockEventPublisher provides a mock implementation of the EventPublisher interface
type MockEventPublisher struct {
	orderEvents      []interfaces.OrderEvent
	tradeEvents      []interfaces.TradeEvent
	marketDataEvents []interfaces.MarketDataEvent
	mu               sync.RWMutex
}

// NewMockEventPublisher creates a new mock event publisher
func NewMockEventPublisher() *MockEventPublisher {
	return &MockEventPublisher{
		orderEvents:      make([]interfaces.OrderEvent, 0),
		tradeEvents:      make([]interfaces.TradeEvent, 0),
		marketDataEvents: make([]interfaces.MarketDataEvent, 0),
	}
}

// Publish publishes a generic event (interface compatibility)
func (m *MockEventPublisher) Publish(ctx context.Context, event interface{}) error {
	// Handle different event types
	switch e := event.(type) {
	case interfaces.OrderEvent:
		return m.PublishOrderEvent(ctx, e)
	case interfaces.TradeEvent:
		return m.PublishTradeEvent(ctx, e)
	case interfaces.MarketDataEvent:
		return m.PublishMarketDataEvent(ctx, e)
	default:
		// For unknown types, just return success
		return nil
	}
}

// PublishBatch publishes multiple events (interface compatibility)
func (m *MockEventPublisher) PublishBatch(ctx context.Context, events []interface{}) error {
	for _, event := range events {
		if err := m.Publish(ctx, event); err != nil {
			return err
		}
	}
	return nil
}

// PublishOrderEvent publishes an order event
func (m *MockEventPublisher) PublishOrderEvent(ctx context.Context, event interfaces.OrderEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.orderEvents = append(m.orderEvents, event)
	return nil
}

// PublishTradeEvent publishes a trade event
func (m *MockEventPublisher) PublishTradeEvent(ctx context.Context, event interfaces.TradeEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.tradeEvents = append(m.tradeEvents, event)
	return nil
}

// PublishMarketDataEvent publishes a market data event
func (m *MockEventPublisher) PublishMarketDataEvent(ctx context.Context, event interfaces.MarketDataEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.marketDataEvents = append(m.marketDataEvents, event)
	return nil
}

// GetOrderEvents returns all order events
func (m *MockEventPublisher) GetOrderEvents() []interfaces.OrderEvent {
	m.mu.RLock()
	defer m.mu.RUnlock()

	events := make([]interfaces.OrderEvent, len(m.orderEvents))
	copy(events, m.orderEvents)
	return events
}

// GetTradeEvents returns all trade events
func (m *MockEventPublisher) GetTradeEvents() []interfaces.TradeEvent {
	m.mu.RLock()
	defer m.mu.RUnlock()

	events := make([]interfaces.TradeEvent, len(m.tradeEvents))
	copy(events, m.tradeEvents)
	return events
}

// GetMarketDataEvents returns all market data events
func (m *MockEventPublisher) GetMarketDataEvents() []interfaces.MarketDataEvent {
	m.mu.RLock()
	defer m.mu.RUnlock()

	events := make([]interfaces.MarketDataEvent, len(m.marketDataEvents))
	copy(events, m.marketDataEvents)
	return events
}

// Clear clears all events
func (m *MockEventPublisher) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.orderEvents = m.orderEvents[:0]
	m.tradeEvents = m.tradeEvents[:0]
	m.marketDataEvents = m.marketDataEvents[:0]
}

// TestDataGenerator provides utilities for generating test data
type TestDataGenerator struct {
	rand *rand.Rand
}

// NewTestDataGenerator creates a new test data generator
func NewTestDataGenerator(seed int64) *TestDataGenerator {
	return &TestDataGenerator{
		rand: rand.New(rand.NewSource(seed)),
	}
}

// GenerateOrder generates a random order
func (g *TestDataGenerator) GenerateOrder() *types.Order {
	sides := []types.OrderSide{types.OrderSideBuy, types.OrderSideSell}
	orderTypes := []types.OrderType{types.OrderTypeLimit, types.OrderTypeMarket}
	symbols := []string{"BTCUSD", "ETHUSD", "ADAUSD", "DOTUSD"}

	order := &types.Order{
		ID:            g.generateID(),
		ClientOrderID: g.generateID(),
		UserID:        g.generateUserID(),
		Symbol:        symbols[g.rand.Intn(len(symbols))],
		Side:          sides[g.rand.Intn(len(sides))],
		Type:          orderTypes[g.rand.Intn(len(orderTypes))],
		Quantity:      g.rand.Float64()*100 + 1, // 1-101
		Status:        types.OrderStatusPending,
		TimeInForce:   types.TimeInForceGTC,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if order.Type == types.OrderTypeLimit {
		order.Price = g.rand.Float64()*50000 + 1000 // 1000-51000
	}

	order.RemainingQuantity = order.Quantity

	return order
}

// GenerateTrade generates a random trade
func (g *TestDataGenerator) GenerateTrade() *types.Trade {
	symbols := []string{"BTCUSD", "ETHUSD", "ADAUSD", "DOTUSD"}
	sides := []types.OrderSide{types.OrderSideBuy, types.OrderSideSell}

	price := g.rand.Float64()*50000 + 1000
	quantity := g.rand.Float64()*10 + 0.1

	trade := &types.Trade{
		ID:           g.generateID(),
		Symbol:       symbols[g.rand.Intn(len(symbols))],
		BuyOrderID:   g.generateID(),
		SellOrderID:  g.generateID(),
		Price:        price,
		Quantity:     quantity,
		Value:        price * quantity,
		Timestamp:    time.Now(),
		BuyUserID:    g.generateUserID(),
		SellUserID:   g.generateUserID(),
		TakerSide:    sides[g.rand.Intn(len(sides))],
		MakerOrderID: g.generateID(),
		TakerOrderID: g.generateID(),
	}

	return trade
}

// GenerateMarketData generates random market data
func (g *TestDataGenerator) GenerateMarketData(symbol string) *types.MarketData {
	basePrice := g.rand.Float64()*50000 + 1000
	spread := basePrice * 0.001 // 0.1% spread

	return &types.MarketData{
		Symbol:           symbol,
		LastPrice:        basePrice,
		BidPrice:         basePrice - spread/2,
		AskPrice:         basePrice + spread/2,
		Volume:           g.rand.Float64() * 1000000,
		High24h:          basePrice * (1 + g.rand.Float64()*0.1),
		Low24h:           basePrice * (1 - g.rand.Float64()*0.1),
		Change24h:        (g.rand.Float64() - 0.5) * basePrice * 0.1,
		ChangePercent24h: (g.rand.Float64() - 0.5) * 10,
		Timestamp:        time.Now(),
	}
}

// GenerateOHLCV generates random OHLCV data
func (g *TestDataGenerator) GenerateOHLCV(symbol, interval string) *types.OHLCV {
	open := g.rand.Float64()*50000 + 1000
	change := (g.rand.Float64() - 0.5) * open * 0.05 // 5% max change
	close := open + change

	high := max(open, close) * (1 + g.rand.Float64()*0.02)
	low := min(open, close) * (1 - g.rand.Float64()*0.02)

	return &types.OHLCV{
		Symbol:    symbol,
		Open:      open,
		High:      high,
		Low:       low,
		Close:     close,
		Volume:    g.rand.Float64() * 100000,
		Timestamp: time.Now(),
		Interval:  interval,
	}
}

// generateID generates a random ID
func (g *TestDataGenerator) generateID() string {
	return fmt.Sprintf("test_%d_%d", time.Now().UnixNano(), g.rand.Int63())
}

// generateUserID generates a random user ID
func (g *TestDataGenerator) generateUserID() string {
	return fmt.Sprintf("user_%d", g.rand.Intn(1000)+1)
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

// TestSuite provides a comprehensive test suite for trading system components
type TestSuite struct {
	logger    *MockLogger
	metrics   *MockMetricsCollector
	publisher *MockEventPublisher
	generator *TestDataGenerator
}

// NewTestSuite creates a new test suite
func NewTestSuite() *TestSuite {
	return &TestSuite{
		logger:    NewMockLogger(),
		metrics:   NewMockMetricsCollector(),
		publisher: NewMockEventPublisher(),
		generator: NewTestDataGenerator(time.Now().UnixNano()),
	}
}

// GetLogger returns the mock logger
func (ts *TestSuite) GetLogger() interfaces.Logger {
	return ts.logger
}

// GetMetrics returns the mock metrics collector
func (ts *TestSuite) GetMetrics() interfaces.MetricsCollector {
	return ts.metrics
}

// GetPublisher returns the mock event publisher
func (ts *TestSuite) GetPublisher() interfaces.EventPublisher {
	return ts.publisher
}

// GetGenerator returns the test data generator
func (ts *TestSuite) GetGenerator() *TestDataGenerator {
	return ts.generator
}

// Reset resets all mock components
func (ts *TestSuite) Reset() {
	ts.logger.Clear()
	ts.metrics.Clear()
	ts.publisher.Clear()
}

// AssertNoErrors asserts that no error logs were recorded
func (ts *TestSuite) AssertNoErrors() error {
	errorLogs := ts.logger.GetLogsByLevel("ERROR")
	if len(errorLogs) > 0 {
		return fmt.Errorf("expected no error logs, but found %d: %v", len(errorLogs), errorLogs)
	}
	return nil
}

// AssertMetricExists asserts that a metric exists
func (ts *TestSuite) AssertMetricExists(name string, tags map[string]string) error {
	counter := ts.metrics.GetCounter(name, tags)
	gauge := ts.metrics.GetGauge(name, tags)
	histogram := ts.metrics.GetHistogram(name, tags)
	timer := ts.metrics.GetTimer(name, tags)

	if counter == 0 && gauge == 0 && len(histogram) == 0 && len(timer) == 0 {
		return fmt.Errorf("metric %s with tags %v not found", name, tags)
	}

	return nil
}

// AssertEventPublished asserts that an event was published
func (ts *TestSuite) AssertEventPublished(eventType string) error {
	switch eventType {
	case "order":
		if len(ts.publisher.GetOrderEvents()) == 0 {
			return fmt.Errorf("no order events published")
		}
	case "trade":
		if len(ts.publisher.GetTradeEvents()) == 0 {
			return fmt.Errorf("no trade events published")
		}
	case "market_data":
		if len(ts.publisher.GetMarketDataEvents()) == 0 {
			return fmt.Errorf("no market data events published")
		}
	default:
		return fmt.Errorf("unknown event type: %s", eventType)
	}

	return nil
}

// LoadTestRunner provides utilities for running load tests
type LoadTestRunner struct {
	concurrency int
	duration    time.Duration
	rampUp      time.Duration
}

// NewLoadTestRunner creates a new load test runner
func NewLoadTestRunner(concurrency int, duration, rampUp time.Duration) *LoadTestRunner {
	return &LoadTestRunner{
		concurrency: concurrency,
		duration:    duration,
		rampUp:      rampUp,
	}
}

// Run runs a load test
func (ltr *LoadTestRunner) Run(testFunc func() error) *LoadTestResults {
	results := &LoadTestResults{
		StartTime:   time.Now(),
		Concurrency: ltr.concurrency,
		Duration:    ltr.duration,
		Requests:    make([]RequestResult, 0),
	}

	var wg sync.WaitGroup
	requestCh := make(chan RequestResult, ltr.concurrency*10)

	// Start workers
	for i := 0; i < ltr.concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			// Ramp up delay
			if ltr.rampUp > 0 {
				delay := time.Duration(int64(ltr.rampUp) * int64(workerID) / int64(ltr.concurrency))
				time.Sleep(delay)
			}

			endTime := results.StartTime.Add(ltr.duration)
			for time.Now().Before(endTime) {
				start := time.Now()
				err := testFunc()
				duration := time.Since(start)

				requestCh <- RequestResult{
					Duration: duration,
					Error:    err,
					WorkerID: workerID,
				}
			}
		}(i)
	}

	// Collect results
	go func() {
		wg.Wait()
		close(requestCh)
	}()

	for result := range requestCh {
		results.Requests = append(results.Requests, result)
		if result.Error != nil {
			results.ErrorCount++
		}
	}

	results.EndTime = time.Now()
	results.calculateStatistics()

	return results
}

// LoadTestResults contains the results of a load test
type LoadTestResults struct {
	StartTime     time.Time
	EndTime       time.Time
	Duration      time.Duration
	Concurrency   int
	Requests      []RequestResult
	ErrorCount    int
	TotalRequests int
	RPS           float64
	AvgLatency    time.Duration
	MinLatency    time.Duration
	MaxLatency    time.Duration
	P95Latency    time.Duration
	P99Latency    time.Duration
}

// RequestResult contains the result of a single request
type RequestResult struct {
	Duration time.Duration
	Error    error
	WorkerID int
}

// calculateStatistics calculates load test statistics
func (ltr *LoadTestResults) calculateStatistics() {
	ltr.TotalRequests = len(ltr.Requests)
	if ltr.TotalRequests == 0 {
		return
	}

	actualDuration := ltr.EndTime.Sub(ltr.StartTime)
	ltr.RPS = float64(ltr.TotalRequests) / actualDuration.Seconds()

	// Calculate latency statistics
	var totalLatency time.Duration
	ltr.MinLatency = time.Hour // Start with a large value

	latencies := make([]time.Duration, len(ltr.Requests))
	for i, req := range ltr.Requests {
		latencies[i] = req.Duration
		totalLatency += req.Duration

		if req.Duration > ltr.MaxLatency {
			ltr.MaxLatency = req.Duration
		}
		if req.Duration < ltr.MinLatency {
			ltr.MinLatency = req.Duration
		}
	}

	ltr.AvgLatency = totalLatency / time.Duration(ltr.TotalRequests)

	// Calculate percentiles (simple implementation)
	if len(latencies) > 0 {
		// Sort latencies for percentile calculation
		for i := 0; i < len(latencies)-1; i++ {
			for j := i + 1; j < len(latencies); j++ {
				if latencies[i] > latencies[j] {
					latencies[i], latencies[j] = latencies[j], latencies[i]
				}
			}
		}

		p95Index := int(float64(len(latencies)) * 0.95)
		p99Index := int(float64(len(latencies)) * 0.99)

		if p95Index < len(latencies) {
			ltr.P95Latency = latencies[p95Index]
		}
		if p99Index < len(latencies) {
			ltr.P99Latency = latencies[p99Index]
		}
	}
}
