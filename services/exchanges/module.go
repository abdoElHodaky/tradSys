// Package exchanges provides exchange services with Fx dependency injection
package exchanges

import (
	"context"
	"time"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

// ExchangeModule provides the Fx module for exchange services
// This module unifies ADX, EGX, and Islamic exchange services
var ExchangeModule = fx.Options(
	// Core types
	fx.Provide(
		NewExchangeService,
		NewADXService,
		NewEGXService,
	),

	// Shared types (exported for reuse)
	fx.Provide(
		NewIslamicCompliance,
		NewUAECompliance,
		NewPerformanceMonitor,
	),

	// Lifecycle management
	fx.Invoke(setupExchangeServices),
)

// ExchangeServices aggregates all exchange services
type ExchangeServices struct {
	ADX  *ADXService
	EGX  *EGXService
	IS   *IslamicCompliance
	UAE  *UAECompliance
}

// NewExchangeService creates the exchange services aggregator
func NewExchangeService(
	adx *ADXService,
	egx *EGXService,
	is *IslamicCompliance,
	uae *UAECompliance,
) *ExchangeServices {
	return &ExchangeServices{
		ADX: adx,
		EGX: egx,
		IS:  is,
		UAE: uae,
	}
}

// setupExchangeServices initializes exchange services on startup
func setupExchangeServices(lc fx.Lifecycle, services *ExchangeServices, logger *zap.Logger) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info("Initializing exchange services")

			// Start ADX service
			if err := services.ADX.connector.Connect(); err != nil {
				logger.Error("Failed to connect to ADX", zap.Error(err))
			}

			// Start EGX service
			if err := services.EGX.connector.Connect(); err != nil {
				logger.Error("Failed to connect to EGX", zap.Error(err))
			}

			// Initialize Islamic compliance
			services.IS.LoadShariaRules()

			logger.Info("All exchange services initialized")
			return nil
		},

		OnStop: func(ctx context.Context) error {
			logger.Info("Shutting down exchange services")

			// Graceful shutdown
			shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()

			services.EGX.Shutdown(shutdownCtx)
			services.ADX.Shutdown(shutdownCtx)

			logger.Info("Exchange services shutdown complete")
			return nil
		},
	})
}

// FederationConfig provides configuration for federated exchange services
type FederationConfig struct {
	Enabled      bool
	SyncInterval time.Duration
	Exchanges    []string
}

// NewFederationConfig creates a new federation configuration
func NewFederationConfig() *FederationConfig {
	return &FederationConfig{
		Enabled:      false,
		SyncInterval: 5 * time.Minute,
		Exchanges:    []string{"ADX", "EGX"},
	}
}