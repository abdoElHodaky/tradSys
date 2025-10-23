package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/abdoElHodaky/tradSys/internal/core/settlement"
	"github.com/abdoElHodaky/tradSys/internal/risk"
	"github.com/abdoElHodaky/tradSys/internal/trading/execution"
	"github.com/abdoElHodaky/tradSys/internal/trading/positions"
	"github.com/abdoElHodaky/tradSys/internal/trading/positions/price_levels"
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
		MaxPositionSize:    10.0,
		MaxOrderSize:       5.0,
		MaxDailyVolume:     1000.0,
		MaxDrawdown:        0.1,
		MaxLeverage:        10.0,
		VaRLimit:           100.0,
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
		if err != nil {
			t.Errorf("Risk check failed: %v", err)
			return
		}
		if !result.Passed {
			t.Error("Buy order should pass risk check")
		}

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
		if err != nil {
			t.Errorf("Risk check failed: %v", err)
			return
		}
		if !result.Passed {
			t.Error("Sell order should pass risk check")
		}
	})

	// Test 2: Trade execution
	t.Run("TradeExecution", func(t *testing.T) {
		ctx := context.Background()

		result, err := executionEngine.ExecuteTrade(ctx, buyOrder, sellOrder, 50050.0)
		if err != nil {
			t.Errorf("Trade execution failed: %v", err)
			return
		}
		if !result.Success {
			t.Error("Trade execution should succeed")
		}
		if result.Trade == nil {
			t.Error("Trade should be created")
		}
		if result.Trade.Quantity != 1.0 {
			t.Errorf("Expected trade quantity 1.0, got %f", result.Trade.Quantity)
		}
		if result.Trade.Price != 50050.0 {
			t.Errorf("Expected trade price 50050.0, got %f", result.Trade.Price)
		}
		if result.LatencyNs >= int64(100*time.Microsecond) {
			t.Errorf("Execution latency %d ns should be < 100μs", result.LatencyNs)
		}
	})

	// Test 3: Settlement
	t.Run("Settlement", func(t *testing.T) {
		// Get the executed trade
		trades := executionEngine.GetTradesBySymbol(symbol)
		if len(trades) != 1 {
			t.Errorf("Expected 1 executed trade, got %d", len(trades))
			return
		}

		trade := trades[0]

		settlement, err := settlementProcessor.SubmitSettlement(
			trade.ID, trade.Symbol, userID1, userID2,
			trade.Quantity, trade.Price, trade.Fees, trade.Commission,
		)
		if err != nil {
			t.Errorf("Settlement submission failed: %v", err)
			return
		}
		if settlement == nil {
			t.Error("Settlement should be created")
			return
		}
		if settlement.TradeID != trade.ID {
			t.Errorf("Expected settlement trade ID %s, got %s", trade.ID, settlement.TradeID)
		}

		// Wait for settlement processing
		time.Sleep(200 * time.Millisecond)

		// Check settlement status
		processedSettlement, exists := settlementProcessor.GetSettlement(settlement.ID)
		if !exists {
			t.Error("Settlement should exist")
			return
		}
		if processedSettlement.Status != settlement.SettlementStatusSettled {
			t.Errorf("Expected settlement status %s, got %s", settlement.SettlementStatusSettled, processedSettlement.Status)
		}
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
		if err != nil {
			t.Errorf("Buyer position update failed: %v", err)
			return
		}

		err = positionManager.UpdatePosition(sellerUpdate)
		if err != nil {
			t.Errorf("Seller position update failed: %v", err)
			return
		}

		// Check buyer position
		buyerPosition, exists := positionManager.GetPosition(userID1, symbol)
		if !exists {
			t.Error("Buyer position should exist")
			return
		}
		if buyerPosition.Quantity != 1.0 {
			t.Errorf("Expected buyer quantity 1.0, got %f", buyerPosition.Quantity)
		}
		if buyerPosition.AvgPrice != 50050.0 {
			t.Errorf("Expected buyer avg price 50050.0, got %f", buyerPosition.AvgPrice)
		}

		// Check seller position
		sellerPosition, exists := positionManager.GetPosition(userID2, symbol)
		if !exists {
			t.Error("Seller position should exist")
			return
		}
		if sellerPosition.Quantity != -1.0 {
			t.Errorf("Expected seller quantity -1.0, got %f", sellerPosition.Quantity)
		}
		if sellerPosition.AvgPrice != 50050.0 {
			t.Errorf("Expected seller avg price 50050.0, got %f", sellerPosition.AvgPrice)
		}
	})

	// Test 5: Performance metrics
	t.Run("PerformanceMetrics", func(t *testing.T) {
		// Check execution engine metrics
		execMetrics := executionEngine.GetPerformanceMetrics()
		if execMetrics["total_executions"].(int64) <= 0 {
			t.Error("Should have executed trades")
		}
		if execMetrics["success_rate"].(float64) != 1.0 {
			t.Errorf("Expected success rate 1.0, got %f", execMetrics["success_rate"].(float64))
		}

		// Check settlement processor metrics
		settlementMetrics := settlementProcessor.GetPerformanceMetrics()
		if settlementMetrics["total_settlements"].(int64) <= 0 {
			t.Error("Should have processed settlements")
		}

		// Check position manager metrics
		positionMetrics := positionManager.GetPerformanceMetrics()
		if positionMetrics["total_positions"].(int64) <= 0 {
			t.Error("Should have positions")
		}

		// Check risk engine metrics
		riskMetrics := riskEngine.GetPerformanceMetrics()
		if riskMetrics["total_checks"].(int64) <= 0 {
			t.Error("Should have performed risk checks")
		}
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
		MaxPositionSize:    1000.0,
		MaxOrderSize:       100.0,
		MaxDailyVolume:     100000.0,
		MaxDrawdown:        0.1,
		MaxLeverage:        10.0,
		VaRLimit:           1000.0,
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
