package performance

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/abdoElHodaky/tradSys/pkg/matching"
	"go.uber.org/zap"
)

// BenchmarkMatchingEngine_SingleThreaded tests single-threaded performance
func BenchmarkMatchingEngine_SingleThreaded(b *testing.B) {
	config := &matching.EngineConfig{
		Symbol:            "AAPL",
		MaxOrderBookDepth: 1000,
		TickSize:          0.01,
		LotSize:           1.0,
	}
	
	logger := zap.NewNop()
	engine, err := matching.NewEngine(matching.EngineTypeAdvanced, config, logger)
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		order := &matching.Order{
			ID:          fmt.Sprintf("order-%d", i),
			UserID:      "user-001",
			Symbol:      "AAPL",
			Side:        matching.SideBuy,
			Type:        matching.TypeLimit,
			Quantity:    100,
			Price:       150.00 + float64(i%1000)*0.01,
			TimeInForce: matching.TimeInForceGTC,
			Timestamp:   time.Now(),
		}

		_, err := engine.ProcessOrder(order)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMatchingEngine_WithMatching tests performance with actual matching
func BenchmarkMatchingEngine_WithMatching(b *testing.B) {
	engine := matching.NewEngine(&matching.Config{
		Symbol:           "AAPL",
		LatencyTargetNS:  100000,
		MaxOrdersPerSec:  100000,
		OrderBookDepth:   1000,
		EnableHFTMode:    true,
		NUMAOptimization: false,
	})

	ctx := context.Background()

	// Pre-populate with buy orders
	for i := 0; i < 1000; i++ {
		buyOrder := &matching.Order{
			ID:          fmt.Sprintf("buy-%d", i),
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

	// Benchmark sell orders that will match
	for i := 0; i < b.N; i++ {
		sellOrder := &matching.Order{
			ID:          fmt.Sprintf("sell-%d", i),
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

// BenchmarkMatchingEngine_MarketOrders tests market order performance
func BenchmarkMatchingEngine_MarketOrders(b *testing.B) {
	engine := matching.NewEngine(&matching.Config{
		Symbol:           "AAPL",
		LatencyTargetNS:  100000,
		MaxOrdersPerSec:  100000,
		OrderBookDepth:   1000,
		EnableHFTMode:    true,
		NUMAOptimization: false,
	})

	ctx := context.Background()

	// Pre-populate with sell orders at different prices
	for i := 0; i < 1000; i++ {
		sellOrder := &matching.Order{
			ID:          fmt.Sprintf("sell-%d", i),
			UserID:      "user-sell",
			Symbol:      "AAPL",
			Side:        matching.SideSell,
			Type:        matching.TypeLimit,
			Quantity:    100,
			Price:       150.00 + float64(i)*0.01,
			TimeInForce: matching.TimeInForceGTC,
			Timestamp:   time.Now(),
		}
		engine.ProcessOrder(ctx, sellOrder)
	}

	b.ResetTimer()
	b.ReportAllocs()

	// Benchmark market buy orders
	for i := 0; i < b.N; i++ {
		marketOrder := &matching.Order{
			ID:          fmt.Sprintf("market-%d", i),
			UserID:      "user-market",
			Symbol:      "AAPL",
			Side:        matching.SideBuy,
			Type:        matching.TypeMarket,
			Quantity:    100,
			TimeInForce: matching.TimeInForceIOC,
			Timestamp:   time.Now(),
		}

		_, err := engine.ProcessOrder(ctx, marketOrder)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMatchingEngine_Concurrent tests concurrent order processing
func BenchmarkMatchingEngine_Concurrent(b *testing.B) {
	engine := matching.NewEngine(&matching.Config{
		Symbol:           "AAPL",
		LatencyTargetNS:  100000,
		MaxOrdersPerSec:  100000,
		OrderBookDepth:   1000,
		EnableHFTMode:    true,
		NUMAOptimization: false,
	})

	ctx := context.Background()
	numGoroutines := runtime.NumCPU()

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		orderID := 0
		for pb.Next() {
			order := &matching.Order{
				ID:          fmt.Sprintf("concurrent-%d-%d", runtime.NumGoroutine(), orderID),
				UserID:      fmt.Sprintf("user-%d", runtime.NumGoroutine()),
				Symbol:      "AAPL",
				Side:        matching.SideBuy,
				Type:        matching.TypeLimit,
				Quantity:    100,
				Price:       150.00 + float64(orderID%1000)*0.01,
				TimeInForce: matching.TimeInForceGTC,
				Timestamp:   time.Now(),
			}

			_, err := engine.ProcessOrder(ctx, order)
			if err != nil {
				b.Fatal(err)
			}
			orderID++
		}
	})

	b.Logf("Concurrent benchmark with %d goroutines", numGoroutines)
}

// BenchmarkMatchingEngine_HighVolume tests high-volume scenarios
func BenchmarkMatchingEngine_HighVolume(b *testing.B) {
	engine := matching.NewEngine(&matching.Config{
		Symbol:           "AAPL",
		LatencyTargetNS:  100000,
		MaxOrdersPerSec:  1000000, // 1M orders/sec target
		OrderBookDepth:   10000,   // Larger order book
		EnableHFTMode:    true,
		NUMAOptimization: true, // Enable for high volume
	})

	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	// Test with alternating buy/sell orders to generate matches
	for i := 0; i < b.N; i++ {
		var side matching.Side
		var price float64

		if i%2 == 0 {
			side = matching.SideBuy
			price = 150.00 + float64(i%100)*0.01
		} else {
			side = matching.SideSell
			price = 150.00 + float64((i-1)%100)*0.01 // Match with previous buy
		}

		order := &matching.Order{
			ID:          fmt.Sprintf("hv-%d", i),
			UserID:      fmt.Sprintf("user-%d", i%1000),
			Symbol:      "AAPL",
			Side:        side,
			Type:        matching.TypeLimit,
			Quantity:    100,
			Price:       price,
			TimeInForce: matching.TimeInForceGTC,
			Timestamp:   time.Now(),
		}

		_, err := engine.ProcessOrder(ctx, order)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMatchingEngine_Latency measures order processing latency
func BenchmarkMatchingEngine_Latency(b *testing.B) {
	engine := matching.NewEngine(&matching.Config{
		Symbol:           "AAPL",
		LatencyTargetNS:  100000, // 100μs target
		MaxOrdersPerSec:  100000,
		OrderBookDepth:   1000,
		EnableHFTMode:    true,
		NUMAOptimization: false,
	})

	ctx := context.Background()
	latencies := make([]time.Duration, b.N)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		order := &matching.Order{
			ID:          fmt.Sprintf("latency-%d", i),
			UserID:      "user-001",
			Symbol:      "AAPL",
			Side:        matching.SideBuy,
			Type:        matching.TypeLimit,
			Quantity:    100,
			Price:       150.00 + float64(i%1000)*0.01,
			TimeInForce: matching.TimeInForceGTC,
			Timestamp:   time.Now(),
		}

		start := time.Now()
		_, err := engine.ProcessOrder(ctx, order)
		latency := time.Since(start)

		if err != nil {
			b.Fatal(err)
		}

		latencies[i] = latency
	}

	b.StopTimer()

	// Calculate latency statistics
	var totalLatency time.Duration
	var maxLatency time.Duration
	var minLatency time.Duration = time.Hour // Initialize to large value

	for _, latency := range latencies {
		totalLatency += latency
		if latency > maxLatency {
			maxLatency = latency
		}
		if latency < minLatency {
			minLatency = latency
		}
	}

	avgLatency := totalLatency / time.Duration(b.N)

	b.Logf("Latency Statistics:")
	b.Logf("  Average: %v", avgLatency)
	b.Logf("  Minimum: %v", minLatency)
	b.Logf("  Maximum: %v", maxLatency)
	b.Logf("  Target:  100μs")

	// Assert latency targets
	if avgLatency > 100*time.Microsecond {
		b.Errorf("Average latency %v exceeds target of 100μs", avgLatency)
	}
}

// BenchmarkMatchingEngine_Throughput measures order throughput
func BenchmarkMatchingEngine_Throughput(b *testing.B) {
	engine := matching.NewEngine(&matching.Config{
		Symbol:           "AAPL",
		LatencyTargetNS:  100000,
		MaxOrdersPerSec:  100000,
		OrderBookDepth:   1000,
		EnableHFTMode:    true,
		NUMAOptimization: false,
	})

	ctx := context.Background()
	orderCount := 100000 // Test with 100k orders

	b.ResetTimer()
	start := time.Now()

	for i := 0; i < orderCount; i++ {
		order := &matching.Order{
			ID:          fmt.Sprintf("throughput-%d", i),
			UserID:      "user-001",
			Symbol:      "AAPL",
			Side:        matching.SideBuy,
			Type:        matching.TypeLimit,
			Quantity:    100,
			Price:       150.00 + float64(i%1000)*0.01,
			TimeInForce: matching.TimeInForceGTC,
			Timestamp:   time.Now(),
		}

		_, err := engine.ProcessOrder(ctx, order)
		if err != nil {
			b.Fatal(err)
		}
	}

	duration := time.Since(start)
	throughput := float64(orderCount) / duration.Seconds()

	b.Logf("Throughput: %.2f orders/second", throughput)
	b.Logf("Duration: %v", duration)
	b.Logf("Target: 100,000 orders/second")

	// Assert throughput target
	if throughput < 100000 {
		b.Errorf("Throughput %.2f orders/sec is below target of 100,000", throughput)
	}
}

// BenchmarkMatchingEngine_MemoryUsage tests memory efficiency
func BenchmarkMatchingEngine_MemoryUsage(b *testing.B) {
	engine := matching.NewEngine(&matching.Config{
		Symbol:           "AAPL",
		LatencyTargetNS:  100000,
		MaxOrdersPerSec:  100000,
		OrderBookDepth:   1000,
		EnableHFTMode:    true,
		NUMAOptimization: false,
	})

	ctx := context.Background()

	// Force garbage collection before starting
	runtime.GC()
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		order := &matching.Order{
			ID:          fmt.Sprintf("memory-%d", i),
			UserID:      "user-001",
			Symbol:      "AAPL",
			Side:        matching.SideBuy,
			Type:        matching.TypeLimit,
			Quantity:    100,
			Price:       150.00 + float64(i%1000)*0.01,
			TimeInForce: matching.TimeInForceGTC,
			Timestamp:   time.Now(),
		}

		_, err := engine.ProcessOrder(ctx, order)
		if err != nil {
			b.Fatal(err)
		}
	}

	b.StopTimer()

	// Measure memory usage after processing
	runtime.GC()
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)

	memoryUsed := m2.Alloc - m1.Alloc
	memoryPerOrder := float64(memoryUsed) / float64(b.N)

	b.Logf("Memory Usage:")
	b.Logf("  Total: %d bytes", memoryUsed)
	b.Logf("  Per Order: %.2f bytes", memoryPerOrder)
	b.Logf("  Allocations: %d", m2.Mallocs-m1.Mallocs)
}

// BenchmarkMatchingEngine_ConcurrentMatching tests concurrent matching performance
func BenchmarkMatchingEngine_ConcurrentMatching(b *testing.B) {
	engine := matching.NewEngine(&matching.Config{
		Symbol:           "AAPL",
		LatencyTargetNS:  100000,
		MaxOrdersPerSec:  100000,
		OrderBookDepth:   1000,
		EnableHFTMode:    true,
		NUMAOptimization: false,
	})

	ctx := context.Background()
	numWorkers := runtime.NumCPU()
	ordersPerWorker := b.N / numWorkers

	b.ResetTimer()
	b.ReportAllocs()

	var wg sync.WaitGroup
	start := time.Now()

	for worker := 0; worker < numWorkers; worker++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for i := 0; i < ordersPerWorker; i++ {
				var side matching.Side
				if (workerID+i)%2 == 0 {
					side = matching.SideBuy
				} else {
					side = matching.SideSell
				}

				order := &matching.Order{
					ID:          fmt.Sprintf("worker-%d-%d", workerID, i),
					UserID:      fmt.Sprintf("user-%d", workerID),
					Symbol:      "AAPL",
					Side:        side,
					Type:        matching.TypeLimit,
					Quantity:    100,
					Price:       150.00 + float64(i%100)*0.01,
					TimeInForce: matching.TimeInForceGTC,
					Timestamp:   time.Now(),
				}

				_, err := engine.ProcessOrder(ctx, order)
				if err != nil {
					b.Error(err)
					return
				}
			}
		}(worker)
	}

	wg.Wait()
	duration := time.Since(start)
	totalOrders := numWorkers * ordersPerWorker
	throughput := float64(totalOrders) / duration.Seconds()

	b.Logf("Concurrent Matching Performance:")
	b.Logf("  Workers: %d", numWorkers)
	b.Logf("  Total Orders: %d", totalOrders)
	b.Logf("  Duration: %v", duration)
	b.Logf("  Throughput: %.2f orders/second", throughput)
}

// BenchmarkMatchingEngine_OrderBookDepth tests performance with different order book depths
func BenchmarkMatchingEngine_OrderBookDepth(b *testing.B) {
	depths := []int{100, 500, 1000, 5000, 10000}

	for _, depth := range depths {
		b.Run(fmt.Sprintf("Depth-%d", depth), func(b *testing.B) {
			engine := matching.NewEngine(&matching.Config{
				Symbol:           "AAPL",
				LatencyTargetNS:  100000,
				MaxOrdersPerSec:  100000,
				OrderBookDepth:   depth,
				EnableHFTMode:    true,
				NUMAOptimization: false,
			})

			ctx := context.Background()

			// Pre-populate order book to specified depth
			for i := 0; i < depth/2; i++ {
				buyOrder := &matching.Order{
					ID:          fmt.Sprintf("buy-depth-%d", i),
					UserID:      "user-buy",
					Symbol:      "AAPL",
					Side:        matching.SideBuy,
					Type:        matching.TypeLimit,
					Quantity:    100,
					Price:       150.00 - float64(i)*0.01,
					TimeInForce: matching.TimeInForceGTC,
					Timestamp:   time.Now(),
				}
				engine.ProcessOrder(ctx, buyOrder)

				sellOrder := &matching.Order{
					ID:          fmt.Sprintf("sell-depth-%d", i),
					UserID:      "user-sell",
					Symbol:      "AAPL",
					Side:        matching.SideSell,
					Type:        matching.TypeLimit,
					Quantity:    100,
					Price:       150.00 + float64(i)*0.01,
					TimeInForce: matching.TimeInForceGTC,
					Timestamp:   time.Now(),
				}
				engine.ProcessOrder(ctx, sellOrder)
			}

			b.ResetTimer()
			b.ReportAllocs()

			// Benchmark order processing with populated book
			for i := 0; i < b.N; i++ {
				order := &matching.Order{
					ID:          fmt.Sprintf("test-%d", i),
					UserID:      "user-test",
					Symbol:      "AAPL",
					Side:        matching.SideBuy,
					Type:        matching.TypeLimit,
					Quantity:    100,
					Price:       150.00 + float64(i%100)*0.01,
					TimeInForce: matching.TimeInForceGTC,
					Timestamp:   time.Now(),
				}

				_, err := engine.ProcessOrder(ctx, order)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
