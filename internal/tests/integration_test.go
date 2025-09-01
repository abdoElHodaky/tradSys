package tests

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/config"
	"github.com/abdoElHodaky/tradSys/internal/micro"
	"github.com/abdoElHodaky/tradSys/internal/performance"
	"github.com/abdoElHodaky/tradSys/internal/resource"
	"github.com/abdoElHodaky/tradSys/internal/trading/order_matching"
	gomicro "go-micro.dev/v4"
	"go-micro.dev/v4/registry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

// TestFrameworkStandardization verifies that the framework standardization works correctly
func TestFrameworkStandardization(t *testing.T) {
	// Create a test logger
	logger := zaptest.NewLogger(t)

	// Create a test config
	cfg := &config.Config{
		Service: config.Service{
			Name:        "test-service",
			Version:     "1.0.0",
			Address:     ":0",
			Environment: "test",
		},
		Registry: config.RegistryConfig{
			Type:             "mdns",
			Addresses:        []string{},
			TTL:              5 * time.Second,
			RegisterInterval: 2 * time.Second,
		},
	}

	// Create a test lifecycle
	lc := fxtest.NewLifecycle(t)

	// Create a service
	service, err := micro.NewService(micro.ServiceParams{
		Logger:    logger,
		Config:    cfg,
		Lifecycle: lc,
	})

	// Assert that the service was created successfully
	assert.NoError(t, err)
	assert.NotNil(t, service)

	// Assert that the service has the correct name
	assert.Equal(t, "test-service", service.Name())

	// Assert that the service has the correct version
	assert.Equal(t, "1.0.0", service.Version())

	// Start the service
	assert.NoError(t, lc.Start(context.Background()))

	// Stop the service
	assert.NoError(t, lc.Stop(context.Background()))
}

// TestServiceRegistration verifies that service registration works correctly
func TestServiceRegistration(t *testing.T) {
	// Skip if running in CI
	if os.Getenv("CI") != "" {
		t.Skip("Skipping test in CI environment")
	}

	// Create a test logger
	logger := zaptest.NewLogger(t)

	// Create a test config
	cfg := &config.Config{
		Service: config.Service{
			Name:        "test-registration-service",
			Version:     "1.0.0",
			Address:     ":0",
			Environment: "test",
		},
		Registry: config.RegistryConfig{
			Type:             "mdns",
			Addresses:        []string{},
			TTL:              5 * time.Second,
			RegisterInterval: 2 * time.Second,
		},
	}

	// Create a test lifecycle
	lc := fxtest.NewLifecycle(t)

	// Create a service
	service, err := micro.NewService(micro.ServiceParams{
		Logger:    logger,
		Config:    cfg,
		Lifecycle: lc,
	})

	// Assert that the service was created successfully
	assert.NoError(t, err)
	assert.NotNil(t, service)

	// Start the service
	assert.NoError(t, lc.Start(context.Background()))

	// Wait for the service to register
	time.Sleep(3 * time.Second)

	// Get the registry
	reg := service.Options().Registry

	// Get services
	services, err := reg.GetService(cfg.Service.Name)
	assert.NoError(t, err)
	assert.NotEmpty(t, services)

	// Assert that the service is registered
	assert.Equal(t, cfg.Service.Name, services[0].Name)
	assert.Equal(t, cfg.Service.Version, services[0].Version)

	// Stop the service
	assert.NoError(t, lc.Stop(context.Background()))
}

// TestServiceDiscovery verifies that service discovery works correctly
func TestServiceDiscovery(t *testing.T) {
	// Skip if running in CI
	if os.Getenv("CI") != "" {
		t.Skip("Skipping test in CI environment")
	}

	// Create a test logger
	logger := zaptest.NewLogger(t)

	// Create a test config for the first service
	cfg1 := &config.Config{
		Service: config.Service{
			Name:        "test-discovery-service-1",
			Version:     "1.0.0",
			Address:     ":0",
			Environment: "test",
		},
		Registry: config.RegistryConfig{
			Type:             "mdns",
			Addresses:        []string{},
			TTL:              5 * time.Second,
			RegisterInterval: 2 * time.Second,
		},
	}

	// Create a test config for the second service
	cfg2 := &config.Config{
		Service: config.Service{
			Name:        "test-discovery-service-2",
			Version:     "1.0.0",
			Address:     ":0",
			Environment: "test",
		},
		Registry: config.RegistryConfig{
			Type:             "mdns",
			Addresses:        []string{},
			TTL:              5 * time.Second,
			RegisterInterval: 2 * time.Second,
		},
	}

	// Create test lifecycles
	lc1 := fxtest.NewLifecycle(t)
	lc2 := fxtest.NewLifecycle(t)

	// Create services
	service1, err := micro.NewService(micro.ServiceParams{
		Logger:    logger,
		Config:    cfg1,
		Lifecycle: lc1,
	})
	assert.NoError(t, err)
	assert.NotNil(t, service1)

	service2, err := micro.NewService(micro.ServiceParams{
		Logger:    logger,
		Config:    cfg2,
		Lifecycle: lc2,
	})
	assert.NoError(t, err)
	assert.NotNil(t, service2)

	// Start the services
	assert.NoError(t, lc1.Start(context.Background()))
	assert.NoError(t, lc2.Start(context.Background()))

	// Wait for the services to register
	time.Sleep(3 * time.Second)

	// Get the registry
	reg := service1.Options().Registry

	// Get services
	services1, err := reg.GetService(cfg1.Service.Name)
	assert.NoError(t, err)
	assert.NotEmpty(t, services1)

	services2, err := reg.GetService(cfg2.Service.Name)
	assert.NoError(t, err)
	assert.NotEmpty(t, services2)

	// Assert that the services are registered
	assert.Equal(t, cfg1.Service.Name, services1[0].Name)
	assert.Equal(t, cfg1.Service.Version, services1[0].Version)

	assert.Equal(t, cfg2.Service.Name, services2[0].Name)
	assert.Equal(t, cfg2.Service.Version, services2[0].Version)

	// Stop the services
	assert.NoError(t, lc1.Stop(context.Background()))
	assert.NoError(t, lc2.Stop(context.Background()))
}

// TestResourceManager verifies that the resource manager works correctly
func TestResourceManager(t *testing.T) {
	// Create a test logger
	logger := zaptest.NewLogger(t)

	// Create a resource manager
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
}

// TestProfiler verifies that the profiler works correctly
func TestProfiler(t *testing.T) {
	// Create a test logger
	logger := zaptest.NewLogger(t)

	// Create a temporary directory for profiles
	tempDir, err := os.MkdirTemp("", "profiler-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create profiler options
	options := performance.DefaultProfilerOptions()
	options.ProfileDir = tempDir

	// Create a profiler
	profiler, err := performance.NewProfiler(options, logger)
	require.NoError(t, err)

	// Start the profiler
	err = profiler.Start()
	require.NoError(t, err)

	// Verify that the profiler is started
	assert.True(t, profiler.IsStarted())

	// Take a snapshot
	err = profiler.TakeSnapshot("test-snapshot")
	require.NoError(t, err)

	// Get memory stats
	memStats := profiler.GetMemoryStats()
	assert.NotNil(t, memStats)
	assert.Contains(t, memStats, "alloc")
	assert.Contains(t, memStats, "totalAlloc")

	// Get goroutine stats
	goroutineStats := profiler.GetGoroutineStats()
	assert.NotNil(t, goroutineStats)
	assert.Contains(t, goroutineStats, "goroutines")

	// Stop the profiler
	err = profiler.Stop()
	require.NoError(t, err)

	// Verify that the profiler is stopped
	assert.False(t, profiler.IsStarted())
}

// TestOrderMatching verifies that the order matching engine works correctly
func TestOrderMatching(t *testing.T) {
	// Create a test logger
	logger := zaptest.NewLogger(t)

	// Create an order matching engine
	engine := order_matching.NewOrderMatchingEngine(logger)

	// Create an order book
	symbol := "BTC-USD"
	orderBook := engine.CreateOrderBook(symbol)
	assert.NotNil(t, orderBook)

	// Create a buy order
	buyOrder := order_matching.NewOrder(symbol, order_matching.OrderTypeLimit, order_matching.OrderSideBuy, 10000.0, 1.0, "user1")

	// Create a sell order
	sellOrder := order_matching.NewOrder(symbol, order_matching.OrderTypeLimit, order_matching.OrderSideSell, 10000.0, 0.5, "user2")

	// Place the orders
	err := engine.PlaceOrder(buyOrder)
	assert.NoError(t, err)

	err = engine.PlaceOrder(sellOrder)
	assert.NoError(t, err)

	// Get the order book snapshot
	snapshot, err := engine.GetOrderBookSnapshot(symbol, 10)
	assert.NoError(t, err)
	assert.NotNil(t, snapshot)

	// Verify that the buy order is partially filled
	buyOrderResult, err := engine.GetOrder(symbol, buyOrder.ID)
	assert.NoError(t, err)
	assert.Equal(t, order_matching.OrderStatusPartiallyFilled, buyOrderResult.Status)
	assert.Equal(t, 0.5, buyOrderResult.RemainingSize)

	// Verify that the sell order is filled
	_, err = engine.GetOrder(symbol, sellOrder.ID)
	assert.Error(t, err) // Order should be removed from the order book

	// Get engine stats
	stats := engine.GetStats()
	assert.NotNil(t, stats)
	assert.Equal(t, uint64(2), stats["ordersProcessed"])

	// Create a context for cleanup
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Run cleanup
	engine.Cleanup(ctx)
}

// TestMessageBatcher verifies that the message batcher works correctly
func TestMessageBatcher(t *testing.T) {
	// Create a test logger
	logger := zaptest.NewLogger(t)

	// Create a channel to receive batches
	batchCh := make(chan *performance.MessageBatch, 10)

	// Create a send function
	sendFunc := func(batch *performance.MessageBatch) error {
		batchCh <- batch
		return nil
	}

	// Create a message batcher
	config := performance.DefaultMessageBatcherConfig()
	config.BatchTimeout = 100 * time.Millisecond
	batcher := performance.NewMessageBatcher(config, sendFunc, logger)

	// Create a context for the batcher
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the batcher
	err := batcher.Start(ctx)
	require.NoError(t, err)

	// Add messages
	for i := 0; i < 5; i++ {
		message := &performance.BatchableMessage{
			Type:     "test",
			Data:     []byte(fmt.Sprintf(`{"id":%d}`, i)),
			Priority: performance.PriorityMedium,
		}
		err := batcher.AddMessage(message)
		assert.NoError(t, err)
	}

	// Wait for the batch to be sent
	select {
	case batch := <-batchCh:
		assert.Len(t, batch.Messages, 5)
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for batch")
	}

	// Add a critical message
	criticalMessage := &performance.BatchableMessage{
		Type:     "critical",
		Data:     []byte(`{"id":999}`),
		Priority: performance.PriorityCritical,
	}
	err = batcher.AddMessage(criticalMessage)
	assert.NoError(t, err)

	// Critical messages should be sent immediately
	select {
	case batch := <-batchCh:
		assert.Len(t, batch.Messages, 1)
		assert.Equal(t, "critical", batch.Messages[0].Type)
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for critical batch")
	}

	// Get batcher stats
	stats := batcher.GetStats()
	assert.NotNil(t, stats)
	assert.Equal(t, uint64(6), stats["messageCount"])

	// Stop the batcher
	err = batcher.Stop()
	require.NoError(t, err)
}

