package integration

import (
	"github.com/abdoElHodaky/tradSys/internal/architecture/cqrs/eventbus"
	"github.com/abdoElHodaky/tradSys/internal/eventsourcing/aggregate"
	"github.com/abdoElHodaky/tradSys/internal/eventsourcing/store"
	"go.uber.org/zap"
)

// CQRSSystem represents a complete CQRS system
type CQRSSystem struct {
	// Core components
	EventStore    store.EventStore
	AggregateRepo aggregate.Repository
	EventBus      eventbus.EventBus
	
	// Adapters
	WatermillAdapter *WatermillCQRSAdapter
	NatsAdapter      *NatsCQRSAdapter
	
	// Compatibility and monitoring
	CompatibilityLayer *CompatibilityLayer
	PerformanceMonitor *PerformanceMonitor
	
	// Logger
	Logger *zap.Logger
}

// CQRSFactory creates CQRS systems
type CQRSFactory struct {
	logger        *zap.Logger
	useWatermill  bool
	useNats       bool
	useCompatLayer bool
	useMonitoring  bool
}

// NewCQRSFactory creates a new CQRS factory
func NewCQRSFactory(
	logger *zap.Logger,
	useWatermill bool,
	useNats bool,
	useCompatLayer bool,
	useMonitoring bool,
) *CQRSFactory {
	return &CQRSFactory{
		logger:        logger,
		useWatermill:  useWatermill,
		useNats:       useNats,
		useCompatLayer: useCompatLayer,
		useMonitoring:  useMonitoring,
	}
}

// CreateCQRSSystem creates a new CQRS system
func (f *CQRSFactory) CreateCQRSSystem() (*CQRSSystem, error) {
	// Create the event store
	eventStore, err := store.NewInMemoryEventStore()
	if err != nil {
		return nil, err
	}
	
	// Create the aggregate repository
	aggregateRepo := aggregate.NewRepository(eventStore)
	
	// Create the system
	system := &CQRSSystem{
		EventStore:    eventStore,
		AggregateRepo: aggregateRepo,
		Logger:        f.logger,
	}
	
	// Create the Watermill adapter if enabled
	if f.useWatermill {
		// Create the Watermill adapter
		watermillConfig := DefaultWatermillCQRSConfig()
		watermillAdapter, err := NewWatermillCQRSAdapter(
			eventStore,
			aggregateRepo,
			f.logger,
			watermillConfig,
		)
		if err != nil {
			return nil, err
		}
		
		// Start the adapter
		err = watermillAdapter.Start()
		if err != nil {
			return nil, err
		}
		
		// Create an event bus adapter
		eventBus := watermillAdapter.CreateEventBusAdapter()
		
		// Set the components
		system.WatermillAdapter = watermillAdapter
		system.EventBus = eventBus
	}
	
	// Create the NATS adapter if enabled
	if f.useNats {
		// Create the NATS adapter
		natsConfig := DefaultNatsCQRSConfig()
		natsAdapter, err := NewNatsCQRSAdapter(
			eventStore,
			aggregateRepo,
			f.logger,
			natsConfig,
		)
		if err != nil {
			return nil, err
		}
		
		// Start the adapter
		err = natsAdapter.Start()
		if err != nil {
			return nil, err
		}
		
		// Create an event bus adapter
		eventBus := natsAdapter.CreateEventBusAdapter()
		
		// Set the components
		system.NatsAdapter = natsAdapter
		system.EventBus = eventBus
	}
	
	// If neither Watermill nor NATS is enabled, create a default event bus
	if !f.useWatermill && !f.useNats {
		// Create a default event bus
		eventBus := eventbus.NewInMemoryEventBus(eventStore, f.logger)
		
		// Set the event bus
		system.EventBus = eventBus
	}
	
	// Create the compatibility layer if enabled
	if f.useCompatLayer {
		// Create the compatibility layer
		compatLayer := NewCompatibilityLayer(
			system.WatermillAdapter,
			eventStore,
			aggregateRepo,
			system.EventBus,
			f.logger,
		)
		
		// Set the compatibility layer
		system.CompatibilityLayer = compatLayer
	}
	
	// Create the performance monitor if enabled
	if f.useMonitoring {
		// Create the performance monitor
		monitor := NewPerformanceMonitor(f.logger, 100)
		
		// Set the performance monitor
		system.PerformanceMonitor = monitor
	}
	
	return system, nil
}

