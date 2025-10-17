package events

import (
	"context"

	"github.com/abdoElHodaky/tradSys/internal/config"
	"go-micro.dev/v4/broker"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// BrokerParams contains parameters for creating a broker
type BrokerParams struct {
	fx.In

	Config *config.Config
	Logger *zap.Logger
	Lifecycle fx.Lifecycle
}

// NewBroker creates a new message broker with fx dependency injection
func NewBroker(p BrokerParams) broker.Broker {
	var b broker.Broker

	// Create broker based on configuration
	switch p.Config.Broker.Type {
	case "nats":
		b = broker.NewBroker(
			broker.Addrs(p.Config.Broker.Address),
		)
	case "kafka":
		b = broker.NewBroker(
			broker.Addrs(p.Config.Broker.Address),
		)
	default:
		// Default to HTTP broker
		b = broker.NewBroker()
	}

	// Add lifecycle hooks
	p.Lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err := b.Connect(); err != nil {
				return err
			}
			p.Logger.Info("Message broker connected",
				zap.String("type", p.Config.Broker.Type))
			return nil
		},
		OnStop: func(ctx context.Context) error {
			if err := b.Disconnect(); err != nil {
				return err
			}
			p.Logger.Info("Message broker disconnected")
			return nil
		},
	})

	return b
}

// BrokerModule provides the broker module for fx
var BrokerModule = fx.Options(
	fx.Provide(NewBroker),
)
