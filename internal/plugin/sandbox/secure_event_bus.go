package sandbox

import (
	"context"
	"fmt"
	"sync"

	"go.uber.org/zap"

	"github.com/abdoElHodaky/tradSys/internal/plugin/cqrs"
)

// SecureEventBus provides controlled event exchange between plugins
type SecureEventBus struct {
	inner           *cqrs.EventBus
	permissionCheck func(publisherID, eventType string, subscriberID string) bool
	logger          *zap.Logger
	mu              sync.RWMutex
}

// NewSecureEventBus creates a new secure event bus
func NewSecureEventBus(inner *cqrs.EventBus, logger *zap.Logger) *SecureEventBus {
	return &SecureEventBus{
		inner:  inner,
		permissionCheck: func(publisherID, eventType string, subscriberID string) bool {
			// Default permission check allows all communication
			return true
		},
		logger: logger,
	}
}

// WithPermissionCheck sets the permission check function
func (b *SecureEventBus) WithPermissionCheck(check func(publisherID, eventType string, subscriberID string) bool) *SecureEventBus {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	b.permissionCheck = check
	return b
}

// Publish publishes an event with publisher verification
func (b *SecureEventBus) Publish(ctx context.Context, event cqrs.Event, publisherID string) {
	// Verify publisher identity
	if !isValidPublisher(ctx, publisherID) {
		b.logger.Warn("Unauthorized event publication attempt",
			zap.String("publisher_id", publisherID),
			zap.String("event_type", event.EventType()))
		return
	}
	
	// Add publisher information to event context
	ctx = context.WithValue(ctx, "publisher_id", publisherID)
	
	// Publish to inner bus
	b.inner.Publish(ctx, event)
	
	b.logger.Debug("Published event",
		zap.String("publisher_id", publisherID),
		zap.String("event_type", event.EventType()))
}

// RegisterHandler registers an event handler with subscriber verification
func (b *SecureEventBus) RegisterHandler(handler cqrs.EventHandler, subscriberID string) {
	// Create a secure wrapper for the handler
	secureHandler := &secureEventHandler{
		inner:           handler,
		subscriberID:    subscriberID,
		permissionCheck: b.permissionCheck,
		logger:          b.logger,
	}
	
	// Register the secure handler with the inner bus
	b.inner.RegisterHandler(secureHandler)
	
	b.logger.Debug("Registered event handler",
		zap.String("subscriber_id", subscriberID),
		zap.String("event_type", handler.EventType()))
}

// secureEventHandler wraps an event handler with security checks
type secureEventHandler struct {
	inner           cqrs.EventHandler
	subscriberID    string
	permissionCheck func(publisherID, eventType string, subscriberID string) bool
	logger          *zap.Logger
}

// EventType returns the type of event this handler can process
func (h *secureEventHandler) EventType() string {
	return h.inner.EventType()
}

// Handle processes the event with security checks
func (h *secureEventHandler) Handle(ctx context.Context, event cqrs.Event) error {
	// Extract publisher ID from context
	publisherID, ok := ctx.Value("publisher_id").(string)
	if !ok {
		h.logger.Warn("Event missing publisher ID",
			zap.String("subscriber_id", h.subscriberID),
			zap.String("event_type", event.EventType()))
		return fmt.Errorf("event missing publisher ID")
	}
	
	// Check permission
	if !h.permissionCheck(publisherID, event.EventType(), h.subscriberID) {
		h.logger.Warn("Unauthorized event subscription",
			zap.String("publisher_id", publisherID),
			zap.String("subscriber_id", h.subscriberID),
			zap.String("event_type", event.EventType()))
		return fmt.Errorf("unauthorized event subscription")
	}
	
	// Handle the event
	return h.inner.Handle(ctx, event)
}

// isValidPublisher checks if a publisher ID is valid
func isValidPublisher(ctx context.Context, publisherID string) bool {
	// In a real implementation, this would verify the publisher's identity
	// using authentication information from the context
	
	// For now, just return true
	return true
}

// SharedDataRegistry manages controlled data sharing between plugins
type SharedDataRegistry struct {
	data       map[string]interface{}
	access     map[string]map[string]AccessLevel
	mu         sync.RWMutex
	logger     *zap.Logger
}

// AccessLevel defines the level of access to shared data
type AccessLevel int

const (
	NoAccess AccessLevel = iota
	ReadAccess
	WriteAccess
	OwnerAccess
)

// NewSharedDataRegistry creates a new shared data registry
func NewSharedDataRegistry(logger *zap.Logger) *SharedDataRegistry {
	return &SharedDataRegistry{
		data:   make(map[string]interface{}),
		access: make(map[string]map[string]AccessLevel),
		logger: logger,
	}
}

// SetData sets shared data with owner access
func (r *SharedDataRegistry) SetData(key string, value interface{}, ownerID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	// Set the data
	r.data[key] = value
	
	// Set owner access
	if _, exists := r.access[key]; !exists {
		r.access[key] = make(map[string]AccessLevel)
	}
	r.access[key][ownerID] = OwnerAccess
	
	r.logger.Debug("Set shared data",
		zap.String("key", key),
		zap.String("owner_id", ownerID))
}

// GetData retrieves shared data with permission check
func (r *SharedDataRegistry) GetData(key string, pluginID string) (interface{}, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	// Check if data exists
	value, exists := r.data[key]
	if !exists {
		return nil, fmt.Errorf("shared data key %s does not exist", key)
	}
	
	// Check access permissions
	accessMap, exists := r.access[key]
	if !exists {
		return nil, fmt.Errorf("shared data key %s does not have access rules", key)
	}
	
	level, hasAccess := accessMap[pluginID]
	if !hasAccess || level < ReadAccess {
		r.logger.Warn("Unauthorized data access attempt",
			zap.String("key", key),
			zap.String("plugin_id", pluginID))
		return nil, fmt.Errorf("plugin %s does not have read access to %s", pluginID, key)
	}
	
	r.logger.Debug("Retrieved shared data",
		zap.String("key", key),
		zap.String("plugin_id", pluginID))
	
	return value, nil
}

// GrantAccess grants access to shared data
func (r *SharedDataRegistry) GrantAccess(key string, pluginID string, level AccessLevel, grantorID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	// Check if data exists
	if _, exists := r.data[key]; !exists {
		return fmt.Errorf("shared data key %s does not exist", key)
	}
	
	// Check if grantor has owner access
	accessMap, exists := r.access[key]
	if !exists {
		return fmt.Errorf("shared data key %s does not have access rules", key)
	}
	
	grantorLevel, hasAccess := accessMap[grantorID]
	if !hasAccess || grantorLevel < OwnerAccess {
		return fmt.Errorf("plugin %s does not have owner access to %s", grantorID, key)
	}
	
	// Grant access
	accessMap[pluginID] = level
	
	r.logger.Debug("Granted data access",
		zap.String("key", key),
		zap.String("plugin_id", pluginID),
		zap.Int("level", int(level)),
		zap.String("grantor_id", grantorID))
	
	return nil
}

// RevokeAccess revokes access to shared data
func (r *SharedDataRegistry) RevokeAccess(key string, pluginID string, revokerID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	// Check if data exists
	if _, exists := r.data[key]; !exists {
		return fmt.Errorf("shared data key %s does not exist", key)
	}
	
	// Check if revoker has owner access
	accessMap, exists := r.access[key]
	if !exists {
		return fmt.Errorf("shared data key %s does not have access rules", key)
	}
	
	revokerLevel, hasAccess := accessMap[revokerID]
	if !hasAccess || revokerLevel < OwnerAccess {
		return fmt.Errorf("plugin %s does not have owner access to %s", revokerID, key)
	}
	
	// Revoke access
	delete(accessMap, pluginID)
	
	r.logger.Debug("Revoked data access",
		zap.String("key", key),
		zap.String("plugin_id", pluginID),
		zap.String("revoker_id", revokerID))
	
	return nil
}
