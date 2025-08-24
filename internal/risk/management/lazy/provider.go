package lazy

import (
	"context"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/architecture/fx/lazy"
	"github.com/abdoElHodaky/tradSys/internal/risk/management"
	"go.uber.org/zap"
)

// RiskManagerProvider provides lazy loading for risk management components
type RiskManagerProvider struct {
	lazyProvider *lazy.EnhancedLazyProvider
	logger       *zap.Logger
}

// NewRiskManagerProvider creates a new provider for risk manager
func NewRiskManagerProvider(
	logger *zap.Logger,
	metrics *lazy.AdaptiveMetrics,
	config management.RiskManagerConfig,
) *RiskManagerProvider {
	return &RiskManagerProvider{
		lazyProvider: lazy.NewEnhancedLazyProvider(
			"risk-manager",
			func(logger *zap.Logger) (interface{}, error) {
				logger.Info("Initializing risk manager")
				startTime := time.Now()
				
				// This is typically an expensive operation
				riskManager, err := management.NewRiskManager(config, logger)
				if err != nil {
					return nil, err
				}
				
				logger.Info("Risk manager initialized",
					zap.Duration("duration", time.Since(startTime)))
				
				return riskManager, nil
			},
			logger,
			metrics,
			lazy.WithPriority(20), // Higher priority (lower number)
			lazy.WithTimeout(45*time.Second),
			lazy.WithMemoryEstimate(50*1024*1024), // 50MB estimate
		),
		logger: logger,
	}
}

// Get returns the risk manager, initializing it if necessary
func (p *RiskManagerProvider) Get() (*management.RiskManager, error) {
	instance, err := p.lazyProvider.Get()
	if err != nil {
		return nil, err
	}
	return instance.(*management.RiskManager), nil
}

// GetWithContext returns the risk manager with context timeout
func (p *RiskManagerProvider) GetWithContext(ctx context.Context) (*management.RiskManager, error) {
	instance, err := p.lazyProvider.GetWithContext(ctx)
	if err != nil {
		return nil, err
	}
	return instance.(*management.RiskManager), nil
}

// IsInitialized returns whether the risk manager has been initialized
func (p *RiskManagerProvider) IsInitialized() bool {
	return p.lazyProvider.IsInitialized()
}

// RiskRuleEngineProvider provides lazy loading for risk rule engines
type RiskRuleEngineProvider struct {
	lazyProvider *lazy.EnhancedLazyProvider
	logger       *zap.Logger
	engineType   string
}

// NewRiskRuleEngineProvider creates a new provider for risk rule engine
func NewRiskRuleEngineProvider(
	logger *zap.Logger,
	metrics *lazy.AdaptiveMetrics,
	engineType string,
	config management.RuleEngineConfig,
) *RiskRuleEngineProvider {
	return &RiskRuleEngineProvider{
		lazyProvider: lazy.NewEnhancedLazyProvider(
			"risk-rule-engine-"+engineType,
			func(logger *zap.Logger) (interface{}, error) {
				logger.Info("Initializing risk rule engine", 
					zap.String("type", engineType))
				startTime := time.Now()
				
				// This is typically an expensive operation
				engine, err := management.NewRuleEngine(engineType, config, logger)
				if err != nil {
					return nil, err
				}
				
				logger.Info("Risk rule engine initialized",
					zap.Duration("duration", time.Since(startTime)),
					zap.String("type", engineType))
				
				return engine, nil
			},
			logger,
			metrics,
			lazy.WithPriority(30), // Medium priority
			lazy.WithTimeout(30*time.Second),
			lazy.WithMemoryEstimate(20*1024*1024), // 20MB estimate
		),
		logger:     logger,
		engineType: engineType,
	}
}

// Get returns the risk rule engine, initializing it if necessary
func (p *RiskRuleEngineProvider) Get() (management.RuleEngine, error) {
	instance, err := p.lazyProvider.Get()
	if err != nil {
		return nil, err
	}
	return instance.(management.RuleEngine), nil
}

// GetWithContext returns the risk rule engine with context timeout
func (p *RiskRuleEngineProvider) GetWithContext(ctx context.Context) (management.RuleEngine, error) {
	instance, err := p.lazyProvider.GetWithContext(ctx)
	if err != nil {
		return nil, err
	}
	return instance.(management.RuleEngine), nil
}

// IsInitialized returns whether the risk rule engine has been initialized
func (p *RiskRuleEngineProvider) IsInitialized() bool {
	return p.lazyProvider.IsInitialized()
}

// GetEngineType returns the engine type
func (p *RiskRuleEngineProvider) GetEngineType() string {
	return p.engineType
}

// RiskLimitProviderFactory creates risk limit providers
type RiskLimitProviderFactory struct {
	logger  *zap.Logger
	metrics *lazy.AdaptiveMetrics
	config  management.LimitConfig
}

// NewRiskLimitProviderFactory creates a new risk limit provider factory
func NewRiskLimitProviderFactory(
	logger *zap.Logger,
	metrics *lazy.AdaptiveMetrics,
	config management.LimitConfig,
) *RiskLimitProviderFactory {
	return &RiskLimitProviderFactory{
		logger:  logger,
		metrics: metrics,
		config:  config,
	}
}

// CreateProvider creates a risk limit provider for a specific limit type
func (f *RiskLimitProviderFactory) CreateProvider(limitType string) *RiskLimitProvider {
	return NewRiskLimitProvider(f.logger, f.metrics, limitType, f.config)
}

// RiskLimitProvider provides lazy loading for risk limits
type RiskLimitProvider struct {
	lazyProvider *lazy.EnhancedLazyProvider
	logger       *zap.Logger
	limitType    string
}

// NewRiskLimitProvider creates a new provider for risk limits
func NewRiskLimitProvider(
	logger *zap.Logger,
	metrics *lazy.AdaptiveMetrics,
	limitType string,
	config management.LimitConfig,
) *RiskLimitProvider {
	return &RiskLimitProvider{
		lazyProvider: lazy.NewEnhancedLazyProvider(
			"risk-limit-"+limitType,
			func(logger *zap.Logger) (interface{}, error) {
				logger.Info("Initializing risk limit", 
					zap.String("type", limitType))
				startTime := time.Now()
				
				// This is typically an expensive operation
				limit, err := management.NewLimit(limitType, config, logger)
				if err != nil {
					return nil, err
				}
				
				logger.Info("Risk limit initialized",
					zap.Duration("duration", time.Since(startTime)),
					zap.String("type", limitType))
				
				return limit, nil
			},
			logger,
			metrics,
			lazy.WithPriority(40), // Lower priority
			lazy.WithTimeout(15*time.Second),
			lazy.WithMemoryEstimate(5*1024*1024), // 5MB estimate
		),
		logger:    logger,
		limitType: limitType,
	}
}

// Get returns the risk limit, initializing it if necessary
func (p *RiskLimitProvider) Get() (management.Limit, error) {
	instance, err := p.lazyProvider.Get()
	if err != nil {
		return nil, err
	}
	return instance.(management.Limit), nil
}

// GetWithContext returns the risk limit with context timeout
func (p *RiskLimitProvider) GetWithContext(ctx context.Context) (management.Limit, error) {
	instance, err := p.lazyProvider.GetWithContext(ctx)
	if err != nil {
		return nil, err
	}
	return instance.(management.Limit), nil
}

// IsInitialized returns whether the risk limit has been initialized
func (p *RiskLimitProvider) IsInitialized() bool {
	return p.lazyProvider.IsInitialized()
}

// GetLimitType returns the limit type
func (p *RiskLimitProvider) GetLimitType() string {
	return p.limitType
}

