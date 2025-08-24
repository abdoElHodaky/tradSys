package lazy

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/architecture/fx/lazy"
	"github.com/abdoElHodaky/tradSys/internal/risk/management"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Module provides fx components for lazy-loaded risk management
var Module = fx.Options(
	fx.Provide(NewRiskManagementModule),
	fx.Provide(NewRiskManagerProvider),
	fx.Provide(NewRiskRuleEngineRegistry),
	fx.Provide(NewRiskLimitProviderFactory),
)

// RiskManagementModule coordinates lazy loading of risk management components
type RiskManagementModule struct {
	logger              *zap.Logger
	metrics             *lazy.AdaptiveMetrics
	initManager         *lazy.InitializationManager
	contextPropagator   *lazy.ContextPropagator
	riskManagerProvider *RiskManagerProvider
	ruleEngineRegistry  *RiskRuleEngineRegistry
	limitFactory        *RiskLimitProviderFactory
	limitProviders      map[string]*RiskLimitProvider
	limitProvidersMu    sync.RWMutex
}

// NewRiskManagementModule creates a new risk management module
func NewRiskManagementModule(
	logger *zap.Logger,
	metrics *lazy.AdaptiveMetrics,
	initManager *lazy.InitializationManager,
	contextPropagator *lazy.ContextPropagator,
	riskManagerProvider *RiskManagerProvider,
	ruleEngineRegistry *RiskRuleEngineRegistry,
	limitFactory *RiskLimitProviderFactory,
) *RiskManagementModule {
	return &RiskManagementModule{
		logger:              logger,
		metrics:             metrics,
		initManager:         initManager,
		contextPropagator:   contextPropagator,
		riskManagerProvider: riskManagerProvider,
		ruleEngineRegistry:  ruleEngineRegistry,
		limitFactory:        limitFactory,
		limitProviders:      make(map[string]*RiskLimitProvider),
	}
}

// Initialize initializes the risk management module
func (m *RiskManagementModule) Initialize(ctx context.Context) error {
	m.logger.Info("Initializing risk management module")
	startTime := time.Now()
	
	// Register providers with initialization manager
	m.initManager.RegisterProvider(m.riskManagerProvider.lazyProvider)
	
	// Register rule engine providers
	for _, provider := range m.ruleEngineRegistry.GetAllProviders() {
		m.initManager.RegisterProvider(provider.lazyProvider)
	}
	
	// Warm up critical components
	warmupCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	
	m.initManager.WarmupComponents(warmupCtx)
	
	m.logger.Info("Risk management module initialized",
		zap.Duration("duration", time.Since(startTime)))
	
	return nil
}

// GetRiskManager returns the risk manager
func (m *RiskManagementModule) GetRiskManager() (*management.RiskManager, error) {
	return m.riskManagerProvider.Get()
}

// GetRiskManagerWithContext returns the risk manager with context
func (m *RiskManagementModule) GetRiskManagerWithContext(ctx context.Context) (*management.RiskManager, error) {
	return m.riskManagerProvider.GetWithContext(ctx)
}

// GetRuleEngine returns a rule engine by type
func (m *RiskManagementModule) GetRuleEngine(engineType string) (management.RuleEngine, error) {
	provider, err := m.ruleEngineRegistry.GetProvider(engineType)
	if err != nil {
		return nil, err
	}
	
	return provider.Get()
}

// GetRuleEngineWithContext returns a rule engine by type with context
func (m *RiskManagementModule) GetRuleEngineWithContext(ctx context.Context, engineType string) (management.RuleEngine, error) {
	provider, err := m.ruleEngineRegistry.GetProvider(engineType)
	if err != nil {
		return nil, err
	}
	
	return provider.GetWithContext(ctx)
}

// GetLimit returns a limit by type
func (m *RiskManagementModule) GetLimit(limitType string) (management.Limit, error) {
	provider, err := m.getOrCreateLimitProvider(limitType)
	if err != nil {
		return nil, err
	}
	
	return provider.Get()
}

// GetLimitWithContext returns a limit by type with context
func (m *RiskManagementModule) GetLimitWithContext(ctx context.Context, limitType string) (management.Limit, error) {
	provider, err := m.getOrCreateLimitProvider(limitType)
	if err != nil {
		return nil, err
	}
	
	return provider.GetWithContext(ctx)
}

// getOrCreateLimitProvider gets or creates a limit provider
func (m *RiskManagementModule) getOrCreateLimitProvider(limitType string) (*RiskLimitProvider, error) {
	// Check if provider exists
	m.limitProvidersMu.RLock()
	provider, ok := m.limitProviders[limitType]
	m.limitProvidersMu.RUnlock()
	
	if ok {
		return provider, nil
	}
	
	// Create new provider
	m.limitProvidersMu.Lock()
	defer m.limitProvidersMu.Unlock()
	
	// Check again in case another goroutine created it
	provider, ok = m.limitProviders[limitType]
	if ok {
		return provider, nil
	}
	
	// Create new provider
	provider = m.limitFactory.CreateProvider(limitType)
	
	// Register with initialization manager
	m.initManager.RegisterProvider(provider.lazyProvider)
	
	// Store provider
	m.limitProviders[limitType] = provider
	
	return provider, nil
}

// RiskRuleEngineRegistry manages rule engine providers
type RiskRuleEngineRegistry struct {
	logger    *zap.Logger
	metrics   *lazy.AdaptiveMetrics
	providers map[string]*RiskRuleEngineProvider
	mu        sync.RWMutex
}

// NewRiskRuleEngineRegistry creates a new rule engine registry
func NewRiskRuleEngineRegistry(
	logger *zap.Logger,
	metrics *lazy.AdaptiveMetrics,
) *RiskRuleEngineRegistry {
	return &RiskRuleEngineRegistry{
		logger:    logger,
		metrics:   metrics,
		providers: make(map[string]*RiskRuleEngineProvider),
	}
}

// RegisterProvider registers a rule engine provider
func (r *RiskRuleEngineRegistry) RegisterProvider(provider *RiskRuleEngineProvider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	r.providers[provider.GetEngineType()] = provider
}

// GetProvider gets a rule engine provider by type
func (r *RiskRuleEngineRegistry) GetProvider(engineType string) (*RiskRuleEngineProvider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	provider, ok := r.providers[engineType]
	if !ok {
		return nil, fmt.Errorf("rule engine provider not found: %s", engineType)
	}
	
	return provider, nil
}

// GetAllProviders gets all rule engine providers
func (r *RiskRuleEngineRegistry) GetAllProviders() []*RiskRuleEngineProvider {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	providers := make([]*RiskRuleEngineProvider, 0, len(r.providers))
	for _, provider := range r.providers {
		providers = append(providers, provider)
	}
	
	return providers
}

// RegisterStandardEngines registers standard rule engines
func (r *RiskRuleEngineRegistry) RegisterStandardEngines(config management.RuleEngineConfig) {
	// Register standard rule engines
	standardEngines := []string{
		"position-limit",
		"exposure-limit",
		"volatility-based",
		"correlation-based",
		"liquidity-based",
	}
	
	for _, engineType := range standardEngines {
		provider := NewRiskRuleEngineProvider(r.logger, r.metrics, engineType, config)
		r.RegisterProvider(provider)
	}
}

