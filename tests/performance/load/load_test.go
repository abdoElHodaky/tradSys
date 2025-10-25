package load

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/orders"
	"github.com/abdoElHodaky/tradSys/internal/risk"
	"github.com/abdoElHodaky/tradSys/pkg/matching"
	"github.com/stretchr/testify/require"
)

// LoadTestConfig defines configuration for load testing
type LoadTestConfig struct {
	Duration           time.Duration
	ConcurrentUsers    int
	OrdersPerSecond    int
	SymbolCount        int
	PriceRange         float64
	BasePrice          float64
	EnableRiskChecks   bool
	EnableCompliance   bool
	ReportInterval     time.Duration
}

// LoadTestMetrics tracks performance metrics during load testing
type LoadTestMetrics struct {
	TotalOrders       int64
	SuccessfulOrders  int64
	FailedOrders      int64
	TotalTrades       int64
	AvgLatency        time.Duration
	MaxLatency        time.Duration
	MinLatency        time.Duration
	P95Latency        time.Duration
	P99Latency        time.Duration
	OrdersPerSecond   float64
	TradesPerSecond   float64
	ErrorRate         float64
	StartTime         time.Time
	EndTime           time.Time
}

// LoadTestResult contains comprehensive load test results
type LoadTestResult struct {
	Config  LoadTestConfig
	Metrics LoadTestMetrics
	Errors  []error
}

// TestLoad_MatchingEngine_HighThroughput tests matching engine under high load
func TestLoad_MatchingEngine_HighThroughput(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	config := LoadTestConfig{
		Duration:           5 * time.Minute,
		ConcurrentUsers:    100,
		OrdersPerSecond:    10000,
		SymbolCount:        10,
		PriceRange:         10.0,
		BasePrice:          150.0,
		EnableRiskChecks:   false, // Disable for pure performance testing
		EnableCompliance:   false,
		ReportInterval:     30 * time.Second,
	}

	result := runMatchingEngineLoadTest(t, config)
	
	// Validate performance targets
	require.Greater(t, result.Metrics.OrdersPerSecond, 100000.0, 
		"Should achieve >100k orders/second, got %.2f", result.Metrics.OrdersPerSecond)
	require.Less(t, result.Metrics.AvgLatency, 100*time.Microsecond,
		"Average latency should be <100μs, got %v", result.Metrics.AvgLatency)
	require.Less(t, result.Metrics.P99Latency, 1*time.Millisecond,
		"P99 latency should be <1ms, got %v", result.Metrics.P99Latency)
	require.Less(t, result.Metrics.ErrorRate, 0.01,
		"Error rate should be <1%%, got %.2f%%", result.Metrics.ErrorRate*100)

	t.Logf("Load Test Results:")
	t.Logf("  Duration: %v", result.Config.Duration)
	t.Logf("  Total Orders: %d", result.Metrics.TotalOrders)
	t.Logf("  Orders/Second: %.2f", result.Metrics.OrdersPerSecond)
	t.Logf("  Trades/Second: %.2f", result.Metrics.TradesPerSecond)
	t.Logf("  Average Latency: %v", result.Metrics.AvgLatency)
	t.Logf("  P95 Latency: %v", result.Metrics.P95Latency)
	t.Logf("  P99 Latency: %v", result.Metrics.P99Latency)
	t.Logf("  Error Rate: %.2f%%", result.Metrics.ErrorRate*100)
}

// TestLoad_OrderService_ConcurrentUsers tests order service with concurrent users
func TestLoad_OrderService_ConcurrentUsers(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	config := LoadTestConfig{
		Duration:           3 * time.Minute,
		ConcurrentUsers:    500,
		OrdersPerSecond:    5000,
		SymbolCount:        20,
		PriceRange:         20.0,
		BasePrice:          100.0,
		EnableRiskChecks:   true,
		EnableCompliance:   true,
		ReportInterval:     30 * time.Second,
	}

	result := runOrderServiceLoadTest(t, config)
	
	// Validate service performance under realistic load
	require.Greater(t, result.Metrics.OrdersPerSecond, 5000.0,
		"Should achieve >5k orders/second with full validation, got %.2f", result.Metrics.OrdersPerSecond)
	require.Less(t, result.Metrics.AvgLatency, 10*time.Millisecond,
		"Average latency should be <10ms with validation, got %v", result.Metrics.AvgLatency)
	require.Less(t, result.Metrics.ErrorRate, 0.05,
		"Error rate should be <5%% with validation, got %.2f%%", result.Metrics.ErrorRate*100)

	t.Logf("Order Service Load Test Results:")
	t.Logf("  Concurrent Users: %d", result.Config.ConcurrentUsers)
	t.Logf("  Total Orders: %d", result.Metrics.TotalOrders)
	t.Logf("  Success Rate: %.2f%%", (1-result.Metrics.ErrorRate)*100)
	t.Logf("  Orders/Second: %.2f", result.Metrics.OrdersPerSecond)
	t.Logf("  Average Latency: %v", result.Metrics.AvgLatency)
}

// TestLoad_EndToEnd_TradingFlow tests complete trading flow under load
func TestLoad_EndToEnd_TradingFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	config := LoadTestConfig{
		Duration:           10 * time.Minute,
		ConcurrentUsers:    200,
		OrdersPerSecond:    2000,
		SymbolCount:        5,
		PriceRange:         5.0,
		BasePrice:          150.0,
		EnableRiskChecks:   true,
		EnableCompliance:   true,
		ReportInterval:     1 * time.Minute,
	}

	result := runEndToEndLoadTest(t, config)
	
	// Validate end-to-end performance
	require.Greater(t, result.Metrics.OrdersPerSecond, 2000.0,
		"Should achieve >2k orders/second end-to-end, got %.2f", result.Metrics.OrdersPerSecond)
	require.Greater(t, result.Metrics.TradesPerSecond, 500.0,
		"Should achieve >500 trades/second, got %.2f", result.Metrics.TradesPerSecond)
	require.Less(t, result.Metrics.ErrorRate, 0.02,
		"Error rate should be <2%% end-to-end, got %.2f%%", result.Metrics.ErrorRate*100)

	t.Logf("End-to-End Load Test Results:")
	t.Logf("  Test Duration: %v", result.Config.Duration)
	t.Logf("  Total Orders: %d", result.Metrics.TotalOrders)
	t.Logf("  Total Trades: %d", result.Metrics.TotalTrades)
	t.Logf("  Trade Ratio: %.2f%%", float64(result.Metrics.TotalTrades)/float64(result.Metrics.TotalOrders)*100)
	t.Logf("  Orders/Second: %.2f", result.Metrics.OrdersPerSecond)
	t.Logf("  Trades/Second: %.2f", result.Metrics.TradesPerSecond)
}

// runMatchingEngineLoadTest executes load test against matching engine
func runMatchingEngineLoadTest(t *testing.T, config LoadTestConfig) *LoadTestResult {
	engine := matching.NewEngine(&matching.Config{
		Symbol:           "AAPL",
		LatencyTargetNS:  100000, // 100μs target
		MaxOrdersPerSec:  1000000,
		OrderBookDepth:   10000,
		EnableHFTMode:    true,
		NUMAOptimization: true,
	})

	ctx, cancel := context.WithTimeout(context.Background(), config.Duration+time.Minute)
	defer cancel()

	metrics := &LoadTestMetrics{
		StartTime:  time.Now(),
		MinLatency: time.Hour, // Initialize to large value
	}

	var wg sync.WaitGroup
	var orderCounter int64
	var tradeCounter int64
	var errorCounter int64
	latencies := make([]time.Duration, 0, config.OrdersPerSecond*int(config.Duration.Seconds()))
	latencyMutex := sync.Mutex{}

	// Start metrics reporter
	go reportMetrics(t, metrics, config.ReportInterval, config.Duration)

	// Calculate orders per goroutine
	ordersPerGoroutine := config.OrdersPerSecond / config.ConcurrentUsers
	if ordersPerGoroutine == 0 {
		ordersPerGoroutine = 1
	}

	// Start load generators
	for i := 0; i < config.ConcurrentUsers; i++ {
		wg.Add(1)
		go func(userID int) {
			defer wg.Done()
			
			ticker := time.NewTicker(time.Second / time.Duration(ordersPerGoroutine))
			defer ticker.Stop()
			
			orderID := 0
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					// Generate order
					side := matching.SideBuy
					if orderID%2 == 1 {
						side = matching.SideSell
					}
					
					price := config.BasePrice + (float64(orderID%int(config.PriceRange*100)) / 100.0)
					
					order := &matching.Order{
						ID:       fmt.Sprintf("user-%d-order-%d", userID, orderID),
						UserID:   fmt.Sprintf("user-%d", userID),
						Symbol:   "AAPL",
						Side:     side,
						Type:     matching.TypeLimit,
						Quantity: 100,
						Price:    price,
						TimeInForce: matching.TimeInForceGTC,
						Timestamp:   time.Now(),
					}
					
					// Measure latency
					start := time.Now()
					trades, err := engine.ProcessOrder(ctx, order)
					latency := time.Since(start)
					
					// Update metrics
					atomic.AddInt64(&orderCounter, 1)
					if err != nil {
						atomic.AddInt64(&errorCounter, 1)
					} else {
						atomic.AddInt64(&tradeCounter, int64(len(trades)))
					}
					
					// Record latency
					latencyMutex.Lock()
					latencies = append(latencies, latency)
					latencyMutex.Unlock()
					
					orderID++
				}
			}
		}(i)
	}

	// Wait for completion
	wg.Wait()
	metrics.EndTime = time.Now()

	// Calculate final metrics
	duration := metrics.EndTime.Sub(metrics.StartTime)
	metrics.TotalOrders = atomic.LoadInt64(&orderCounter)
	metrics.SuccessfulOrders = metrics.TotalOrders - atomic.LoadInt64(&errorCounter)
	metrics.FailedOrders = atomic.LoadInt64(&errorCounter)
	metrics.TotalTrades = atomic.LoadInt64(&tradeCounter)
	metrics.OrdersPerSecond = float64(metrics.TotalOrders) / duration.Seconds()
	metrics.TradesPerSecond = float64(metrics.TotalTrades) / duration.Seconds()
	metrics.ErrorRate = float64(metrics.FailedOrders) / float64(metrics.TotalOrders)

	// Calculate latency statistics
	if len(latencies) > 0 {
		metrics.AvgLatency = calculateAvgLatency(latencies)
		metrics.MinLatency = calculateMinLatency(latencies)
		metrics.MaxLatency = calculateMaxLatency(latencies)
		metrics.P95Latency = calculatePercentileLatency(latencies, 0.95)
		metrics.P99Latency = calculatePercentileLatency(latencies, 0.99)
	}

	return &LoadTestResult{
		Config:  config,
		Metrics: *metrics,
	}
}

// runOrderServiceLoadTest executes load test against order service
func runOrderServiceLoadTest(t *testing.T, config LoadTestConfig) *LoadTestResult {
	orderService := orders.NewService(&orders.Config{
		MaxOrdersPerUser: 10000,
		MaxOrderValue:    10000000,
		EnableRiskChecks: config.EnableRiskChecks,
		EnableCompliance: config.EnableCompliance,
		OrderTimeout:     30 * time.Minute,
	})

	ctx, cancel := context.WithTimeout(context.Background(), config.Duration+time.Minute)
	defer cancel()

	metrics := &LoadTestMetrics{
		StartTime:  time.Now(),
		MinLatency: time.Hour,
	}

	var wg sync.WaitGroup
	var orderCounter int64
	var errorCounter int64
	latencies := make([]time.Duration, 0, config.OrdersPerSecond*int(config.Duration.Seconds()))
	latencyMutex := sync.Mutex{}

	// Start metrics reporter
	go reportMetrics(t, metrics, config.ReportInterval, config.Duration)

	// Calculate orders per goroutine
	ordersPerGoroutine := config.OrdersPerSecond / config.ConcurrentUsers
	if ordersPerGoroutine == 0 {
		ordersPerGoroutine = 1
	}

	// Start load generators
	for i := 0; i < config.ConcurrentUsers; i++ {
		wg.Add(1)
		go func(userID int) {
			defer wg.Done()
			
			ticker := time.NewTicker(time.Second / time.Duration(ordersPerGoroutine))
			defer ticker.Stop()
			
			orderID := 0
			symbols := generateSymbols(config.SymbolCount)
			
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					// Generate order request
					side := orders.SideBuy
					if orderID%2 == 1 {
						side = orders.SideSell
					}
					
					symbol := symbols[orderID%len(symbols)]
					price := config.BasePrice + (float64(orderID%int(config.PriceRange*100)) / 100.0)
					
					orderReq := &orders.CreateOrderRequest{
						UserID:      fmt.Sprintf("user-%d", userID),
						ClientOrderID: fmt.Sprintf("client-%d-%d", userID, orderID),
						Symbol:      symbol,
						Side:        side,
						Type:        orders.TypeLimit,
						Quantity:    100,
						Price:       price,
						TimeInForce: orders.TimeInForceGTC,
					}
					
					// Measure latency
					start := time.Now()
					_, err := orderService.CreateOrder(ctx, orderReq)
					latency := time.Since(start)
					
					// Update metrics
					atomic.AddInt64(&orderCounter, 1)
					if err != nil {
						atomic.AddInt64(&errorCounter, 1)
					}
					
					// Record latency
					latencyMutex.Lock()
					latencies = append(latencies, latency)
					latencyMutex.Unlock()
					
					orderID++
				}
			}
		}(i)
	}

	// Wait for completion
	wg.Wait()
	metrics.EndTime = time.Now()

	// Calculate final metrics
	duration := metrics.EndTime.Sub(metrics.StartTime)
	metrics.TotalOrders = atomic.LoadInt64(&orderCounter)
	metrics.SuccessfulOrders = metrics.TotalOrders - atomic.LoadInt64(&errorCounter)
	metrics.FailedOrders = atomic.LoadInt64(&errorCounter)
	metrics.OrdersPerSecond = float64(metrics.TotalOrders) / duration.Seconds()
	metrics.ErrorRate = float64(metrics.FailedOrders) / float64(metrics.TotalOrders)

	// Calculate latency statistics
	if len(latencies) > 0 {
		metrics.AvgLatency = calculateAvgLatency(latencies)
		metrics.MinLatency = calculateMinLatency(latencies)
		metrics.MaxLatency = calculateMaxLatency(latencies)
		metrics.P95Latency = calculatePercentileLatency(latencies, 0.95)
		metrics.P99Latency = calculatePercentileLatency(latencies, 0.99)
	}

	return &LoadTestResult{
		Config:  config,
		Metrics: *metrics,
	}
}

// runEndToEndLoadTest executes comprehensive end-to-end load test
func runEndToEndLoadTest(t *testing.T, config LoadTestConfig) *LoadTestResult {
	// Initialize all components
	orderService := orders.NewService(&orders.Config{
		MaxOrdersPerUser: 10000,
		MaxOrderValue:    10000000,
		EnableRiskChecks: config.EnableRiskChecks,
		EnableCompliance: config.EnableCompliance,
		OrderTimeout:     30 * time.Minute,
	})

	riskCalculator := risk.NewCalculator(&risk.Config{
		VaRConfidence:       0.95,
		CalculationInterval: time.Second,
		MaxPositionSize:     1000000,
		ConcentrationLimit:  0.3,
		EnableRealTimeCalc:  config.EnableRiskChecks,
	})

	matchingEngine := matching.NewEngine(&matching.Config{
		Symbol:           "AAPL",
		LatencyTargetNS:  100000,
		MaxOrdersPerSec:  100000,
		OrderBookDepth:   1000,
		EnableHFTMode:    true,
	})

	ctx, cancel := context.WithTimeout(context.Background(), config.Duration+time.Minute)
	defer cancel()

	metrics := &LoadTestMetrics{
		StartTime:  time.Now(),
		MinLatency: time.Hour,
	}

	var wg sync.WaitGroup
	var orderCounter int64
	var tradeCounter int64
	var errorCounter int64
	latencies := make([]time.Duration, 0, config.OrdersPerSecond*int(config.Duration.Seconds()))
	latencyMutex := sync.Mutex{}

	// Start metrics reporter
	go reportMetrics(t, metrics, config.ReportInterval, config.Duration)

	// Calculate orders per goroutine
	ordersPerGoroutine := config.OrdersPerSecond / config.ConcurrentUsers
	if ordersPerGoroutine == 0 {
		ordersPerGoroutine = 1
	}

	// Start load generators
	for i := 0; i < config.ConcurrentUsers; i++ {
		wg.Add(1)
		go func(userID int) {
			defer wg.Done()
			
			ticker := time.NewTicker(time.Second / time.Duration(ordersPerGoroutine))
			defer ticker.Stop()
			
			orderID := 0
			symbols := generateSymbols(config.SymbolCount)
			
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					start := time.Now()
					
					// Step 1: Create order
					side := orders.SideBuy
					if orderID%2 == 1 {
						side = orders.SideSell
					}
					
					symbol := symbols[orderID%len(symbols)]
					price := config.BasePrice + (float64(orderID%int(config.PriceRange*100)) / 100.0)
					
					orderReq := &orders.CreateOrderRequest{
						UserID:      fmt.Sprintf("user-%d", userID),
						ClientOrderID: fmt.Sprintf("client-%d-%d", userID, orderID),
						Symbol:      symbol,
						Side:        side,
						Type:        orders.TypeLimit,
						Quantity:    100,
						Price:       price,
						TimeInForce: orders.TimeInForceGTC,
					}
					
					order, err := orderService.CreateOrder(ctx, orderReq)
					if err != nil {
						atomic.AddInt64(&errorCounter, 1)
						continue
					}
					
					// Step 2: Risk check (if enabled)
					if config.EnableRiskChecks {
						portfolio := &risk.Portfolio{
							UserID: fmt.Sprintf("user-%d", userID),
							Positions: []risk.Position{
								{
									Symbol:        symbol,
									Quantity:      500,
									AveragePrice:  price * 0.95,
									CurrentPrice:  price,
									MarketValue:   price * 500,
									UnrealizedPnL: price * 500 * 0.05,
								},
							},
							TotalMarketValue:   price * 500,
							TotalUnrealizedPnL: price * 500 * 0.05,
							Cash:              100000,
							TotalValue:        100000 + price*500,
						}
						
						orderRisk := &risk.OrderRisk{
							UserID:    fmt.Sprintf("user-%d", userID),
							Symbol:    symbol,
							Side:      string(side),
							Quantity:  100,
							Price:     price,
							OrderType: "limit",
						}
						
						riskResult, err := riskCalculator.CalculateOrderRisk(ctx, orderRisk, portfolio)
						if err != nil || !riskResult.IsAcceptable {
							atomic.AddInt64(&errorCounter, 1)
							continue
						}
					}
					
					// Step 3: Submit to matching engine
					matchingOrder := &matching.Order{
						ID:          order.ID,
						UserID:      order.UserID,
						Symbol:      order.Symbol,
						Side:        matching.Side(order.Side),
						Type:        matching.Type(order.Type),
						Quantity:    order.Quantity,
						Price:       order.Price,
						TimeInForce: matching.TimeInForce(order.TimeInForce),
						Timestamp:   order.CreatedAt,
					}
					
					trades, err := matchingEngine.ProcessOrder(ctx, matchingOrder)
					if err != nil {
						atomic.AddInt64(&errorCounter, 1)
						continue
					}
					
					// Record metrics
					latency := time.Since(start)
					atomic.AddInt64(&orderCounter, 1)
					atomic.AddInt64(&tradeCounter, int64(len(trades)))
					
					// Record latency
					latencyMutex.Lock()
					latencies = append(latencies, latency)
					latencyMutex.Unlock()
					
					orderID++
				}
			}
		}(i)
	}

	// Wait for completion
	wg.Wait()
	metrics.EndTime = time.Now()

	// Calculate final metrics
	duration := metrics.EndTime.Sub(metrics.StartTime)
	metrics.TotalOrders = atomic.LoadInt64(&orderCounter)
	metrics.SuccessfulOrders = metrics.TotalOrders - atomic.LoadInt64(&errorCounter)
	metrics.FailedOrders = atomic.LoadInt64(&errorCounter)
	metrics.TotalTrades = atomic.LoadInt64(&tradeCounter)
	metrics.OrdersPerSecond = float64(metrics.TotalOrders) / duration.Seconds()
	metrics.TradesPerSecond = float64(metrics.TotalTrades) / duration.Seconds()
	metrics.ErrorRate = float64(metrics.FailedOrders) / float64(metrics.TotalOrders)

	// Calculate latency statistics
	if len(latencies) > 0 {
		metrics.AvgLatency = calculateAvgLatency(latencies)
		metrics.MinLatency = calculateMinLatency(latencies)
		metrics.MaxLatency = calculateMaxLatency(latencies)
		metrics.P95Latency = calculatePercentileLatency(latencies, 0.95)
		metrics.P99Latency = calculatePercentileLatency(latencies, 0.99)
	}

	return &LoadTestResult{
		Config:  config,
		Metrics: *metrics,
	}
}

// Helper functions

func reportMetrics(t *testing.T, metrics *LoadTestMetrics, interval, totalDuration time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	
	startTime := time.Now()
	
	for {
		select {
		case <-ticker.C:
			elapsed := time.Since(startTime)
			if elapsed >= totalDuration {
				return
			}
			
			currentOrders := atomic.LoadInt64(&metrics.TotalOrders)
			currentTrades := atomic.LoadInt64(&metrics.TotalTrades)
			currentErrors := atomic.LoadInt64(&metrics.FailedOrders)
			
			ordersPerSec := float64(currentOrders) / elapsed.Seconds()
			tradesPerSec := float64(currentTrades) / elapsed.Seconds()
			errorRate := float64(currentErrors) / float64(currentOrders) * 100
			
			t.Logf("[%v] Orders: %d (%.0f/s), Trades: %d (%.0f/s), Errors: %.1f%%, Memory: %s",
				elapsed.Round(time.Second),
				currentOrders, ordersPerSec,
				currentTrades, tradesPerSec,
				errorRate,
				formatMemoryUsage())
		}
	}
}

func generateSymbols(count int) []string {
	symbols := make([]string, count)
	baseSymbols := []string{"AAPL", "GOOGL", "MSFT", "AMZN", "TSLA", "META", "NVDA", "NFLX", "ADBE", "CRM"}
	
	for i := 0; i < count; i++ {
		if i < len(baseSymbols) {
			symbols[i] = baseSymbols[i]
		} else {
			symbols[i] = fmt.Sprintf("SYM%d", i)
		}
	}
	
	return symbols
}

func formatMemoryUsage() string {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return fmt.Sprintf("%.1fMB", float64(m.Alloc)/1024/1024)
}

func calculateAvgLatency(latencies []time.Duration) time.Duration {
	var total time.Duration
	for _, latency := range latencies {
		total += latency
	}
	return total / time.Duration(len(latencies))
}

func calculateMinLatency(latencies []time.Duration) time.Duration {
	min := latencies[0]
	for _, latency := range latencies {
		if latency < min {
			min = latency
		}
	}
	return min
}

func calculateMaxLatency(latencies []time.Duration) time.Duration {
	max := latencies[0]
	for _, latency := range latencies {
		if latency > max {
			max = latency
		}
	}
	return max
}

func calculatePercentileLatency(latencies []time.Duration, percentile float64) time.Duration {
	// Simple percentile calculation - in production, use a proper algorithm
	sorted := make([]time.Duration, len(latencies))
	copy(sorted, latencies)
	
	// Simple bubble sort for demonstration
	for i := 0; i < len(sorted); i++ {
		for j := 0; j < len(sorted)-1-i; j++ {
			if sorted[j] > sorted[j+1] {
				sorted[j], sorted[j+1] = sorted[j+1], sorted[j]
			}
		}
	}
	
	index := int(float64(len(sorted)) * percentile)
	if index >= len(sorted) {
		index = len(sorted) - 1
	}
	
	return sorted[index]
}
