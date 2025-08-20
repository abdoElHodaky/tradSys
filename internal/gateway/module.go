package gateway

import (
	"go.uber.org/fx"
)

// Module provides the API Gateway module for fx
var Module = fx.Options(
	fx.Provide(NewServer),
	fx.Provide(NewRouter),
	fx.Provide(NewServiceProxy),
	fx.Provide(NewMiddleware),
)

