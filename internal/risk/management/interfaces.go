package management

import (
	"context"
)

// RiskManager manages risk for trading operations
type RiskManager struct {
	// Configuration for the risk manager
	Config RiskManagerConfig
	
	// Rule engines for risk evaluation
	RuleEngines map[string]RuleEngine
	
	// Limits for risk control
	Limits map[string]Limit
}

// RiskManagerConfig contains configuration for the risk manager
type RiskManagerConfig struct {
	// Default risk limits
	DefaultLimits map[string]float64
	
	// Circuit breaker thresholds
	CircuitBreakerThresholds map[string]float64
	
	// Risk evaluation frequency
	EvaluationFrequencyMs int
	
	// Enable/disable features
	Features map[string]bool
}

// RuleEngine defines the interface for risk rule engines
type RuleEngine interface {
	// Evaluate evaluates risk for a given context
	Evaluate(ctx context.Context, params map[string]interface{}) (RiskEvaluation, error)
	
	// GetType returns the type of rule engine
	GetType() string
	
	// GetConfig returns the configuration for the rule engine
	GetConfig() RuleEngineConfig
}

// RuleEngineConfig contains configuration for rule engines
type RuleEngineConfig struct {
	// Parameters for the rule engine
	Parameters map[string]interface{}
	
	// Thresholds for risk levels
	Thresholds map[string]float64
	
	// Priority of the rule engine
	Priority int
}

// RiskEvaluation represents the result of a risk evaluation
type RiskEvaluation struct {
	// Risk level (0-100)
	RiskLevel float64
	
	// Risk factors contributing to the evaluation
	RiskFactors map[string]float64
	
	// Whether the risk is acceptable
	Acceptable bool
	
	// Reason for the evaluation result
	Reason string
}

// Limit defines the interface for risk limits
type Limit interface {
	// Check checks if an operation is within limits
	Check(ctx context.Context, params map[string]interface{}) (bool, error)
	
	// GetType returns the type of limit
	GetType() string
	
	// GetValue returns the current value of the limit
	GetValue() float64
	
	// GetThreshold returns the threshold for the limit
	GetThreshold() float64
}

// NewRiskManager creates a new risk manager
func NewRiskManager(config RiskManagerConfig) *RiskManager {
	return &RiskManager{
		Config:      config,
		RuleEngines: make(map[string]RuleEngine),
		Limits:      make(map[string]Limit),
	}
}

// RegisterRuleEngine registers a rule engine with the risk manager
func (rm *RiskManager) RegisterRuleEngine(engine RuleEngine) {
	rm.RuleEngines[engine.GetType()] = engine
}

// RegisterLimit registers a limit with the risk manager
func (rm *RiskManager) RegisterLimit(limit Limit) {
	rm.Limits[limit.GetType()] = limit
}

// EvaluateRisk evaluates risk for a given context
func (rm *RiskManager) EvaluateRisk(ctx context.Context, params map[string]interface{}) (RiskEvaluation, error) {
	// Default evaluation
	evaluation := RiskEvaluation{
		RiskLevel:   0,
		RiskFactors: make(map[string]float64),
		Acceptable:  true,
		Reason:      "No risk factors evaluated",
	}
	
	// Apply each rule engine
	for _, engine := range rm.RuleEngines {
		engineEval, err := engine.Evaluate(ctx, params)
		if err != nil {
			return evaluation, err
		}
		
		// Combine risk factors
		for factor, value := range engineEval.RiskFactors {
			evaluation.RiskFactors[factor] = value
		}
		
		// Update risk level (use max of all engines)
		if engineEval.RiskLevel > evaluation.RiskLevel {
			evaluation.RiskLevel = engineEval.RiskLevel
		}
		
		// If any engine says unacceptable, the overall result is unacceptable
		if !engineEval.Acceptable {
			evaluation.Acceptable = false
			evaluation.Reason = engineEval.Reason
		}
	}
	
	return evaluation, nil
}

// CheckLimits checks if an operation is within all limits
func (rm *RiskManager) CheckLimits(ctx context.Context, params map[string]interface{}) (bool, string, error) {
	for limitType, limit := range rm.Limits {
		withinLimit, err := limit.Check(ctx, params)
		if err != nil {
			return false, "", err
		}
		
		if !withinLimit {
			return false, limitType + " limit exceeded", nil
		}
	}
	
	return true, "", nil
}

