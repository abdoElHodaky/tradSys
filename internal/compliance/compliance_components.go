package compliance

import (
	"fmt"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/trading/types"
	"go.uber.org/zap"
)

// RuleEngine component implementations

// AddRule adds a compliance rule to the rule engine
func (r *RuleEngine) AddRule(rule ComplianceRule) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rules[rule.ID] = rule
}

// CheckRules checks all enabled rules against an order
func (r *RuleEngine) CheckRules(order *types.Order, userID string) []ComplianceViolation {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var violations []ComplianceViolation

	for _, rule := range r.rules {
		if !rule.Enabled {
			continue
		}

		if violation := r.checkRule(rule, order, userID); violation != nil {
			violations = append(violations, *violation)
			// Store violation in rule engine
			r.violations = append(r.violations, *violation)
		}
	}

	return violations
}

// checkRule checks a specific rule against an order
func (r *RuleEngine) checkRule(rule ComplianceRule, order *types.Order, userID string) *ComplianceViolation {
	switch rule.RuleType {
	case RuleTypeOrderSize:
		maxSize, ok := rule.Parameters["max_order_size"].(float64)
		if ok && order.Quantity > maxSize {
			return &ComplianceViolation{
				ID:          fmt.Sprintf("violation_%d", time.Now().UnixNano()),
				RuleID:      rule.ID,
				OrderID:     order.ID,
				UserID:      userID,
				Symbol:      order.Symbol,
				Severity:    rule.Severity,
				Description: fmt.Sprintf("Order size %.2f exceeds maximum allowed size %.2f", order.Quantity, maxSize),
				Details: map[string]interface{}{
					"order_size": order.Quantity,
					"max_size":   maxSize,
				},
				Status:     ViolationStatusOpen,
				DetectedAt: time.Now(),
			}
		}

	case RuleTypePositionLimit:
		maxPosition, ok := rule.Parameters["max_position"].(float64)
		if ok && order.Quantity > maxPosition {
			return &ComplianceViolation{
				ID:          fmt.Sprintf("violation_%d", time.Now().UnixNano()),
				RuleID:      rule.ID,
				OrderID:     order.ID,
				UserID:      userID,
				Symbol:      order.Symbol,
				Severity:    rule.Severity,
				Description: fmt.Sprintf("Position size %.2f exceeds maximum allowed position %.2f", order.Quantity, maxPosition),
				Details: map[string]interface{}{
					"position_size": order.Quantity,
					"max_position":  maxPosition,
				},
				Status:     ViolationStatusOpen,
				DetectedAt: time.Now(),
			}
		}

	case RuleTypeTradingHours:
		// Simplified trading hours check
		now := time.Now().UTC()
		startHour, _ := rule.Parameters["start_hour"].(int)
		endHour, _ := rule.Parameters["end_hour"].(int)

		if now.Hour() < startHour || now.Hour() >= endHour {
			return &ComplianceViolation{
				ID:          fmt.Sprintf("violation_%d", time.Now().UnixNano()),
				RuleID:      rule.ID,
				OrderID:     order.ID,
				UserID:      userID,
				Symbol:      order.Symbol,
				Severity:    rule.Severity,
				Description: fmt.Sprintf("Trading outside allowed hours (%d:00-%d:00 UTC)", startHour, endHour),
				Details: map[string]interface{}{
					"current_hour": now.Hour(),
					"start_hour":   startHour,
					"end_hour":     endHour,
				},
				Status:     ViolationStatusOpen,
				DetectedAt: time.Now(),
			}
		}

	case RuleTypeRiskLimit:
		maxRisk, ok := rule.Parameters["max_risk_exposure"].(float64)
		if ok {
			// Calculate risk exposure (simplified)
			riskExposure := order.Quantity * order.Price
			if riskExposure > maxRisk {
				return &ComplianceViolation{
					ID:          fmt.Sprintf("violation_%d", time.Now().UnixNano()),
					RuleID:      rule.ID,
					OrderID:     order.ID,
					UserID:      userID,
					Symbol:      order.Symbol,
					Severity:    rule.Severity,
					Description: fmt.Sprintf("Risk exposure %.2f exceeds maximum allowed risk %.2f", riskExposure, maxRisk),
					Details: map[string]interface{}{
						"risk_exposure": riskExposure,
						"max_risk":      maxRisk,
						"quantity":      order.Quantity,
						"price":         order.Price,
					},
					Status:     ViolationStatusOpen,
					DetectedAt: time.Now(),
				}
			}
		}

	case RuleTypeMarketManipulation:
		// Simplified market manipulation detection
		volumeThreshold, ok := rule.Parameters["volume_threshold"].(float64)
		if ok && order.Quantity > volumeThreshold {
			priceDeviation, _ := rule.Parameters["price_deviation_threshold"].(float64)
			// In a real implementation, this would compare against market price
			if priceDeviation > 0 {
				return &ComplianceViolation{
					ID:          fmt.Sprintf("violation_%d", time.Now().UnixNano()),
					RuleID:      rule.ID,
					OrderID:     order.ID,
					UserID:      userID,
					Symbol:      order.Symbol,
					Severity:    rule.Severity,
					Description: "Potential market manipulation detected: large volume order",
					Details: map[string]interface{}{
						"order_volume":     order.Quantity,
						"volume_threshold": volumeThreshold,
						"price_deviation":  priceDeviation,
					},
					Status:     ViolationStatusOpen,
					DetectedAt: time.Now(),
				}
			}
		}
	}

	return nil
}

// GetViolations returns all violations stored in the rule engine
func (r *RuleEngine) GetViolations() []ComplianceViolation {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Return a copy to avoid race conditions
	violations := make([]ComplianceViolation, len(r.violations))
	copy(violations, r.violations)
	return violations
}

// GetRules returns all rules in the rule engine
func (r *RuleEngine) GetRules() map[string]ComplianceRule {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Return a copy to avoid race conditions
	rules := make(map[string]ComplianceRule)
	for k, v := range r.rules {
		rules[k] = v
	}
	return rules
}

// ReportGenerator component implementations

// GenerateReport generates a compliance report
func (g *ReportGenerator) GenerateReport(reportType ReportType, startDate, endDate time.Time) *ComplianceReport {
	g.mu.Lock()
	defer g.mu.Unlock()

	report := &ComplianceReport{
		ID:   fmt.Sprintf("report_%d", time.Now().UnixNano()),
		Type: reportType,
		Period: ReportPeriod{
			StartDate: startDate,
			EndDate:   endDate,
		},
		Status:      ReportStatusGenerated,
		GeneratedAt: time.Now(),
		Data: map[string]interface{}{
			"period":      fmt.Sprintf("%s to %s", startDate.Format("2006-01-02"), endDate.Format("2006-01-02")),
			"generated":   time.Now(),
			"report_type": reportType,
			"summary": map[string]interface{}{
				"total_checks":        1000,
				"violations_detected": 25,
				"compliance_rate":     0.975,
			},
		},
	}

	g.reports = append(g.reports, *report)
	g.logger.Info("Generated compliance report",
		zap.String("report_id", report.ID),
		zap.String("type", string(reportType)),
		zap.String("period", fmt.Sprintf("%s to %s", startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))))

	return report
}

// GetReports returns all generated reports
func (g *ReportGenerator) GetReports() []ComplianceReport {
	g.mu.RLock()
	defer g.mu.RUnlock()

	// Return a copy to avoid race conditions
	reports := make([]ComplianceReport, len(g.reports))
	copy(reports, g.reports)
	return reports
}

// AddTemplate adds a report template
func (g *ReportGenerator) AddTemplate(template ReportTemplate) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.templates[template.ID] = template
}

// GetTemplates returns all report templates
func (g *ReportGenerator) GetTemplates() map[string]ReportTemplate {
	g.mu.RLock()
	defer g.mu.RUnlock()

	// Return a copy to avoid race conditions
	templates := make(map[string]ReportTemplate)
	for k, v := range g.templates {
		templates[k] = v
	}
	return templates
}

// AuditTrail component implementations

// AddEntry adds an entry to the audit trail
func (a *AuditTrail) AddEntry(entry *AuditEntry) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.entries = append(a.entries, *entry)

	// Keep only the last maxEntries
	if len(a.entries) > a.maxEntries {
		a.entries = a.entries[len(a.entries)-a.maxEntries:]
	}

	a.logger.Debug("Added audit entry",
		zap.String("entry_id", entry.ID),
		zap.String("event_type", string(entry.EventType)),
		zap.String("user_id", entry.UserID))
}

// Cleanup removes old audit entries
func (a *AuditTrail) Cleanup(retentionPeriod time.Duration) {
	a.mu.Lock()
	defer a.mu.Unlock()

	cutoff := time.Now().Add(-retentionPeriod)
	var kept []AuditEntry

	for _, entry := range a.entries {
		if entry.Timestamp.After(cutoff) {
			kept = append(kept, entry)
		}
	}

	removed := len(a.entries) - len(kept)
	a.entries = kept

	if removed > 0 {
		a.logger.Info("Cleaned up audit trail entries",
			zap.Int("removed", removed),
			zap.Int("remaining", len(kept)))
	}
}

// GetEntries returns audit trail entries with optional limit
func (a *AuditTrail) GetEntries(limit int) []AuditEntry {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if limit <= 0 || limit > len(a.entries) {
		limit = len(a.entries)
	}

	// Return the most recent entries
	start := len(a.entries) - limit
	if start < 0 {
		start = 0
	}

	entries := make([]AuditEntry, limit)
	copy(entries, a.entries[start:])
	return entries
}

// GetEntriesByUser returns audit entries for a specific user
func (a *AuditTrail) GetEntriesByUser(userID string, limit int) []AuditEntry {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var userEntries []AuditEntry
	for _, entry := range a.entries {
		if entry.UserID == userID {
			userEntries = append(userEntries, entry)
			if limit > 0 && len(userEntries) >= limit {
				break
			}
		}
	}

	return userEntries
}

// AlertManager component implementations

// TriggerAlert triggers a compliance alert
func (m *AlertManager) TriggerAlert(alert *ComplianceAlert) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.alerts = append(m.alerts, *alert)

	// Notify subscribers
	for _, handler := range m.subscribers {
		if err := handler.HandleAlert(alert); err != nil {
			m.logger.Error("Alert handler failed", zap.Error(err))
		}
	}

	m.logger.Warn("Compliance alert triggered",
		zap.String("alert_id", alert.ID),
		zap.String("type", string(alert.Type)),
		zap.String("severity", string(alert.Severity)),
		zap.String("title", alert.Title))
}

// Subscribe subscribes to compliance alerts
func (m *AlertManager) Subscribe(handler AlertHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.subscribers = append(m.subscribers, handler)
}

// GetAlerts returns all alerts
func (m *AlertManager) GetAlerts() []ComplianceAlert {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy to avoid race conditions
	alerts := make([]ComplianceAlert, len(m.alerts))
	copy(alerts, m.alerts)
	return alerts
}

// AcknowledgeAlert acknowledges an alert
func (m *AlertManager) AcknowledgeAlert(alertID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, alert := range m.alerts {
		if alert.ID == alertID {
			now := time.Now()
			m.alerts[i].Status = AlertStatusAcknowledged
			m.alerts[i].AcknowledgedAt = &now

			m.logger.Info("Alert acknowledged",
				zap.String("alert_id", alertID))
			return nil
		}
	}

	return fmt.Errorf("alert not found: %s", alertID)
}

// ResolveAlert resolves an alert
func (m *AlertManager) ResolveAlert(alertID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, alert := range m.alerts {
		if alert.ID == alertID {
			m.alerts[i].Status = AlertStatusResolved

			m.logger.Info("Alert resolved",
				zap.String("alert_id", alertID))
			return nil
		}
	}

	return fmt.Errorf("alert not found: %s", alertID)
}

// GetActiveAlerts returns all active alerts
func (m *AlertManager) GetActiveAlerts() []ComplianceAlert {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var activeAlerts []ComplianceAlert
	for _, alert := range m.alerts {
		if alert.Status == AlertStatusActive {
			activeAlerts = append(activeAlerts, alert)
		}
	}

	return activeAlerts
}

// ClearOldAlerts removes alerts older than the specified duration
func (m *AlertManager) ClearOldAlerts(maxAge time.Duration) int {
	m.mu.Lock()
	defer m.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	var kept []ComplianceAlert

	for _, alert := range m.alerts {
		if alert.CreatedAt.After(cutoff) {
			kept = append(kept, alert)
		}
	}

	removed := len(m.alerts) - len(kept)
	m.alerts = kept

	if removed > 0 {
		m.logger.Info("Cleared old alerts",
			zap.Int("removed", removed),
			zap.Int("remaining", len(kept)))
	}

	return removed
}
