package orders

import (
	"go.uber.org/fx"
)

// Module provides the orders module for fx
var Module = fx.Options(
	fx.Provide(NewHandler),
)

// ServiceModule provides the orders service module for fx
var ServiceModule = fx.Options(
	fx.Provide(NewService),
)
