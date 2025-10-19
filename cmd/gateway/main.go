package main

import (
	"github.com/abdoElHodaky/tradSys/internal/unified-config"
	"github.com/abdoElHodaky/tradSys/internal/gateway"
	"github.com/abdoElHodaky/tradSys/internal/micro"
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
		// TODO: Add monitoring.MetricsModule when available
		gateway.Module,
		fx.Invoke(func(server *gateway.Server) {
			// This function is invoked when the application starts
			// The server is already started by the gateway.Module
			logger.Info("API Gateway started")
		}),
	)

	app.Run()
}
