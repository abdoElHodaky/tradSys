package testing

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/hft/metrics"
	"github.com/abdoElHodaky/tradSys/internal/hft/pools"
)

// LoadTestConfig contains load testing configuration
type LoadTestConfig struct {
	// Test parameters
	Duration        time.Duration `yaml:"duration" default:"60s"`
	Concurrency     int           `yaml:"concurrency" default:"100"`
	RequestsPerSec  int           `yaml:"requests_per_sec" default:"1000"`
	RampUpTime      time.Duration `yaml:"ramp_up_time" default:"10s"`
	RampDownTime    time.Duration `yaml:"ramp_down_time" default:"10s"`
	
	// Target configuration
	TargetURL       string `yaml:"target_url" default:"http://localhost:8080"`
	Endpoints       []EndpointConfig `yaml:"endpoints"`
	
	// Test scenarios
	Scenarios       []ScenarioConfig `yaml:"scenarios"`
	
	// Reporting
	ReportInterval  time.Duration `yaml:"report_interval" default:"5s"`
	EnableMetrics   bool          `yaml:"enable_metrics" default:"true"`
	OutputFile      string        `yaml:"output_file" default:"load_test_results.json"`
}

// EndpointConfig contains endpoint-specific configuration
type EndpointConfig struct {
	Path        string            `yaml:"path"`
	Method      string            `yaml:"method" default:"GET"`
	Headers     map[string]string `yaml:"headers"`
	Body        string            `yaml:"body"`
	Weight      int               `yaml:"weight" default:"1"`
	Timeout     time.Duration     `yaml:"timeout" default:"10s"`
}

// ScenarioConfig contains test scenario configuration
type ScenarioConfig struct {
	Name        string  `yaml:"name"`
	Weight      int     `yaml:"weight" default:"1"`
	Endpoints   []string `yaml:"endpoints"`
	UserFlow    []string `yaml:"user_flow"`
	ThinkTime   time.Duration `yaml:"think_time" default:"100ms"`
}

// LoadTestResults contains the results of a load test
type LoadTestResults struct {
	// Test configuration
	Config LoadTestConfig `json:"config"`
	
	// Overall statistics
	TotalRequests     int64         `json:"total_requests"`
	SuccessfulRequests int64        `json:"successful_requests"`
	FailedRequests    int64         `json:"failed_requests"`
	TotalDuration     time.Duration `json:"total_duration"`
	
	// Performance metrics
	AverageLatency    time.Duration `json:"average_latency"`
	P50Latency        time.Duration `json:"p50_latency"`
	P95Latency        time.Duration `json:"p95_latency"`
	P99Latency        time.Duration `json:"p99_latency"`
	MaxLatency        time.Duration `json:"max_latency"`
	MinLatency        time.Duration `json:"min_latency"`
	
	// Throughput metrics
	RequestsPerSecond float64 `json:"requests_per_second"`
	BytesPerSecond    float64 `json:"bytes_per_second"`
	
	// Error statistics
	ErrorRate         float64            `json:"error_rate"`
	ErrorsByType      map[string]int64   `json:"errors_by_type"`
	
	// Endpoint-specific results
	EndpointResults   map[string]*EndpointResults `json:"endpoint_results"`
	
	// Timeline data
	Timeline          []TimelinePoint `json:"timeline"`
	
	Timestamp         time.Time `json:"timestamp"`
}

// EndpointResults contains results for a specific endpoint
type EndpointResults struct {
	Path              string        `json:"path"`
	Method            string        `json:"method"`
	TotalRequests     int64         `json:"total_requests"`
	SuccessfulRequests int64        `json:"successful_requests"`
	FailedRequests    int64         `json:"failed_requests"`
	AverageLatency    time.Duration `json:"average_latency"`
	P95Latency        time.Duration `json:"p95_latency"`
	P99Latency        time.Duration `json:"p99_latency"`
	ErrorRate         float64       `json:"error_rate"`
}

// TimelinePoint represents a point in time during the test
type TimelinePoint struct {
	Timestamp         time.Time     `json:"timestamp"`
	RequestsPerSecond float64       `json:"requests_per_second"`
	AverageLatency    time.Duration `json:"average_latency"`
	ErrorRate         float64       `json:"error_rate"`
	ActiveUsers       int           `json:"active_users"`
}

// HFTLoadTester provides high-frequency trading load testing capabilities
type HFTLoadTester struct {
	config *LoadTestConfig
	
	// Test state
	ctx        context.Context
	cancel     context.CancelFunc
	startTime  time.Time
	
	// Statistics
	totalRequests     int64
	successfulRequests int64
	failedRequests    int64
	totalLatency      int64
	latencies         []time.Duration
	errors            map[string]int64
	endpointStats     map[string]*EndpointResults
	timeline          []TimelinePoint
	
	// Synchronization
	mu sync.RWMutex
	wg sync.WaitGroup
	
	// Pools for efficiency
	requestPool  *pools.WebSocketMessagePool
	responsePool *pools.WebSocketMessagePool
}

// NewHFTLoadTester creates a new HFT load tester
func NewHFTLoadTester(config *LoadTestConfig) *HFTLoadTester {
	if config == nil {
		config = &LoadTestConfig{
			Duration:       60 * time.Second,
			Concurrency:    100,
			RequestsPerSec: 1000,
			RampUpTime:     10 * time.Second,
			RampDownTime:   10 * time.Second,
			TargetURL:      "http://localhost:8080",
			ReportInterval: 5 * time.Second,
			EnableMetrics:  true,
			OutputFile:     "load_test_results.json",
		}
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	return &HFTLoadTester{
		config:        config,
		ctx:           ctx,
		cancel:        cancel,
		errors:        make(map[string]int64),
		endpointStats: make(map[string]*EndpointResults),
		latencies:     make([]time.Duration, 0, 100000), // Pre-allocate for efficiency
		requestPool:   pools.NewWebSocketMessagePool(),
		responsePool:  pools.NewWebSocketMessagePool(),
	}
}

// RunLoadTest executes the load test
func (lt *HFTLoadTester) RunLoadTest() (*LoadTestResults, error) {
	lt.startTime = time.Now()
	
	fmt.Printf("Starting HFT load test...\n")
	fmt.Printf("Duration: %v, Concurrency: %d, Target RPS: %d\n",
		lt.config.Duration, lt.config.Concurrency, lt.config.RequestsPerSec)
	
	// Start reporting goroutine
	if lt.config.EnableMetrics {
		lt.wg.Add(1)
		go lt.reportingLoop()
	}
	
	// Start timeline collection
	lt.wg.Add(1)
	go lt.timelineLoop()
	
	// Execute test phases
	if err := lt.executeTest(); err != nil {
		return nil, fmt.Errorf("test execution failed: %w", err)
	}
	
	// Wait for reporting to finish
	lt.cancel()
	lt.wg.Wait()
	
	// Generate results
	results := lt.generateResults()
	
	fmt.Printf("Load test completed!\n")
	fmt.Printf("Total requests: %d, Success rate: %.2f%%, Average latency: %v\n",
		results.TotalRequests,
		float64(results.SuccessfulRequests)/float64(results.TotalRequests)*100,
		results.AverageLatency)
	
	return results, nil
}

// executeTest executes the main test logic
func (lt *HFTLoadTester) executeTest() error {
	// Calculate phase durations
	totalDuration := lt.config.Duration + lt.config.RampUpTime + lt.config.RampDownTime
	
	// Create test context with timeout
	testCtx, testCancel := context.WithTimeout(lt.ctx, totalDuration)
	defer testCancel()
	
	// Phase 1: Ramp up
	if lt.config.RampUpTime > 0 {
		if err := lt.rampUpPhase(testCtx); err != nil {
			return fmt.Errorf("ramp up phase failed: %w", err)
		}
	}
	
	// Phase 2: Steady state
	if err := lt.steadyStatePhase(testCtx); err != nil {
		return fmt.Errorf("steady state phase failed: %w", err)
	}
	
	// Phase 3: Ramp down
	if lt.config.RampDownTime > 0 {
		if err := lt.rampDownPhase(testCtx); err != nil {
			return fmt.Errorf("ramp down phase failed: %w", err)
		}
	}
	
	return nil
}

// rampUpPhase gradually increases load
func (lt *HFTLoadTester) rampUpPhase(ctx context.Context) error {
	fmt.Printf("Ramp up phase: %v\n", lt.config.RampUpTime)
	
	rampCtx, rampCancel := context.WithTimeout(ctx, lt.config.RampUpTime)
	defer rampCancel()
	
	startConcurrency := 1
	targetConcurrency := lt.config.Concurrency
	
	return lt.executePhase(rampCtx, startConcurrency, targetConcurrency, lt.config.RampUpTime)
}

// steadyStatePhase maintains constant load
func (lt *HFTLoadTester) steadyStatePhase(ctx context.Context) error {
	fmt.Printf("Steady state phase: %v\n", lt.config.Duration)
	
	steadyCtx, steadyCancel := context.WithTimeout(ctx, lt.config.Duration)
	defer steadyCancel()
	
	return lt.executePhase(steadyCtx, lt.config.Concurrency, lt.config.Concurrency, lt.config.Duration)
}

// rampDownPhase gradually decreases load
func (lt *HFTLoadTester) rampDownPhase(ctx context.Context) error {
	fmt.Printf("Ramp down phase: %v\n", lt.config.RampDownTime)
	
	rampCtx, rampCancel := context.WithTimeout(ctx, lt.config.RampDownTime)
	defer rampCancel()
	
	startConcurrency := lt.config.Concurrency
	targetConcurrency := 1
	
	return lt.executePhase(rampCtx, startConcurrency, targetConcurrency, lt.config.RampDownTime)
}

// executePhase executes a test phase with specified concurrency
func (lt *HFTLoadTester) executePhase(ctx context.Context, startConcurrency, targetConcurrency int, duration time.Duration) error {
	// Calculate rate limiting
	requestInterval := time.Second / time.Duration(lt.config.RequestsPerSec)
	
	// Start workers
	workerCtx, workerCancel := context.WithCancel(ctx)
	defer workerCancel()
	
	// Gradually adjust concurrency
	concurrencyStep := float64(targetConcurrency-startConcurrency) / float64(duration/time.Second)
	currentConcurrency := float64(startConcurrency)
	
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	
	activeWorkers := 0
	
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			targetWorkers := int(currentConcurrency)
			
			// Adjust worker count
			if targetWorkers > activeWorkers {
				// Start new workers
				for i := activeWorkers; i < targetWorkers; i++ {
					lt.wg.Add(1)
					go lt.worker(workerCtx, requestInterval)
					activeWorkers++
				}
			}
			
			currentConcurrency += concurrencyStep
			if currentConcurrency > float64(targetConcurrency) {
				currentConcurrency = float64(targetConcurrency)
			}
		}
	}
}

// worker executes requests at the specified rate
func (lt *HFTLoadTester) worker(ctx context.Context, requestInterval time.Duration) {
	defer lt.wg.Done()
	
	ticker := time.NewTicker(requestInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			lt.executeRequest()
		}
	}
}

// executeRequest executes a single request
func (lt *HFTLoadTester) executeRequest() {
	// Select endpoint based on weights
	endpoint := lt.selectEndpoint()
	if endpoint == nil {
		return
	}
	
	start := time.Now()
	
	// Execute request (this would be implemented based on your specific needs)
	success, err := lt.makeRequest(endpoint)
	
	latency := time.Since(start)
	
	// Record statistics
	atomic.AddInt64(&lt.totalRequests, 1)
	atomic.AddInt64(&lt.totalLatency, int64(latency))
	
	if success {
		atomic.AddInt64(&lt.successfulRequests, 1)
	} else {
		atomic.AddInt64(&lt.failedRequests, 1)
		
		// Record error
		errorType := "unknown"
		if err != nil {
			errorType = err.Error()
		}
		
		lt.mu.Lock()
		lt.errors[errorType]++
		lt.mu.Unlock()
	}
	
	// Record latency (with sampling to avoid memory issues)
	if len(lt.latencies) < cap(lt.latencies) {
		lt.mu.Lock()
		lt.latencies = append(lt.latencies, latency)
		lt.mu.Unlock()
	}
	
	// Update endpoint statistics
	lt.updateEndpointStats(endpoint, success, latency)
}

// selectEndpoint selects an endpoint based on weights
func (lt *HFTLoadTester) selectEndpoint() *EndpointConfig {
	if len(lt.config.Endpoints) == 0 {
		return nil
	}
	
	// Simple random selection for now
	// In a real implementation, this would consider weights
	index := rand.Intn(len(lt.config.Endpoints))
	return &lt.config.Endpoints[index]
}

// makeRequest makes an HTTP request to the specified endpoint
func (lt *HFTLoadTester) makeRequest(endpoint *EndpointConfig) (bool, error) {
	// This is a placeholder - implement actual HTTP request logic
	// For HFT testing, you might want to use a custom HTTP client
	// with connection pooling and other optimizations
	
	// Simulate request processing
	time.Sleep(time.Millisecond * time.Duration(rand.Intn(10)+1))
	
	// Simulate success/failure (90% success rate)
	success := rand.Float64() < 0.9
	
	var err error
	if !success {
		err = fmt.Errorf("simulated error")
	}
	
	return success, err
}

// updateEndpointStats updates statistics for a specific endpoint
func (lt *HFTLoadTester) updateEndpointStats(endpoint *EndpointConfig, success bool, latency time.Duration) {
	lt.mu.Lock()
	defer lt.mu.Unlock()
	
	key := fmt.Sprintf("%s %s", endpoint.Method, endpoint.Path)
	stats, exists := lt.endpointStats[key]
	if !exists {
		stats = &EndpointResults{
			Path:   endpoint.Path,
			Method: endpoint.Method,
		}
		lt.endpointStats[key] = stats
	}
	
	stats.TotalRequests++
	if success {
		stats.SuccessfulRequests++
	} else {
		stats.FailedRequests++
	}
	
	// Update latency (simplified - in reality you'd want proper percentile calculation)
	if stats.AverageLatency == 0 {
		stats.AverageLatency = latency
	} else {
		stats.AverageLatency = (stats.AverageLatency + latency) / 2
	}
	
	stats.ErrorRate = float64(stats.FailedRequests) / float64(stats.TotalRequests)
}

// reportingLoop provides periodic progress reports
func (lt *HFTLoadTester) reportingLoop() {
	defer lt.wg.Done()
	
	ticker := time.NewTicker(lt.config.ReportInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-lt.ctx.Done():
			return
		case <-ticker.C:
			lt.printProgress()
		}
	}
}

// timelineLoop collects timeline data
func (lt *HFTLoadTester) timelineLoop() {
	defer lt.wg.Done()
	
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-lt.ctx.Done():
			return
		case <-ticker.C:
			lt.recordTimelinePoint()
		}
	}
}

// recordTimelinePoint records a timeline data point
func (lt *HFTLoadTester) recordTimelinePoint() {
	totalReqs := atomic.LoadInt64(&lt.totalRequests)
	successReqs := atomic.LoadInt64(&lt.successfulRequests)
	failedReqs := atomic.LoadInt64(&lt.failedRequests)
	
	elapsed := time.Since(lt.startTime).Seconds()
	rps := float64(totalReqs) / elapsed
	errorRate := float64(failedReqs) / float64(totalReqs)
	
	avgLatency := time.Duration(0)
	if totalReqs > 0 {
		avgLatency = time.Duration(atomic.LoadInt64(&lt.totalLatency) / totalReqs)
	}
	
	point := TimelinePoint{
		Timestamp:         time.Now(),
		RequestsPerSecond: rps,
		AverageLatency:    avgLatency,
		ErrorRate:         errorRate,
		ActiveUsers:       lt.config.Concurrency, // Simplified
	}
	
	lt.mu.Lock()
	lt.timeline = append(lt.timeline, point)
	lt.mu.Unlock()
}

// printProgress prints current test progress
func (lt *HFTLoadTester) printProgress() {
	totalReqs := atomic.LoadInt64(&lt.totalRequests)
	successReqs := atomic.LoadInt64(&lt.successfulRequests)
	failedReqs := atomic.LoadInt64(&lt.failedRequests)
	
	elapsed := time.Since(lt.startTime)
	rps := float64(totalReqs) / elapsed.Seconds()
	successRate := float64(successReqs) / float64(totalReqs) * 100
	
	avgLatency := time.Duration(0)
	if totalReqs > 0 {
		avgLatency = time.Duration(atomic.LoadInt64(&lt.totalLatency) / totalReqs)
	}
	
	fmt.Printf("[%v] Requests: %d, RPS: %.1f, Success: %.1f%%, Avg Latency: %v\n",
		elapsed.Truncate(time.Second), totalReqs, rps, successRate, avgLatency)
}

// generateResults generates the final test results
func (lt *HFTLoadTester) generateResults() *LoadTestResults {
	totalReqs := atomic.LoadInt64(&lt.totalRequests)
	successReqs := atomic.LoadInt64(&lt.successfulRequests)
	failedReqs := atomic.LoadInt64(&lt.failedRequests)
	totalDuration := time.Since(lt.startTime)
	
	results := &LoadTestResults{
		Config:             *lt.config,
		TotalRequests:      totalReqs,
		SuccessfulRequests: successReqs,
		FailedRequests:     failedReqs,
		TotalDuration:      totalDuration,
		RequestsPerSecond:  float64(totalReqs) / totalDuration.Seconds(),
		ErrorRate:          float64(failedReqs) / float64(totalReqs),
		ErrorsByType:       make(map[string]int64),
		EndpointResults:    make(map[string]*EndpointResults),
		Timestamp:          time.Now(),
	}
	
	// Calculate latency percentiles
	if len(lt.latencies) > 0 {
		results.AverageLatency = time.Duration(atomic.LoadInt64(&lt.totalLatency) / totalReqs)
		
		// Sort latencies for percentile calculation
		// In a real implementation, you'd use a more efficient percentile algorithm
		results.MinLatency = lt.latencies[0]
		results.MaxLatency = lt.latencies[0]
		
		for _, latency := range lt.latencies {
			if latency < results.MinLatency {
				results.MinLatency = latency
			}
			if latency > results.MaxLatency {
				results.MaxLatency = latency
			}
		}
		
		// Simplified percentile calculation
		if len(lt.latencies) > 10 {
			results.P50Latency = lt.latencies[len(lt.latencies)/2]
			results.P95Latency = lt.latencies[len(lt.latencies)*95/100]
			results.P99Latency = lt.latencies[len(lt.latencies)*99/100]
		}
	}
	
	// Copy error statistics
	lt.mu.RLock()
	for errorType, count := range lt.errors {
		results.ErrorsByType[errorType] = count
	}
	
	// Copy endpoint results
	for key, stats := range lt.endpointStats {
		results.EndpointResults[key] = stats
	}
	
	// Copy timeline
	results.Timeline = make([]TimelinePoint, len(lt.timeline))
	copy(results.Timeline, lt.timeline)
	lt.mu.RUnlock()
	
	return results
}

// Stop stops the load test
func (lt *HFTLoadTester) Stop() {
	lt.cancel()
}
