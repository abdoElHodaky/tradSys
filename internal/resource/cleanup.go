package resource

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
)

// ResourceType represents the type of resource
type ResourceType string

const (
	// ResourceTypeConnection represents a connection resource
	ResourceTypeConnection ResourceType = "connection"
	// ResourceTypeFile represents a file resource
	ResourceTypeFile ResourceType = "file"
	// ResourceTypeMemory represents a memory resource
	ResourceTypeMemory ResourceType = "memory"
	// ResourceTypeGoroutine represents a goroutine resource
	ResourceTypeGoroutine ResourceType = "goroutine"
	// ResourceTypeCustom represents a custom resource
	ResourceTypeCustom ResourceType = "custom"
)

// Resource represents a resource that needs to be cleaned up
type Resource struct {
	// ID is the unique identifier for the resource
	ID string

	// Type is the type of resource
	Type ResourceType

	// CleanupFunc is the function to call to clean up the resource
	CleanupFunc func() error

	// CreatedAt is the time the resource was created
	CreatedAt time.Time

	// LastUsedAt is the time the resource was last used
	LastUsedAt time.Time

	// Metadata is additional metadata about the resource
	Metadata map[string]interface{}
}

// ResourceManagerConfig contains configuration for the resource manager
type ResourceManagerConfig struct {
	// CleanupInterval is the interval at which to run cleanup
	CleanupInterval time.Duration

	// ResourceTimeout is the timeout after which a resource is considered unused
	ResourceTimeout time.Duration

	// MaxResources is the maximum number of resources to track
	MaxResources int

	// EnableMetrics enables metrics collection
	EnableMetrics bool
}

// DefaultResourceManagerConfig returns the default resource manager configuration
func DefaultResourceManagerConfig() ResourceManagerConfig {
	return ResourceManagerConfig{
		CleanupInterval: 5 * time.Minute,
		ResourceTimeout: 30 * time.Minute,
		MaxResources:    1000,
		EnableMetrics:   true,
	}
}

// ResourceManager manages resources and ensures they are properly cleaned up
type ResourceManager struct {
	// Configuration
	config ResourceManagerConfig

	// Resources
	resources map[string]*Resource
	mu        sync.RWMutex

	// Metrics
	resourcesCreated   int
	resourcesCleaned   int
	resourcesLeaked    int
	lastCleanupTime    time.Time
	totalCleanupTime   time.Duration
	cleanupCount       int
	resourceTypeCount  map[ResourceType]int
	resourceTypeLeaked map[ResourceType]int

	// Logger
	logger *zap.Logger

	// Context for cancellation
	ctx        context.Context
	cancelFunc context.CancelFunc
}

// NewResourceManager creates a new resource manager
func NewResourceManager(config ResourceManagerConfig, logger *zap.Logger) *ResourceManager {
	if logger == nil {
		logger = zap.NewNop()
	}

	ctx, cancel := context.WithCancel(context.Background())

	rm := &ResourceManager{
		config:             config,
		resources:          make(map[string]*Resource),
		resourceTypeCount:  make(map[ResourceType]int),
		resourceTypeLeaked: make(map[ResourceType]int),
		logger:             logger,
		ctx:                ctx,
		cancelFunc:         cancel,
	}

	// Start the cleanup goroutine
	go rm.cleanupLoop()

	return rm
}

// RegisterResource registers a resource with the manager
func (rm *ResourceManager) RegisterResource(id string, resourceType ResourceType, cleanupFunc func() error, metadata map[string]interface{}) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	// Check if we're at the maximum number of resources
	if len(rm.resources) >= rm.config.MaxResources {
		rm.logger.Warn("Maximum number of resources reached, cleaning up oldest resources",
			zap.Int("maxResources", rm.config.MaxResources),
		)
		rm.cleanupOldestResourcesLocked(10) // Clean up 10 oldest resources
	}

	// Create the resource
	resource := &Resource{
		ID:          id,
		Type:        resourceType,
		CleanupFunc: cleanupFunc,
		CreatedAt:   time.Now(),
		LastUsedAt:  time.Now(),
		Metadata:    metadata,
	}

	// Add the resource
	rm.resources[id] = resource
	rm.resourcesCreated++
	rm.resourceTypeCount[resourceType]++

	rm.logger.Debug("Registered resource",
		zap.String("id", id),
		zap.String("type", string(resourceType)),
	)
}

// UpdateResourceUsage updates the last used time for a resource
func (rm *ResourceManager) UpdateResourceUsage(id string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	// Find the resource
	resource, exists := rm.resources[id]
	if !exists {
		return
	}

	// Update the last used time
	resource.LastUsedAt = time.Now()
}

// CleanupResource cleans up a resource
func (rm *ResourceManager) CleanupResource(id string) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	// Find the resource
	resource, exists := rm.resources[id]
	if !exists {
		return nil
	}

	// Clean up the resource
	err := resource.CleanupFunc()
	if err != nil {
		rm.logger.Error("Failed to clean up resource",
			zap.String("id", id),
			zap.String("type", string(resource.Type)),
			zap.Error(err),
		)
		return err
	}

	// Remove the resource
	delete(rm.resources, id)
	rm.resourcesCleaned++
	rm.resourceTypeCount[resource.Type]--

	rm.logger.Debug("Cleaned up resource",
		zap.String("id", id),
		zap.String("type", string(resource.Type)),
	)

	return nil
}

// CleanupResourcesByType cleans up all resources of a specific type
func (rm *ResourceManager) CleanupResourcesByType(resourceType ResourceType) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	// Find all resources of the specified type
	for id, resource := range rm.resources {
		if resource.Type == resourceType {
			// Clean up the resource
			err := resource.CleanupFunc()
			if err != nil {
				rm.logger.Error("Failed to clean up resource",
					zap.String("id", id),
					zap.String("type", string(resource.Type)),
					zap.Error(err),
				)
				continue
			}

			// Remove the resource
			delete(rm.resources, id)
			rm.resourcesCleaned++
			rm.resourceTypeCount[resource.Type]--

			rm.logger.Debug("Cleaned up resource",
				zap.String("id", id),
				zap.String("type", string(resource.Type)),
			)
		}
	}
}

// CleanupUnusedResources cleans up resources that haven't been used for a while
func (rm *ResourceManager) CleanupUnusedResources() {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	startTime := time.Now()
	cleaned := 0

	// Find all unused resources
	for id, resource := range rm.resources {
		if time.Since(resource.LastUsedAt) > rm.config.ResourceTimeout {
			// Clean up the resource
			err := resource.CleanupFunc()
			if err != nil {
				rm.logger.Error("Failed to clean up resource",
					zap.String("id", id),
					zap.String("type", string(resource.Type)),
					zap.Error(err),
				)
				rm.resourcesLeaked++
				rm.resourceTypeLeaked[resource.Type]++
				continue
			}

			// Remove the resource
			delete(rm.resources, id)
			rm.resourcesCleaned++
			rm.resourceTypeCount[resource.Type]--
			cleaned++

			rm.logger.Debug("Cleaned up unused resource",
				zap.String("id", id),
				zap.String("type", string(resource.Type)),
				zap.Duration("unusedTime", time.Since(resource.LastUsedAt)),
			)
		}
	}

	// Update metrics
	rm.lastCleanupTime = time.Now()
	rm.totalCleanupTime += time.Since(startTime)
	rm.cleanupCount++

	if cleaned > 0 {
		rm.logger.Info("Cleaned up unused resources",
			zap.Int("cleaned", cleaned),
			zap.Duration("duration", time.Since(startTime)),
		)
	}
}

// cleanupOldestResourcesLocked cleans up the oldest resources (must be called with lock held)
func (rm *ResourceManager) cleanupOldestResourcesLocked(count int) {
	// Create a slice of resources sorted by creation time
	type resourceWithID struct {
		id       string
		resource *Resource
	}
	resources := make([]resourceWithID, 0, len(rm.resources))
	for id, resource := range rm.resources {
		resources = append(resources, resourceWithID{id, resource})
	}

	// Sort by creation time (oldest first)
	sort.Slice(resources, func(i, j int) bool {
		return resources[i].resource.CreatedAt.Before(resources[j].resource.CreatedAt)
	})

	// Clean up the oldest resources
	for i := 0; i < count && i < len(resources); i++ {
		id := resources[i].id
		resource := resources[i].resource

		// Clean up the resource
		err := resource.CleanupFunc()
		if err != nil {
			rm.logger.Error("Failed to clean up resource",
				zap.String("id", id),
				zap.String("type", string(resource.Type)),
				zap.Error(err),
			)
			rm.resourcesLeaked++
			rm.resourceTypeLeaked[resource.Type]++
			continue
		}

		// Remove the resource
		delete(rm.resources, id)
		rm.resourcesCleaned++
		rm.resourceTypeCount[resource.Type]--

		rm.logger.Debug("Cleaned up oldest resource",
			zap.String("id", id),
			zap.String("type", string(resource.Type)),
			zap.Time("createdAt", resource.CreatedAt),
		)
	}
}

// cleanupLoop runs the cleanup process periodically
func (rm *ResourceManager) cleanupLoop() {
	ticker := time.NewTicker(rm.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rm.CleanupUnusedResources()
		case <-rm.ctx.Done():
			return
		}
	}
}

// Shutdown shuts down the resource manager and cleans up all resources
func (rm *ResourceManager) Shutdown() {
	// Cancel the cleanup goroutine
	rm.cancelFunc()

	rm.mu.Lock()
	defer rm.mu.Unlock()

	rm.logger.Info("Shutting down resource manager",
		zap.Int("resourceCount", len(rm.resources)),
	)

	// Clean up all resources
	for id, resource := range rm.resources {
		// Clean up the resource
		err := resource.CleanupFunc()
		if err != nil {
			rm.logger.Error("Failed to clean up resource during shutdown",
				zap.String("id", id),
				zap.String("type", string(resource.Type)),
				zap.Error(err),
			)
			rm.resourcesLeaked++
			rm.resourceTypeLeaked[resource.Type]++
			continue
		}

		// Remove the resource
		delete(rm.resources, id)
		rm.resourcesCleaned++
		rm.resourceTypeCount[resource.Type]--

		rm.logger.Debug("Cleaned up resource during shutdown",
			zap.String("id", id),
			zap.String("type", string(resource.Type)),
		)
	}

	rm.logger.Info("Resource manager shutdown complete",
		zap.Int("resourcesCleaned", rm.resourcesCleaned),
		zap.Int("resourcesLeaked", rm.resourcesLeaked),
	)
}

// GetStats gets the resource manager statistics
func (rm *ResourceManager) GetStats() map[string]interface{} {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	stats := make(map[string]interface{})
	stats["resourceCount"] = len(rm.resources)
	stats["resourcesCreated"] = rm.resourcesCreated
	stats["resourcesCleaned"] = rm.resourcesCleaned
	stats["resourcesLeaked"] = rm.resourcesLeaked
	stats["lastCleanupTime"] = rm.lastCleanupTime
	stats["totalCleanupTime"] = rm.totalCleanupTime
	stats["cleanupCount"] = rm.cleanupCount
	stats["resourceTypeCount"] = rm.resourceTypeCount
	stats["resourceTypeLeaked"] = rm.resourceTypeLeaked

	// Calculate average cleanup time
	if rm.cleanupCount > 0 {
		stats["averageCleanupTime"] = rm.totalCleanupTime / time.Duration(rm.cleanupCount)
	} else {
		stats["averageCleanupTime"] = time.Duration(0)
	}

	return stats
}

// GetResourceCount gets the number of resources
func (rm *ResourceManager) GetResourceCount() int {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	return len(rm.resources)
}

// GetResourceCountByType gets the number of resources by type
func (rm *ResourceManager) GetResourceCountByType(resourceType ResourceType) int {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	return rm.resourceTypeCount[resourceType]
}

// GetResourcesCreated gets the number of resources created
func (rm *ResourceManager) GetResourcesCreated() int {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	return rm.resourcesCreated
}

// GetResourcesCleaned gets the number of resources cleaned
func (rm *ResourceManager) GetResourcesCleaned() int {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	return rm.resourcesCleaned
}

// GetResourcesLeaked gets the number of resources leaked
func (rm *ResourceManager) GetResourcesLeaked() int {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	return rm.resourcesLeaked
}

