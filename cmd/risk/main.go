package main

import (
	"github.com/abdoElHodaky/tradSys/internal/common"
	"github.com/abdoElHodaky/tradSys/internal/config"
	"github.com/abdoElHodaky/tradSys/internal/db/repositories"
	"github.com/abdoElHodaky/tradSys/internal/micro"
	"github.com/abdoElHodaky/tradSys/internal/risk"
	"go.uber.org/fx"
)

func main() {
	app := common.MicroserviceApp("risk",
		config.Module,
		micro.Module,
		repositories.RiskRepositoryModule,
		risk.Module,
		risk.ServiceModule,
		common.RegisterServiceHandler("risk", risk.RegisterRiskServiceHandler),
	)

	app.Run()
}
