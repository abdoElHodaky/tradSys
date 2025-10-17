package risk

import (
	"go.uber.org/fx"
)

// Module provides the risk module for fx
var Module = fx.Options(
	fx.Provide(NewHandler),
)

// ServiceModule provides the risk service module for fx
var ServiceModule = fx.Options(
	fx.Provide(NewService),
)
