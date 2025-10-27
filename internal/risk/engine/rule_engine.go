package engine

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// RuleEngine handles risk rule evaluation and management
type RuleEngine struct {
	config     *RiskEngineConfig
	logger     *zap.Logger
	rules      map[string]*RiskRule
	ruleGroups map[string]*RuleGroup
	mu         sync.RWMutex

	// Rule execution metrics
	rulesExecuted int64
	rulesFailed   int64
	executionTime int64 // nanoseconds
}

// RiskRule represents a single risk rule
type RiskRule struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Type        RiskRuleType           `json:"type"`
	Condition   *RuleCondition         `json:"condition"`
	Action      *RuleAction            `json:"action"`
	Priority    int                    `json:"priority"` // Higher number = higher priority
	Enabled     bool                   `json:"enabled"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// RiskRuleType defines the type of risk rule
type RiskRuleType int

const (
	RiskRuleTypePreTrade RiskRuleType = iota
	RiskRuleTypePostTrade
	RiskRuleTypePosition
	RiskRuleTypeMarketData
	RiskRuleTypePortfolio
	RiskRuleTypeCompliance
)

// String returns the string representation of the risk rule type
func (rrt RiskRuleType) String() string {
	switch rrt {
	case RiskRuleTypePreTrade:
		return "pre_trade"
	case RiskRuleTypePostTrade:
		return "post_trade"
	case RiskRuleTypePosition:
		return "position"
	case RiskRuleTypeMarketData:
		return "market_data"
	case RiskRuleTypePortfolio:
		return "portfolio"
	case RiskRuleTypeCompliance:
		return "compliance"
	default:
		return "unknown"
	}
}

// RuleCondition defines the condition for a risk rule
type RuleCondition struct {
	Field    string           `json:"field"`         // e.g., "order.quantity", "position.value"
	Operator string           `json:"operator"`      // e.g., ">", "<", "==", "!=", "in", "not_in"
	Value    interface{}      `json:"value"`         // The value to compare against
	And      []*RuleCondition `json:"and,omitempty"` // AND conditions
	Or       []*RuleCondition `json:"or,omitempty"`  // OR conditions
}

// RuleAction defines the action to take when a rule is triggered
type RuleAction struct {
	Type       RuleActionType         `json:"type"`
	Parameters map[string]interface{} `json:"parameters"`
}

// RuleActionType defines the type of action to take
type RuleActionType int

const (
	RuleActionReject RuleActionType = iota
	RuleActionLimit
	RuleActionWarn
	RuleActionLog
	RuleActionNotify
	RuleActionCircuitBreaker
)

// String returns the string representation of the rule action type
func (rat RuleActionType) String() string {
	switch rat {
	case RuleActionReject:
		return "reject"
	case RuleActionLimit:
		return "limit"
	case RuleActionWarn:
		return "warn"
	case RuleActionLog:
		return "log"
	case RuleActionNotify:
		return "notify"
	case RuleActionCircuitBreaker:
		return "circuit_breaker"
	default:
		return "unknown"
	}
}

// RuleGroup represents a group of related rules
type RuleGroup struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	RuleIDs     []string  `json:"rule_ids"`
	Enabled     bool      `json:"enabled"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// RuleEvaluationContext contains context for rule evaluation
type RuleEvaluationContext struct {
	Event     *RiskEvent             `json:"event"`
	UserID    string                 `json:"user_id"`
	Symbol    string                 `json:"symbol"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}

// RuleEvaluationResult contains the result of rule evaluation
type RuleEvaluationResult struct {
	RuleID        string                 `json:"rule_id"`
	Triggered     bool                   `json:"triggered"`
	Action        *RuleAction            `json:"action"`
	Message       string                 `json:"message"`
	Metadata      map[string]interface{} `json:"metadata"`
	ExecutionTime time.Duration          `json:"execution_time"`
}

// NewRuleEngine creates a new rule engine
func NewRuleEngine(config *RiskEngineConfig, logger *zap.Logger) *RuleEngine {
	return &RuleEngine{
		config:     config,
		logger:     logger,
		rules:      make(map[string]*RiskRule),
		ruleGroups: make(map[string]*RuleGroup),
	}
}

// AddRule adds a new risk rule
func (re *RuleEngine) AddRule(rule *RiskRule) error {
	re.mu.Lock()
	defer re.mu.Unlock()

	if rule.ID == "" {
		return fmt.Errorf("rule ID cannot be empty")
	}

	if _, exists := re.rules[rule.ID]; exists {
		return fmt.Errorf("rule with ID %s already exists", rule.ID)
	}

	// Set timestamps
	now := time.Now()
	rule.CreatedAt = now
	rule.UpdatedAt = now

	re.rules[rule.ID] = rule

	re.logger.Info("Risk rule added",
		zap.String("ruleID", rule.ID),
		zap.String("name", rule.Name),
		zap.String("type", rule.Type.String()),
		zap.Int("priority", rule.Priority),
	)

	return nil
}

// UpdateRule updates an existing risk rule
func (re *RuleEngine) UpdateRule(rule *RiskRule) error {
	re.mu.Lock()
	defer re.mu.Unlock()

	if _, exists := re.rules[rule.ID]; !exists {
		return fmt.Errorf("rule with ID %s not found", rule.ID)
	}

	rule.UpdatedAt = time.Now()
	re.rules[rule.ID] = rule

	re.logger.Info("Risk rule updated",
		zap.String("ruleID", rule.ID),
		zap.String("name", rule.Name),
	)

	return nil
}

// RemoveRule removes a risk rule
func (re *RuleEngine) RemoveRule(ruleID string) error {
	re.mu.Lock()
	defer re.mu.Unlock()

	if _, exists := re.rules[ruleID]; !exists {
		return fmt.Errorf("rule with ID %s not found", ruleID)
	}

	delete(re.rules, ruleID)

	re.logger.Info("Risk rule removed", zap.String("ruleID", ruleID))
	return nil
}

// GetRule retrieves a risk rule by ID
func (re *RuleEngine) GetRule(ruleID string) (*RiskRule, error) {
	re.mu.RLock()
	defer re.mu.RUnlock()

	rule, exists := re.rules[ruleID]
	if !exists {
		return nil, fmt.Errorf("rule with ID %s not found", ruleID)
	}

	return rule, nil
}

// ListRules returns all rules of a specific type
func (re *RuleEngine) ListRules(ruleType RiskRuleType) []*RiskRule {
	re.mu.RLock()
	defer re.mu.RUnlock()

	var rules []*RiskRule
	for _, rule := range re.rules {
		if rule.Type == ruleType && rule.Enabled {
			rules = append(rules, rule)
		}
	}

	return rules
}

// EvaluateRules evaluates all applicable rules for a given context
func (re *RuleEngine) EvaluateRules(ctx context.Context, evalCtx *RuleEvaluationContext) ([]*RuleEvaluationResult, error) {
	// Determine rule type based on event type
	var ruleType RiskRuleType
	switch evalCtx.Event.Type {
	case RiskEventPreTrade:
		ruleType = RiskRuleTypePreTrade
	case RiskEventPostTrade:
		ruleType = RiskRuleTypePostTrade
	case RiskEventPositionUpdate:
		ruleType = RiskRuleTypePosition
	case RiskEventMarketData:
		ruleType = RiskRuleTypeMarketData
	default:
		return nil, fmt.Errorf("unsupported event type for rule evaluation: %s", evalCtx.Event.Type.String())
	}

	// Get applicable rules
	rules := re.ListRules(ruleType)
	if len(rules) == 0 {
		return []*RuleEvaluationResult{}, nil
	}

	// Sort rules by priority (higher priority first)
	re.sortRulesByPriority(rules)

	// Evaluate rules
	results := make([]*RuleEvaluationResult, 0, len(rules))
	for _, rule := range rules {
		result, err := re.evaluateRule(ctx, rule, evalCtx)
		if err != nil {
			re.logger.Error("Failed to evaluate rule",
				zap.String("ruleID", rule.ID),
				zap.Error(err),
			)
			continue
		}
		results = append(results, result)

		// If rule triggered and action is reject, stop evaluation
		if result.Triggered && result.Action.Type == RuleActionReject {
			break
		}
	}

	return results, nil
}

// evaluateRule evaluates a single rule
func (re *RuleEngine) evaluateRule(ctx context.Context, rule *RiskRule, evalCtx *RuleEvaluationContext) (*RuleEvaluationResult, error) {
	startTime := time.Now()

	result := &RuleEvaluationResult{
		RuleID:   rule.ID,
		Action:   rule.Action,
		Metadata: make(map[string]interface{}),
	}

	// Evaluate condition
	triggered, err := re.evaluateCondition(rule.Condition, evalCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate condition: %w", err)
	}

	result.Triggered = triggered
	result.ExecutionTime = time.Since(startTime)

	if triggered {
		result.Message = fmt.Sprintf("Rule %s triggered: %s", rule.ID, rule.Description)
		re.logger.Warn("Risk rule triggered",
			zap.String("ruleID", rule.ID),
			zap.String("name", rule.Name),
			zap.String("userID", evalCtx.UserID),
			zap.String("symbol", evalCtx.Symbol),
			zap.String("action", rule.Action.Type.String()),
		)
	}

	return result, nil
}

// evaluateCondition evaluates a rule condition
func (re *RuleEngine) evaluateCondition(condition *RuleCondition, evalCtx *RuleEvaluationContext) (bool, error) {
	if condition == nil {
		return true, nil
	}

	// Evaluate main condition
	mainResult, err := re.evaluateSingleCondition(condition, evalCtx)
	if err != nil {
		return false, err
	}

	// Evaluate AND conditions
	if len(condition.And) > 0 {
		for _, andCondition := range condition.And {
			andResult, err := re.evaluateCondition(andCondition, evalCtx)
			if err != nil {
				return false, err
			}
			mainResult = mainResult && andResult
		}
	}

	// Evaluate OR conditions
	if len(condition.Or) > 0 {
		orResult := false
		for _, orCondition := range condition.Or {
			orConditionResult, err := re.evaluateCondition(orCondition, evalCtx)
			if err != nil {
				return false, err
			}
			orResult = orResult || orConditionResult
		}
		mainResult = mainResult || orResult
	}

	return mainResult, nil
}

// evaluateSingleCondition evaluates a single condition
func (re *RuleEngine) evaluateSingleCondition(condition *RuleCondition, evalCtx *RuleEvaluationContext) (bool, error) {
	// Get field value from context
	fieldValue, err := re.getFieldValue(condition.Field, evalCtx)
	if err != nil {
		return false, err
	}

	// Compare based on operator
	switch condition.Operator {
	case ">":
		return re.compareGreater(fieldValue, condition.Value)
	case "<":
		return re.compareLess(fieldValue, condition.Value)
	case ">=":
		return re.compareGreaterEqual(fieldValue, condition.Value)
	case "<=":
		return re.compareLessEqual(fieldValue, condition.Value)
	case "==":
		return re.compareEqual(fieldValue, condition.Value)
	case "!=":
		return re.compareNotEqual(fieldValue, condition.Value)
	case "in":
		return re.compareIn(fieldValue, condition.Value)
	case "not_in":
		return re.compareNotIn(fieldValue, condition.Value)
	default:
		return false, fmt.Errorf("unsupported operator: %s", condition.Operator)
	}
}

// getFieldValue extracts a field value from the evaluation context
func (re *RuleEngine) getFieldValue(field string, evalCtx *RuleEvaluationContext) (interface{}, error) {
	switch field {
	case "order.quantity":
		return evalCtx.Event.Quantity, nil
	case "order.price":
		return evalCtx.Event.Price, nil
	case "order.side":
		return evalCtx.Event.Side.String(), nil
	case "user.id":
		return evalCtx.UserID, nil
	case "symbol":
		return evalCtx.Symbol, nil
	default:
		// Check in data map
		if value, exists := evalCtx.Data[field]; exists {
			return value, nil
		}
		return nil, fmt.Errorf("unknown field: %s", field)
	}
}

// Comparison functions
func (re *RuleEngine) compareGreater(a, b interface{}) (bool, error) {
	aFloat, aOk := a.(float64)
	bFloat, bOk := b.(float64)
	if aOk && bOk {
		return aFloat > bFloat, nil
	}
	return false, fmt.Errorf("cannot compare non-numeric values with >")
}

func (re *RuleEngine) compareLess(a, b interface{}) (bool, error) {
	aFloat, aOk := a.(float64)
	bFloat, bOk := b.(float64)
	if aOk && bOk {
		return aFloat < bFloat, nil
	}
	return false, fmt.Errorf("cannot compare non-numeric values with <")
}

func (re *RuleEngine) compareGreaterEqual(a, b interface{}) (bool, error) {
	aFloat, aOk := a.(float64)
	bFloat, bOk := b.(float64)
	if aOk && bOk {
		return aFloat >= bFloat, nil
	}
	return false, fmt.Errorf("cannot compare non-numeric values with >=")
}

func (re *RuleEngine) compareLessEqual(a, b interface{}) (bool, error) {
	aFloat, aOk := a.(float64)
	bFloat, bOk := b.(float64)
	if aOk && bOk {
		return aFloat <= bFloat, nil
	}
	return false, fmt.Errorf("cannot compare non-numeric values with <=")
}

func (re *RuleEngine) compareEqual(a, b interface{}) (bool, error) {
	return a == b, nil
}

func (re *RuleEngine) compareNotEqual(a, b interface{}) (bool, error) {
	return a != b, nil
}

func (re *RuleEngine) compareIn(a, b interface{}) (bool, error) {
	bSlice, ok := b.([]interface{})
	if !ok {
		return false, fmt.Errorf("'in' operator requires array value")
	}

	for _, item := range bSlice {
		if a == item {
			return true, nil
		}
	}
	return false, nil
}

func (re *RuleEngine) compareNotIn(a, b interface{}) (bool, error) {
	result, err := re.compareIn(a, b)
	return !result, err
}

// sortRulesByPriority sorts rules by priority (higher first)
func (re *RuleEngine) sortRulesByPriority(rules []*RiskRule) {
	for i := 0; i < len(rules)-1; i++ {
		for j := i + 1; j < len(rules); j++ {
			if rules[i].Priority < rules[j].Priority {
				rules[i], rules[j] = rules[j], rules[i]
			}
		}
	}
}

// GetMetrics returns rule engine metrics
func (re *RuleEngine) GetMetrics() map[string]interface{} {
	re.mu.RLock()
	defer re.mu.RUnlock()

	return map[string]interface{}{
		"total_rules":       len(re.rules),
		"rules_executed":    re.rulesExecuted,
		"rules_failed":      re.rulesFailed,
		"execution_time_ns": re.executionTime,
	}
}
