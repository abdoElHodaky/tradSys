// Package assets provides unified asset system with Fx dependency injection
package assets

import (
	"context"
	"time"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

// AssetsModule provides the Fx module for assets service
// This module integrates the unified asset system with fx for dependency injection
var AssetsModule = fx.Options(
	// Core service providers
	fx.Provide(
		NewAssetRegistry,
		NewUnifiedPricingEngine,
		NewCrossExchangePortfolioManager,
		NewUnifiedAnalyticsEngine,
		NewUnifiedComplianceManager,
		NewUnifiedLicensingManager,
		NewUnifiedConfigManager,
		NewUnifiedReportingEngine,
		NewUnifiedPerformanceMonitor,
	),

	// Main service provider
	fx.Provide(NewUnifiedAssetSystem),

	// Lifecycle management
	fx.Invoke(setupAssetsServices),
)

// AssetsServices aggregates all assets service components
type AssetsServices struct {
	AssetSystem    *UnifiedAssetSystem
	AssetRegistry  *AssetRegistry
	PricingEngine  *UnifiedPricingEngine
	PortfolioMgr   *CrossExchangePortfolioManager
	AnalyticsEngine *UnifiedAnalyticsEngine
	ComplianceMgr  *UnifiedComplianceManager
	LicensingMgr   *UnifiedLicensingManager
	ConfigMgr      *UnifiedConfigManager
}

// setupAssetsServices initializes assets services on startup
func setupAssetsServices(lc fx.Lifecycle, services *AssetsServices, logger *zap.Logger) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info("Initializing assets services")
			logger.Info("Assets services initialized")
			return nil
		},

		OnStop: func(ctx context.Context) error {
			logger.Info("Shutting down assets services")
			shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()
			services.AssetSystem.Shutdown(shutdownCtx)
			logger.Info("Assets services shutdown complete")
			return nil
		},
	})
}

// NewAssetsServices creates an aggregated services object
func NewAssetsServices(
	system *UnifiedAssetSystem,
	registry *AssetRegistry,
	pricing *UnifiedPricingEngine,
	portfolio *CrossExchangePortfolioManager,
	analytics *UnifiedAnalyticsEngine,
	compliance *UnifiedComplianceManager,
	licensing *UnifiedLicensingManager,
	config *UnifiedConfigManager,
) *AssetsServices {
	return &AssetsServices{
		AssetSystem:     system,
		AssetRegistry:   registry,
		PricingEngine:   pricing,
		PortfolioMgr:    portfolio,
		AnalyticsEngine: analytics,
		ComplianceMgr:   compliance,
		LicensingMgr:    licensing,
		ConfigMgr:       config,
	}
}