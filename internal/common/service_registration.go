package common

import (
	"context"

	"go-micro.dev/v4/server"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// ServiceHandler represents a generic service handler interface
type ServiceHandler interface{}

// ServiceRegistrar represents a function that can register a handler with a server
type ServiceRegistrar func(server.Server, ServiceHandler) error

// ServiceRegistrationParams contains parameters for service registration
type ServiceRegistrationParams struct {
	fx.In

	Lifecycle fx.Lifecycle
	Logger    *zap.Logger
	Server    server.Server
	Handler   ServiceHandler
	Registrar ServiceRegistrar
}

// RegisterServiceHandler provides a unified way to register service handlers
func RegisterServiceHandler(serviceName string, registrar ServiceRegistrar) fx.Option {
	return fx.Invoke(func(
		lc fx.Lifecycle,
		logger *zap.Logger,
		server server.Server,
		handler ServiceHandler,
	) {
		// Register the handler with the service
		if err := registrar(server, handler); err != nil {
			logger.Fatal("Failed to register handler",
				zap.String("service", serviceName),
				zap.Error(err))
		}

		logger.Info("Service registered successfully",
			zap.String("service", serviceName))
	})
}

// MicroserviceApp creates a standardized fx application for microservices
func MicroserviceApp(serviceName string, modules ...fx.Option) *fx.App {
	logger, _ := zap.NewProduction()

	// Base modules that all services need
	baseModules := []fx.Option{
		fx.Supply(logger),
		fx.Provide(func() *zap.Logger { return logger }),
	}

	// Combine base modules with service-specific modules
	allModules := append(baseModules, modules...)

	// Add lifecycle management
	allModules = append(allModules, fx.Invoke(func(lc fx.Lifecycle, logger *zap.Logger) {
		lc.Append(fx.Hook{
			OnStart: func(ctx context.Context) error {
				logger.Info("Starting microservice", zap.String("service", serviceName))
				return nil
			},
			OnStop: func(ctx context.Context) error {
				logger.Info("Stopping microservice", zap.String("service", serviceName))
				return nil
			},
		})
	}))

	return fx.New(allModules...)
}
