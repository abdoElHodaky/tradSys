// Package routing implements Plan 5: Intelligent Routing System for TradSys v3
// Provides multi-dimensional routing with context-aware decisions
package routing

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// IntelligentRouter provides multi-dimensional routing capabilities
type IntelligentRouter struct {
	routingEngine      *RoutingEngine
	contextAnalyzer    *ContextAnalyzer
	decisionEngine     *DecisionEngine
	routingCache       *RoutingCache
	loadBalancer       *LoadBalancer
	circuitBreaker     *CircuitBreaker
	performanceMonitor *PerformanceMonitor
	mu                 sync.RWMutex
}

// RoutingContext contains all context information for routing decisions
type RoutingContext struct {
	UserID           string
	SessionID        string
	Exchange         ExchangeType
	AssetType        AssetType
	OrderType        OrderType
	LicenseTier      LicenseTier
	IslamicCompliant bool
	Region           string
	Priority         RoutingPriority
	Metadata         map[string]interface{}
	Timestamp        time.Time
}

// RoutingDecision represents the result of a routing decision
type RoutingDecision struct {
	TargetService    string
	RoutingStrategy  RoutingStrategy
	LoadBalanceType  LoadBalanceType
	Confidence       float64
	Latency          time.Duration
	Metadata         map[string]interface{}
	FallbackServices []string
}

// ExchangeType defines supported exchanges
type ExchangeType int

const (
	ExchangeEGX ExchangeType = iota
	ExchangeADX
	ExchangeUnified
)

// AssetType defines supported asset types
type AssetType int

const (
	AssetStock AssetType = iota
	AssetBond
	AssetETF
	AssetMutualFund
	AssetREIT
	AssetCrypto
	AssetForex
	AssetCommodity
	AssetSukuk
	AssetIslamicFund
)

// OrderType defines order types
type OrderType int

const (
	OrderMarket OrderType = iota
	OrderLimit
	OrderStop
	OrderStopLimit
)

// LicenseTier defines license tiers
type LicenseTier int

const (
	LicenseBasic LicenseTier = iota
	LicenseProfessional
	LicenseEnterprise
	LicenseIslamic
)

// RoutingPriority defines routing priority levels
type RoutingPriority int

const (
	PriorityLow RoutingPriority = iota
	PriorityNormal
	PriorityHigh
	PriorityCritical
)

// RoutingStrategy defines routing strategies
type RoutingStrategy int

const (
	StrategyRoundRobin RoutingStrategy = iota
	StrategyWeightedRoundRobin
	StrategyLeastConnections
	StrategyLatencyBased
	StrategyRegionBased
	StrategyLicenseBased
	StrategyIslamicCompliant
)

// LoadBalanceType defines load balancing types
type LoadBalanceType int

const (
	LoadBalanceRoundRobin LoadBalanceType = iota
	LoadBalanceWeighted
	LoadBalanceLeastConnections
	LoadBalanceLatency
	LoadBalanceResource
)

// NewIntelligentRouter creates a new intelligent router instance
func NewIntelligentRouter() *IntelligentRouter {
	return &IntelligentRouter{
		routingEngine:      NewRoutingEngine(),
		contextAnalyzer:    NewContextAnalyzer(),
		decisionEngine:     NewDecisionEngine(),
		routingCache:       NewRoutingCache(),
		loadBalancer:       NewLoadBalancer(),
		circuitBreaker:     NewCircuitBreaker(),
		performanceMonitor: NewPerformanceMonitor(),
	}
}

// Route performs intelligent routing based on context
func (ir *IntelligentRouter) Route(ctx context.Context, routingCtx *RoutingContext) (*RoutingDecision, error) {
	startTime := time.Now()

	// Analyze context for routing dimensions
	analysis, err := ir.contextAnalyzer.Analyze(ctx, routingCtx)
	if err != nil {
		return nil, fmt.Errorf("context analysis failed: %w", err)
	}

	// Check routing cache first
	if cached := ir.routingCache.Get(analysis.CacheKey); cached != nil {
		ir.performanceMonitor.RecordCacheHit(routingCtx.Exchange, time.Since(startTime))
		return cached, nil
	}

	// Make routing decision
	decision, err := ir.decisionEngine.Decide(ctx, analysis)
	if err != nil {
		return nil, fmt.Errorf("routing decision failed: %w", err)
	}

	// Apply load balancing
	finalDecision, err := ir.loadBalancer.Balance(ctx, decision, routingCtx)
	if err != nil {
		return nil, fmt.Errorf("load balancing failed: %w", err)
	}

	// Check circuit breaker
	if !ir.circuitBreaker.AllowRequest(finalDecision.TargetService) {
		// Use fallback service
		if len(finalDecision.FallbackServices) > 0 {
			finalDecision.TargetService = finalDecision.FallbackServices[0]
		} else {
			return nil, fmt.Errorf("circuit breaker open and no fallback available")
		}
	}

	// Cache the decision
	ir.routingCache.Set(analysis.CacheKey, finalDecision, 5*time.Minute)

	// Record performance metrics
	finalDecision.Latency = time.Since(startTime)
	ir.performanceMonitor.RecordRouting(routingCtx.Exchange, finalDecision.Latency)

	return finalDecision, nil
}

// RouteToExchange routes requests to specific exchange services
func (ir *IntelligentRouter) RouteToExchange(ctx context.Context, exchange ExchangeType, routingCtx *RoutingContext) (*RoutingDecision, error) {
	routingCtx.Exchange = exchange

	switch exchange {
	case ExchangeEGX:
		return ir.routeToEGX(ctx, routingCtx)
	case ExchangeADX:
		return ir.routeToADX(ctx, routingCtx)
	case ExchangeUnified:
		return ir.routeToUnified(ctx, routingCtx)
	default:
		return nil, fmt.Errorf("unsupported exchange: %v", exchange)
	}
}

// routeToEGX handles EGX-specific routing
func (ir *IntelligentRouter) routeToEGX(ctx context.Context, routingCtx *RoutingContext) (*RoutingDecision, error) {
	// Apply EGX-specific routing logic
	decision := &RoutingDecision{
		TargetService:   "egx-service-1",
		RoutingStrategy: StrategyRegionBased,
		LoadBalanceType: LoadBalanceLatency,
		Confidence:      0.9,
		Metadata: map[string]interface{}{
			"exchange":     "EGX",
			"region":       "Cairo",
			"optimization": "latency",
		},
		FallbackServices: []string{"egx-service-2", "egx-service-3"},
	}

	return decision, nil
}

// routeToADX handles ADX-specific routing
func (ir *IntelligentRouter) routeToADX(ctx context.Context, routingCtx *RoutingContext) (*RoutingDecision, error) {
	// Apply ADX-specific routing logic with Islamic finance focus
	decision := &RoutingDecision{
		TargetService:   "adx-service-1",
		RoutingStrategy: StrategyIslamicCompliant,
		LoadBalanceType: LoadBalanceWeighted,
		Confidence:      0.95,
		Metadata: map[string]interface{}{
			"exchange":          "ADX",
			"region":            "UAE",
			"islamic_compliant": routingCtx.IslamicCompliant,
			"optimization":      "sharia_compliance",
		},
		FallbackServices: []string{"adx-service-2", "adx-service-3"},
	}

	return decision, nil
}

// routeToUnified handles unified cross-exchange routing
func (ir *IntelligentRouter) routeToUnified(ctx context.Context, routingCtx *RoutingContext) (*RoutingDecision, error) {
	// Apply unified routing logic
	decision := &RoutingDecision{
		TargetService:   "unified-service-1",
		RoutingStrategy: StrategyLatencyBased,
		LoadBalanceType: LoadBalanceLeastConnections,
		Confidence:      0.85,
		Metadata: map[string]interface{}{
			"exchange":     "UNIFIED",
			"optimization": "best_execution",
			"arbitrage":    true,
		},
		FallbackServices: []string{"unified-service-2", "unified-service-3"},
	}

	return decision, nil
}

// GetRoutingStats returns routing statistics
func (ir *IntelligentRouter) GetRoutingStats() map[string]interface{} {
	ir.mu.RLock()
	defer ir.mu.RUnlock()

	stats := make(map[string]interface{})

	// Get performance metrics
	stats["performance"] = ir.performanceMonitor.GetStats()

	// Get cache statistics
	stats["cache"] = ir.routingCache.GetStats()

	// Get circuit breaker status
	stats["circuit_breakers"] = ir.circuitBreaker.GetStatus()

	// Get load balancer metrics
	stats["load_balancer"] = ir.loadBalancer.GetMetrics()

	stats["timestamp"] = time.Now()

	return stats
}

// ValidateRoutingContext validates routing context
func (ir *IntelligentRouter) ValidateRoutingContext(ctx *RoutingContext) error {
	if ctx.UserID == "" {
		return fmt.Errorf("user ID is required")
	}
	if ctx.SessionID == "" {
		return fmt.Errorf("session ID is required")
	}
	if ctx.Timestamp.IsZero() {
		ctx.Timestamp = time.Now()
	}
	if ctx.Metadata == nil {
		ctx.Metadata = make(map[string]interface{})
	}
	return nil
}

// UpdateRoutingRules updates routing rules dynamically
func (ir *IntelligentRouter) UpdateRoutingRules(rules map[string]interface{}) error {
	ir.mu.Lock()
	defer ir.mu.Unlock()

	// Update routing engine rules
	if err := ir.routingEngine.UpdateRules(rules); err != nil {
		return fmt.Errorf("failed to update routing rules: %w", err)
	}

	// Clear cache to force re-evaluation
	ir.routingCache.Clear()

	log.Printf("Routing rules updated successfully")
	return nil
}

// GetHealthStatus returns the health status of the routing system
func (ir *IntelligentRouter) GetHealthStatus() map[string]interface{} {
	status := make(map[string]interface{})

	status["routing_engine"] = ir.routingEngine.IsHealthy()
	status["context_analyzer"] = ir.contextAnalyzer.IsHealthy()
	status["decision_engine"] = ir.decisionEngine.IsHealthy()
	status["cache"] = ir.routingCache.IsHealthy()
	status["load_balancer"] = ir.loadBalancer.IsHealthy()
	status["circuit_breaker"] = ir.circuitBreaker.IsHealthy()
	status["performance_monitor"] = ir.performanceMonitor.IsHealthy()
	status["timestamp"] = time.Now()

	return status
}
