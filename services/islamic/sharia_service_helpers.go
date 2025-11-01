// Package islamic provides Islamic finance services for TradSys v3
package islamic

import (
	"fmt"
	"time"

	"github.com/abdoElHodaky/tradSys/pkg/types"
)

// NewShariaBoard creates a default Sharia board
func NewShariaBoard() *ShariaBoard {
	return &ShariaBoard{
		Name:        "TradSys Islamic Finance Advisory Board",
		Established: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		Members: []ShariaBoardMember{
			{
				Name:           "Dr. Ahmed Al-Rashid",
				Title:          "Chairman",
				Qualifications: []string{"PhD Islamic Finance", "Certified Sharia Advisor"},
				Experience:     15,
			},
			{
				Name:           "Sheikh Mohammed Al-Qasimi",
				Title:          "Senior Advisor",
				Qualifications: []string{"Islamic Jurisprudence", "Financial Markets Expert"},
				Experience:     20,
			},
		},
		Certifications: []string{"AAOIFI Certified", "IFSB Compliant"},
		ContactInfo: ContactInfo{
			Email:   "sharia@tradsys.com",
			Phone:   "+971-4-123-4567",
			Address: "Dubai International Financial Centre, UAE",
			Website: "https://tradsys.com/islamic-finance",
		},
	}
}

// GetDefaultShariaConfig returns default Sharia configuration
func GetDefaultShariaConfig() *ShariaConfig {
	return &ShariaConfig{
		EnableScreening:   true,
		EnableZakat:       true,
		EnableShariaBoard: true,
		ZakatRate:         0.025, // 2.5%
		NisabThreshold:    10000, // $10,000 minimum threshold
		Currency:          "USD",
		ShariaStandard:    "AAOIFI",
		ComplianceLevel:   MODERATE,
		ScreeningRules: []ShariaRule{
			{
				ID:              "interest_prohibition",
				Name:            "Interest Prohibition",
				Description:     "Prohibits investment in interest-based activities",
				Category:        "interest_based",
				AssetTypes:      []types.AssetType{types.STOCK, types.BOND, types.ETF},
				ComplianceLevel: STRICT,
				IsActive:        true,
				CreatedAt:       time.Now(),
				UpdatedAt:       time.Now(),
			},
			{
				ID:              "prohibited_activities",
				Name:            "Prohibited Business Activities",
				Description:     "Prohibits investment in alcohol, gambling, pork, etc.",
				Category:        "prohibited_activities",
				AssetTypes:      []types.AssetType{types.STOCK, types.ETF, types.REIT},
				ComplianceLevel: STRICT,
				IsActive:        true,
				CreatedAt:       time.Now(),
				UpdatedAt:       time.Now(),
			},
			{
				ID:              "debt_ratio_check",
				Name:            "Debt Ratio Check",
				Description:     "Ensures debt-to-equity ratio is within acceptable limits",
				Category:        "debt_ratio",
				AssetTypes:      []types.AssetType{types.STOCK, types.REIT},
				ComplianceLevel: MODERATE,
				IsActive:        true,
				CreatedAt:       time.Now(),
				UpdatedAt:       time.Now(),
			},
		},
	}
}

// GetStrictShariaConfig returns strict Sharia configuration
func GetStrictShariaConfig() *ShariaConfig {
	config := GetDefaultShariaConfig()
	config.ComplianceLevel = STRICT

	// Add additional strict rules
	strictRules := []ShariaRule{
		{
			ID:              "revenue_screening",
			Name:            "Revenue Screening",
			Description:     "Ensures non-compliant revenue is below 5%",
			Category:        "revenue_screening",
			AssetTypes:      []types.AssetType{types.STOCK, types.ETF, types.REIT},
			ComplianceLevel: STRICT,
			IsActive:        true,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		},
		{
			ID:              "cash_ratio_check",
			Name:            "Cash Ratio Check",
			Description:     "Ensures cash and interest-bearing securities are below 33%",
			Category:        "cash_ratio",
			AssetTypes:      []types.AssetType{types.STOCK, types.REIT},
			ComplianceLevel: STRICT,
			IsActive:        true,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		},
	}

	config.ScreeningRules = append(config.ScreeningRules, strictRules...)
	return config
}

// GetFlexibleShariaConfig returns flexible Sharia configuration
func GetFlexibleShariaConfig() *ShariaConfig {
	config := GetDefaultShariaConfig()
	config.ComplianceLevel = FLEXIBLE
	config.ZakatRate = 0.02 // Reduced rate for flexible compliance

	// Make some rules less strict
	for i := range config.ScreeningRules {
		if config.ScreeningRules[i].ID == "debt_ratio_check" {
			config.ScreeningRules[i].ComplianceLevel = FLEXIBLE
		}
	}

	return config
}

// CreateCustomShariaConfig creates a custom Sharia configuration
func CreateCustomShariaConfig(enableScreening, enableZakat, enableBoard bool, zakatRate, nisabThreshold float64, currency, standard string, level ComplianceLevel) *ShariaConfig {
	return &ShariaConfig{
		EnableScreening:   enableScreening,
		EnableZakat:       enableZakat,
		EnableShariaBoard: enableBoard,
		ZakatRate:         zakatRate,
		NisabThreshold:    nisabThreshold,
		Currency:          currency,
		ShariaStandard:    standard,
		ComplianceLevel:   level,
		ScreeningRules:    []ShariaRule{}, // Empty rules, to be added separately
	}
}

// AddScreeningRule adds a screening rule to the configuration
func (config *ShariaConfig) AddScreeningRule(rule ShariaRule) {
	config.ScreeningRules = append(config.ScreeningRules, rule)
}

// RemoveScreeningRule removes a screening rule by ID
func (config *ShariaConfig) RemoveScreeningRule(ruleID string) {
	for i, rule := range config.ScreeningRules {
		if rule.ID == ruleID {
			config.ScreeningRules = append(config.ScreeningRules[:i], config.ScreeningRules[i+1:]...)
			break
		}
	}
}

// UpdateScreeningRule updates a screening rule
func (config *ShariaConfig) UpdateScreeningRule(ruleID string, updatedRule ShariaRule) {
	for i, rule := range config.ScreeningRules {
		if rule.ID == ruleID {
			updatedRule.UpdatedAt = time.Now()
			config.ScreeningRules[i] = updatedRule
			break
		}
	}
}

// GetActiveRules returns only active screening rules
func (config *ShariaConfig) GetActiveRules() []ShariaRule {
	var activeRules []ShariaRule
	for _, rule := range config.ScreeningRules {
		if rule.IsActive {
			activeRules = append(activeRules, rule)
		}
	}
	return activeRules
}

// GetRulesByCategory returns rules filtered by category
func (config *ShariaConfig) GetRulesByCategory(category string) []ShariaRule {
	var categoryRules []ShariaRule
	for _, rule := range config.ScreeningRules {
		if rule.Category == category {
			categoryRules = append(categoryRules, rule)
		}
	}
	return categoryRules
}

// GetRulesByComplianceLevel returns rules filtered by compliance level
func (config *ShariaConfig) GetRulesByComplianceLevel(level ComplianceLevel) []ShariaRule {
	var levelRules []ShariaRule
	for _, rule := range config.ScreeningRules {
		if rule.ComplianceLevel == level {
			levelRules = append(levelRules, rule)
		}
	}
	return levelRules
}

// ValidateConfig validates the Sharia configuration
func (config *ShariaConfig) ValidateConfig() error {
	if config.ZakatRate < 0 || config.ZakatRate > 1 {
		return fmt.Errorf("invalid Zakat rate: must be between 0 and 1")
	}

	if config.NisabThreshold < 0 {
		return fmt.Errorf("invalid Nisab threshold: must be non-negative")
	}

	if config.Currency == "" {
		return fmt.Errorf("currency is required")
	}

	if config.ShariaStandard == "" {
		return fmt.Errorf("Sharia standard is required")
	}

	// Validate compliance level
	switch config.ComplianceLevel {
	case STRICT, MODERATE, FLEXIBLE:
		// Valid levels
	default:
		return fmt.Errorf("invalid compliance level: %s", config.ComplianceLevel)
	}

	return nil
}

// Clone creates a deep copy of the Sharia configuration
func (config *ShariaConfig) Clone() *ShariaConfig {
	clone := &ShariaConfig{
		EnableScreening:   config.EnableScreening,
		EnableZakat:       config.EnableZakat,
		EnableShariaBoard: config.EnableShariaBoard,
		ZakatRate:         config.ZakatRate,
		NisabThreshold:    config.NisabThreshold,
		Currency:          config.Currency,
		ShariaStandard:    config.ShariaStandard,
		ComplianceLevel:   config.ComplianceLevel,
		ScreeningRules:    make([]ShariaRule, len(config.ScreeningRules)),
	}

	// Deep copy screening rules
	copy(clone.ScreeningRules, config.ScreeningRules)

	return clone
}
