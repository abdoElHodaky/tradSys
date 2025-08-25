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
	),
)

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
}

// ProvideCoordinationComponents provides the coordination components
func ProvideCoordinationComponents(params Params) Result {
	// Create memory manager
	memoryManager := NewMemoryManager(1024*1024*1024*4, params.Logger) // 4GB default

	// Create timeout manager
	timeoutManager := NewTimeoutManager(params.Logger)

	// Create metrics collector
	metricsConfig := DefaultMetricsConfig()
	metricsCollector := NewUnifiedMetricsCollector(metricsConfig, params.Logger)

	// Create lock manager
	lockManager := NewLockManager(params.Logger)

	// Create component coordinator
	coordinatorConfig := DefaultCoordinatorConfig()
	coordinator := &ComponentCoordinator{
		components:       make(map[string]*ComponentInfo),
		memoryManager:    memoryManager,
		timeoutManager:   timeoutManager,
		metricsCollector: metricsCollector.collector,
		logger:           params.Logger,
		config:           coordinatorConfig,
	}

	return Result{
		Coordinator:      coordinator,
		MemoryManager:    memoryManager,
		TimeoutManager:   timeoutManager,
		MetricsCollector: metricsCollector,
		LockManager:      lockManager,
	}
}

