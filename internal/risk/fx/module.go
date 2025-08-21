package fx

import (
	"context"

	"github.com/abdoElHodaky/tradSys/internal/architecture/fx/resilience"
	"github.com/abdoElHodaky/tradSys/internal/risk"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Module provides the risk management components
var Module = fx.Options(
	// Provide the position limit manager
	fx.Provide(NewPositionLimitManager),

	// Provide the exposure tracker
	fx.Provide(NewExposureTracker),

	// Provide the risk validator
	fx.Provide(NewRiskValidator),

	// Provide the risk reporter
	fx.Provide(NewRiskReporter),

	// Provide the risk manager
	fx.Provide(NewRiskManager),

	// Register lifecycle hooks
	fx.Invoke(registerRiskHooks),
)

// PositionLimitManagerParams contains parameters for creating a PositionLimitManager
type PositionLimitManagerParams struct {
	fx.In

	Logger *zap.Logger
}

// NewPositionLimitManager creates a new PositionLimitManager
func NewPositionLimitManager(params PositionLimitManagerParams) *risk.PositionLimitManager {
	return risk.NewPositionLimitManager(params.Logger)
}

// ExposureTrackerParams contains parameters for creating an ExposureTracker
type ExposureTrackerParams struct {
	fx.In

	Logger *zap.Logger
}

// NewExposureTracker creates a new ExposureTracker
func NewExposureTracker(params ExposureTrackerParams) *risk.ExposureTracker {
	return risk.NewExposureTracker(params.Logger)
}

// RiskValidatorParams contains parameters for creating a RiskValidator
type RiskValidatorParams struct {
	fx.In

	Logger              *zap.Logger
	PositionLimitManager *risk.PositionLimitManager
	ExposureTracker     *risk.ExposureTracker
	CircuitBreakerFactory *resilience.CircuitBreakerFactory
}

// NewRiskValidator creates a new RiskValidator
func NewRiskValidator(params RiskValidatorParams) *risk.RiskValidator {
	return risk.NewRiskValidator(
		params.Logger,
		params.PositionLimitManager,
		params.ExposureTracker,
		params.CircuitBreakerFactory,
	)
}

// RiskReporterParams contains parameters for creating a RiskReporter
type RiskReporterParams struct {
	fx.In

	Logger          *zap.Logger
	ExposureTracker *risk.ExposureTracker
}

// NewRiskReporter creates a new RiskReporter
func NewRiskReporter(params RiskReporterParams) *risk.RiskReporter {
	return risk.NewRiskReporter(
		params.Logger,
		params.ExposureTracker,
	)
}

// RiskManagerParams contains parameters for creating a RiskManager
type RiskManagerParams struct {
	fx.In

	Logger              *zap.Logger
	PositionLimitManager *risk.PositionLimitManager
	ExposureTracker     *risk.ExposureTracker
	RiskValidator       *risk.RiskValidator
	RiskReporter        *risk.RiskReporter
}

// NewRiskManager creates a new RiskManager
func NewRiskManager(params RiskManagerParams) *risk.RiskManager {
	return risk.NewRiskManager(
		params.Logger,
		params.PositionLimitManager,
		params.ExposureTracker,
		params.RiskValidator,
		params.RiskReporter,
	)
}

// registerRiskHooks registers lifecycle hooks for risk components
func registerRiskHooks(
	lc fx.Lifecycle,
	logger *zap.Logger,
	manager *risk.RiskManager,
) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info("Starting risk management components")
			return manager.Start(ctx)
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("Stopping risk management components")
			return manager.Stop(ctx)
		},
	})
}

