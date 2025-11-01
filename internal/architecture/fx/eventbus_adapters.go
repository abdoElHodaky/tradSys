package fx

import (
	"context"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-nats/pkg/nats"
	"github.com/nats-io/nats.go"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// EventBusAdaptersModule provides the event bus adapters
var EventBusAdaptersModule = fx.Options(
	// Provide the NATS event bus
	fx.Provide(NewNatsEventBus),

	// Provide the Watermill event bus
	fx.Provide(NewWatermillEventBus),

	// Register lifecycle hooks
	fx.Invoke(registerEventBusAdaptersHooks),
)

// NatsEventBusConfig contains configuration for the NATS event bus
type NatsEventBusConfig struct {
	// URLs is a list of NATS server URLs
	URLs []string

	// TopicPrefix is the prefix for all topics
	TopicPrefix string

	// UseJetStream determines if JetStream should be used
	UseJetStream bool
}

// DefaultNatsEventBusConfig returns the default NATS event bus configuration
func DefaultNatsEventBusConfig() NatsEventBusConfig {
	return NatsEventBusConfig{
		URLs:         []string{nats.DefaultURL},
		TopicPrefix:  "events.",
		UseJetStream: true,
	}
}

// NewNatsEventBus creates a new NATS event bus
func NewNatsEventBus(
	eventStore store.EventStore,
	logger *zap.Logger,
) (*eventbus.NatsEventBus, error) {
	config := eventbus.DefaultNatsEventBusConfig()
	return eventbus.NewNatsEventBus(eventStore, logger, config)
}

// WatermillEventBusConfig contains configuration for the Watermill event bus
type WatermillEventBusConfig struct {
	// NatsURL is the URL of the NATS server
	NatsURL string

	// TopicPrefix is the prefix for all topics
	TopicPrefix string
}

// DefaultWatermillEventBusConfig returns the default Watermill event bus configuration
func DefaultWatermillEventBusConfig() WatermillEventBusConfig {
	return WatermillEventBusConfig{
		NatsURL:     nats.DefaultURL,
		TopicPrefix: "events.",
	}
}

// NewWatermillEventBus creates a new Watermill event bus
func NewWatermillEventBus(
	eventStore store.EventStore,
	logger *zap.Logger,
) (*eventbus.WatermillEventBus, error) {
	config := DefaultWatermillEventBusConfig()

	// Create a Watermill logger
	watermillLogger := watermill.NewStdLogger(false, false)

	// Create a NATS publisher
	publisherConfig := nats.PublisherConfig{
		URL:       config.NatsURL,
		Marshaler: nats.GobMarshaler{},
	}

	publisher, err := nats.NewPublisher(publisherConfig, watermillLogger)
	if err != nil {
		return nil, err
	}

	// Create a NATS subscriber
	subscriberConfig := nats.SubscriberConfig{
		URL:         config.NatsURL,
		Unmarshaler: nats.GobMarshaler{},
		QueueGroup:  "tradSys",
	}

	subscriber, err := nats.NewSubscriber(subscriberConfig, watermillLogger)
	if err != nil {
		return nil, err
	}

	// Create the Watermill event bus
	return eventbus.NewWatermillEventBus(eventStore, publisher, subscriber, logger, config.TopicPrefix)
}

// registerEventBusAdaptersHooks registers lifecycle hooks for the event bus adapters
func registerEventBusAdaptersHooks(
	lc fx.Lifecycle,
	logger *zap.Logger,
	natsEventBus *eventbus.NatsEventBus,
	watermillEventBus *eventbus.WatermillEventBus,
) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info("Starting event bus adapters")

			// Start the NATS event bus
			err := natsEventBus.Start()
			if err != nil {
				return err
			}

			// Start the Watermill event bus
			err = watermillEventBus.Start()
			if err != nil {
				return err
			}

			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("Stopping event bus adapters")

			// Stop the NATS event bus
			err := natsEventBus.Stop()
			if err != nil {
				logger.Error("Failed to stop NATS event bus", zap.Error(err))
			}

			// Stop the Watermill event bus
			err = watermillEventBus.Stop()
			if err != nil {
				logger.Error("Failed to stop Watermill event bus", zap.Error(err))
			}

			return nil
		},
	})
}
