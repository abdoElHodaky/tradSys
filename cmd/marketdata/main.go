package main

import (
	"github.com/abdoElHodaky/tradSys/internal/config"
	"github.com/abdoElHodaky/tradSys/internal/db/repositories"
	"github.com/abdoElHodaky/tradSys/internal/marketdata"
	"github.com/abdoElHodaky/tradSys/internal/micro"
	"github.com/abdoElHodaky/tradSys/proto/marketdata"
	"go-micro.dev/v4"
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
		repositories.MarketDataRepositoryModule,
		marketdata.Module,
		marketdata.ServiceModule,
		fx.Invoke(registerMarketDataHandler),
	)

	app.Run()
}

func registerMarketDataHandler(
	lc fx.Lifecycle,
	logger *zap.Logger,
	service *micro.Service,
	handler *marketdata.Handler,
) {
	// Register the handler with the service
	if err := marketdata.RegisterMarketDataServiceHandler(service.Server(), handler); err != nil {
		logger.Fatal("Failed to register handler", zap.Error(err))
	}

	logger.Info("Market data service registered")
}
