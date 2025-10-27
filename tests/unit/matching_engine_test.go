package unit

import (
	"context"
	"testing"
	"time"

	"github.com/abdoElHodaky/tradSys/pkg/matching"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMatchingEngine_BasicMatching(t *testing.T) {
	engine := matching.NewEngine(&matching.Config{
		Symbol:           "AAPL",
		LatencyTargetNS:  100000, // 100μs
		MaxOrdersPerSec:  100000,
		OrderBookDepth:   1000,
		EnableHFTMode:    true,
		NUMAOptimization: false, // Disable for testing
	})

	ctx := context.Background()

	// Test buy order
	buyOrder := &matching.Order{
		ID:          "buy-001",
		UserID:      "user-001",
		Symbol:      "AAPL",
		Side:        matching.SideBuy,
		Type:        matching.TypeLimit,
		Quantity:    100,
		Price:       150.50,
		TimeInForce: matching.TimeInForceGTC,
		Timestamp:   time.Now(),
	}

	trades, err := engine.ProcessOrder(ctx, buyOrder)
	require.NoError(t, err)
	assert.Empty(t, trades, "No trades should occur with single buy order")

	// Test sell order that should match
	sellOrder := &matching.Order{
		ID:          "sell-001",
		UserID:      "user-002",
		Symbol:      "AAPL",
		Side:        matching.SideSell,
		Type:        matching.TypeLimit,
		Quantity:    50,
		Price:       150.50,
		TimeInForce: matching.TimeInForceGTC,
		Timestamp:   time.Now(),
	}

	trades, err = engine.ProcessOrder(ctx, sellOrder)
	require.NoError(t, err)
	assert.Len(t, trades, 1, "Should generate one trade")

	trade := trades[0]
	assert.Equal(t, "buy-001", trade.BuyOrderID)
	assert.Equal(t, "sell-001", trade.SellOrderID)
	assert.Equal(t, float64(50), trade.Quantity)
	assert.Equal(t, 150.50, trade.Price)
}

func TestMatchingEngine_PriceTimePriority(t *testing.T) {
	engine := matching.NewEngine(&matching.Config{
		Symbol:          "AAPL",
		LatencyTargetNS: 100000,
		MaxOrdersPerSec: 100000,
		OrderBookDepth:  1000,
		EnableHFTMode:   true,
	})

	ctx := context.Background()

	// Add first buy order at lower price
	buyOrder1 := &matching.Order{
		ID:          "buy-001",
		UserID:      "user-001",
		Symbol:      "AAPL",
		Side:        matching.SideBuy,
		Type:        matching.TypeLimit,
		Quantity:    100,
		Price:       150.00,
		TimeInForce: matching.TimeInForceGTC,
		Timestamp:   time.Now(),
	}

	_, err := engine.ProcessOrder(ctx, buyOrder1)
	require.NoError(t, err)

	// Add second buy order at higher price (should have priority)
	buyOrder2 := &matching.Order{
		ID:          "buy-002",
		UserID:      "user-002",
		Symbol:      "AAPL",
		Side:        matching.SideBuy,
		Type:        matching.TypeLimit,
		Quantity:    100,
		Price:       150.50,
		TimeInForce: matching.TimeInForceGTC,
		Timestamp:   time.Now(),
	}

	_, err = engine.ProcessOrder(ctx, buyOrder2)
	require.NoError(t, err)

	// Add sell order that should match with higher price buy order
	sellOrder := &matching.Order{
		ID:          "sell-001",
		UserID:      "user-003",
		Symbol:      "AAPL",
		Side:        matching.SideSell,
		Type:        matching.TypeLimit,
		Quantity:    50,
		Price:       150.25,
		TimeInForce: matching.TimeInForceGTC,
		Timestamp:   time.Now(),
	}

	trades, err := engine.ProcessOrder(ctx, sellOrder)
	require.NoError(t, err)
	assert.Len(t, trades, 1)

	trade := trades[0]
	assert.Equal(t, "buy-002", trade.BuyOrderID, "Should match with higher price buy order")
	assert.Equal(t, 150.50, trade.Price, "Should execute at buy order price")
}

func TestMatchingEngine_PartialFill(t *testing.T) {
	engine := matching.NewEngine(&matching.Config{
		Symbol:          "AAPL",
		LatencyTargetNS: 100000,
		MaxOrdersPerSec: 100000,
		OrderBookDepth:  1000,
		EnableHFTMode:   true,
	})

	ctx := context.Background()

	// Large buy order
	buyOrder := &matching.Order{
		ID:          "buy-001",
		UserID:      "user-001",
		Symbol:      "AAPL",
		Side:        matching.SideBuy,
		Type:        matching.TypeLimit,
		Quantity:    1000,
		Price:       150.50,
		TimeInForce: matching.TimeInForceGTC,
		Timestamp:   time.Now(),
	}

	_, err := engine.ProcessOrder(ctx, buyOrder)
	require.NoError(t, err)

	// Smaller sell order (partial fill)
	sellOrder := &matching.Order{
		ID:          "sell-001",
		UserID:      "user-002",
		Symbol:      "AAPL",
		Side:        matching.SideSell,
		Type:        matching.TypeLimit,
		Quantity:    300,
		Price:       150.50,
		TimeInForce: matching.TimeInForceGTC,
		Timestamp:   time.Now(),
	}

	trades, err := engine.ProcessOrder(ctx, sellOrder)
	require.NoError(t, err)
	assert.Len(t, trades, 1)

	trade := trades[0]
	assert.Equal(t, float64(300), trade.Quantity)

	// Check remaining quantity in order book
	orderBook := engine.GetOrderBook()
	assert.NotNil(t, orderBook)

	// The buy order should still have 700 remaining
	buyLevels := orderBook.GetBuyLevels()
	assert.NotEmpty(t, buyLevels)

	topBuyLevel := buyLevels[0]
	assert.Equal(t, 150.50, topBuyLevel.Price)
	assert.Equal(t, float64(700), topBuyLevel.Quantity)
}

func TestMatchingEngine_MarketOrder(t *testing.T) {
	engine := matching.NewEngine(&matching.Config{
		Symbol:          "AAPL",
		LatencyTargetNS: 100000,
		MaxOrdersPerSec: 100000,
		OrderBookDepth:  1000,
		EnableHFTMode:   true,
	})

	ctx := context.Background()

	// Add limit sell orders at different prices
	sellOrder1 := &matching.Order{
		ID:          "sell-001",
		UserID:      "user-001",
		Symbol:      "AAPL",
		Side:        matching.SideSell,
		Type:        matching.TypeLimit,
		Quantity:    100,
		Price:       150.50,
		TimeInForce: matching.TimeInForceGTC,
		Timestamp:   time.Now(),
	}

	_, err := engine.ProcessOrder(ctx, sellOrder1)
	require.NoError(t, err)

	sellOrder2 := &matching.Order{
		ID:          "sell-002",
		UserID:      "user-002",
		Symbol:      "AAPL",
		Side:        matching.SideSell,
		Type:        matching.TypeLimit,
		Quantity:    100,
		Price:       150.75,
		TimeInForce: matching.TimeInForceGTC,
		Timestamp:   time.Now(),
	}

	_, err = engine.ProcessOrder(ctx, sellOrder2)
	require.NoError(t, err)

	// Market buy order should match with best ask price
	marketBuyOrder := &matching.Order{
		ID:          "buy-market-001",
		UserID:      "user-003",
		Symbol:      "AAPL",
		Side:        matching.SideBuy,
		Type:        matching.TypeMarket,
		Quantity:    150,
		TimeInForce: matching.TimeInForceIOC,
		Timestamp:   time.Now(),
	}

	trades, err := engine.ProcessOrder(ctx, marketBuyOrder)
	require.NoError(t, err)
	assert.Len(t, trades, 2, "Should generate two trades")

	// First trade should be at 150.50 for 100 shares
	trade1 := trades[0]
	assert.Equal(t, "sell-001", trade1.SellOrderID)
	assert.Equal(t, float64(100), trade1.Quantity)
	assert.Equal(t, 150.50, trade1.Price)

	// Second trade should be at 150.75 for 50 shares
	trade2 := trades[1]
	assert.Equal(t, "sell-002", trade2.SellOrderID)
	assert.Equal(t, float64(50), trade2.Quantity)
	assert.Equal(t, 150.75, trade2.Price)
}

func TestMatchingEngine_OrderCancellation(t *testing.T) {
	engine := matching.NewEngine(&matching.Config{
		Symbol:          "AAPL",
		LatencyTargetNS: 100000,
		MaxOrdersPerSec: 100000,
		OrderBookDepth:  1000,
		EnableHFTMode:   true,
	})

	ctx := context.Background()

	// Add buy order
	buyOrder := &matching.Order{
		ID:          "buy-001",
		UserID:      "user-001",
		Symbol:      "AAPL",
		Side:        matching.SideBuy,
		Type:        matching.TypeLimit,
		Quantity:    100,
		Price:       150.50,
		TimeInForce: matching.TimeInForceGTC,
		Timestamp:   time.Now(),
	}

	_, err := engine.ProcessOrder(ctx, buyOrder)
	require.NoError(t, err)

	// Cancel the order
	err = engine.CancelOrder(ctx, "buy-001", "user-001")
	require.NoError(t, err)

	// Try to match with sell order - should not match
	sellOrder := &matching.Order{
		ID:          "sell-001",
		UserID:      "user-002",
		Symbol:      "AAPL",
		Side:        matching.SideSell,
		Type:        matching.TypeLimit,
		Quantity:    100,
		Price:       150.50,
		TimeInForce: matching.TimeInForceGTC,
		Timestamp:   time.Now(),
	}

	trades, err := engine.ProcessOrder(ctx, sellOrder)
	require.NoError(t, err)
	assert.Empty(t, trades, "No trades should occur after cancellation")
}

func TestMatchingEngine_TimeInForceIOC(t *testing.T) {
	engine := matching.NewEngine(&matching.Config{
		Symbol:          "AAPL",
		LatencyTargetNS: 100000,
		MaxOrdersPerSec: 100000,
		OrderBookDepth:  1000,
		EnableHFTMode:   true,
	})

	ctx := context.Background()

	// Add IOC order that cannot be fully filled
	iocOrder := &matching.Order{
		ID:          "ioc-001",
		UserID:      "user-001",
		Symbol:      "AAPL",
		Side:        matching.SideBuy,
		Type:        matching.TypeLimit,
		Quantity:    1000,
		Price:       150.50,
		TimeInForce: matching.TimeInForceIOC,
		Timestamp:   time.Now(),
	}

	trades, err := engine.ProcessOrder(ctx, iocOrder)
	require.NoError(t, err)
	assert.Empty(t, trades, "IOC order should not generate trades when no matching orders")

	// Order should not remain in book
	orderBook := engine.GetOrderBook()
	buyLevels := orderBook.GetBuyLevels()
	assert.Empty(t, buyLevels, "IOC order should not remain in order book")
}

func TestMatchingEngine_Performance(t *testing.T) {
	engine := matching.NewEngine(&matching.Config{
		Symbol:          "AAPL",
		LatencyTargetNS: 100000, // 100μs target
		MaxOrdersPerSec: 100000,
		OrderBookDepth:  1000,
		EnableHFTMode:   true,
	})

	ctx := context.Background()

	// Measure order processing latency
	start := time.Now()

	order := &matching.Order{
		ID:          "perf-001",
		UserID:      "user-001",
		Symbol:      "AAPL",
		Side:        matching.SideBuy,
		Type:        matching.TypeLimit,
		Quantity:    100,
		Price:       150.50,
		TimeInForce: matching.TimeInForceGTC,
		Timestamp:   time.Now(),
	}

	_, err := engine.ProcessOrder(ctx, order)
	require.NoError(t, err)

	latency := time.Since(start)

	// Assert latency is under target (100μs = 100,000ns)
	assert.Less(t, latency.Nanoseconds(), int64(100000),
		"Order processing should be under 100μs, got %v", latency)
}

func BenchmarkMatchingEngine_ProcessOrder(b *testing.B) {
	engine := matching.NewEngine(&matching.Config{
		Symbol:          "AAPL",
		LatencyTargetNS: 100000,
		MaxOrdersPerSec: 100000,
		OrderBookDepth:  1000,
		EnableHFTMode:   true,
	})

	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		order := &matching.Order{
			ID:          "bench-" + string(rune(i)),
			UserID:      "user-001",
			Symbol:      "AAPL",
			Side:        matching.SideBuy,
			Type:        matching.TypeLimit,
			Quantity:    100,
			Price:       150.50 + float64(i%100)*0.01,
			TimeInForce: matching.TimeInForceGTC,
			Timestamp:   time.Now(),
		}

		_, err := engine.ProcessOrder(ctx, order)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMatchingEngine_MatchingThroughput(b *testing.B) {
	engine := matching.NewEngine(&matching.Config{
		Symbol:          "AAPL",
		LatencyTargetNS: 100000,
		MaxOrdersPerSec: 100000,
		OrderBookDepth:  1000,
		EnableHFTMode:   true,
	})

	ctx := context.Background()

	// Pre-populate order book with buy orders
	for i := 0; i < 1000; i++ {
		buyOrder := &matching.Order{
			ID:          "buy-" + string(rune(i)),
			UserID:      "user-buy",
			Symbol:      "AAPL",
			Side:        matching.SideBuy,
			Type:        matching.TypeLimit,
			Quantity:    100,
			Price:       150.00 + float64(i)*0.01,
			TimeInForce: matching.TimeInForceGTC,
			Timestamp:   time.Now(),
		}
		engine.ProcessOrder(ctx, buyOrder)
	}

	b.ResetTimer()
	b.ReportAllocs()

	// Benchmark matching with sell orders
	for i := 0; i < b.N; i++ {
		sellOrder := &matching.Order{
			ID:          "sell-" + string(rune(i)),
			UserID:      "user-sell",
			Symbol:      "AAPL",
			Side:        matching.SideSell,
			Type:        matching.TypeLimit,
			Quantity:    100,
			Price:       150.00 + float64(i%1000)*0.01,
			TimeInForce: matching.TimeInForceGTC,
			Timestamp:   time.Now(),
		}

		_, err := engine.ProcessOrder(ctx, sellOrder)
		if err != nil {
			b.Fatal(err)
		}
	}
}
