package marketdata

import (
	"context"
	"errors"
	"reflect"

	"github.com/abdoElHodaky/tradSys/internal/architecture/cqrs"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var (
	// ErrInvalidCommand is returned when an invalid command is provided
	ErrInvalidCommand = errors.New("invalid command")
)

// CQRSModule provides the market data service CQRS components
var CQRSModule = fx.Options(
	// Provide the market data service
	fx.Provide(NewService),
	
	// Register command and query handlers
	fx.Invoke(RegisterHandlers),
)

// ServiceParams contains the parameters for creating a market data service
type ServiceParams struct {
	fx.In
	
	Logger     *zap.Logger
	CommandBus *cqrs.CommandBus
}

// RegisterHandlers registers command and query handlers for the market data service
func RegisterHandlers(
	lifecycle fx.Lifecycle,
	service *Service,
	commandBus *cqrs.CommandBus,
	logger *zap.Logger,
) {
	// Register lifecycle hooks
	lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info("Registering market data service handlers")
			
			// Register command handlers
			registerCommandHandlers(service, commandBus, logger)
			
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("Stopping market data service")
			return nil
		},
	})
}

// registerCommandHandlers registers command handlers for the market data service
func registerCommandHandlers(service *Service, commandBus *cqrs.CommandBus, logger *zap.Logger) {
	// Example: Register a command handler for adding a market data source
	err := commandBus.RegisterHandlerFunc(
		reflect.TypeOf(&AddMarketDataSourceCommand{}),
		func(ctx context.Context, cmd cqrs.Command) error {
			addCmd, ok := cmd.(*AddMarketDataSourceCommand)
			if !ok {
				return ErrInvalidCommand
			}
			return service.AddMarketDataSource(ctx, addCmd.Source, addCmd.Config)
		},
	)
	if err != nil {
		logger.Error("Failed to register AddMarketDataSourceCommand handler", zap.Error(err))
	}
	
	// Register other command handlers...
}



// AddMarketDataSourceCommand is a command to add a market data source
type AddMarketDataSourceCommand struct {
	Source string
	Config map[string]interface{}
}

// CommandName returns the name of the command
func (c *AddMarketDataSourceCommand) CommandName() string {
	return "AddMarketDataSource"
}

// GetMarketDataQuery is a query to get market data
type GetMarketDataQuery struct {
	Symbol    string
	TimeRange string
}

// QueryName returns the name of the query
func (q *GetMarketDataQuery) QueryName() string {
	return "GetMarketData"
}
