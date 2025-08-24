package lazy

import (
	"fmt"
	"reflect"
	"sync"
	"time"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

// LazyProvider wraps a component constructor for lazy initialization
type LazyProvider struct {
	constructor interface{}
	instance    interface{}
	once        sync.Once
	logger      *zap.Logger
	initialized bool
	err         error
	name        string
	metrics     *LazyLoadingMetrics
}

// NewLazyProvider creates a new lazy provider
func NewLazyProvider(name string, constructor interface{}, logger *zap.Logger, metrics *LazyLoadingMetrics) *LazyProvider {
	return &LazyProvider{
		constructor: constructor,
		logger:      logger,
		name:        name,
		metrics:     metrics,
	}
}

// Get returns the lazily initialized component
func (p *LazyProvider) Get() (interface{}, error) {
	p.once.Do(func() {
		startTime := time.Now()
		p.logger.Debug("Lazily initializing component", zap.String("component", p.name))

		constructorValue := reflect.ValueOf(p.constructor)
		constructorType := constructorValue.Type()

		// Check if the constructor is a function
		if constructorType.Kind() != reflect.Func {
			p.err = fmt.Errorf("constructor must be a function, got %s", constructorType.Kind())
			p.metrics.RecordInitialization(p.name, time.Since(startTime), p.err)
			return
		}

		// Get the parameter values from the dependency injection container
		var params []reflect.Value

		// Call the constructor
		results := constructorValue.Call(params)

		// Check the results
		if len(results) == 0 {
			p.err = fmt.Errorf("constructor must return at least one value")
			p.metrics.RecordInitialization(p.name, time.Since(startTime), p.err)
			return
		}

		// If the last result is an error, check it
		if len(results) > 1 && !results[len(results)-1].IsNil() && results[len(results)-1].Type().Implements(reflect.TypeOf((*error)(nil)).Elem()) {
			p.err = results[len(results)-1].Interface().(error)
			p.metrics.RecordInitialization(p.name, time.Since(startTime), p.err)
			return
		}

		// Store the result
		p.instance = results[0].Interface()
		p.initialized = true
		p.metrics.RecordInitialization(p.name, time.Since(startTime), nil)
		p.logger.Info("Component lazily initialized", zap.String("component", p.name), zap.Duration("duration", time.Since(startTime)))
	})

	return p.instance, p.err
}

// IsInitialized returns whether the component has been initialized
func (p *LazyProvider) IsInitialized() bool {
	return p.initialized
}

// AsOption returns an fx.Option that registers the lazy provider
func (p *LazyProvider) AsOption() fx.Option {
	return fx.Provide(func() *LazyProvider {
		return p
	})
}

// LazyLoadingMetrics collects metrics for lazy loading
type LazyLoadingMetrics struct {
	mu                sync.RWMutex
	initializations   map[string]int64
	initializationErr map[string]int64
	initializationTimes map[string][]time.Duration
}

// NewLazyLoadingMetrics creates a new LazyLoadingMetrics
func NewLazyLoadingMetrics() *LazyLoadingMetrics {
	return &LazyLoadingMetrics{
		initializations:   make(map[string]int64),
		initializationErr: make(map[string]int64),
		initializationTimes: make(map[string][]time.Duration),
	}
}

// RecordInitialization records an initialization
func (m *LazyLoadingMetrics) RecordInitialization(name string, duration time.Duration, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.initializations[name]++
	
	if err != nil {
		m.initializationErr[name]++
	}

	if _, ok := m.initializationTimes[name]; !ok {
		m.initializationTimes[name] = make([]time.Duration, 0, 10)
	}
	
	m.initializationTimes[name] = append(m.initializationTimes[name], duration)
	
	// Keep only the last 10 initialization times
	if len(m.initializationTimes[name]) > 10 {
		m.initializationTimes[name] = m.initializationTimes[name][1:]
	}
}

// GetInitializationCount returns the number of initializations
func (m *LazyLoadingMetrics) GetInitializationCount(name string) int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	return m.initializations[name]
}

// GetInitializationErrorCount returns the number of initialization errors
func (m *LazyLoadingMetrics) GetInitializationErrorCount(name string) int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	return m.initializationErr[name]
}

// GetAverageInitializationTime returns the average initialization time
func (m *LazyLoadingMetrics) GetAverageInitializationTime(name string) time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	times, ok := m.initializationTimes[name]
	if !ok || len(times) == 0 {
		return 0
	}
	
	var sum time.Duration
	for _, t := range times {
		sum += t
	}
	
	return sum / time.Duration(len(times))
}

// Reset resets all metrics
func (m *LazyLoadingMetrics) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.initializations = make(map[string]int64)
	m.initializationErr = make(map[string]int64)
	m.initializationTimes = make(map[string][]time.Duration)
}

