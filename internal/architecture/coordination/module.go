package coordination

import (
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Module provides the fx module for the coordination system
var Module = fx.Options(
	fx.Provide(
		NewCoordinationComponents,
	),
)

// CoordinationComponents contains the coordination system components
type CoordinationComponents struct {
	Coordinator   *ComponentCoordinator
	MemoryManager *MemoryManager
	TimeoutManager *TimeoutManager
	LockManager   *LockManager
}

// NewCoordinationComponents creates the coordination system components
func NewCoordinationComponents(logger *zap.Logger) *CoordinationComponents {
	// Create the lock manager
	lockManager := NewLockManager(DefaultLockManagerConfig(), logger)
	
	// Create the component coordinator
	coordinator := NewComponentCoordinator(DefaultCoordinatorConfig(), logger)
	
	return &CoordinationComponents{
		Coordinator:   coordinator,
		MemoryManager: coordinator.GetMemoryManager(),
		TimeoutManager: coordinator.GetTimeoutManager(),
		LockManager:   lockManager,
	}
}

