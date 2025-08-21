package main

import (
	"github.com/abdoElHodaky/tradSys/internal/api"
	"github.com/abdoElHodaky/tradSys/internal/auth"
	"github.com/abdoElHodaky/tradSys/internal/config"
	"github.com/abdoElHodaky/tradSys/internal/db"
	"github.com/abdoElHodaky/tradSys/internal/gateway"
	"github.com/abdoElHodaky/tradSys/internal/orders"
	"github.com/abdoElHodaky/tradSys/internal/risk"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func main() {
	app := fx.New(
		// Provide core components
		fx.Provide(
			config.NewConfig,
			newLogger,
			newGinEngine,
		),
		
		// Include modules
		db.Module,
		auth.Module,
		orders.Module,
		risk.Module,
		api.Module,
		gateway.Module,
		
		// Start the server
		fx.Invoke(func(server *gateway.Server) {
			server.Start()
		}),
	)
	
	app.Run()
}

// newLogger creates a new logger
func newLogger(config *config.Config) *zap.Logger {
	var logger *zap.Logger
	var err error
	
	if config.Environment == "production" {
		logger, err = zap.NewProduction()
	} else {
		logger, err = zap.NewDevelopment()
	}
	
	if err != nil {
		panic(err)
	}
	
	return logger
}

// newGinEngine creates a new Gin engine
func newGinEngine(config *config.Config) *gin.Engine {
	if config.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	
	return gin.Default()
}

