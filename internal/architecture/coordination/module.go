package coordination

import (
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Module provides the fx module for the coordination package
var Module = fx.Options(
	fx.Provide(
		NewComponentCoordinator,
		NewMemoryManager,
		NewTimeoutManager,
		NewUnifiedMetricsCollector,
		NewLockManager,
		ProvideCoordinationComponents,
	),
)

// CoordinationComponents contains the coordination system components
type CoordinationComponents struct {
	Coordinator     *ComponentCoordinator
	MemoryManager   *MemoryManager
	TimeoutManager  *TimeoutManager
	MetricsCollector *UnifiedMetricsCollector
	LockManager     *LockManager
}

// Params contains the parameters for the coordination module
type Params struct {
	fx.In

	Logger *zap.Logger
}

// Result contains the results of the coordination module
type Result struct {
	fx.Out

	Coordinator      *ComponentCoordinator
	MemoryManager    *MemoryManager
	TimeoutManager   *TimeoutManager
	MetricsCollector *UnifiedMetricsCollector
	LockManager      *LockManager
	Components       *CoordinationComponents
}

// ProvideCoordinationComponents provides the coordination components
func ProvideCoordinationComponents(params Params) Result {
	// Create memory manager
	memoryManager := NewMemoryManager(DefaultMemoryManagerConfig(), params.Logger)

	// Create timeout manager
	timeoutManager := NewTimeoutManager(params.Logger)

	// Create metrics collector
	metricsConfig := DefaultMetricsConfig()
	metricsCollector := NewUnifiedMetricsCollector(metricsConfig, params.Logger)

	// Create lock manager
	lockManager := NewLockManager(DefaultLockManagerConfig(), params.Logger)

	// Create component coordinator
	coordinatorConfig := DefaultCoordinatorConfig()
	coordinator := NewComponentCoordinator(coordinatorConfig, params.Logger)

	// Create components struct for backward compatibility
	components := &CoordinationComponents{
		Coordinator:     coordinator,
		MemoryManager:   memoryManager,
		TimeoutManager:  timeoutManager,
		MetricsCollector: metricsCollector,
		LockManager:     lockManager,
	}

	return Result{
		Coordinator:      coordinator,
		MemoryManager:    memoryManager,
		TimeoutManager:   timeoutManager,
		MetricsCollector: metricsCollector,
		LockManager:      lockManager,
		Components:       components,
	}
}
