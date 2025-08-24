package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/architecture/fx/lazy"
	"go.uber.org/zap"
)

// This example demonstrates how to use lazy loading in a standalone application

// Config contains configuration for the component
type Config struct {
	Name     string
	Timeout  time.Duration
	MaxItems int
}

// Component is a resource-intensive component
type Component struct {
	config Config
	logger *zap.Logger
	data   map[string]interface{}
}

// NewComponent creates a new component
func NewComponent(config Config, logger *zap.Logger) (*Component, error) {
	logger.Info("Initializing component", zap.String("name", config.Name))
	
	// Simulate resource-intensive initialization
	time.Sleep(2 * time.Second)
	
	return &Component{
		config: config,
		logger: logger,
		data:   make(map[string]interface{}),
	}, nil
}

// DoSomething does something with the component
func (c *Component) DoSomething() error {
	c.logger.Info("Doing something with component", zap.String("name", c.config.Name))
	
	// Simulate work
	time.Sleep(500 * time.Millisecond)
	
	return nil
}

// Cleanup cleans up the component
func (c *Component) Cleanup() error {
	c.logger.Info("Cleaning up component", zap.String("name", c.config.Name))
	
	// Simulate cleanup
	time.Sleep(500 * time.Millisecond)
	
	return nil
}

func main() {
	// Create a logger
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()
	
	// Create metrics
	metrics := lazy.NewLazyLoadingMetrics()
	
	// Create a lazy provider
	provider := lazy.NewLazyProvider(
		"example-component",
		func(logger *zap.Logger) (*Component, error) {
			config := Config{
				Name:     "example",
				Timeout:  5 * time.Second,
				MaxItems: 100,
			}
			return NewComponent(config, logger)
		},
		logger,
		metrics,
	)
	
	// The component is not initialized yet
	fmt.Println("Component initialized:", provider.IsInitialized())
	
	// Get the component (this will initialize it)
	component, err := getComponent(provider)
	if err != nil {
		log.Fatalf("Failed to get component: %v", err)
	}
	
	// The component is now initialized
	fmt.Println("Component initialized:", provider.IsInitialized())
	
	// Use the component
	if err := component.DoSomething(); err != nil {
		log.Fatalf("Failed to do something: %v", err)
	}
	
	// Get metrics
	fmt.Println("Initialization count:", metrics.GetInitializationCount("example-component"))
	fmt.Println("Initialization error count:", metrics.GetInitializationErrorCount("example-component"))
	fmt.Println("Average initialization time:", metrics.GetAverageInitializationTime("example-component"))
	
	// Clean up
	if err := component.Cleanup(); err != nil {
		log.Fatalf("Failed to clean up: %v", err)
	}
}

// getComponent gets the component, initializing it if necessary
func getComponent(provider *lazy.LazyProvider) (*Component, error) {
	instance, err := provider.Get()
	if err != nil {
		return nil, err
	}
	return instance.(*Component), nil
}

// Example of using lazy loading with context
func exampleWithContext() {
	// Create a logger
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()
	
	// Create metrics
	metrics := lazy.NewLazyLoadingMetrics()
	
	// Create a lazy provider
	provider := lazy.NewLazyProvider(
		"example-component",
		func(logger *zap.Logger) (*Component, error) {
			config := Config{
				Name:     "example",
				Timeout:  5 * time.Second,
				MaxItems: 100,
			}
			return NewComponent(config, logger)
		},
		logger,
		metrics,
	)
	
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	
	// Try to get the component with timeout
	component, err := getComponentWithContext(ctx, provider)
	if err != nil {
		log.Printf("Failed to get component: %v", err)
		return
	}
	
	// Use the component
	if err := component.DoSomething(); err != nil {
		log.Printf("Failed to do something: %v", err)
		return
	}
	
	// Clean up
	if err := component.Cleanup(); err != nil {
		log.Printf("Failed to clean up: %v", err)
		return
	}
}

// getComponentWithContext gets the component with context
func getComponentWithContext(ctx context.Context, provider *lazy.LazyProvider) (*Component, error) {
	// Create a channel for the result
	resultCh := make(chan struct {
		component *Component
		err       error
	})
	
	// Get the component in a goroutine
	go func() {
		instance, err := provider.Get()
		if err != nil {
			resultCh <- struct {
				component *Component
				err       error
			}{nil, err}
			return
		}
		
		resultCh <- struct {
			component *Component
			err       error
		}{instance.(*Component), nil}
	}()
	
	// Wait for the result or context cancellation
	select {
	case result := <-resultCh:
		return result.component, result.err
	case <-ctx.Done():
		return nil, fmt.Errorf("context canceled: %w", ctx.Err())
	}
}

