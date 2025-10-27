package integration

import (
	"context"
	"testing"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/matching"
	"github.com/abdoElHodaky/tradSys/internal/services"
	"github.com/abdoElHodaky/tradSys/pkg/config"
	"github.com/abdoElHodaky/tradSys/pkg/interfaces"
	"github.com/abdoElHodaky/tradSys/pkg/testing"
	"github.com/abdoElHodaky/tradSys/pkg/types"
)

// TradingSystemIntegrationTest tests the complete trading flow
func TestTradingSystemIntegration(t *testing.T) {
	// Setup test environment
	testSuite := testing.NewTestSuite()
	ctx := context.Background()

	// Create configuration
	cfg := config.DefaultConfig()
	cfg.Matching.EngineType = "unified"
	cfg.Matching.MaxOrdersPerSymbol = 1000
	cfg.Matching.WorkerCount = 2

	// Initialize service registry
	registry := services.NewServiceRegistry(
		cfg,
		testSuite.GetLogger(),
		testSuite.GetMetrics(),
		testSuite.GetPublisher(),
	)

	// Initialize services
	err := registry.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize service registry: %v", err)
	}

	// Start services
	err = registry.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start services: %v", err)
	}
	defer registry.Stop(ctx)

	// Run integration tests
	t.Run("OrderProcessingFlow", func(t *testing.T) {
		testOrderProcessingFlow(t, registry, testSuite)
	})

	t.Run("MarketDataFlow", func(t *testing.T) {
		testMarketDataFlow(t, registry, testSuite)
	})

	t.Run("TradeExecutionFlow", func(t *testing.T) {
		testTradeExecutionFlow(t, registry, testSuite)
	})

	t.Run("PerformanceUnderLoad", func(t *testing.T) {
		testPerformanceUnderLoad(t, registry, testSuite)
	})
}

func testOrderProcessingFlow(t *testing.T, registry *services.ServiceRegistry, testSuite *testing.TestSuite) {
	ctx := context.Background()
	generator := testSuite.GetGenerator()

	// Get services
	matchingEngine := registry.GetMatchingEngine()
	marketDataService := registry.GetMarketDataService()

	if matchingEngine == nil {
		t.Skip("Matching engine not available")
	}

	if marketDataService == nil {
		t.Skip("Market data service not available")
	}

	// Add test symbol
	symbol := &types.Symbol{
		Symbol:      "BTCUSD",
		BaseAsset:   "BTC",
		QuoteAsset:  "USD",
		Status:      "TRADING",
		MinPrice:    0.01,
		MaxPrice:    100000.0,
		TickSize:    0.01,
		MinQuantity: 0.001,
		MaxQuantity: 1000.0,
		StepSize:    0.001,
		MinNotional: 10.0,
	}

	if mds, ok := marketDataService.(*services.MarketDataService); ok {
		err := mds.AddSymbol(symbol)
		if err != nil {
			t.Fatalf("Failed to add symbol: %v", err)
		}
	}

	// Test order creation and processing
	order := generator.GenerateOrder()
	order.Symbol = "BTCUSD"
	order.Type = types.OrderTypeLimit
	order.Price = 50000.0
	order.Quantity = 1.0

	// Process order through matching engine
	trades, err := matchingEngine.ProcessOrder(ctx, order)
	if err != nil {
		t.Errorf("Failed to process order: %v", err)
	}

	// Verify order was processed
	if order.Status != types.OrderStatusPending {
		t.Errorf("Expected order status to be pending, got %s", order.Status)
	}

	// Verify no trades generated (no matching orders)
	if len(trades) != 0 {
		t.Errorf("Expected no trades, got %d", len(trades))
	}

	// Verify events were published
	orderEvents := testSuite.GetPublisher().(*testing.MockEventPublisher).GetOrderEvents()
	if len(orderEvents) == 0 {
		t.Error("Expected order events to be published")
	}

	// Verify metrics were recorded
	if testSuite.GetMetrics().GetCounter("matching_engine.orders_processed", nil) == 0 {
		t.Error("Expected order processing metrics to be recorded")
	}
}

func testMarketDataFlow(t *testing.T, registry *services.ServiceRegistry, testSuite *testing.TestSuite) {
	ctx := context.Background()
	generator := testSuite.GetGenerator()

	marketDataService := registry.GetMarketDataService()
	if marketDataService == nil {
		t.Skip("Market data service not available")
	}

	// Test market data updates
	marketData := generator.GenerateMarketData("BTCUSD")

	if mds, ok := marketDataService.(*services.MarketDataService); ok {
		err := mds.UpdateMarketData("BTCUSD", marketData)
		if err != nil {
			t.Errorf("Failed to update market data: %v", err)
		}

		// Retrieve market data
		retrievedData, err := mds.GetMarketData(ctx, "BTCUSD")
		if err != nil {
			t.Errorf("Failed to get market data: %v", err)
		}

		if retrievedData.Symbol != "BTCUSD" {
			t.Errorf("Expected symbol BTCUSD, got %s", retrievedData.Symbol)
		}

		// Test OHLCV data
		ohlcv := generator.GenerateOHLCV("BTCUSD", "1m")
		err = mds.UpdateOHLCV("BTCUSD", "1m", ohlcv)
		if err != nil {
			t.Errorf("Failed to update OHLCV data: %v", err)
		}

		// Retrieve OHLCV data
		ohlcvData, err := mds.GetOHLCV(ctx, "BTCUSD", "1m", 10)
		if err != nil {
			t.Errorf("Failed to get OHLCV data: %v", err)
		}

		if len(ohlcvData) == 0 {
			t.Error("Expected OHLCV data to be returned")
		}
	}

	// Verify market data events were published
	marketDataEvents := testSuite.GetPublisher().(*testing.MockEventPublisher).GetMarketDataEvents()
	if len(marketDataEvents) == 0 {
		t.Error("Expected market data events to be published")
	}
}

func testTradeExecutionFlow(t *testing.T, registry *services.ServiceRegistry, testSuite *testing.TestSuite) {
	ctx := context.Background()
	generator := testSuite.GetGenerator()

	tradeService := registry.GetTradeService()
	if tradeService == nil {
		t.Skip("Trade service not available")
	}

	// Create a test trade
	trade := generator.GenerateTrade()
	trade.Symbol = "BTCUSD"

	// Process trade
	if ts, ok := tradeService.(*services.TradeServiceUnified); ok {
		err := ts.CreateTrade(ctx, trade)
		if err != nil {
			t.Errorf("Failed to create trade: %v", err)
		}

		// Retrieve trade
		retrievedTrade, err := ts.GetTrade(ctx, trade.ID)
		if err != nil {
			t.Errorf("Failed to get trade: %v", err)
		}

		if retrievedTrade.ID != trade.ID {
			t.Errorf("Expected trade ID %s, got %s", trade.ID, retrievedTrade.ID)
		}

		// Test trade statistics
		filters := &interfaces.TradeFilters{
			Symbol: "BTCUSD",
			Limit:  10,
		}

		trades, err := ts.ListTrades(ctx, filters)
		if err != nil {
			t.Errorf("Failed to list trades: %v", err)
		}

		if len(trades) == 0 {
			t.Error("Expected trades to be returned")
		}
	}

	// Verify trade events were published
	tradeEvents := testSuite.GetPublisher().(*testing.MockEventPublisher).GetTradeEvents()
	if len(tradeEvents) == 0 {
		t.Error("Expected trade events to be published")
	}
}

func testPerformanceUnderLoad(t *testing.T, registry *services.ServiceRegistry, testSuite *testing.TestSuite) {
	ctx := context.Background()
	generator := testSuite.GetGenerator()

	matchingEngine := registry.GetMatchingEngine()
	if matchingEngine == nil {
		t.Skip("Matching engine not available")
	}

	// Create load test runner
	loadTestRunner := testing.NewLoadTestRunner(
		10,            // 10 concurrent workers
		5*time.Second, // 5 second duration
		1*time.Second, // 1 second ramp-up
	)

	// Define test function
	testFunc := func() error {
		order := generator.GenerateOrder()
		order.Symbol = "BTCUSD"
		order.Type = types.OrderTypeLimit
		order.Price = 50000.0 + (generator.GenerateOrder().Price-50000.0)*0.1 // Small price variation

		_, err := matchingEngine.ProcessOrder(ctx, order)
		return err
	}

	// Run load test
	results := loadTestRunner.Run(testFunc)

	// Verify performance metrics
	if results.TotalRequests == 0 {
		t.Error("Expected requests to be processed during load test")
	}

	if results.ErrorCount > results.TotalRequests/10 { // Allow up to 10% errors
		t.Errorf("Too many errors during load test: %d/%d", results.ErrorCount, results.TotalRequests)
	}

	if results.RPS < 100 { // Expect at least 100 RPS
		t.Errorf("Performance below expectations: %.2f RPS", results.RPS)
	}

	if results.AvgLatency > 10*time.Millisecond { // Expect average latency under 10ms
		t.Errorf("Latency above expectations: %v", results.AvgLatency)
	}

	t.Logf("Load test results: %d requests, %.2f RPS, %v avg latency, %d errors",
		results.TotalRequests, results.RPS, results.AvgLatency, results.ErrorCount)
}

// TestServiceHealthChecks tests the health checking functionality
func TestServiceHealthChecks(t *testing.T) {
	testSuite := testing.NewTestSuite()
	ctx := context.WithValue(context.Background(), "timestamp", time.Now())

	cfg := config.DefaultConfig()
	registry := services.NewServiceRegistry(
		cfg,
		testSuite.GetLogger(),
		testSuite.GetMetrics(),
		testSuite.GetPublisher(),
	)

	err := registry.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize service registry: %v", err)
	}

	err = registry.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start services: %v", err)
	}
	defer registry.Stop(ctx)

	// Test health status
	healthStatus := registry.GetHealthStatus(ctx)
	if healthStatus == nil {
		t.Error("Expected health status to be returned")
	}

	// Test service statistics
	stats := registry.GetServiceStatistics()
	if stats == nil {
		t.Error("Expected service statistics to be returned")
	}

	if !stats.Started {
		t.Error("Expected services to be started")
	}
}

// TestConcurrentOperations tests concurrent access to services
func TestConcurrentOperations(t *testing.T) {
	testSuite := testing.NewTestSuite()
	ctx := context.Background()

	cfg := config.DefaultConfig()
	registry := services.NewServiceRegistry(
		cfg,
		testSuite.GetLogger(),
		testSuite.GetMetrics(),
		testSuite.GetPublisher(),
	)

	err := registry.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize service registry: %v", err)
	}

	err = registry.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start services: %v", err)
	}
	defer registry.Stop(ctx)

	matchingEngine := registry.GetMatchingEngine()
	if matchingEngine == nil {
		t.Skip("Matching engine not available")
	}

	generator := testSuite.GetGenerator()
	const numWorkers = 5
	const ordersPerWorker = 10

	// Run concurrent operations
	errChan := make(chan error, numWorkers)
	for i := 0; i < numWorkers; i++ {
		go func(workerID int) {
			for j := 0; j < ordersPerWorker; j++ {
				order := generator.GenerateOrder()
				order.Symbol = "BTCUSD"
				order.Type = types.OrderTypeLimit
				order.Price = 50000.0

				_, err := matchingEngine.ProcessOrder(ctx, order)
				if err != nil {
					errChan <- err
					return
				}
			}
			errChan <- nil
		}(i)
	}

	// Collect results
	errorCount := 0
	for i := 0; i < numWorkers; i++ {
		if err := <-errChan; err != nil {
			errorCount++
			t.Errorf("Worker error: %v", err)
		}
	}

	if errorCount > 0 {
		t.Errorf("Concurrent operations failed with %d errors", errorCount)
	}

	// Verify metrics
	totalOrders := numWorkers * ordersPerWorker
	processedOrders := testSuite.GetMetrics().GetCounter("matching_engine.orders_processed", nil)

	if int(processedOrders) != totalOrders {
		t.Errorf("Expected %d processed orders, got %.0f", totalOrders, processedOrders)
	}
}

// TestErrorHandlingAndRecovery tests error scenarios and recovery
func TestErrorHandlingAndRecovery(t *testing.T) {
	testSuite := testing.NewTestSuite()
	ctx := context.Background()

	cfg := config.DefaultConfig()
	registry := services.NewServiceRegistry(
		cfg,
		testSuite.GetLogger(),
		testSuite.GetMetrics(),
		testSuite.GetPublisher(),
	)

	err := registry.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize service registry: %v", err)
	}

	err = registry.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start services: %v", err)
	}
	defer registry.Stop(ctx)

	matchingEngine := registry.GetMatchingEngine()
	if matchingEngine == nil {
		t.Skip("Matching engine not available")
	}

	// Test invalid order handling
	invalidOrder := &types.Order{
		ID:       "invalid-order",
		Symbol:   "", // Invalid: empty symbol
		Side:     types.OrderSideBuy,
		Type:     types.OrderTypeLimit,
		Price:    -100.0, // Invalid: negative price
		Quantity: 0,      // Invalid: zero quantity
	}

	_, err = matchingEngine.ProcessOrder(ctx, invalidOrder)
	if err == nil {
		t.Error("Expected error for invalid order, but got none")
	}

	// Test order cancellation for non-existent order
	err = matchingEngine.CancelOrder(ctx, "non-existent-order")
	if err == nil {
		t.Error("Expected error for non-existent order cancellation, but got none")
	}

	// Verify error logs were generated
	errorLogs := testSuite.GetLogger().(*testing.MockLogger).GetLogsByLevel("ERROR")
	if len(errorLogs) == 0 {
		t.Error("Expected error logs to be generated for invalid operations")
	}
}
