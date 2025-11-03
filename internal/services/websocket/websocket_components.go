// Package websocket provides supporting components for WebSocket system
package websocket

import (
	"fmt"
	"sync"
	"time"
)

// ConnectionManager manages WebSocket connections
type ConnectionManager struct {
	connections map[string]*WebSocketConnection
	mu          sync.RWMutex
}

// NewConnectionManager creates a new connection manager
func NewConnectionManager() *ConnectionManager {
	return &ConnectionManager{
		connections: make(map[string]*WebSocketConnection),
	}
}

// RegisterConnection registers a new WebSocket connection
func (cm *ConnectionManager) RegisterConnection(conn *WebSocketConnection) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.connections[conn.ID] = conn
}

// UnregisterConnection removes a WebSocket connection
func (cm *ConnectionManager) UnregisterConnection(connectionID string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	delete(cm.connections, connectionID)
}

// GetConnection retrieves a connection by ID
func (cm *ConnectionManager) GetConnection(connectionID string) (*WebSocketConnection, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	conn, exists := cm.connections[connectionID]
	return conn, exists
}

// GetActiveConnections returns all active connections
func (cm *ConnectionManager) GetActiveConnections() []*WebSocketConnection {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	var active []*WebSocketConnection
	for _, conn := range cm.connections {
		if conn.IsActive {
			active = append(active, conn)
		}
	}
	return active
}

// SubscriptionManager manages WebSocket subscriptions
type SubscriptionManager struct {
	subscriptions map[string]*Subscription
	mu            sync.RWMutex
}

// NewSubscriptionManager creates a new subscription manager
func NewSubscriptionManager() *SubscriptionManager {
	return &SubscriptionManager{
		subscriptions: make(map[string]*Subscription),
	}
}

// AddSubscription adds a new subscription
func (sm *SubscriptionManager) AddSubscription(subscription *Subscription) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.subscriptions[subscription.ID] = subscription
}

// RemoveSubscription removes a subscription
func (sm *SubscriptionManager) RemoveSubscription(subscriptionID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.subscriptions, subscriptionID)
}

// GetSubscription retrieves a subscription by ID
func (sm *SubscriptionManager) GetSubscription(subscriptionID string) (*Subscription, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	sub, exists := sm.subscriptions[subscriptionID]
	return sub, exists
}

// GetSubscriptionsByChannel returns all subscriptions for a channel
func (sm *SubscriptionManager) GetSubscriptionsByChannel(channel string) []*Subscription {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	var channelSubs []*Subscription
	for _, sub := range sm.subscriptions {
		if sub.Channel == channel {
			channelSubs = append(channelSubs, sub)
		}
	}
	return channelSubs
}

// LicenseValidator validates WebSocket access licenses
type LicenseValidator struct {
	cache map[string]*LicenseValidationResult
	mu    sync.RWMutex
}

// LicenseValidationResult represents license validation result
type LicenseValidationResult struct {
	Valid     bool
	Tier      LicenseTier
	ExpiresAt time.Time
	Features  []string
}

// NewLicenseValidator creates a new license validator
func NewLicenseValidator() *LicenseValidator {
	return &LicenseValidator{
		cache: make(map[string]*LicenseValidationResult),
	}
}

// ValidateLicense validates a user's license for WebSocket access
func (lv *LicenseValidator) ValidateLicense(userID string, tier LicenseTier) (bool, error) {
	lv.mu.RLock()
	if result, exists := lv.cache[userID]; exists {
		if time.Now().Before(result.ExpiresAt) {
			lv.mu.RUnlock()
			return result.Valid && result.Tier >= tier, nil
		}
	}
	lv.mu.RUnlock()

	// Simulate license validation
	result := &LicenseValidationResult{
		Valid:     true,
		Tier:      tier,
		ExpiresAt: time.Now().Add(1 * time.Hour),
		Features:  getLicenseFeatures(tier),
	}

	lv.mu.Lock()
	lv.cache[userID] = result
	lv.mu.Unlock()

	return result.Valid, nil
}

// GetLicenseInfo returns license information for a user
func (lv *LicenseValidator) GetLicenseInfo(userID string) (*LicenseValidationResult, error) {
	lv.mu.RLock()
	defer lv.mu.RUnlock()

	if result, exists := lv.cache[userID]; exists {
		if time.Now().Before(result.ExpiresAt) {
			return result, nil
		}
	}

	return nil, fmt.Errorf("license not found or expired for user: %s", userID)
}

// InvalidateLicense removes a license from cache
func (lv *LicenseValidator) InvalidateLicense(userID string) {
	lv.mu.Lock()
	defer lv.mu.Unlock()
	delete(lv.cache, userID)
}

// IslamicFilter filters WebSocket messages for Islamic compliance
type IslamicFilter struct {
	rules map[string]FilterRule
	mu    sync.RWMutex
}

// FilterRule represents an Islamic finance filtering rule
type FilterRule struct {
	Name      string
	Condition func(interface{}) bool
	Action    FilterAction
	Priority  int
}

// FilterAction defines filtering actions
type FilterAction int

const (
	FilterActionAllow FilterAction = iota
	FilterActionBlock
	FilterActionModify
)

// NewIslamicFilter creates a new Islamic finance filter
func NewIslamicFilter() *IslamicFilter {
	filter := &IslamicFilter{
		rules: make(map[string]FilterRule),
	}

	// Initialize default Islamic finance rules
	filter.initializeDefaultRules()

	return filter
}

// FilterMessage filters a WebSocket message for Islamic compliance
func (if_ *IslamicFilter) FilterMessage(message *WebSocketMessage, ctx *WebSocketConnectionContext) (*WebSocketMessage, error) {
	if !ctx.IslamicCompliant {
		return message, nil
	}

	// Apply Islamic finance filtering rules
	filteredMessage := *message

	// Example filtering logic
	if data, ok := message.Data.(map[string]interface{}); ok {
		// Filter out non-halal assets
		if assetType, exists := data["asset_type"]; exists {
			if assetType == "alcohol" || assetType == "gambling" || assetType == "pork" {
				return nil, fmt.Errorf("asset not halal compliant")
			}
		}

		// Filter interest-based instruments
		if instrument, exists := data["instrument_type"]; exists {
			if instrument == "conventional_bond" || instrument == "interest_derivative" {
				return nil, fmt.Errorf("instrument not sharia compliant")
			}
		}

		// Check debt-to-equity ratio for stocks
		if debtRatio, exists := data["debt_equity_ratio"]; exists {
			if ratio, ok := debtRatio.(float64); ok && ratio > 0.33 {
				return nil, fmt.Errorf("debt-to-equity ratio exceeds Islamic finance limits")
			}
		}

		// Filter based on business activities
		if activities, exists := data["business_activities"]; exists {
			if activitiesList, ok := activities.([]interface{}); ok {
				for _, activity := range activitiesList {
					if activityStr, ok := activity.(string); ok {
						if !if_.isHalalActivity(activityStr) {
							return nil, fmt.Errorf("business activity not halal compliant: %s", activityStr)
						}
					}
				}
			}
		}
	}

	return &filteredMessage, nil
}

// initializeDefaultRules sets up default Islamic finance filtering rules
func (if_ *IslamicFilter) initializeDefaultRules() {
	if_.rules["alcohol_filter"] = FilterRule{
		Name: "Alcohol Filter",
		Condition: func(data interface{}) bool {
			if dataMap, ok := data.(map[string]interface{}); ok {
				if sector, exists := dataMap["sector"]; exists {
					return sector == "alcohol" || sector == "beverages_alcoholic"
				}
			}
			return false
		},
		Action:   FilterActionBlock,
		Priority: 1,
	}

	if_.rules["gambling_filter"] = FilterRule{
		Name: "Gambling Filter",
		Condition: func(data interface{}) bool {
			if dataMap, ok := data.(map[string]interface{}); ok {
				if sector, exists := dataMap["sector"]; exists {
					return sector == "gambling" || sector == "casinos"
				}
			}
			return false
		},
		Action:   FilterActionBlock,
		Priority: 1,
	}

	if_.rules["interest_filter"] = FilterRule{
		Name: "Interest-based Instruments Filter",
		Condition: func(data interface{}) bool {
			if dataMap, ok := data.(map[string]interface{}); ok {
				if instrument, exists := dataMap["instrument_type"]; exists {
					return instrument == "conventional_bond" || instrument == "interest_derivative"
				}
			}
			return false
		},
		Action:   FilterActionBlock,
		Priority: 1,
	}
}

// isHalalActivity checks if a business activity is halal
func (if_ *IslamicFilter) isHalalActivity(activity string) bool {
	halalActivities := map[string]bool{
		"technology":         true,
		"healthcare":         true,
		"education":          true,
		"manufacturing":      true,
		"retail":             true,
		"telecommunications": true,
		"utilities":          true,
		"real_estate":        true,
		"transportation":     true,
		"agriculture":        true,
	}

	haramActivities := map[string]bool{
		"alcohol":              false,
		"gambling":             false,
		"pork":                 false,
		"adult_entertainment":  false,
		"conventional_banking": false,
		"insurance":            false,
		"tobacco":              false,
	}

	if _, isHaram := haramActivities[activity]; isHaram {
		return false
	}

	if _, isHalal := halalActivities[activity]; isHalal {
		return true
	}

	// Default to requiring manual review for unknown activities
	return false
}

// ComplianceEngine handles regulatory compliance for WebSocket connections
type ComplianceEngine struct {
	rules map[string]ComplianceRule
	mu    sync.RWMutex
}

// ComplianceRule represents a regulatory compliance rule
type ComplianceRule struct {
	Region      string
	Exchange    ExchangeType
	Requirement string
	Validator   func(*WebSocketConnectionContext) bool
	Severity    ComplianceSeverity
}

// ComplianceSeverity defines compliance rule severity
type ComplianceSeverity int

const (
	SeverityInfo ComplianceSeverity = iota
	SeverityWarning
	SeverityError
	SeverityCritical
)

// NewComplianceEngine creates a new compliance engine
func NewComplianceEngine() *ComplianceEngine {
	engine := &ComplianceEngine{
		rules: make(map[string]ComplianceRule),
	}

	// Initialize default compliance rules
	engine.initializeDefaultRules()

	return engine
}

// ValidateCompliance validates regulatory compliance for a connection
func (ce *ComplianceEngine) ValidateCompliance(ctx *WebSocketConnectionContext) error {
	ce.mu.RLock()
	defer ce.mu.RUnlock()

	var violations []string

	for _, rule := range ce.rules {
		if rule.Exchange == ctx.Exchange || rule.Exchange == ExchangeTypeUnified {
			if !rule.Validator(ctx) {
				violation := fmt.Sprintf("compliance validation failed: %s", rule.Requirement)
				violations = append(violations, violation)

				if rule.Severity == SeverityCritical {
					return fmt.Errorf("critical compliance violation: %s", rule.Requirement)
				}
			}
		}
	}

	if len(violations) > 0 {
		return fmt.Errorf("compliance violations: %v", violations)
	}

	return nil
}

// initializeDefaultRules sets up default compliance rules
func (ce *ComplianceEngine) initializeDefaultRules() {
	ce.rules["egx_kyc"] = ComplianceRule{
		Region:      "Egypt",
		Exchange:    ExchangeTypeEGX,
		Requirement: "KYC verification required for EGX trading",
		Validator: func(ctx *WebSocketConnectionContext) bool {
			// Check if user has completed KYC
			return ctx.UserID != "" // Simplified check
		},
		Severity: SeverityCritical,
	}

	ce.rules["adx_islamic_compliance"] = ComplianceRule{
		Region:      "UAE",
		Exchange:    ExchangeTypeADX,
		Requirement: "Islamic compliance verification for ADX Islamic instruments",
		Validator: func(ctx *WebSocketConnectionContext) bool {
			// For Islamic tier, ensure compliance is enabled
			if ctx.LicenseTier == LicenseTierIslamic {
				return ctx.IslamicCompliant
			}
			return true
		},
		Severity: SeverityError,
	}

	ce.rules["license_validation"] = ComplianceRule{
		Region:      "Global",
		Exchange:    ExchangeTypeUnified,
		Requirement: "Valid license required for trading access",
		Validator: func(ctx *WebSocketConnectionContext) bool {
			return ctx.LicenseTier >= LicenseTierBasic
		},
		Severity: SeverityCritical,
	}
}

// AnalyticsEngine provides analytics for WebSocket connections
type AnalyticsEngine struct {
	metrics map[string]*ConnectionMetrics
	events  []AnalyticsEvent
	mu      sync.RWMutex
}

// ConnectionMetrics represents analytics metrics for connections
type ConnectionMetrics struct {
	ConnectionID     string
	UserID           string
	MessageCount     int64
	BytesTransferred int64
	SessionDuration  time.Duration
	LastActivity     time.Time
	Exchange         ExchangeType
	LicenseTier      LicenseTier
}

// AnalyticsEvent represents an analytics event
type AnalyticsEvent struct {
	ID        string
	Type      string
	UserID    string
	Data      map[string]interface{}
	Timestamp time.Time
}

// NewAnalyticsEngine creates a new analytics engine
func NewAnalyticsEngine() *AnalyticsEngine {
	return &AnalyticsEngine{
		metrics: make(map[string]*ConnectionMetrics),
		events:  make([]AnalyticsEvent, 0),
	}
}

// RecordConnection records connection analytics
func (ae *AnalyticsEngine) RecordConnection(conn *WebSocketConnection) {
	ae.mu.Lock()
	defer ae.mu.Unlock()

	ae.metrics[conn.ID] = &ConnectionMetrics{
		ConnectionID:     conn.ID,
		UserID:           conn.UserID,
		MessageCount:     0,
		BytesTransferred: 0,
		SessionDuration:  0,
		LastActivity:     time.Now(),
		Exchange:         conn.Context.Exchange,
		LicenseTier:      conn.Context.LicenseTier,
	}

	// Record connection event
	event := AnalyticsEvent{
		ID:     fmt.Sprintf("conn_event_%d", time.Now().UnixNano()),
		Type:   "connection_established",
		UserID: conn.UserID,
		Data: map[string]interface{}{
			"connection_id": conn.ID,
			"exchange":      conn.Context.Exchange,
			"license_tier":  conn.Context.LicenseTier,
			"client_ip":     conn.Context.ClientIP,
		},
		Timestamp: time.Now(),
	}
	ae.events = append(ae.events, event)
}

// RecordMessage records message analytics
func (ae *AnalyticsEngine) RecordMessage(conn *WebSocketConnection, message *WebSocketMessage, decision interface{}) {
	ae.mu.Lock()
	defer ae.mu.Unlock()

	if metrics, exists := ae.metrics[conn.ID]; exists {
		metrics.MessageCount++
		metrics.LastActivity = time.Now()
		metrics.SessionDuration = time.Since(conn.CreatedAt)
	}

	// Record message event
	event := AnalyticsEvent{
		ID:     fmt.Sprintf("msg_event_%d", time.Now().UnixNano()),
		Type:   "message_processed",
		UserID: conn.UserID,
		Data: map[string]interface{}{
			"connection_id": conn.ID,
			"message_type":  message.Type,
			"channel":       message.Channel,
			"message_size":  len(fmt.Sprintf("%+v", message.Data)),
		},
		Timestamp: time.Now(),
	}
	ae.events = append(ae.events, event)
}

// GetConnectionMetrics returns metrics for a specific connection
func (ae *AnalyticsEngine) GetConnectionMetrics(connectionID string) (*ConnectionMetrics, bool) {
	ae.mu.RLock()
	defer ae.mu.RUnlock()

	metrics, exists := ae.metrics[connectionID]
	return metrics, exists
}

// GetAggregatedMetrics returns aggregated analytics metrics
func (ae *AnalyticsEngine) GetAggregatedMetrics() map[string]interface{} {
	ae.mu.RLock()
	defer ae.mu.RUnlock()

	totalConnections := len(ae.metrics)
	totalMessages := int64(0)
	totalBytes := int64(0)

	exchangeDistribution := make(map[ExchangeType]int)
	licenseDistribution := make(map[LicenseTier]int)

	for _, metrics := range ae.metrics {
		totalMessages += metrics.MessageCount
		totalBytes += metrics.BytesTransferred
		exchangeDistribution[metrics.Exchange]++
		licenseDistribution[metrics.LicenseTier]++
	}

	return map[string]interface{}{
		"total_connections":     totalConnections,
		"total_messages":        totalMessages,
		"total_bytes":           totalBytes,
		"exchange_distribution": exchangeDistribution,
		"license_distribution":  licenseDistribution,
		"total_events":          len(ae.events),
		"timestamp":             time.Now(),
	}
}

// Helper functions

// getLicenseFeatures returns features available for a license tier
func getLicenseFeatures(tier LicenseTier) []string {
	switch tier {
	case LicenseTierBasic:
		return []string{"basic_trading", "market_data"}
	case LicenseTierProfessional:
		return []string{"basic_trading", "market_data", "real_time_data", "advanced_analytics"}
	case LicenseTierEnterprise:
		return []string{"basic_trading", "market_data", "real_time_data", "advanced_analytics", "api_access", "white_label"}
	case LicenseTierIslamic:
		return []string{"islamic_trading", "sukuk_data", "sharia_compliance", "zakat_calculation", "halal_screening"}
	default:
		return []string{}
	}
}
