package monitoring

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// AlertLevel represents the severity level of an alert
type AlertLevel string

const (
	// AlertLevelInfo represents an informational alert
	AlertLevelInfo AlertLevel = "INFO"
	
	// AlertLevelWarning represents a warning alert
	AlertLevelWarning AlertLevel = "WARNING"
	
	// AlertLevelError represents an error alert
	AlertLevelError AlertLevel = "ERROR"
	
	// AlertLevelCritical represents a critical alert
	AlertLevelCritical AlertLevel = "CRITICAL"
)

// Alert represents a system alert
type Alert struct {
	ID        string
	Level     AlertLevel
	Source    string
	Message   string
	Details   map[string]interface{}
	Timestamp time.Time
	Resolved  bool
}

// AlertHandler is a function that handles alerts
type AlertHandler func(alert *Alert) error

// AlertManager manages system alerts
type AlertManager struct {
	logger   *zap.Logger
	handlers map[string]AlertHandler
	alerts   []*Alert
	mu       sync.RWMutex
}

// NewAlertManager creates a new alert manager
func NewAlertManager(logger *zap.Logger) *AlertManager {
	return &AlertManager{
		logger:   logger,
		handlers: make(map[string]AlertHandler),
		alerts:   make([]*Alert, 0),
	}
}

// RegisterHandler registers an alert handler
func (m *AlertManager) RegisterHandler(name string, handler AlertHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.handlers[name] = handler
	
	m.logger.Info("Alert handler registered", zap.String("name", name))
}

// UnregisterHandler unregisters an alert handler
func (m *AlertManager) UnregisterHandler(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	delete(m.handlers, name)
	
	m.logger.Info("Alert handler unregistered", zap.String("name", name))
}

// Trigger triggers an alert
func (m *AlertManager) Trigger(ctx context.Context, level AlertLevel, source, message string, details map[string]interface{}) (*Alert, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Create alert
	alert := &Alert{
		ID:        fmt.Sprintf("ALERT-%d", time.Now().UnixNano()),
		Level:     level,
		Source:    source,
		Message:   message,
		Details:   details,
		Timestamp: time.Now(),
		Resolved:  false,
	}
	
	// Add to alerts
	m.alerts = append(m.alerts, alert)
	
	// Log alert
	m.logAlert(alert)
	
	// Notify handlers
	for name, handler := range m.handlers {
		go func(n string, h AlertHandler, a *Alert) {
			if err := h(a); err != nil {
				m.logger.Error("Failed to handle alert",
					zap.Error(err),
					zap.String("handler", n),
					zap.String("alert_id", a.ID))
			}
		}(name, handler, alert)
	}
	
	return alert, nil
}

// Resolve resolves an alert
func (m *AlertManager) Resolve(ctx context.Context, alertID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Find alert
	var alert *Alert
	for _, a := range m.alerts {
		if a.ID == alertID {
			alert = a
			break
		}
	}
	
	if alert == nil {
		return fmt.Errorf("alert not found: %s", alertID)
	}
	
	// Resolve alert
	alert.Resolved = true
	
	m.logger.Info("Alert resolved",
		zap.String("alert_id", alertID),
		zap.String("source", alert.Source),
		zap.String("message", alert.Message))
	
	return nil
}

// GetAlerts returns all alerts
func (m *AlertManager) GetAlerts(includeResolved bool) []*Alert {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	var alerts []*Alert
	
	for _, alert := range m.alerts {
		if includeResolved || !alert.Resolved {
			alerts = append(alerts, alert)
		}
	}
	
	return alerts
}

// GetActiveAlerts returns active (unresolved) alerts
func (m *AlertManager) GetActiveAlerts() []*Alert {
	return m.GetAlerts(false)
}

// logAlert logs an alert
func (m *AlertManager) logAlert(alert *Alert) {
	// Create logger fields
	fields := []zap.Field{
		zap.String("alert_id", alert.ID),
		zap.String("source", alert.Source),
	}
	
	// Add details as fields
	for k, v := range alert.Details {
		fields = append(fields, zap.Any(k, v))
	}
	
	// Log based on level
	switch alert.Level {
	case AlertLevelInfo:
		m.logger.Info(alert.Message, fields...)
	case AlertLevelWarning:
		m.logger.Warn(alert.Message, fields...)
	case AlertLevelError:
		m.logger.Error(alert.Message, fields...)
	case AlertLevelCritical:
		m.logger.Error(fmt.Sprintf("CRITICAL: %s", alert.Message), fields...)
	default:
		m.logger.Info(alert.Message, fields...)
	}
}
