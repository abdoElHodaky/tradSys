// Package exchanges provides compliance infrastructure for Islamic finance and regulatory compliance
package exchanges

import (
	"fmt"
	"time"
)

// BaseCompliance provides common compliance functionality
// following DRY principle - shared logic between Egyptian, UAE, and other compliance types
type BaseCompliance struct {
	regulatoryRules map[string]ComplianceRule
	kycRequirements KYCRequirements
	reportingRules  ReportingRules
	lastUpdated     time.Time
}

// NewBaseCompliance creates a base compliance instance
func NewBaseCompliance() *BaseCompliance {
	return &BaseCompliance{
		regulatoryRules: make(map[string]ComplianceRule),
		kycRequirements: KYCRequirements{
			VerificationLevel: 1,
			RenewalPeriod:     365 * 24 * time.Hour,
		},
	}
}

// ValidateOrder validates an order against compliance rules
func (c *BaseCompliance) ValidateOrder(order *Order) error {
	for _, rule := range c.regulatoryRules {
		applies := containsAssetType(rule.AssetTypes, order.AssetType)
		if applies && !rule.Validator(order) {
			if rule.Severity == SeverityCritical {
				return fmt.Errorf("critical compliance violation: %s", rule.Description)
			}
		}
	}
	return nil
}

// AddRule adds a compliance rule to the registry
func (c *BaseCompliance) AddRule(rule ComplianceRule) {
	c.regulatoryRules[rule.RuleID] = rule
}

// GetRules returns all registered compliance rules
func (c *BaseCompliance) GetRules() map[string]ComplianceRule {
	return c.regulatoryRules
}

// Ensure BaseCompliance implements ComplianceService
var _ ComplianceService = (*BaseCompliance)(nil)
