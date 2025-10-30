package compliance

import (
	"fmt"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/trading/types"
	"go.uber.org/zap"
)

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
		}
	}

	return violations
}

// checkRule checks a specific rule against an order
func (r *RuleEngine) checkRule(rule ComplianceRule, order *types.Order, userID string) *ComplianceViolation {
	switch rule.Type {
	case RuleTypePositionLimit:
		maxSize, ok := rule.Parameters["max_position_size"].(float64)
		if ok && order.Quantity > maxSize {
			return &ComplianceViolation{
				ID:            fmt.Sprintf("violation_%d", time.Now().UnixNano()),
				RuleID:        rule.ID,
				OrderID:       order.ID,
				UserID:        userID,
				ViolationType: string(rule.Type),
				Severity:      SeverityHigh,
				Description:   fmt.Sprintf("Order size %.2f exceeds maximum allowed size %.2f", order.Quantity, maxSize),
				Details: map[string]interface{}{
					"order_size": order.Quantity,
					"max_size":   maxSize,
				},
				Status:     ViolationStatusOpen,
				DetectedAt: time.Now(),
			}
		}

	case RuleTypeTradingHours:
		// Simplified trading hours check
		now := time.Now().UTC()
		startTime, _ := rule.Parameters["start_time"].(string)
		endTime, _ := rule.Parameters["end_time"].(string)

		// Parse time strings (simplified - assumes HH:MM format)
		startHour := 9  // Default 09:00
		endHour := 16   // Default 16:00

		if now.Hour() < startHour || now.Hour() >= endHour {
			return &ComplianceViolation{
				ID:            fmt.Sprintf("violation_%d", time.Now().UnixNano()),
				RuleID:        rule.ID,
				OrderID:       order.ID,
				UserID:        userID,
				ViolationType: string(rule.Type),
				Severity:      SeverityMedium,
				Description:   fmt.Sprintf("Trading outside allowed hours (%s-%s UTC)", startTime, endTime),
				Details: map[string]interface{}{
					"current_hour": now.Hour(),
					"start_time":   startTime,
					"end_time":     endTime,
				},
				Status:     ViolationStatusOpen,
				DetectedAt: time.Now(),
			}
		}

	case RuleTypeRiskLimit:
		maxLoss, ok := rule.Parameters["max_daily_loss"].(float64)
		if ok {
			// Simplified risk check - in real implementation would check actual P&L
			orderValue := order.Quantity * order.Price
			if orderValue > maxLoss {
				return &ComplianceViolation{
					ID:            fmt.Sprintf("violation_%d", time.Now().UnixNano()),
					RuleID:        rule.ID,
					OrderID:       order.ID,
					UserID:        userID,
					ViolationType: string(rule.Type),
					Severity:      SeverityCritical,
					Description:   fmt.Sprintf("Order value %.2f exceeds daily risk limit %.2f", orderValue, maxLoss),
					Details: map[string]interface{}{
						"order_value": orderValue,
						"risk_limit":  maxLoss,
					},
					Status:     ViolationStatusOpen,
					DetectedAt: time.Now(),
				}
			}
		}

	case RuleTypeKYC:
		// KYC compliance check - simplified
		return &ComplianceViolation{
			ID:            fmt.Sprintf("violation_%d", time.Now().UnixNano()),
			RuleID:        rule.ID,
			OrderID:       order.ID,
			UserID:        userID,
			ViolationType: string(rule.Type),
			Severity:      SeverityHigh,
			Description:   "KYC verification required",
			Details: map[string]interface{}{
				"user_id": userID,
			},
			Status:     ViolationStatusOpen,
			DetectedAt: time.Now(),
		}

	case RuleTypeAML:
		// AML compliance check - simplified
		return &ComplianceViolation{
			ID:            fmt.Sprintf("violation_%d", time.Now().UnixNano()),
			RuleID:        rule.ID,
			OrderID:       order.ID,
			UserID:        userID,
			ViolationType: string(rule.Type),
			Severity:      SeverityHigh,
			Description:   "AML screening required",
			Details: map[string]interface{}{
				"user_id": userID,
			},
			Status:     ViolationStatusOpen,
			DetectedAt: time.Now(),
		}
	}

	return nil
}

// GenerateReport generates a compliance report
func (g *ReportGenerator) GenerateReport(reportType ReportType, startDate, endDate time.Time) *ComplianceReport {
	g.mu.Lock()
	defer g.mu.Unlock()

	report := &ComplianceReport{
		ID:   fmt.Sprintf("report_%d", time.Now().UnixNano()),
		Type: reportType,
		Title: fmt.Sprintf("%s Compliance Report", reportType),
		Description: fmt.Sprintf("Compliance report for period %s to %s", 
			startDate.Format("2006-01-02"), endDate.Format("2006-01-02")),
		Period: ReportPeriod{
			StartDate: startDate,
			EndDate:   endDate,
		},
		Status:      ReportStatusCompleted,
		GeneratedAt: time.Now(),
		GeneratedBy: "system",
		Data: map[string]interface{}{
			"period":      fmt.Sprintf("%s to %s", startDate.Format("2006-01-02"), endDate.Format("2006-01-02")),
			"generated":   time.Now(),
			"report_type": reportType,
			"summary": map[string]interface{}{
				"total_checks":        0,
				"violations_detected": 0,
				"alerts_triggered":    0,
			},
		},
	}

	g.logger.Info("Compliance report generated",
		zap.String("report_id", report.ID),
		zap.String("type", string(reportType)),
		zap.String("period", fmt.Sprintf("%s to %s", startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))))

	return report
}

// AddEntry adds an entry to the audit trail
func (a *AuditTrail) AddEntry(entry *AuditEntry) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.entries = append(a.entries, *entry)

	// Keep only the last maxEntries
	if len(a.entries) > a.maxEntries {
		a.entries = a.entries[len(a.entries)-a.maxEntries:]
	}

	a.logger.Debug("Audit entry added",
		zap.String("entry_id", entry.ID),
		zap.String("event_type", string(entry.EventType)),
		zap.String("user_id", entry.UserID))
}

// GetEntries returns audit entries for a specific period
func (a *AuditTrail) GetEntries(startDate, endDate time.Time) []AuditEntry {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var filtered []AuditEntry
	for _, entry := range a.entries {
		if entry.Timestamp.After(startDate) && entry.Timestamp.Before(endDate) {
			filtered = append(filtered, entry)
		}
	}

	return filtered
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

// TriggerAlert triggers a compliance alert
func (m *AlertManager) TriggerAlert(alert *ComplianceAlert) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Notify handlers
	for _, handler := range m.handlers {
		if err := handler.HandleAlert(alert); err != nil {
			m.logger.Error("Alert handler failed", 
				zap.Error(err),
				zap.String("alert_id", alert.ID))
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
	m.handlers = append(m.handlers, handler)

	m.logger.Info("Alert handler subscribed",
		zap.Int("total_handlers", len(m.handlers)))
}

// GetActiveAlerts returns all active alerts
func (m *AlertManager) GetActiveAlerts() []ComplianceAlert {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var activeAlerts []ComplianceAlert
	// In a real implementation, this would filter from a stored collection
	// For now, return empty slice as alerts are processed immediately
	return activeAlerts
}

// AcknowledgeAlert acknowledges an alert
func (m *AlertManager) AcknowledgeAlert(alertID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// In a real implementation, this would update the alert status
	m.logger.Info("Alert acknowledged", zap.String("alert_id", alertID))
	return nil
}

// ResolveAlert resolves an alert
func (m *AlertManager) ResolveAlert(alertID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// In a real implementation, this would update the alert status
	m.logger.Info("Alert resolved", zap.String("alert_id", alertID))
	return nil
}
