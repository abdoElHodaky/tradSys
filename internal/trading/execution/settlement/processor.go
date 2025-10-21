package settlement

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

// SettlementRequest represents a settlement request
type SettlementRequest struct {
	ID          string    `json:"id"`
	TradeID     string    `json:"trade_id"`
	BuyerID     string    `json:"buyer_id"`
	SellerID    string    `json:"seller_id"`
	Symbol      string    `json:"symbol"`
	Quantity    float64   `json:"quantity"`
	Price       float64   `json:"price"`
	Fee         float64   `json:"fee"`
	Commission  float64   `json:"commission"`
	Status      string    `json:"status"` // "pending", "processing", "settled", "failed"
	CreatedAt   time.Time `json:"created_at"`
	ProcessedAt time.Time `json:"processed_at,omitempty"`
	RetryCount  int       `json:"retry_count"`
}

// SettlementResult represents the result of a settlement
type SettlementResult struct {
	Success     bool      `json:"success"`
	RequestID   string    `json:"request_id"`
	Error       string    `json:"error,omitempty"`
	ProcessedAt time.Time `json:"processed_at"`
	Latency     time.Duration `json:"latency"`
}

// Processor handles T+0 real-time settlement processing
type Processor struct {
	requests        map[string]*SettlementRequest
	mutex           sync.RWMutex
	workers         int
	workerPool      chan struct{}
	requestQueue    chan *SettlementRequest
	metrics         map[string]interface{}
	totalSettlements int64
	successfulSettlements int64
	failedSettlements int64
	running         bool
	ctx             context.Context
	cancel          context.CancelFunc
	logger          *zap.Logger
	wg              sync.WaitGroup
}

// NewProcessor creates a new settlement processor
func NewProcessor(logger *zap.Logger) *Processor {
	workers := 10 // Default number of workers
	ctx, cancel := context.WithCancel(context.Background())
	
	sp := &Processor{
		requests:     make(map[string]*SettlementRequest),
		workers:      workers,
		workerPool:   make(chan struct{}, workers),
		requestQueue: make(chan *SettlementRequest, 1000),
		metrics:      make(map[string]interface{}),
		ctx:          ctx,
		cancel:       cancel,
		logger:       logger,
	}
	
	// Initialize worker pool
	for i := 0; i < workers; i++ {
		sp.workerPool <- struct{}{}
	}
	
	return sp
}

// Start starts the settlement processor
func (sp *Processor) Start() {
	sp.mutex.Lock()
	if sp.running {
		sp.mutex.Unlock()
		return
	}
	sp.running = true
	sp.mutex.Unlock()
	
	// Start worker goroutines
	for i := 0; i < sp.workers; i++ {
		sp.wg.Add(1)
		go sp.worker()
	}
}

// Stop stops the settlement processor
func (sp *Processor) Stop() {
	sp.mutex.Lock()
	if !sp.running {
		sp.mutex.Unlock()
		return
	}
	sp.running = false
	sp.mutex.Unlock()
	
	sp.cancel()
	close(sp.requestQueue)
	sp.wg.Wait()
}

// Shutdown gracefully shuts down the processor with timeout
func (sp *Processor) Shutdown(timeout time.Duration) error {
	done := make(chan struct{})
	go func() {
		sp.Stop()
		close(done)
	}()
	
	select {
	case <-done:
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("shutdown timeout after %v", timeout)
	}
}

// worker processes settlement requests
func (sp *Processor) worker() {
	defer sp.wg.Done()
	
	for {
		select {
		case <-sp.ctx.Done():
			return
		case request, ok := <-sp.requestQueue:
			if !ok {
				return
			}
			
			// Get worker token
			<-sp.workerPool
			
			// Process settlement
			result := sp.processSettlement(request)
			
			// Update request status
			sp.mutex.Lock()
			if result.Success {
				request.Status = "settled"
				atomic.AddInt64(&sp.successfulSettlements, 1)
			} else {
				request.Status = "failed"
				atomic.AddInt64(&sp.failedSettlements, 1)
			}
			request.ProcessedAt = result.ProcessedAt
			sp.requests[request.ID] = request
			sp.updateMetrics()
			sp.mutex.Unlock()
			
			// Return worker token
			sp.workerPool <- struct{}{}
		}
	}
}

// ProcessSettlement processes a settlement request
func (sp *Processor) ProcessSettlement(ctx context.Context, request *SettlementRequest) (*SettlementResult, error) {
	if !sp.running {
		return nil, fmt.Errorf("settlement processor is not running")
	}
	
	// Generate ID if not provided
	if request.ID == "" {
		request.ID = fmt.Sprintf("settlement_%d_%s", time.Now().UnixNano(), request.TradeID)
	}
	
	request.Status = "pending"
	request.CreatedAt = time.Now()
	
	// Store request
	sp.mutex.Lock()
	sp.requests[request.ID] = request
	atomic.AddInt64(&sp.totalSettlements, 1)
	sp.mutex.Unlock()
	
	// Queue for processing
	select {
	case sp.requestQueue <- request:
		// Successfully queued
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(100 * time.Millisecond):
		return nil, fmt.Errorf("settlement queue is full")
	}
	
	// Wait for processing (simplified for demo)
	time.Sleep(1 * time.Millisecond) // Simulate T+0 processing
	
	return &SettlementResult{
		Success:     true,
		RequestID:   request.ID,
		ProcessedAt: time.Now(),
		Latency:     time.Since(request.CreatedAt),
	}, nil
}

// processSettlement performs the actual settlement processing
func (sp *Processor) processSettlement(request *SettlementRequest) *SettlementResult {
	start := time.Now()
	
	// Update status to processing
	request.Status = "processing"
	
	// Simulate settlement processing (T+0)
	// In a real system, this would involve:
	// 1. Validating balances
	// 2. Transferring assets
	// 3. Updating positions
	// 4. Recording transactions
	// 5. Notifying parties
	
	// For demo purposes, we'll simulate a very fast settlement
	time.Sleep(500 * time.Microsecond) // 500μs processing time
	
	// Simulate occasional failures (1% failure rate)
	success := true
	errorMsg := ""
	
	if request.RetryCount > 2 {
		success = false
		errorMsg = "max retries exceeded"
	}
	
	return &SettlementResult{
		Success:     success,
		RequestID:   request.ID,
		Error:       errorMsg,
		ProcessedAt: time.Now(),
		Latency:     time.Since(start),
	}
}

// GetSettlementRequest retrieves a settlement request by ID
func (sp *Processor) GetSettlementRequest(requestID string) (*SettlementRequest, bool) {
	sp.mutex.RLock()
	defer sp.mutex.RUnlock()
	
	request, exists := sp.requests[requestID]
	return request, exists
}

// GetSettlementsByTrade returns all settlements for a trade
func (sp *Processor) GetSettlementsByTrade(tradeID string) []*SettlementRequest {
	sp.mutex.RLock()
	defer sp.mutex.RUnlock()
	
	var settlements []*SettlementRequest
	for _, request := range sp.requests {
		if request.TradeID == tradeID {
			settlements = append(settlements, request)
		}
	}
	
	return settlements
}

// GetSettlementsByUser returns all settlements for a user
func (sp *Processor) GetSettlementsByUser(userID string) []*SettlementRequest {
	sp.mutex.RLock()
	defer sp.mutex.RUnlock()
	
	var settlements []*SettlementRequest
	for _, request := range sp.requests {
		if request.BuyerID == userID || request.SellerID == userID {
			settlements = append(settlements, request)
		}
	}
	
	return settlements
}

// RetrySettlement retries a failed settlement
func (sp *Processor) RetrySettlement(requestID string) error {
	sp.mutex.Lock()
	defer sp.mutex.Unlock()
	
	request, exists := sp.requests[requestID]
	if !exists {
		return fmt.Errorf("settlement request %s not found", requestID)
	}
	
	if request.Status != "failed" {
		return fmt.Errorf("can only retry failed settlements")
	}
	
	if request.RetryCount >= 3 {
		return fmt.Errorf("maximum retry attempts exceeded")
	}
	
	request.RetryCount++
	request.Status = "pending"
	
	// Re-queue for processing
	select {
	case sp.requestQueue <- request:
		return nil
	default:
		return fmt.Errorf("settlement queue is full")
	}
}

// updateMetrics updates internal performance metrics
func (sp *Processor) updateMetrics() {
	totalSettlements := atomic.LoadInt64(&sp.totalSettlements)
	successfulSettlements := atomic.LoadInt64(&sp.successfulSettlements)
	failedSettlements := atomic.LoadInt64(&sp.failedSettlements)
	
	var successRate float64
	if totalSettlements > 0 {
		successRate = float64(successfulSettlements) / float64(totalSettlements)
	}
	
	sp.metrics["total_settlements"] = totalSettlements
	sp.metrics["successful_settlements"] = successfulSettlements
	sp.metrics["failed_settlements"] = failedSettlements
	sp.metrics["success_rate"] = successRate
	sp.metrics["queue_size"] = int64(len(sp.requestQueue))
	sp.metrics["workers"] = int64(sp.workers)
	sp.metrics["last_settlement"] = time.Now()
}

// GetPerformanceMetrics returns settlement processor performance metrics
func (sp *Processor) GetPerformanceMetrics() map[string]interface{} {
	sp.mutex.RLock()
	defer sp.mutex.RUnlock()
	
	// Update metrics before returning
	sp.updateMetrics()
	
	metrics := make(map[string]interface{})
	for k, v := range sp.metrics {
		metrics[k] = v
	}
	
	return metrics
}

// GetSettlementStats returns settlement statistics
func (sp *Processor) GetSettlementStats() map[string]interface{} {
	sp.mutex.RLock()
	defer sp.mutex.RUnlock()
	
	stats := make(map[string]interface{})
	stats["total_requests"] = len(sp.requests)
	stats["workers"] = sp.workers
	stats["queue_capacity"] = cap(sp.requestQueue)
	stats["queue_size"] = len(sp.requestQueue)
	stats["running"] = sp.running
	
	// Calculate status distribution
	statusCounts := make(map[string]int)
	for _, request := range sp.requests {
		statusCounts[request.Status]++
	}
	stats["status_distribution"] = statusCounts
	
	return stats
}

// GetPendingSettlements returns all pending settlements
func (sp *Processor) GetPendingSettlements() []*SettlementRequest {
	sp.mutex.RLock()
	defer sp.mutex.RUnlock()
	
	var pending []*SettlementRequest
	for _, request := range sp.requests {
		if request.Status == "pending" || request.Status == "processing" {
			pending = append(pending, request)
		}
	}
	
	return pending
}

// ProcessTrade processes a trade for settlement (simplified interface for unified engine)
func (sp *Processor) ProcessTrade(tradeID, symbol string, quantity, price float64) error {
	request := &SettlementRequest{
		TradeID:   tradeID,
		Symbol:    symbol,
		Quantity:  quantity,
		Price:     price,
		Status:    "pending",
		CreatedAt: time.Now(),
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	
	result, err := sp.ProcessSettlement(ctx, request)
	if err != nil {
		return err
	}
	
	if !result.Success {
		return fmt.Errorf("settlement failed: %s", result.Error)
	}
	
	return nil
}
