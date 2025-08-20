package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/abdoElHodaky/tradSys/internal/architecture/fx"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func main() {
	// Create a logger
	logger, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	// Create an application with Fx
	app := fx.New(
		// Provide the logger
		fx.Provide(func() *zap.Logger {
			return logger
		}),

		// Include the core architecture module
		fx.Options(fx.Module),

		// Include the service modules
		fx.Options(
			fx.NewMarketDataModule(),
			fx.NewOrdersModule(),
			fx.NewRiskModule(),
		),

		// Register a signal handler to gracefully shut down the application
		fx.Invoke(func(lc fx.Lifecycle, logger *zap.Logger) {
			lc.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					// Create a channel to listen for signals
					sigChan := make(chan os.Signal, 1)
					signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

					// Start a goroutine to listen for signals
					go func() {
						sig := <-sigChan
						logger.Info("Received signal", zap.String("signal", sig.String()))
						// Stop the application
						_ = app.Stop(context.Background())
					}()

					return nil
				},
				OnStop: func(ctx context.Context) error {
					logger.Info("Application stopped")
					return nil
				},
			})
		}),
	)

	// Start the application
	if err := app.Start(context.Background()); err != nil {
		logger.Fatal("Failed to start application", zap.Error(err))
	}

	// Block until the application is stopped
	<-app.Done()
}

