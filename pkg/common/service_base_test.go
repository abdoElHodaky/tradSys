package common

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/zap/zaptest"
)

func TestBaseService_NewBaseService(t *testing.T) {
	logger := zaptest.NewLogger(t)
	service := NewBaseService("test-service", "1.0.0", logger)

	if service == nil {
		t.Fatal("Expected service to be created")
	}

	if service.Name() != "test-service" {
		t.Errorf("Expected name 'test-service', got '%s'", service.Name())
	}

	if service.Version() != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%s'", service.Version())
	}

	if service.State() != ServiceStateStopped {
		t.Errorf("Expected initial state to be stopped, got %s", service.State().String())
	}
}

func TestBaseService_StartStop(t *testing.T) {
	logger := zaptest.NewLogger(t)
	service := NewBaseService("test-service", "1.0.0", logger)
	ctx := context.Background()

	// Test start
	err := service.Start(ctx)
	if err != nil {
		t.Errorf("Expected no error on start, got %v", err)
	}

	if service.State() != ServiceStateRunning {
		t.Errorf("Expected state to be running, got %s", service.State().String())
	}

	if !service.IsRunning() {
		t.Error("Expected service to be running")
	}

	// Test stop
	err = service.Stop(ctx)
	if err != nil {
		t.Errorf("Expected no error on stop, got %v", err)
	}

	if service.State() != ServiceStateStopped {
		t.Errorf("Expected state to be stopped, got %s", service.State().String())
	}

	if service.IsRunning() {
		t.Error("Expected service to not be running")
	}
}

func TestBaseService_StartAlreadyStarted(t *testing.T) {
	logger := zaptest.NewLogger(t)
	service := NewBaseService("test-service", "1.0.0", logger)
	ctx := context.Background()

	// Start service
	err := service.Start(ctx)
	if err != nil {
		t.Errorf("Expected no error on first start, got %v", err)
	}

	// Try to start again
	err = service.Start(ctx)
	if err != ErrServiceAlreadyStarted {
		t.Errorf("Expected ErrServiceAlreadyStarted, got %v", err)
	}
}

func TestBaseService_StopNotRunning(t *testing.T) {
	logger := zaptest.NewLogger(t)
	service := NewBaseService("test-service", "1.0.0", logger)
	ctx := context.Background()

	// Try to stop without starting
	err := service.Stop(ctx)
	if err != ErrServiceNotRunning {
		t.Errorf("Expected ErrServiceNotRunning, got %v", err)
	}
}

func TestBaseService_StartHook(t *testing.T) {
	logger := zaptest.NewLogger(t)
	service := NewBaseService("test-service", "1.0.0", logger)
	ctx := context.Background()

	hookCalled := false
	service.SetStartHook(func(ctx context.Context) error {
		hookCalled = true
		return nil
	})

	err := service.Start(ctx)
	if err != nil {
		t.Errorf("Expected no error on start, got %v", err)
	}

	if !hookCalled {
		t.Error("Expected start hook to be called")
	}
}

func TestBaseService_StartHookError(t *testing.T) {
	logger := zaptest.NewLogger(t)
	service := NewBaseService("test-service", "1.0.0", logger)
	ctx := context.Background()

	expectedError := errors.New("start hook error")
	service.SetStartHook(func(ctx context.Context) error {
		return expectedError
	})

	err := service.Start(ctx)
	if err != expectedError {
		t.Errorf("Expected start hook error, got %v", err)
	}

	if service.State() != ServiceStateError {
		t.Errorf("Expected state to be error, got %s", service.State().String())
	}
}

func TestBaseService_StopHook(t *testing.T) {
	logger := zaptest.NewLogger(t)
	service := NewBaseService("test-service", "1.0.0", logger)
	ctx := context.Background()

	hookCalled := false
	service.SetStopHook(func(ctx context.Context) error {
		hookCalled = true
		return nil
	})

	// Start and then stop
	service.Start(ctx)
	err := service.Stop(ctx)
	if err != nil {
		t.Errorf("Expected no error on stop, got %v", err)
	}

	if !hookCalled {
		t.Error("Expected stop hook to be called")
	}
}

func TestBaseService_Health(t *testing.T) {
	logger := zaptest.NewLogger(t)
	service := NewBaseService("test-service", "1.0.0", logger)

	health := service.Health()
	if health.Status != "healthy" {
		t.Errorf("Expected status 'healthy', got '%s'", health.Status)
	}

	if health.Message != "Service initialized" {
		t.Errorf("Expected message 'Service initialized', got '%s'", health.Message)
	}
}

func TestBaseService_UpdateHealthDetails(t *testing.T) {
	logger := zaptest.NewLogger(t)
	service := NewBaseService("test-service", "1.0.0", logger)

	service.UpdateHealthDetails("connections", "5")
	service.UpdateHealthDetails("memory_usage", "50MB")

	health := service.Health()
	if health.Details["connections"] != "5" {
		t.Errorf("Expected connections '5', got '%s'", health.Details["connections"])
	}

	if health.Details["memory_usage"] != "50MB" {
		t.Errorf("Expected memory_usage '50MB', got '%s'", health.Details["memory_usage"])
	}
}

func TestBaseService_SetHealthStatus(t *testing.T) {
	logger := zaptest.NewLogger(t)
	service := NewBaseService("test-service", "1.0.0", logger)

	service.SetHealthStatus("degraded", "High memory usage")

	health := service.Health()
	if health.Status != "degraded" {
		t.Errorf("Expected status 'degraded', got '%s'", health.Status)
	}

	if health.Message != "High memory usage" {
		t.Errorf("Expected message 'High memory usage', got '%s'", health.Message)
	}
}

func TestBaseService_WorkerManagement(t *testing.T) {
	logger := zaptest.NewLogger(t)
	service := NewBaseService("test-service", "1.0.0", logger)
	ctx := context.Background()

	// Start service
	service.Start(ctx)

	// Add workers
	numWorkers := 3
	for i := 0; i < numWorkers; i++ {
		service.AddWorker()
		go func() {
			defer service.WorkerDone()
			time.Sleep(100 * time.Millisecond)
		}()
	}

	// Stop service (should wait for workers)
	start := time.Now()
	service.Stop(ctx)
	duration := time.Since(start)

	// Should have waited for workers to finish
	if duration < 100*time.Millisecond {
		t.Error("Expected service to wait for workers to finish")
	}
}

func TestBaseService_Context(t *testing.T) {
	logger := zaptest.NewLogger(t)
	service := NewBaseService("test-service", "1.0.0", logger)
	ctx := context.Background()

	// Context should be nil before start
	if service.Context() != nil {
		t.Error("Expected context to be nil before start")
	}

	// Start service
	service.Start(ctx)

	// Context should be available after start
	if service.Context() == nil {
		t.Error("Expected context to be available after start")
	}

	// Context should be cancelled after stop
	serviceCtx := service.Context()
	service.Stop(ctx)

	select {
	case <-serviceCtx.Done():
		// Expected
	case <-time.After(100 * time.Millisecond):
		t.Error("Expected context to be cancelled after stop")
	}
}

func TestBaseService_WaitForShutdown(t *testing.T) {
	logger := zaptest.NewLogger(t)
	service := NewBaseService("test-service", "1.0.0", logger)
	ctx := context.Background()

	// Start service
	service.Start(ctx)

	// Start goroutine to wait for shutdown
	shutdownComplete := make(chan bool)
	go func() {
		service.WaitForShutdown()
		shutdownComplete <- true
	}()

	// Stop service after a delay
	go func() {
		time.Sleep(50 * time.Millisecond)
		service.Stop(ctx)
	}()

	// Wait for shutdown to complete
	select {
	case <-shutdownComplete:
		// Expected
	case <-time.After(200 * time.Millisecond):
		t.Error("Expected shutdown to complete")
	}
}

func TestServiceState_String(t *testing.T) {
	tests := []struct {
		state    ServiceState
		expected string
	}{
		{ServiceStateStopped, "stopped"},
		{ServiceStateStarting, "starting"},
		{ServiceStateRunning, "running"},
		{ServiceStateStopping, "stopping"},
		{ServiceStateError, "error"},
		{ServiceState(999), "unknown"},
	}

	for _, test := range tests {
		if test.state.String() != test.expected {
			t.Errorf("Expected %s, got %s", test.expected, test.state.String())
		}
	}
}

func BenchmarkBaseService_StartStop(b *testing.B) {
	logger := zaptest.NewLogger(b)
	service := NewBaseService("test-service", "1.0.0", logger)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.Start(ctx)
		service.Stop(ctx)
	}
}

func BenchmarkBaseService_Health(b *testing.B) {
	logger := zaptest.NewLogger(b)
	service := NewBaseService("test-service", "1.0.0", logger)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = service.Health()
	}
}

func BenchmarkBaseService_UpdateHealthDetails(b *testing.B) {
	logger := zaptest.NewLogger(b)
	service := NewBaseService("test-service", "1.0.0", logger)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.UpdateHealthDetails("key", "value")
	}
}
