package main

import (
	"github.com/abdoElHodaky/tradSys/internal/config"
	"github.com/abdoElHodaky/tradSys/internal/db/repositories"
	"github.com/abdoElHodaky/tradSys/internal/micro"
	"github.com/abdoElHodaky/tradSys/internal/orders"
	"github.com/abdoElHodaky/tradSys/proto/orders"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	app := fx.New(
		fx.Supply(logger),
		config.Module,
		micro.Module,
		repositories.OrderRepositoryModule,
		orders.Module,
		orders.ServiceModule,
		fx.Invoke(registerOrderHandler),
	)

	app.Run()
}

func registerOrderHandler(
	lc fx.Lifecycle,
	logger *zap.Logger,
	service *micro.Service,
	handler *orders.Handler,
) {
	// Register the handler with the service
	if err := orders.RegisterOrderServiceHandler(service.Server(), handler); err != nil {
		logger.Fatal("Failed to register handler", zap.Error(err))
	}

	logger.Info("Order service registered")
}

