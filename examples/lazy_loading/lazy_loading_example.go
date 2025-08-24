package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/architecture/fx/lazy"
	strategyfx "github.com/abdoElHodaky/tradSys/internal/strategy/fx"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// This example demonstrates how to use lazy loading with fx

func main() {
	// Create a logger
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// Create an fx application with lazy loading
	app := fx.New(
		// Provide the logger
		fx.Provide(func() *zap.Logger {
			return logger
		}),

		// Include the lazy loading module
		lazy.Module,

		// Include the lazy strategy module
		strategyfx.LazyModule,

		// Register a custom lazy component
		lazy.ProvideLazy("expensive-component", func() (*ExpensiveComponent, error) {
			logger.Info("Initializing expensive component (this should be deferred)")
			return NewExpensiveComponent(logger), nil
		}),

		// Register a component that uses the lazy component
		fx.Provide(func(provider *lazy.LazyProvider) *ComponentUser {
			return NewComponentUser(provider, logger)
		}),

		// Register lifecycle hooks
		fx.Invoke(func(
			lifecycle fx.Lifecycle,
			componentUser *ComponentUser,
			strategyManagerProvider *lazy.LazyProvider,
		) {
			lifecycle.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					// Start a goroutine to demonstrate lazy loading
					go func() {
						// Wait a bit to let the application start
						time.Sleep(2 * time.Second)
						logger.Info("Application started, now using components")

						// Use the component user, which will trigger lazy initialization
						componentUser.UseComponent()

						// Get the strategy manager, which will trigger lazy initialization
						manager, err := strategyfx.GetStrategyManager(strategyManagerProvider)
						if err != nil {
							logger.Error("Failed to get strategy manager", zap.Error(err))
							return
						}

						logger.Info("Got strategy manager", zap.String("manager", fmt.Sprintf("%T", manager)))
					}()
					return nil
				},
			})
		}),
	)

	// Start the application
	startCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := app.Start(startCtx); err != nil {
		logger.Fatal("Failed to start application", zap.Error(err))
	}

	// Wait for signal to stop
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	// Stop the application
	stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := app.Stop(stopCtx); err != nil {
		logger.Fatal("Failed to stop application", zap.Error(err))
	}
}

// ExpensiveComponent is a component that is expensive to initialize
type ExpensiveComponent struct {
	logger *zap.Logger
}

// NewExpensiveComponent creates a new expensive component
func NewExpensiveComponent(logger *zap.Logger) *ExpensiveComponent {
	// Simulate expensive initialization
	time.Sleep(1 * time.Second)
	return &ExpensiveComponent{
		logger: logger,
	}
}

// DoSomething does something with the component
func (c *ExpensiveComponent) DoSomething() {
	c.logger.Info("Doing something with expensive component")
}

// ComponentUser uses the expensive component
type ComponentUser struct {
	provider *lazy.LazyProvider
	logger   *zap.Logger
}

// NewComponentUser creates a new component user
func NewComponentUser(provider *lazy.LazyProvider, logger *zap.Logger) *ComponentUser {
	return &ComponentUser{
		provider: provider,
		logger:   logger,
	}
}

// UseComponent uses the expensive component
func (u *ComponentUser) UseComponent() {
	u.logger.Info("Using expensive component")

	// Get the component, which will trigger lazy initialization
	component, err := u.provider.Get()
	if err != nil {
		u.logger.Error("Failed to get component", zap.Error(err))
		return
	}

	// Use the component
	expensiveComponent := component.(*ExpensiveComponent)
	expensiveComponent.DoSomething()
}

