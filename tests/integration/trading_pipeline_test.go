package integration

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/abdoElHodaky/tradSys/internal/trading/execution"
	"github.com/abdoElHodaky/tradSys/internal/trading/order_matching"
	"github.com/abdoElHodaky/tradSys/internal/trading/positions"
	"github.com/abdoElHodaky/tradSys/internal/trading/price_levels"
	"github.com/abdoElHodaky/tradSys/internal/trading/settlement"
	"github.com/abdoElHodaky/tradSys/internal/risk"
)

// TestTradingPipelineIntegration tests the complete trading pipeline
func TestTradingPipelineIntegration(t *testing.T) {
	logger := zap.NewNop()
	
	// Initialize components
	priceLevelManager := price_levels.NewPriceLevelManager(logger)
	executionEngine := execution.NewExecutionEngine(logger)
	settlementProcessor := settlement.NewSettlementProcessor(logger, 2)
	positionManager := positions.NewPositionManager(logger)
	riskEngine := risk.NewRiskEngine(logger)
	
	// Set up test data
	symbol := "BTCUSD"
	userID1 := "user1"
	userID2 := "user2"
	
	// Set up risk limits
	riskLimits := &risk.RiskLimits{
		MaxPositionSize: 10.0,
		MaxOrderSize:    5.0,
		MaxDailyVolume:  1000.0,
		MaxDrawdown:     0.1,
		MaxLeverage:     10.0,
		VaRLimit:        100.0,
		ConcentrationLimit: 0.3,
	}
	riskEngine.SetUserLimits(userID1, riskLimits)
	riskEngine.SetUserLimits(userID2, riskLimits)
	
	// Update market data
	priceLevelManager.UpdatePriceLevel(symbol, "buy", 50000.0, 1.0)
	priceLevelManager.UpdatePriceLevel(symbol, "sell", 50100.0, 1.0)
	riskEngine.UpdateMarketData(symbol, 50050.0, 0.02)
	
	// Create test orders
	buyOrder := &execution.Order{
		ID:        "buy_order_1",
		Symbol:    symbol,
		Side:      "buy",
		Type:      "limit",
		Quantity:  1.0,
		Price:     50050.0,
		Status:    "new",
		CreatedAt: time.Now(),
		UserID:    userID1,
	}
	
	sellOrder := &execution.Order{
		ID:        "sell_order_1",
		Symbol:    symbol,
		Side:      "sell",
		Type:      "limit",
		Quantity:  1.0,
		Price:     50050.0,
		Status:    "new",
		CreatedAt: time.Now(),
		UserID:    userID2,
	}
	
	// Test 1: Risk check
	t.Run("RiskCheck", func(t *testing.T) {
		ctx := context.Background()
		
		buyOrderRisk := &risk.OrderRiskCheck{
			UserID:       buyOrder.UserID,
			Symbol:       buyOrder.Symbol,
			Side:         buyOrder.Side,
			Quantity:     buyOrder.Quantity,
			Price:        buyOrder.Price,
			OrderType:    buyOrder.Type,
			Value:        buyOrder.Price * buyOrder.Quantity,
			CurrentPrice: 50050.0,
		}
		
		result, err := riskEngine.CheckOrderRisk(ctx, buyOrderRisk)
		require.NoError(t, err)
		assert.True(t, result.Passed, "Buy order should pass risk check")
		
		sellOrderRisk := &risk.OrderRiskCheck{
			UserID:       sellOrder.UserID,
			Symbol:       sellOrder.Symbol,
			Side:         sellOrder.Side,
			Quantity:     sellOrder.Quantity,
			Price:        sellOrder.Price,
			OrderType:    sellOrder.Type,
			Value:        sellOrder.Price * sellOrder.Quantity,
			CurrentPrice: 50050.0,
		}
		
		result, err = riskEngine.CheckOrderRisk(ctx, sellOrderRisk)
		require.NoError(t, err)
		assert.True(t, result.Passed, "Sell order should pass risk check")
	})
	
	// Test 2: Trade execution
	t.Run("TradeExecution", func(t *testing.T) {
		ctx := context.Background()
		
		result, err := executionEngine.ExecuteTrade(ctx, buyOrder, sellOrder, 50050.0)
		require.NoError(t, err)
		assert.True(t, result.Success, "Trade execution should succeed")
		assert.NotNil(t, result.Trade, "Trade should be created")
		assert.Equal(t, 1.0, result.Trade.Quantity, "Trade quantity should match")
		assert.Equal(t, 50050.0, result.Trade.Price, "Trade price should match")
		assert.Less(t, result.LatencyNs, int64(100*time.Microsecond), "Execution latency should be < 100μs")
	})
	
	// Test 3: Settlement
	t.Run("Settlement", func(t *testing.T) {
		// Get the executed trade
		trades := executionEngine.GetTradesBySymbol(symbol)
		require.Len(t, trades, 1, "Should have one executed trade")
		
		trade := trades[0]
		
		settlement, err := settlementProcessor.SubmitSettlement(
			trade.ID, trade.Symbol, userID1, userID2,
			trade.Quantity, trade.Price, trade.Fees, trade.Commission,
		)
		require.NoError(t, err)
		assert.NotNil(t, settlement, "Settlement should be created")
		assert.Equal(t, trade.ID, settlement.TradeID, "Settlement should reference correct trade")
		
		// Wait for settlement processing
		time.Sleep(200 * time.Millisecond)
		
		// Check settlement status
		processedSettlement, exists := settlementProcessor.GetSettlement(settlement.ID)
		require.True(t, exists, "Settlement should exist")
		assert.Equal(t, settlement.SettlementStatusSettled, processedSettlement.Status, "Settlement should be completed")
	})
	
	// Test 4: Position updates
	t.Run("PositionUpdates", func(t *testing.T) {
		// Update positions based on the trade
		buyerUpdate := &positions.PositionUpdate{
			UserID:    userID1,
			Symbol:    symbol,
			Quantity:  1.0, // Long position
			Price:     50050.0,
			Timestamp: time.Now(),
			TradeID:   "trade_1",
		}
		
		sellerUpdate := &positions.PositionUpdate{
			UserID:    userID2,
			Symbol:    symbol,
			Quantity:  -1.0, // Short position
			Price:     50050.0,
			Timestamp: time.Now(),
			TradeID:   "trade_1",
		}
		
		err := positionManager.UpdatePosition(buyerUpdate)
		require.NoError(t, err)
		
		err = positionManager.UpdatePosition(sellerUpdate)
		require.NoError(t, err)
		
		// Check buyer position
		buyerPosition, exists := positionManager.GetPosition(userID1, symbol)
		require.True(t, exists, "Buyer position should exist")
		assert.Equal(t, 1.0, buyerPosition.Quantity, "Buyer should have long position")
		assert.Equal(t, 50050.0, buyerPosition.AvgPrice, "Buyer average price should be correct")
		
		// Check seller position
		sellerPosition, exists := positionManager.GetPosition(userID2, symbol)
		require.True(t, exists, "Seller position should exist")
		assert.Equal(t, -1.0, sellerPosition.Quantity, "Seller should have short position")
		assert.Equal(t, 50050.0, sellerPosition.AvgPrice, "Seller average price should be correct")
	})
	
	// Test 5: Performance metrics
	t.Run("PerformanceMetrics", func(t *testing.T) {
		// Check execution engine metrics
		execMetrics := executionEngine.GetPerformanceMetrics()
		assert.Greater(t, execMetrics["total_executions"].(int64), int64(0), "Should have executed trades")
		assert.Equal(t, 1.0, execMetrics["success_rate"].(float64), "Success rate should be 100%")
		
		// Check settlement processor metrics
		settlementMetrics := settlementProcessor.GetPerformanceMetrics()
		assert.Greater(t, settlementMetrics["total_settlements"].(int64), int64(0), "Should have processed settlements")
		
		// Check position manager metrics
		positionMetrics := positionManager.GetPerformanceMetrics()
		assert.Greater(t, positionMetrics["total_positions"].(int64), int64(0), "Should have positions")
		
		// Check risk engine metrics
		riskMetrics := riskEngine.GetPerformanceMetrics()
		assert.Greater(t, riskMetrics["total_checks"].(int64), int64(0), "Should have performed risk checks")
	})
	
	// Cleanup
	defer func() {
		settlementProcessor.Shutdown(5 * time.Second)
	}()
}

// BenchmarkTradingPipeline benchmarks the complete trading pipeline
func BenchmarkTradingPipeline(b *testing.B) {
	logger := zap.NewNop()
	
	// Initialize components
	executionEngine := execution.NewExecutionEngine(logger)
	riskEngine := risk.NewRiskEngine(logger)
	
	// Set up test data
	symbol := "BTCUSD"
	userID1 := "user1"
	userID2 := "user2"
	
	// Set up risk limits
	riskLimits := &risk.RiskLimits{
		MaxPositionSize: 1000.0,
		MaxOrderSize:    100.0,
		MaxDailyVolume:  100000.0,
		MaxDrawdown:     0.1,
		MaxLeverage:     10.0,
		VaRLimit:        1000.0,
		ConcentrationLimit: 0.3,
	}
	riskEngine.SetUserLimits(userID1, riskLimits)
	riskEngine.SetUserLimits(userID2, riskLimits)
	riskEngine.UpdateMarketData(symbol, 50000.0, 0.02)
	
	b.ResetTimer()
	b.ReportAllocs()
	
	b.RunParallel(func(pb *testing.PB) {
		orderID := 0
		for pb.Next() {
			orderID++
			
			// Create orders
			buyOrder := &execution.Order{
				ID:        fmt.Sprintf("buy_order_%d", orderID),
				Symbol:    symbol,
				Side:      "buy",
				Type:      "limit",
				Quantity:  1.0,
				Price:     50000.0,
				Status:    "new",
				CreatedAt: time.Now(),
				UserID:    userID1,
			}
			
			sellOrder := &execution.Order{
				ID:        fmt.Sprintf("sell_order_%d", orderID),
				Symbol:    symbol,
				Side:      "sell",
				Type:      "limit",
				Quantity:  1.0,
				Price:     50000.0,
				Status:    "new",
				CreatedAt: time.Now(),
				UserID:    userID2,
			}
			
			// Risk check
			ctx := context.Background()
			buyOrderRisk := &risk.OrderRiskCheck{
				UserID:       buyOrder.UserID,
				Symbol:       buyOrder.Symbol,
				Side:         buyOrder.Side,
				Quantity:     buyOrder.Quantity,
				Price:        buyOrder.Price,
				OrderType:    buyOrder.Type,
				Value:        buyOrder.Price * buyOrder.Quantity,
				CurrentPrice: 50000.0,
			}
			
			result, err := riskEngine.CheckOrderRisk(ctx, buyOrderRisk)
			if err != nil || !result.Passed {
				b.Error("Risk check failed")
				continue
			}
			
			// Execute trade
			start := time.Now()
			execResult, err := executionEngine.ExecuteTrade(ctx, buyOrder, sellOrder, 50000.0)
			latency := time.Since(start)
			
			if err != nil || !execResult.Success {
				b.Error("Trade execution failed")
				continue
			}
			
			// Check latency target
			if latency > 100*time.Microsecond {
				b.Errorf("Latency %v exceeds target 100μs", latency)
			}
		}
	})
}

