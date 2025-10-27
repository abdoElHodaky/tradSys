package common

import (
	"context"
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/pkg/types"
	"go.uber.org/zap"
)

// Go124ServiceBase provides a base implementation for services using Go 1.24 features
// This consolidates functionality from both internal/common and pkg/common
type Go124ServiceBase struct {
	mu          sync.RWMutex
	name        string
	version     string
	logger      *zap.Logger
	running     bool
	startTime   time.Time
	stopTime    time.Time
	ctx         context.Context
	cancel      context.CancelFunc
	metrics     types.Metadata
	healthCheck types.Validator[*Go124ServiceBase]
	
	// Go 1.24 enhanced features
	attributes  types.StringAttributes
	metadata    types.Metadata
}

// NewGo124ServiceBase creates a new service base with Go 1.24 optimizations
func NewGo124ServiceBase(name, version string, logger *zap.Logger) *Go124ServiceBase {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &Go124ServiceBase{
		name:        name,
		version:     version,
		logger:      logger,
		ctx:         ctx,
		cancel:      cancel,
		metrics:     make(types.Metadata),
		attributes:  make(types.StringAttributes),
		metadata:    make(types.Metadata),
	}
}

// Start implements the ServiceInterface
func (s *Go124ServiceBase) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return types.ErrServiceAlreadyStarted
	}

	s.running = true
	s.startTime = time.Now()
	s.updateMetric("started_at", s.startTime)
	s.updateMetric("start_count", s.getMetricValue("start_count", 0).(int)+1)

	s.logger.Info("Service started",
		zap.String("name", s.name),
		zap.String("version", s.version),
		zap.Time("started_at", s.startTime),
	)

	return nil
}

// Stop implements the ServiceInterface
func (s *Go124ServiceBase) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return types.ErrServiceNotRunning
	}

	s.running = false
	s.stopTime = time.Now()
	s.cancel()
	s.updateMetric("stopped_at", s.stopTime)
	s.updateMetric("uptime_seconds", s.stopTime.Sub(s.startTime).Seconds())

	s.logger.Info("Service stopped",
		zap.String("name", s.name),
		zap.Duration("uptime", s.stopTime.Sub(s.startTime)),
	)

	return nil
}

// Health implements the ServiceInterface with Go 1.24 enhancements
func (s *Go124ServiceBase) Health() types.HealthStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	status := "healthy"
	message := "Service is operating normally"

	if !s.running {
		status = "stopped"
		message = "Service is not running"
	}

	// Run custom health check if provided
	if s.healthCheck != nil {
		if err := s.healthCheck(context.Background(), s); err != nil {
			status = "unhealthy"
			message = err.Error()
		}
	}

	details := make(types.Metadata)
	details["name"] = s.name
	details["version"] = s.version
	details["running"] = s.running
	details["uptime_seconds"] = s.getUptime().Seconds()
	
	// Add custom attributes and metadata
	for k, v := range s.attributes {
		details[k] = v
	}
	for k, v := range s.metadata {
		details["meta_"+k] = v
	}

	return types.HealthStatus{
		Status:    status,
		Message:   message,
		Timestamp: time.Now(),
		Details:   details,
	}
}

// Name implements the ServiceInterface
func (s *Go124ServiceBase) Name() string {
	return s.name
}

// Version implements the ServiceInterface
func (s *Go124ServiceBase) Version() string {
	return s.version
}

// Go 1.24 Enhanced Methods

// SetAttribute sets a service attribute using Go 1.24 generic types
func (s *Go124ServiceBase) SetAttribute(key string, value interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.attributes[key] = value
}

// GetAttribute gets a service attribute with optional type
func (s *Go124ServiceBase) GetAttribute(key string) types.Option[interface{}] {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if value, exists := s.attributes[key]; exists {
		return types.Some(value)
	}
	return types.None[interface{}]()
}

// SetMetadata sets service metadata
func (s *Go124ServiceBase) SetMetadata(key string, value interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.metadata[key] = value
}

// GetMetadata gets service metadata
func (s *Go124ServiceBase) GetMetadata(key string) types.Option[interface{}] {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if value, exists := s.metadata[key]; exists {
		return types.Some(value)
	}
	return types.None[interface{}]()
}

// GetAllMetrics returns all service metrics
func (s *Go124ServiceBase) GetAllMetrics() types.Metadata {
	s.mu.RLock()
	defer s.mu.RUnlock()

	metricsCopy := make(types.Metadata)
	for k, v := range s.metrics {
		metricsCopy[k] = v
	}
	
	// Add runtime metrics
	metricsCopy["current_time"] = time.Now()
	metricsCopy["uptime_seconds"] = s.getUptime().Seconds()
	metricsCopy["running"] = s.running
	
	return metricsCopy
}

// SetHealthCheck sets a custom health check function
func (s *Go124ServiceBase) SetHealthCheck(check types.Validator[*Go124ServiceBase]) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.healthCheck = check
}

// GetLogger returns the service logger
func (s *Go124ServiceBase) GetLogger() *zap.Logger {
	return s.logger
}

// GetContext returns the service context
func (s *Go124ServiceBase) GetContext() context.Context {
	return s.ctx
}

// IsRunning returns whether the service is running
func (s *Go124ServiceBase) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// GetStartTime returns when the service was started
func (s *Go124ServiceBase) GetStartTime() types.Option[time.Time] {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if s.startTime.IsZero() {
		return types.None[time.Time]()
	}
	return types.Some(s.startTime)
}

// GetUptime returns the service uptime
func (s *Go124ServiceBase) GetUptime() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.getUptime()
}

// Private helper methods

func (s *Go124ServiceBase) updateMetric(key string, value interface{}) {
	s.metrics[key] = value
}

func (s *Go124ServiceBase) getMetricValue(key string, defaultValue interface{}) interface{} {
	if value, exists := s.metrics[key]; exists {
		return value
	}
	return defaultValue
}

func (s *Go124ServiceBase) getUptime() time.Duration {
	if !s.running || s.startTime.IsZero() {
		return 0
	}
	return time.Since(s.startTime)
}

// ServiceRegistry manages multiple services using Go 1.24 generics
type ServiceRegistry[T any] struct {
	mu       sync.RWMutex
	services map[string]T
	logger   *zap.Logger
}

// NewServiceRegistry creates a new service registry
func NewServiceRegistry[T any](logger *zap.Logger) *ServiceRegistry[T] {
	return &ServiceRegistry[T]{
		services: make(map[string]T),
		logger:   logger,
	}
}

// Register registers a service
func (r *ServiceRegistry[T]) Register(name string, service T) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.services[name]; exists {
		return types.NewError("service_already_registered", "Service already registered: "+name)
	}

	r.services[name] = service
	r.logger.Info("Service registered", zap.String("name", name))
	return nil
}

// Unregister removes a service
func (r *ServiceRegistry[T]) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.services[name]; !exists {
		return types.NewError("service_not_found", "Service not found: "+name)
	}

	delete(r.services, name)
	r.logger.Info("Service unregistered", zap.String("name", name))
	return nil
}

// Get retrieves a service
func (r *ServiceRegistry[T]) Get(name string) types.Option[T] {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if service, exists := r.services[name]; exists {
		return types.Some(service)
	}
	return types.None[T]()
}

// List returns all services
func (r *ServiceRegistry[T]) List() map[string]T {
	r.mu.RLock()
	defer r.mu.RUnlock()

	servicesCopy := make(map[string]T)
	for k, v := range r.services {
		servicesCopy[k] = v
	}
	return servicesCopy
}

// Count returns the number of registered services
func (r *ServiceRegistry[T]) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.services)
}

// GetNames returns all service names
func (r *ServiceRegistry[T]) GetNames() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.services))
	for name := range r.services {
		names = append(names, name)
	}
	return names
}
