package fx

import (
	"github.com/abdoElHodaky/tradSys/internal/architecture/fx/lazy"
	"github.com/abdoElHodaky/tradSys/internal/risk"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// LazyRiskModule provides lazily loaded risk management components
var LazyRiskModule = fx.Options(
	// Provide lazily loaded risk components
	provideLazyRiskManager,
	provideLazyRiskLimitChecker,
	provideLazyRiskReporter,
	
	// Register lifecycle hooks
	fx.Invoke(registerLazyRiskHooks),
)

// provideLazyRiskManager provides a lazily loaded risk manager
func provideLazyRiskManager(logger *zap.Logger, metrics *lazy.LazyLoadingMetrics) *lazy.LazyProvider {
	return lazy.NewLazyProvider(
		"risk-manager",
		func(config *risk.RiskConfig, logger *zap.Logger) (*risk.RiskManager, error) {
			logger.Info("Lazily initializing risk manager")
			return risk.NewRiskManager(config, logger)
		},
		logger,
		metrics,
	)
}

// provideLazyRiskLimitChecker provides a lazily loaded risk limit checker
func provideLazyRiskLimitChecker(logger *zap.Logger, metrics *lazy.LazyLoadingMetrics) *lazy.LazyProvider {
	return lazy.NewLazyProvider(
		"risk-limit-checker",
		func(config *risk.RiskConfig, logger *zap.Logger) (*risk.RiskLimitChecker, error) {
			logger.Info("Lazily initializing risk limit checker")
			return risk.NewRiskLimitChecker(config, logger)
		},
		logger,
		metrics,
	)
}

// provideLazyRiskReporter provides a lazily loaded risk reporter
func provideLazyRiskReporter(logger *zap.Logger, metrics *lazy.LazyLoadingMetrics) *lazy.LazyProvider {
	return lazy.NewLazyProvider(
		"risk-reporter",
		func(logger *zap.Logger) (*risk.RiskReporter, error) {
			logger.Info("Lazily initializing risk reporter")
			return risk.NewRiskReporter(logger)
		},
		logger,
		metrics,
	)
}

// registerLazyRiskHooks registers lifecycle hooks for the lazy risk components
func registerLazyRiskHooks(
	lc fx.Lifecycle,
	logger *zap.Logger,
	riskManagerProvider *lazy.LazyProvider,
	riskLimitCheckerProvider *lazy.LazyProvider,
	riskReporterProvider *lazy.LazyProvider,
) {
	logger.Info("Registering lazy risk component hooks")
}

// GetRiskManager gets the risk manager, initializing it if necessary
func GetRiskManager(provider *lazy.LazyProvider) (*risk.RiskManager, error) {
	instance, err := provider.Get()
	if err != nil {
		return nil, err
	}
	return instance.(*risk.RiskManager), nil
}

// GetRiskLimitChecker gets the risk limit checker, initializing it if necessary
func GetRiskLimitChecker(provider *lazy.LazyProvider) (*risk.RiskLimitChecker, error) {
	instance, err := provider.Get()
	if err != nil {
		return nil, err
	}
	return instance.(*risk.RiskLimitChecker), nil
}

// GetRiskReporter gets the risk reporter, initializing it if necessary
func GetRiskReporter(provider *lazy.LazyProvider) (*risk.RiskReporter, error) {
	instance, err := provider.Get()
	if err != nil {
		return nil, err
	}
	return instance.(*risk.RiskReporter), nil
}

