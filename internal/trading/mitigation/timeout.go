package mitigation

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
)

// TimeoutConfig represents the configuration for a timeout handler
type TimeoutConfig struct {
	// DefaultTimeout is the default timeout duration
	DefaultTimeout time.Duration
	// TimeoutsByOperation maps operation names to specific timeout durations
	TimeoutsByOperation map[string]time.Duration
}

// DefaultTimeoutConfig returns a default configuration for a timeout handler
func DefaultTimeoutConfig() TimeoutConfig {
	return TimeoutConfig{
		DefaultTimeout: 5 * time.Second,
		TimeoutsByOperation: map[string]time.Duration{
			"read":  2 * time.Second,
			"write": 3 * time.Second,
		},
	}
}

// TimeoutMetrics tracks metrics for the timeout handler
type TimeoutMetrics struct {
	// Completed is the number of operations that completed successfully
	Completed int64
	// TimedOut is the number of operations that timed out
	TimedOut int64
	// ByOperation maps operation names to their timeout counts
	ByOperation map[string]int64
	// AverageExecutionTime maps operation names to their average execution time
	AverageExecutionTime map[string]time.Duration
	// TotalExecutionTime maps operation names to their total execution time
	TotalExecutionTime map[string]time.Duration
	// OperationCounts maps operation names to their execution counts
	OperationCounts map[string]int64
}

// TimeoutHandler implements timeout handling for operations
type TimeoutHandler struct {
	name      string
	config    TimeoutConfig
	metrics   *TimeoutMetrics
	mutex     sync.RWMutex
	logger    *zap.Logger
}

// NewTimeoutHandler creates a new timeout handler with the given name and configuration
func NewTimeoutHandler(name string, config TimeoutConfig, logger *zap.Logger) *TimeoutHandler {
	if logger == nil {
		logger, _ = zap.NewProduction()
	}

	return &TimeoutHandler{
		name:   name,
		config: config,
		metrics: &TimeoutMetrics{
			Completed:           0,
			TimedOut:            0,
			ByOperation:         make(map[string]int64),
			AverageExecutionTime: make(map[string]time.Duration),
			TotalExecutionTime:  make(map[string]time.Duration),
			OperationCounts:     make(map[string]int64),
		},
		logger: logger.With(zap.String("component", "timeout_handler"), zap.String("name", name)),
	}
}

// Execute executes the given function with a timeout
func (t *TimeoutHandler) Execute(ctx context.Context, operation string, fn func(ctx context.Context) error) error {
	// Get timeout duration for the operation
	timeout := t.getTimeout(operation)
	
	// Create a context with timeout if not already set
	var cancel context.CancelFunc
	if _, ok := ctx.Deadline(); !ok {
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	
	// Record start time
	start := time.Now()
	
	// Create a channel for the result
	resultCh := make(chan error, 1)
	
	// Execute the function in a goroutine
	go func() {
		resultCh <- fn(ctx)
	}()
	
	// Wait for result or timeout
	var err error
	select {
	case err = <-resultCh:
		// Function completed
		t.recordCompletion(operation, time.Since(start))
	case <-ctx.Done():
		// Context cancelled or timed out
		if ctx.Err() == context.DeadlineExceeded {
			t.recordTimeout(operation)
			err = ErrOperationTimeout
			t.logger.Debug("Operation timed out",
				zap.String("operation", operation),
				zap.Duration("timeout", timeout))
		} else {
			err = ctx.Err()
		}
	}
	
	return err
}

// getTimeout returns the timeout duration for the given operation
func (t *TimeoutHandler) getTimeout(operation string) time.Duration {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	
	if timeout, ok := t.config.TimeoutsByOperation[operation]; ok {
		return timeout
	}
	
	return t.config.DefaultTimeout
}

// recordCompletion records a successful completion of an operation
func (t *TimeoutHandler) recordCompletion(operation string, duration time.Duration) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	
	t.metrics.Completed++
	t.metrics.OperationCounts[operation]++
	t.metrics.TotalExecutionTime[operation] += duration
	t.metrics.AverageExecutionTime[operation] = t.metrics.TotalExecutionTime[operation] / time.Duration(t.metrics.OperationCounts[operation])
}

// recordTimeout records a timeout for an operation
func (t *TimeoutHandler) recordTimeout(operation string) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	
	t.metrics.TimedOut++
	t.metrics.ByOperation[operation]++
}

// GetMetrics returns a copy of the current metrics
func (t *TimeoutHandler) GetMetrics() TimeoutMetrics {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	
	byOperation := make(map[string]int64)
	avgExecutionTime := make(map[string]time.Duration)
	totalExecutionTime := make(map[string]time.Duration)
	operationCounts := make(map[string]int64)
	
	for k, v := range t.metrics.ByOperation {
		byOperation[k] = v
	}
	
	for k, v := range t.metrics.AverageExecutionTime {
		avgExecutionTime[k] = v
	}
	
	for k, v := range t.metrics.TotalExecutionTime {
		totalExecutionTime[k] = v
	}
	
	for k, v := range t.metrics.OperationCounts {
		operationCounts[k] = v
	}
	
	return TimeoutMetrics{
		Completed:           t.metrics.Completed,
		TimedOut:            t.metrics.TimedOut,
		ByOperation:         byOperation,
		AverageExecutionTime: avgExecutionTime,
		TotalExecutionTime:  totalExecutionTime,
		OperationCounts:     operationCounts,
	}
}

// SetTimeout sets the timeout for a specific operation
func (t *TimeoutHandler) SetTimeout(operation string, timeout time.Duration) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	
	t.config.TimeoutsByOperation[operation] = timeout
	t.logger.Info("Timeout updated",
		zap.String("operation", operation),
		zap.Duration("timeout", timeout))
}

// SetDefaultTimeout sets the default timeout
func (t *TimeoutHandler) SetDefaultTimeout(timeout time.Duration) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	
	t.config.DefaultTimeout = timeout
	t.logger.Info("Default timeout updated",
		zap.Duration("timeout", timeout))
}

// Reset resets the timeout handler metrics
func (t *TimeoutHandler) Reset() {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	
	t.metrics.Completed = 0
	t.metrics.TimedOut = 0
	t.metrics.ByOperation = make(map[string]int64)
	t.metrics.AverageExecutionTime = make(map[string]time.Duration)
	t.metrics.TotalExecutionTime = make(map[string]time.Duration)
	t.metrics.OperationCounts = make(map[string]int64)
	
	t.logger.Info("Timeout handler metrics reset", zap.String("name", t.name))
}

// ErrOperationTimeout is returned when an operation times out
var ErrOperationTimeout = TimeoutError{Err: context.DeadlineExceeded}

