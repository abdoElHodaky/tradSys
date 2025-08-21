package resilience

import (
	"go.uber.org/fx"
)

// Module provides all resilience components in a single Fx module
// This follows Fx's pattern of composing modules
var Module = fx.Options(
	// Include the circuit breaker module
	CircuitBreakerModule,
	
	// Additional resilience modules can be added here
	// BulkheadModule,
	// RetryModule,
	// TimeoutModule,
	// etc.
)

