package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Module provides the metrics components
var Module = fx.Options(
	// Provide the Prometheus registry
	fx.Provide(NewPrometheusRegistry),
	
	// Provide the metrics components
	fx.Provide(NewWebSocketMetrics),
	fx.Provide(NewPeerJSMetrics),
	
	// Register the metrics HTTP handler
	fx.Invoke(RegisterMetricsHandler),
)

// NewPrometheusRegistry creates a new Prometheus registry
func NewPrometheusRegistry() *prometheus.Registry {
	return prometheus.NewRegistry()
}

// RegisterMetricsHandler registers the metrics HTTP handler
func RegisterMetricsHandler(
	lifecycle fx.Lifecycle,
	registry *prometheus.Registry,
	logger *zap.Logger,
) {
	// Create the HTTP handler
	handler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	
	// Create the HTTP server
	server := &http.Server{
		Addr:    ":9090",
		Handler: handler,
	}
	
	// Register lifecycle hooks
	lifecycle.Append(fx.Hook{
		OnStart: func(context.Context) error {
			logger.Info("Starting metrics server", zap.String("addr", server.Addr))
			
			// Start the server in a goroutine
			go func() {
				if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					logger.Error("Metrics server error", zap.Error(err))
				}
			}()
			
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("Stopping metrics server")
			return server.Shutdown(ctx)
		},
	})
}

// MetricsParams contains parameters for metrics components
type MetricsParams struct {
	fx.In
	
	Registry *prometheus.Registry
	Logger   *zap.Logger
}

// NewWebSocketMetrics creates a new WebSocketMetrics
func NewWebSocketMetrics(params MetricsParams) *WebSocketMetrics {
	return NewWebSocketMetrics(params.Registry, params.Logger)
}

// NewPeerJSMetrics creates a new PeerJSMetrics
func NewPeerJSMetrics(params MetricsParams) *PeerJSMetrics {
	return NewPeerJSMetrics(params.Registry, params.Logger)
}

