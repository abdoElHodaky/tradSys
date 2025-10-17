package main

import (
	"github.com/abdoElHodaky/tradSys/internal/common"
	"github.com/abdoElHodaky/tradSys/internal/config"
	"github.com/abdoElHodaky/tradSys/internal/db/repositories"
	"github.com/abdoElHodaky/tradSys/internal/marketdata"
	"github.com/abdoElHodaky/tradSys/internal/micro"
	"go.uber.org/fx"
)

func main() {
	app := common.MicroserviceApp("marketdata",
		config.Module,
		micro.Module,
		repositories.MarketDataRepositoryModule,
		marketdata.Module,
		marketdata.ServiceModule,
		common.RegisterServiceHandler("marketdata", marketdata.RegisterMarketDataServiceHandler),
	)

	app.Run()
}
