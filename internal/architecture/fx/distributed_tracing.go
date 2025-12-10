package fx

import (
	"context"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

// TracingModule provides the distributed tracing components
var TracingModule = fx.Options(
	// Provide the distributed tracer
	fx.Provide(NewDistributedTracer),

	// Register lifecycle hooks
	fx.Invoke(registerTracingHooks),
)

// TracingConfig contains configuration for distributed tracing
type TracingConfig struct {
	// Enabled determines if tracing is enabled
	Enabled bool

	// SamplingRate is the rate at which traces are sampled (1 in N)
	SamplingRate int

	// ExportEndpoint is the endpoint to export traces to
	ExportEndpoint string

	// ServiceName is the name of the service
	ServiceName string
}

// DefaultTracingConfig returns the default tracing configuration
func DefaultTracingConfig() TracingConfig {
	return TracingConfig{
		Enabled:        true,
		SamplingRate:   10,
		ExportEndpoint: "http://localhost:14268/api/traces",
		ServiceName:    "tradSys",
	}
}

// NewDistributedTracer creates a new distributed tracer
func NewDistributedTracer(logger *zap.Logger) *integration.DistributedTracer {
	// Create the tracing configuration
	config := integration.DefaultTracingConfig()

	// Create the distributed tracer
	return integration.NewDistributedTracer(logger, config)
}

// registerTracingHooks registers lifecycle hooks for the distributed tracer
func registerTracingHooks(
	lc fx.Lifecycle,
	logger *zap.Logger,
	tracer *integration.DistributedTracer,
) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info("Starting distributed tracer")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("Stopping distributed tracer")
			return nil
		},
	})
}
