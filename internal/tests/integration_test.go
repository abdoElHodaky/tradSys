package tests

import (
	"context"
	"testing"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/config"
	"github.com/abdoElHodaky/tradSys/internal/micro"
	gomicro "go-micro.dev/v4"
	"go-micro.dev/v4/registry"
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
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Verify that the service was created with the correct options
	if service.Options().Server.Options().Name != "test-service" {
		t.Errorf("Expected service name to be 'test-service', got '%s'", service.Options().Server.Options().Name)
	}

	if service.Options().Server.Options().Version != "1.0.0" {
		t.Errorf("Expected service version to be '1.0.0', got '%s'", service.Options().Server.Options().Version)
	}

	// Create a registry
	reg := micro.NewRegistry(micro.RegistryParams{
		Logger:    logger,
		Config:    cfg,
		Lifecycle: lc,
	})

	// Verify that the registry was created
	if reg == nil {
		t.Fatal("Expected registry to be created, got nil")
	}

	// Start the lifecycle
	ctx := context.Background()
	if err := lc.Start(ctx); err != nil {
		t.Fatalf("Failed to start lifecycle: %v", err)
	}

	// Stop the lifecycle
	if err := lc.Stop(ctx); err != nil {
		t.Fatalf("Failed to stop lifecycle: %v", err)
	}
}

// TestRegistryConfiguration verifies that the registry configuration works correctly
func TestRegistryConfiguration(t *testing.T) {
	// Create a test logger
	logger := zaptest.NewLogger(t)

	// Create a test config with custom TTL and interval
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
			TTL:              10 * time.Second,
			RegisterInterval: 5 * time.Second,
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
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Verify that the service was created with the correct TTL and interval
	opts := service.Options()
	if opts.RegisterTTL != 10*time.Second {
		t.Errorf("Expected TTL to be 10s, got %v", opts.RegisterTTL)
	}

	if opts.RegisterInterval != 5*time.Second {
		t.Errorf("Expected interval to be 5s, got %v", opts.RegisterInterval)
	}
}

// TestServiceCommunication verifies that services can communicate with each other
func TestServiceCommunication(t *testing.T) {
	// Skip in CI environment
	t.Skip("Skipping integration test in CI environment")

	// Create a test logger
	logger := zaptest.NewLogger(t)

	// Create a test app
	app := fx.New(
		fx.Supply(logger),
		fx.Supply(&config.Config{
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
		}),
		fx.Provide(micro.NewService),
		fx.Provide(micro.NewRegistry),
		fx.Invoke(func(service *micro.Service) {
			// Register a handler
			if err := service.Server().Handle(
				service.Server().NewHandler(
					&testHandler{},
				),
			); err != nil {
				t.Fatalf("Failed to register handler: %v", err)
			}
		}),
	)

	// Start the app
	ctx := context.Background()
	if err := app.Start(ctx); err != nil {
		t.Fatalf("Failed to start app: %v", err)
	}

	// Stop the app when the test is done
	defer func() {
		if err := app.Stop(ctx); err != nil {
			t.Fatalf("Failed to stop app: %v", err)
		}
	}()

	// Create a client
	client := gomicro.NewService(
		gomicro.Name("test-client"),
		gomicro.Registry(registry.NewRegistry()),
	)

	// Initialize the client
	if err := client.Init(); err != nil {
		t.Fatalf("Failed to initialize client: %v", err)
	}

	// Wait for service discovery
	time.Sleep(1 * time.Second)

	// Create a request
	req := client.Client().NewRequest("test-service", "TestHandler.Test", &TestRequest{
		Message: "Hello",
	})

	// Create a response
	rsp := &TestResponse{}

	// Call the service
	if err := client.Client().Call(ctx, req, rsp); err != nil {
		t.Fatalf("Failed to call service: %v", err)
	}

	// Verify the response
	if rsp.Message != "Hello World" {
		t.Errorf("Expected response message to be 'Hello World', got '%s'", rsp.Message)
	}
}

// TestRequest is a test request
type TestRequest struct {
	Message string `json:"message"`
}

// TestResponse is a test response
type TestResponse struct {
	Message string `json:"message"`
}

// testHandler is a test handler
type testHandler struct{}

// Test is a test method
func (h *testHandler) Test(ctx context.Context, req *TestRequest, rsp *TestResponse) error {
	rsp.Message = req.Message + " World"
	return nil
}

