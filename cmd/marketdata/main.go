package main

import (
	"context"
	"fmt"

	"github.com/abdoElHodaky/tradSys/internal/unified-config"
	"github.com/abdoElHodaky/tradSys/internal/db/repositories"
	"github.com/abdoElHodaky/tradSys/internal/grpc/server"
	"github.com/abdoElHodaky/tradSys/internal/marketdata"
	marketdatapb "github.com/abdoElHodaky/tradSys/proto/marketdata"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	app := fx.New(
		fx.Supply(logger),
		config.Module,
		repositories.RepositoriesModule,
		marketdata.Module,
		marketdata.HandlerModule,
		fx.Provide(func(logger *zap.Logger) *server.Server {
			return server.NewServer(logger, server.DefaultServerOptions())
		}),
		fx.Invoke(registerMarketDataHandler),
		fx.Invoke(startGRPCServer),
	)

	app.Run()
}

func registerMarketDataHandler(
	logger *zap.Logger,
	grpcServer *server.Server,
	handler *marketdata.Handler,
) {
	// Register the handler with the gRPC server
	marketdatapb.RegisterMarketDataServiceServer(grpcServer.GetServer(), handler)

	logger.Info("Market data service registered")
}

func startGRPCServer(
	lc fx.Lifecycle,
	logger *zap.Logger,
	grpcServer *server.Server,
	config *config.Config,
) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				address := ":8080" // Default address
				if config.Server.Port != 0 {
					address = fmt.Sprintf(":%d", config.Server.Port)
				}
				
				logger.Info("Starting gRPC server", zap.String("address", address))
				if err := grpcServer.Start(ctx, address); err != nil {
					logger.Fatal("Failed to start gRPC server", zap.Error(err))
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("Stopping gRPC server")
			grpcServer.Stop()
			return nil
		},
	})
}
