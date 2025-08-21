package workerpool

import (
	"go.uber.org/fx"
)

// Module provides all worker pool components in a single Fx module
// This follows Fx's pattern of composing modules
var Module = fx.Options(
	// Include the worker pool module
	WorkerPoolModule,
	
	// Additional worker pool related modules can be added here
)

