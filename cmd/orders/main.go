package main

import (
	"github.com/abdoElHodaky/tradSys/internal/common"
	"github.com/abdoElHodaky/tradSys/internal/config"
	"github.com/abdoElHodaky/tradSys/internal/db/repositories"
	"github.com/abdoElHodaky/tradSys/internal/micro"
	"github.com/abdoElHodaky/tradSys/internal/orders"
	"go.uber.org/fx"
)

func main() {
	app := common.MicroserviceApp("orders",
		config.Module,
		micro.Module,
		repositories.OrderRepositoryModule,
		orders.Module,
		orders.ServiceModule,
		common.RegisterServiceHandler("orders", orders.RegisterOrderServiceHandler),
	)

	app.Run()
}
