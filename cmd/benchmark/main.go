package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/risk"
	riskengine "github.com/abdoElHodaky/tradSys/internal/risk/engine"
	"github.com/abdoElHodaky/tradSys/pkg/common"
	"github.com/abdoElHodaky/tradSys/services/exchanges"
	"go.uber.org/zap"
)

// BenchmarkSuite provides comprehensive performance benchmarking
type BenchmarkSuite struct {
	logger *zap.Logger
	ctx    context.Context
}

// NewBenchmarkSuite creates a new benchmark suite
func NewBenchmarkSuite(logger *zap.Logger) *BenchmarkSuite {
	return &BenchmarkSuite{
		logger: logger,
		ctx:    context.Background(),
	}
}

// BenchmarkResult represents the result of a benchmark
type BenchmarkResult struct {
	Name           string        `json:"name"`
	Operations     int64         `json:"operations"`
	Duration       time.Duration `json:"duration"`
	OpsPerSecond   float64       `json:"ops_per_second"`
	AvgLatency     time.Duration `json:"avg_latency"`
	MinLatency     time.Duration `json:"min_latency"`
	MaxLatency     time.Duration `json:"max_latency"`
	P95Latency     time.Duration `json:"p95_latency"`
	P99Latency     time.Duration `json:"p99_latency"`
	MemoryUsage    int64         `json:"memory_usage"`
	AllocationsOps int64         `json:"allocations_ops"`
}

// RunAllBenchmarks runs all performance benchmarks
func (bs *BenchmarkSuite) RunAllBenchmarks() ([]*BenchmarkResult, error) {
	var results []*BenchmarkResult

	bs.logger.Info("Starting comprehensive performance benchmarks")

	// Service Framework Benchmarks
	serviceResults, err := bs.benchmarkServiceFramework()
	if err != nil {
		bs.logger.Error("Service framework benchmark failed", zap.Error(err))
	} else {
		results = append(results, serviceResults...)
	}

	// Risk Engine Benchmarks
	riskResults, err := bs.benchmarkRiskEngine()
	if err != nil {
		bs.logger.Error("Risk engine benchmark failed", zap.Error(err))
	} else {
		results = append(results, riskResults...)
	}

	// Connection Manager Benchmarks
	connResults, err := bs.benchmarkConnectionManager()
	if err != nil {
		bs.logger.Error("Connection manager benchmark failed", zap.Error(err))
	} else {
		results = append(results, connResults...)
	}

	// Concurrent Access Benchmarks
	concurrentResults, err := bs.benchmarkConcurrentAccess()
	if err != nil {
		bs.logger.Error("Concurrent access benchmark failed", zap.Error(err))
	} else {
		results = append(results, concurrentResults...)
	}

	return results, nil
}

// benchmarkServiceFramework benchmarks the service framework
func (bs *BenchmarkSuite) benchmarkServiceFramework() ([]*BenchmarkResult, error) {
	var results []*BenchmarkResult

	// Benchmark service creation
	result := bs.runBenchmark("Service Creation", func() {
		service := common.NewBaseService("test-service", "1.0.0", bs.logger)
		_ = service
	}, 10000)
	results = append(results, result)

	// Benchmark service lifecycle
	result = bs.runBenchmark("Service Lifecycle", func() {
		service := common.NewBaseService("test-service", "1.0.0", bs.logger)
		service.Start(bs.ctx)
		service.Stop(bs.ctx)
	}, 1000)
	results = append(results, result)

	// Benchmark health checks
	service := common.NewBaseService("test-service", "1.0.0", bs.logger)
	service.Start(bs.ctx)
	defer service.Stop(bs.ctx)

	result = bs.runBenchmark("Health Check", func() {
		_ = service.Health()
	}, 100000)
	results = append(results, result)

	return results, nil
}

// benchmarkRiskEngine benchmarks the risk engine components
func (bs *BenchmarkSuite) benchmarkRiskEngine() ([]*BenchmarkResult, error) {
	var results []*BenchmarkResult

	// Setup risk components
	config := &riskengine.EventProcessorConfig{
		BufferSize:    1000,
		WorkerCount:   4,
		BatchSize:     10,
		FlushInterval: 100 * time.Millisecond,
	}

	processor := riskengine.NewEventProcessor(config, bs.logger)
	processor.Start(bs.ctx)
	defer processor.Stop(bs.ctx)

	// Benchmark event processing
	result := bs.runBenchmark("Risk Event Processing", func() {
		event := &riskengine.RiskEvent{
			Type:      riskengine.RiskEventPreTrade,
			OrderID:   "test-order",
			UserID:    "test-user",
			Symbol:    "BTCUSD",
			Quantity:  100.0,
			Price:     50000.0,
			Timestamp: time.Now(),
		}
		processor.SubmitEvent(event)
	}, 50000)
	results = append(results, result)

	// Benchmark limit manager
	limitManager := risk.NewLimitManager(bs.logger)

	// Add some limits
	for i := 0; i < 100; i++ {
		limit := &risk.RiskLimit{
			ID:     fmt.Sprintf("limit-%d", i),
			UserID: "test-user",
			Symbol: "BTCUSD",
			Type:   risk.RiskLimitTypeMaxOrderSize,
			Value:  1000.0,
		}
		limitManager.AddRiskLimit(bs.ctx, limit)
	}

	result = bs.runBenchmark("Risk Limit Check", func() {
		_, _, _ = limitManager.CheckRiskLimit(bs.ctx, "test-user", "BTCUSD", 500.0, "buy")
	}, 10000)
	results = append(results, result)

	return results, nil
}

// benchmarkConnectionManager benchmarks the connection manager
func (bs *BenchmarkSuite) benchmarkConnectionManager() ([]*BenchmarkResult, error) {
	var results []*BenchmarkResult

	// Setup connection manager
	config := &exchanges.ADXConnectionConfig{
		MaxConnections:       10,
		ConnectionTimeout:    30 * time.Second,
		HealthCheckInterval:  10 * time.Second,
		ReconnectInterval:    5 * time.Second,
		MaxReconnectAttempts: 3,
	}

	manager := exchanges.NewADXConnectionManager(config, bs.logger)

	// Benchmark connection health checks
	result := bs.runBenchmark("Connection Health Check", func() {
		_ = manager.GetHealthyConnections()
	}, 10000)
	results = append(results, result)

	// Benchmark metrics collection
	result = bs.runBenchmark("Connection Metrics", func() {
		_ = manager.GetMetrics()
	}, 50000)
	results = append(results, result)

	return results, nil
}

// benchmarkConcurrentAccess benchmarks concurrent access patterns
func (bs *BenchmarkSuite) benchmarkConcurrentAccess() ([]*BenchmarkResult, error) {
	var results []*BenchmarkResult

	// Benchmark concurrent service operations
	service := common.NewBaseService("test-service", "1.0.0", bs.logger)
	service.Start(bs.ctx)
	defer service.Stop(bs.ctx)

	result := bs.runConcurrentBenchmark("Concurrent Health Checks", func() {
		_ = service.Health()
	}, 1000, 10)
	results = append(results, result)

	// Benchmark concurrent risk limit checks
	limitManager := risk.NewLimitManager(bs.logger)
	limit := &risk.RiskLimit{
		ID:     "concurrent-limit",
		UserID: "test-user",
		Symbol: "BTCUSD",
		Type:   risk.RiskLimitTypeMaxOrderSize,
		Value:  1000.0,
	}
	limitManager.AddRiskLimit(bs.ctx, limit)

	result = bs.runConcurrentBenchmark("Concurrent Limit Checks", func() {
		_, _, _ = limitManager.CheckRiskLimit(bs.ctx, "test-user", "BTCUSD", 500.0, "buy")
	}, 1000, 10)
	results = append(results, result)

	return results, nil
}

// runBenchmark runs a single benchmark
func (bs *BenchmarkSuite) runBenchmark(name string, operation func(), iterations int) *BenchmarkResult {
	// Warm up
	for i := 0; i < 100; i++ {
		operation()
	}

	// Collect garbage before benchmark
	runtime.GC()

	var memBefore, memAfter runtime.MemStats
	runtime.ReadMemStats(&memBefore)

	latencies := make([]time.Duration, iterations)

	start := time.Now()
	for i := 0; i < iterations; i++ {
		opStart := time.Now()
		operation()
		latencies[i] = time.Since(opStart)
	}
	duration := time.Since(start)

	runtime.ReadMemStats(&memAfter)

	// Calculate statistics
	opsPerSecond := float64(iterations) / duration.Seconds()
	avgLatency := duration / time.Duration(iterations)

	// Sort latencies for percentiles
	sortLatencies(latencies)
	minLatency := latencies[0]
	maxLatency := latencies[len(latencies)-1]
	p95Latency := latencies[int(float64(len(latencies))*0.95)]
	p99Latency := latencies[int(float64(len(latencies))*0.99)]

	memoryUsage := int64(memAfter.Alloc - memBefore.Alloc)
	allocationsOps := int64(memAfter.Mallocs - memBefore.Mallocs)

	result := &BenchmarkResult{
		Name:           name,
		Operations:     int64(iterations),
		Duration:       duration,
		OpsPerSecond:   opsPerSecond,
		AvgLatency:     avgLatency,
		MinLatency:     minLatency,
		MaxLatency:     maxLatency,
		P95Latency:     p95Latency,
		P99Latency:     p99Latency,
		MemoryUsage:    memoryUsage,
		AllocationsOps: allocationsOps,
	}

	bs.logger.Info("Benchmark completed",
		zap.String("name", name),
		zap.Float64("ops_per_second", opsPerSecond),
		zap.Duration("avg_latency", avgLatency),
		zap.Duration("p95_latency", p95Latency),
	)

	return result
}

// runConcurrentBenchmark runs a concurrent benchmark
func (bs *BenchmarkSuite) runConcurrentBenchmark(name string, operation func(), iterations int, goroutines int) *BenchmarkResult {
	// Warm up
	for i := 0; i < 100; i++ {
		operation()
	}

	runtime.GC()

	var memBefore, memAfter runtime.MemStats
	runtime.ReadMemStats(&memBefore)

	latencies := make([]time.Duration, iterations*goroutines)
	latencyIndex := int64(0)
	var latencyMutex sync.Mutex

	var wg sync.WaitGroup
	start := time.Now()

	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				opStart := time.Now()
				operation()
				latency := time.Since(opStart)

				latencyMutex.Lock()
				latencies[latencyIndex] = latency
				latencyIndex++
				latencyMutex.Unlock()
			}
		}()
	}

	wg.Wait()
	duration := time.Since(start)

	runtime.ReadMemStats(&memAfter)

	totalOps := iterations * goroutines
	opsPerSecond := float64(totalOps) / duration.Seconds()
	avgLatency := duration / time.Duration(totalOps)

	// Sort latencies for percentiles
	sortLatencies(latencies)
	minLatency := latencies[0]
	maxLatency := latencies[len(latencies)-1]
	p95Latency := latencies[int(float64(len(latencies))*0.95)]
	p99Latency := latencies[int(float64(len(latencies))*0.99)]

	memoryUsage := int64(memAfter.Alloc - memBefore.Alloc)
	allocationsOps := int64(memAfter.Mallocs - memBefore.Mallocs)

	result := &BenchmarkResult{
		Name:           name,
		Operations:     int64(totalOps),
		Duration:       duration,
		OpsPerSecond:   opsPerSecond,
		AvgLatency:     avgLatency,
		MinLatency:     minLatency,
		MaxLatency:     maxLatency,
		P95Latency:     p95Latency,
		P99Latency:     p99Latency,
		MemoryUsage:    memoryUsage,
		AllocationsOps: allocationsOps,
	}

	bs.logger.Info("Concurrent benchmark completed",
		zap.String("name", name),
		zap.Int("goroutines", goroutines),
		zap.Float64("ops_per_second", opsPerSecond),
		zap.Duration("avg_latency", avgLatency),
		zap.Duration("p95_latency", p95Latency),
	)

	return result
}

// sortLatencies sorts latencies in ascending order
func sortLatencies(latencies []time.Duration) {
	// Simple bubble sort for small arrays
	n := len(latencies)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if latencies[j] > latencies[j+1] {
				latencies[j], latencies[j+1] = latencies[j+1], latencies[j]
			}
		}
	}
}

// generateReport generates a performance report
func (bs *BenchmarkSuite) generateReport(results []*BenchmarkResult) string {
	report := "# 📊 TradSys v3 Performance Baseline Report\n\n"
	report += fmt.Sprintf("**Generated**: %s\n", time.Now().Format(time.RFC3339))
	report += fmt.Sprintf("**Go Version**: %s\n", runtime.Version())
	report += fmt.Sprintf("**GOMAXPROCS**: %d\n\n", runtime.GOMAXPROCS(0))

	report += "## 🎯 Performance Summary\n\n"
	report += "| Component | Operations/sec | Avg Latency | P95 Latency | P99 Latency |\n"
	report += "|-----------|----------------|-------------|-------------|-------------|\n"

	for _, result := range results {
		report += fmt.Sprintf("| %s | %.0f | %v | %v | %v |\n",
			result.Name,
			result.OpsPerSecond,
			result.AvgLatency,
			result.P95Latency,
			result.P99Latency,
		)
	}

	report += "\n## 📈 Detailed Results\n\n"

	for _, result := range results {
		report += fmt.Sprintf("### %s\n\n", result.Name)
		report += fmt.Sprintf("- **Total Operations**: %d\n", result.Operations)
		report += fmt.Sprintf("- **Duration**: %v\n", result.Duration)
		report += fmt.Sprintf("- **Operations/sec**: %.2f\n", result.OpsPerSecond)
		report += fmt.Sprintf("- **Average Latency**: %v\n", result.AvgLatency)
		report += fmt.Sprintf("- **Min Latency**: %v\n", result.MinLatency)
		report += fmt.Sprintf("- **Max Latency**: %v\n", result.MaxLatency)
		report += fmt.Sprintf("- **P95 Latency**: %v\n", result.P95Latency)
		report += fmt.Sprintf("- **P99 Latency**: %v\n", result.P99Latency)
		report += fmt.Sprintf("- **Memory Usage**: %d bytes\n", result.MemoryUsage)
		report += fmt.Sprintf("- **Allocations**: %d\n\n", result.AllocationsOps)
	}

	report += "## 🎯 Performance Targets\n\n"
	report += "- **Event Processing**: Target <10μs (HFT requirement)\n"
	report += "- **Service Operations**: Target <1ms for lifecycle operations\n"
	report += "- **Risk Checks**: Target <100μs for limit validation\n"
	report += "- **Connection Health**: Target <10μs for health checks\n\n"

	report += "## 📊 Analysis\n\n"

	// Analyze results
	for _, result := range results {
		if result.P95Latency < 10*time.Microsecond {
			report += fmt.Sprintf("✅ **%s**: Excellent performance (P95: %v)\n", result.Name, result.P95Latency)
		} else if result.P95Latency < 100*time.Microsecond {
			report += fmt.Sprintf("🟡 **%s**: Good performance (P95: %v)\n", result.Name, result.P95Latency)
		} else if result.P95Latency < 1*time.Millisecond {
			report += fmt.Sprintf("🟠 **%s**: Acceptable performance (P95: %v)\n", result.Name, result.P95Latency)
		} else {
			report += fmt.Sprintf("🔴 **%s**: Needs optimization (P95: %v)\n", result.Name, result.P95Latency)
		}
	}

	return report
}

func main() {
	var (
		output  = flag.String("output", "PERFORMANCE_BASELINE.md", "Output file for the report")
		verbose = flag.Bool("verbose", false, "Enable verbose logging")
	)
	flag.Parse()

	// Setup logger
	var logger *zap.Logger
	var err error

	if *verbose {
		logger, err = zap.NewDevelopment()
	} else {
		logger, err = zap.NewProduction()
	}

	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Sync()

	// Create benchmark suite
	suite := NewBenchmarkSuite(logger)

	logger.Info("Starting performance baseline benchmarks")

	// Run all benchmarks
	results, err := suite.RunAllBenchmarks()
	if err != nil {
		logger.Fatal("Benchmarks failed", zap.Error(err))
	}

	// Generate report
	report := suite.generateReport(results)

	// Write report to file
	err = os.WriteFile(*output, []byte(report), 0644)
	if err != nil {
		logger.Fatal("Failed to write report", zap.Error(err))
	}

	logger.Info("Performance baseline complete",
		zap.String("report", *output),
		zap.Int("benchmarks", len(results)),
	)
}
