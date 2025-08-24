package lazy

import (
	"context"
	"sync"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

// LazyLifecycle manages the lifecycle of lazily loaded components
type LazyLifecycle struct {
	logger   *zap.Logger
	metrics  *LazyLoadingMetrics
	modules  map[string]*ProxyModule
	mu       sync.RWMutex
}

// NewLazyLifecycle creates a new LazyLifecycle
func NewLazyLifecycle(logger *zap.Logger, metrics *LazyLoadingMetrics) *LazyLifecycle {
	return &LazyLifecycle{
		logger:  logger,
		metrics: metrics,
		modules: make(map[string]*ProxyModule),
	}
}

// RegisterModule registers a module with the lazy lifecycle
func (l *LazyLifecycle) RegisterModule(name string, module *ProxyModule) {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	l.modules[name] = module
	l.logger.Debug("Registered lazy module", zap.String("name", name))
}

// GetModule returns a module by name
func (l *LazyLifecycle) GetModule(name string) (*ProxyModule, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	
	module, ok := l.modules[name]
	return module, ok
}

// StartModule starts a module by name
func (l *LazyLifecycle) StartModule(ctx context.Context, name string) error {
	module, ok := l.GetModule(name)
	if !ok {
		return nil
	}
	
	// Get the module instance, which will initialize it if needed
	instance, err := module.Get()
	if err != nil {
		return err
	}
	
	// Check if the instance has a Start method
	if starter, ok := instance.(interface {
		Start(context.Context) error
	}); ok {
		l.logger.Debug("Starting lazy module", zap.String("name", name))
		return starter.Start(ctx)
	}
	
	return nil
}

// StopModule stops a module by name
func (l *LazyLifecycle) StopModule(ctx context.Context, name string) error {
	module, ok := l.GetModule(name)
	if !ok {
		return nil
	}
	
	// If the module is not initialized, there's nothing to stop
	if !module.GetProvider().IsInitialized() {
		return nil
	}
	
	// Get the module instance
	instance, err := module.Get()
	if err != nil {
		return err
	}
	
	// Check if the instance has a Stop method
	if stopper, ok := instance.(interface {
		Stop(context.Context) error
	}); ok {
		l.logger.Debug("Stopping lazy module", zap.String("name", name))
		return stopper.Stop(ctx)
	}
	
	return nil
}

// StartAllModules starts all registered modules
func (l *LazyLifecycle) StartAllModules(ctx context.Context) error {
	l.mu.RLock()
	defer l.mu.RUnlock()
	
	for name := range l.modules {
		if err := l.StartModule(ctx, name); err != nil {
			return err
		}
	}
	
	return nil
}

// StopAllModules stops all registered modules
func (l *LazyLifecycle) StopAllModules(ctx context.Context) error {
	l.mu.RLock()
	defer l.mu.RUnlock()
	
	for name := range l.modules {
		if err := l.StopModule(ctx, name); err != nil {
			return err
		}
	}
	
	return nil
}

// AsLifecycle returns an fx.Lifecycle that can be used with fx
func (l *LazyLifecycle) AsLifecycle() fx.Option {
	return fx.Invoke(func(lifecycle fx.Lifecycle) {
		lifecycle.Append(fx.Hook{
			OnStop: func(ctx context.Context) error {
				return l.StopAllModules(ctx)
			},
		})
	})
}

