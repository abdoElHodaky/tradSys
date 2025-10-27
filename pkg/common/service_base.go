package common

import (
	"context"
	"errors"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Service errors
var (
	ErrServiceAlreadyStarted = errors.New("service already started")
	ErrServiceNotRunning     = errors.New("service not running")
)

// ServiceInterface defines the standard interface for all services
type ServiceInterface interface {
	// Start starts the service
	Start(ctx context.Context) error
	// Stop stops the service gracefully
	Stop(ctx context.Context) error
	// Health returns the health status of the service
	Health() HealthStatus
	// Name returns the service name
	Name() string
	// Version returns the service version
	Version() string
}

// ServiceHealthStatus represents the health status of a service
type ServiceHealthStatus struct {
	Status    string            `json:"status"`    // "healthy", "degraded", "unhealthy"
	Message   string            `json:"message"`   // Human-readable status message
	Timestamp time.Time         `json:"timestamp"` // When the status was last updated
	Details   map[string]string `json:"details"`   // Additional health details
}

// ServiceState represents the current state of a service
type ServiceState int

const (
	ServiceStateStopped ServiceState = iota
	ServiceStateStarting
	ServiceStateRunning
	ServiceStateStopping
	ServiceStateError
)

// String returns the string representation of the service state
func (s ServiceState) String() string {
	switch s {
	case ServiceStateStopped:
		return "stopped"
	case ServiceStateStarting:
		return "starting"
	case ServiceStateRunning:
		return "running"
	case ServiceStateStopping:
		return "stopping"
	case ServiceStateError:
		return "error"
	default:
		return "unknown"
	}
}

// BaseService provides a standard implementation of common service functionality
type BaseService struct {
	name    string
	version string
	logger  *zap.Logger
	
	// State management
	state     ServiceState
	stateMu   sync.RWMutex
	
	// Context management
	ctx    context.Context
	cancel context.CancelFunc
	
	// Health monitoring
	health     ServiceHealthStatus
	healthMu   sync.RWMutex
	
	// Lifecycle hooks
	startHook func(ctx context.Context) error
	stopHook  func(ctx context.Context) error
	
	// Wait group for graceful shutdown
	wg sync.WaitGroup
}

// NewBaseService creates a new base service
func NewBaseService(name, version string, logger *zap.Logger) *BaseService {
	return &BaseService{
		name:    name,
		version: version,
		logger:  logger,
		state:   ServiceStateStopped,
		health: ServiceHealthStatus{
			Status:    "healthy",
			Message:   "Service initialized",
			Timestamp: time.Now(),
			Details:   make(map[string]string),
		},
	}
}

// SetStartHook sets the function to be called when the service starts
func (bs *BaseService) SetStartHook(hook func(ctx context.Context) error) {
	bs.startHook = hook
}

// SetStopHook sets the function to be called when the service stops
func (bs *BaseService) SetStopHook(hook func(ctx context.Context) error) {
	bs.stopHook = hook
}

// Start starts the service
func (bs *BaseService) Start(ctx context.Context) error {
	bs.stateMu.Lock()
	defer bs.stateMu.Unlock()
	
	if bs.state != ServiceStateStopped {
		return ErrServiceAlreadyStarted
	}
	
	bs.state = ServiceStateStarting
	bs.ctx, bs.cancel = context.WithCancel(ctx)
	
	bs.logger.Info("Starting service", zap.String("service", bs.name))
	
	// Call start hook if provided
	if bs.startHook != nil {
		if err := bs.startHook(bs.ctx); err != nil {
			bs.state = ServiceStateError
			bs.updateHealth("unhealthy", "Failed to start: "+err.Error())
			return err
		}
	}
	
	bs.state = ServiceStateRunning
	bs.updateHealth("healthy", "Service running")
	
	bs.logger.Info("Service started successfully", zap.String("service", bs.name))
	return nil
}

// Stop stops the service gracefully
func (bs *BaseService) Stop(ctx context.Context) error {
	bs.stateMu.Lock()
	defer bs.stateMu.Unlock()
	
	if bs.state != ServiceStateRunning {
		return ErrServiceNotRunning
	}
	
	bs.state = ServiceStateStopping
	bs.logger.Info("Stopping service", zap.String("service", bs.name))
	
	// Cancel context to signal shutdown
	if bs.cancel != nil {
		bs.cancel()
	}
	
	// Call stop hook if provided
	if bs.stopHook != nil {
		if err := bs.stopHook(ctx); err != nil {
			bs.logger.Error("Error during service stop", zap.Error(err))
			// Continue with shutdown even if hook fails
		}
	}
	
	// Wait for all goroutines to finish
	done := make(chan struct{})
	go func() {
		bs.wg.Wait()
		close(done)
	}()
	
	select {
	case <-done:
		// All goroutines finished
	case <-ctx.Done():
		// Timeout during shutdown
		bs.logger.Warn("Service shutdown timeout", zap.String("service", bs.name))
	}
	
	bs.state = ServiceStateStopped
	bs.updateHealth("healthy", "Service stopped")
	
	bs.logger.Info("Service stopped", zap.String("service", bs.name))
	return nil
}

// Health returns the current health status
func (bs *BaseService) Health() HealthStatus {
	bs.healthMu.RLock()
	defer bs.healthMu.RUnlock()
	return bs.health
}

// Name returns the service name
func (bs *BaseService) Name() string {
	return bs.name
}

// Version returns the service version
func (bs *BaseService) Version() string {
	return bs.version
}

// State returns the current service state
func (bs *BaseService) State() ServiceState {
	bs.stateMu.RLock()
	defer bs.stateMu.RUnlock()
	return bs.state
}

// Context returns the service context
func (bs *BaseService) Context() context.Context {
	return bs.ctx
}

// Logger returns the service logger
func (bs *BaseService) Logger() *zap.Logger {
	return bs.logger
}

// AddWorker adds a worker goroutine to the wait group
func (bs *BaseService) AddWorker() {
	bs.wg.Add(1)
}

// WorkerDone signals that a worker goroutine has finished
func (bs *BaseService) WorkerDone() {
	bs.wg.Done()
}

// updateHealth updates the health status
func (bs *BaseService) updateHealth(status, message string) {
	bs.healthMu.Lock()
	defer bs.healthMu.Unlock()
	
	bs.health.Status = status
	bs.health.Message = message
	bs.health.Timestamp = time.Now()
}

// UpdateHealthDetails updates specific health details
func (bs *BaseService) UpdateHealthDetails(key, value string) {
	bs.healthMu.Lock()
	defer bs.healthMu.Unlock()
	
	if bs.health.Details == nil {
		bs.health.Details = make(map[string]string)
	}
	bs.health.Details[key] = value
	bs.health.Timestamp = time.Now()
}

// SetHealthStatus sets the overall health status
func (bs *BaseService) SetHealthStatus(status, message string) {
	bs.updateHealth(status, message)
}

// IsRunning returns true if the service is running
func (bs *BaseService) IsRunning() bool {
	bs.stateMu.RLock()
	defer bs.stateMu.RUnlock()
	return bs.state == ServiceStateRunning
}

// WaitForShutdown waits for the service to be shut down
func (bs *BaseService) WaitForShutdown() {
	if bs.ctx != nil {
		<-bs.ctx.Done()
	}
}
