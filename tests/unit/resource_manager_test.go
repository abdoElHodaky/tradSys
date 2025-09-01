package unit

import (
	"context"
	"testing"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestResourceManager(t *testing.T) {
	// Create a test logger
	logger := zaptest.NewLogger(t)

	t.Run("Basic Resource Management", func(t *testing.T) {
		// Create a resource manager with fast cleanup
		config := resource.DefaultResourceManagerConfig()
		config.CleanupInterval = 100 * time.Millisecond
		config.ResourceTimeout = 200 * time.Millisecond
		rm := resource.NewResourceManager(config, logger)

		// Register a resource
		resourceID := "test-resource"
		cleanupCalled := false
		rm.RegisterResource(resourceID, resource.ResourceTypeMemory, func() error {
			cleanupCalled = true
			return nil
		}, nil)

		// Verify that the resource is registered
		assert.Equal(t, 1, rm.GetResourceCount())
		assert.Equal(t, 1, rm.GetResourceCountByType(resource.ResourceTypeMemory))

		// Wait for the resource to be cleaned up
		time.Sleep(500 * time.Millisecond)

		// Verify that the resource was cleaned up
		assert.Equal(t, 0, rm.GetResourceCount())
		assert.True(t, cleanupCalled)

		// Shutdown the resource manager
		rm.Shutdown()
	})

	t.Run("Resource Usage Tracking", func(t *testing.T) {
		// Create a resource manager with fast cleanup
		config := resource.DefaultResourceManagerConfig()
		config.CleanupInterval = 100 * time.Millisecond
		config.ResourceTimeout = 200 * time.Millisecond
		rm := resource.NewResourceManager(config, logger)

		// Register a resource
		resourceID := "test-resource"
		cleanupCalled := false
		rm.RegisterResource(resourceID, resource.ResourceTypeMemory, func() error {
			cleanupCalled = true
			return nil
		}, nil)

		// Update resource usage to prevent cleanup
		for i := 0; i < 5; i++ {
			rm.UpdateResourceUsage(resourceID)
			time.Sleep(100 * time.Millisecond)
		}

		// Verify that the resource is still registered
		assert.Equal(t, 1, rm.GetResourceCount())
		assert.False(t, cleanupCalled)

		// Wait for the resource to be cleaned up
		time.Sleep(300 * time.Millisecond)

		// Verify that the resource was cleaned up
		assert.Equal(t, 0, rm.GetResourceCount())
		assert.True(t, cleanupCalled)

		// Shutdown the resource manager
		rm.Shutdown()
	})

	t.Run("Multiple Resource Types", func(t *testing.T) {
		// Create a resource manager
		config := resource.DefaultResourceManagerConfig()
		rm := resource.NewResourceManager(config, logger)

		// Register resources of different types
		rm.RegisterResource("conn1", resource.ResourceTypeConnection, func() error { return nil }, nil)
		rm.RegisterResource("file1", resource.ResourceTypeFile, func() error { return nil }, nil)
		rm.RegisterResource("mem1", resource.ResourceTypeMemory, func() error { return nil }, nil)
		rm.RegisterResource("conn2", resource.ResourceTypeConnection, func() error { return nil }, nil)

		// Verify resource counts
		assert.Equal(t, 4, rm.GetResourceCount())
		assert.Equal(t, 2, rm.GetResourceCountByType(resource.ResourceTypeConnection))
		assert.Equal(t, 1, rm.GetResourceCountByType(resource.ResourceTypeFile))
		assert.Equal(t, 1, rm.GetResourceCountByType(resource.ResourceTypeMemory))

		// Cleanup resources by type
		rm.CleanupResourcesByType(resource.ResourceTypeConnection)

		// Verify updated resource counts
		assert.Equal(t, 2, rm.GetResourceCount())
		assert.Equal(t, 0, rm.GetResourceCountByType(resource.ResourceTypeConnection))
		assert.Equal(t, 1, rm.GetResourceCountByType(resource.ResourceTypeFile))
		assert.Equal(t, 1, rm.GetResourceCountByType(resource.ResourceTypeMemory))

		// Shutdown the resource manager
		rm.Shutdown()
	})

	t.Run("Resource Cleanup Error Handling", func(t *testing.T) {
		// Create a resource manager
		config := resource.DefaultResourceManagerConfig()
		rm := resource.NewResourceManager(config, logger)

		// Register a resource with a failing cleanup function
		rm.RegisterResource("error-resource", resource.ResourceTypeCustom, func() error {
			return assert.AnError
		}, nil)

		// Attempt to clean up the resource
		err := rm.CleanupResource("error-resource")
		assert.Error(t, err)

		// Verify that the resource is still registered
		assert.Equal(t, 1, rm.GetResourceCount())

		// Shutdown the resource manager
		rm.Shutdown()

		// Verify that the resource was leaked
		stats := rm.GetStats()
		assert.Equal(t, 1, stats["resourcesLeaked"])
	})

	t.Run("Resource Manager Shutdown", func(t *testing.T) {
		// Create a resource manager
		config := resource.DefaultResourceManagerConfig()
		rm := resource.NewResourceManager(config, logger)

		// Register multiple resources
		cleanupCount := 0
		for i := 0; i < 5; i++ {
			rm.RegisterResource(
				fmt.Sprintf("resource-%d", i),
				resource.ResourceTypeMemory,
				func() error {
					cleanupCount++
					return nil
				},
				nil,
			)
		}

		// Verify initial resource count
		assert.Equal(t, 5, rm.GetResourceCount())

		// Shutdown the resource manager
		rm.Shutdown()

		// Verify that all resources were cleaned up
		assert.Equal(t, 0, rm.GetResourceCount())
		assert.Equal(t, 5, cleanupCount)
	})

	t.Run("Resource Manager Context Cancellation", func(t *testing.T) {
		// Create a resource manager with fast cleanup
		config := resource.DefaultResourceManagerConfig()
		config.CleanupInterval = 50 * time.Millisecond
		rm := resource.NewResourceManager(config, logger)

		// Create a context that will be cancelled
		ctx, cancel := context.WithCancel(context.Background())

		// Register resources
		for i := 0; i < 10; i++ {
			rm.RegisterResource(
				fmt.Sprintf("resource-%d", i),
				resource.ResourceTypeMemory,
				func() error { return nil },
				nil,
			)
		}

		// Cancel the context after a short delay
		go func() {
			time.Sleep(75 * time.Millisecond)
			cancel()
		}()

		// Run cleanup with the cancellable context
		rm.Cleanup(ctx)

		// Shutdown the resource manager
		rm.Shutdown()
	})
}

func TestResourceManagerMaxResources(t *testing.T) {
	// Create a test logger
	logger := zaptest.NewLogger(t)

	// Create a resource manager with a small max resources limit
	config := resource.DefaultResourceManagerConfig()
	config.MaxResources = 5
	rm := resource.NewResourceManager(config, logger)

	// Register resources up to the limit
	for i := 0; i < config.MaxResources; i++ {
		rm.RegisterResource(
			fmt.Sprintf("resource-%d", i),
			resource.ResourceTypeMemory,
			func() error { return nil },
			nil,
		)
	}

	// Verify resource count
	assert.Equal(t, config.MaxResources, rm.GetResourceCount())

	// Register one more resource, which should trigger cleanup of the oldest
	rm.RegisterResource(
		"one-more-resource",
		resource.ResourceTypeMemory,
		func() error { return nil },
		nil,
	)

	// Verify that we still have MaxResources resources
	assert.Equal(t, config.MaxResources, rm.GetResourceCount())

	// Shutdown the resource manager
	rm.Shutdown()
}

func TestResourceManagerMetadata(t *testing.T) {
	// Create a test logger
	logger := zaptest.NewLogger(t)

	// Create a resource manager
	config := resource.DefaultResourceManagerConfig()
	rm := resource.NewResourceManager(config, logger)

	// Register a resource with metadata
	metadata := map[string]interface{}{
		"owner":      "test-user",
		"created_at": time.Now(),
		"size":       1024,
		"tags":       []string{"test", "resource"},
	}

	rm.RegisterResource(
		"metadata-resource",
		resource.ResourceTypeCustom,
		func() error { return nil },
		metadata,
	)

	// Verify that the resource is registered
	assert.Equal(t, 1, rm.GetResourceCount())

	// Shutdown the resource manager
	rm.Shutdown()
}

func TestResourceManagerStats(t *testing.T) {
	// Create a test logger
	logger := zaptest.NewLogger(t)

	// Create a resource manager
	config := resource.DefaultResourceManagerConfig()
	rm := resource.NewResourceManager(config, logger)

	// Register resources of different types
	rm.RegisterResource("conn1", resource.ResourceTypeConnection, func() error { return nil }, nil)
	rm.RegisterResource("file1", resource.ResourceTypeFile, func() error { return nil }, nil)
	rm.RegisterResource("mem1", resource.ResourceTypeMemory, func() error { return nil }, nil)

	// Clean up one resource
	rm.CleanupResource("conn1")

	// Get stats
	stats := rm.GetStats()

	// Verify stats
	assert.Equal(t, 2, stats["resourceCount"])
	assert.Equal(t, 3, stats["resourcesCreated"])
	assert.Equal(t, 1, stats["resourcesCleaned"])
	assert.Equal(t, 0, stats["resourcesLeaked"])

	// Verify resource type counts
	resourceTypeCount := stats["resourceTypeCount"].(map[resource.ResourceType]int)
	assert.Equal(t, 0, resourceTypeCount[resource.ResourceTypeConnection])
	assert.Equal(t, 1, resourceTypeCount[resource.ResourceTypeFile])
	assert.Equal(t, 1, resourceTypeCount[resource.ResourceTypeMemory])

	// Shutdown the resource manager
	rm.Shutdown()
}

