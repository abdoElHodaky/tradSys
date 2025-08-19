package integration

import (
	"github.com/abdoElHodaky/tradSys/internal/architecture/cqrs/command"
	"github.com/abdoElHodaky/tradSys/internal/architecture/cqrs/eventbus"
	"github.com/abdoElHodaky/tradSys/internal/architecture/cqrs/query"
	"github.com/abdoElHodaky/tradSys/internal/eventsourcing/aggregate"
	"github.com/abdoElHodaky/tradSys/internal/eventsourcing/projection"
	"github.com/abdoElHodaky/tradSys/internal/eventsourcing/store"
	"go.uber.org/zap"
)

// CQRSFactory creates CQRS components
type CQRSFactory struct {
	logger *zap.Logger
	useWatermill bool
}

// NewCQRSFactory creates a new CQRS factory
func NewCQRSFactory(logger *zap.Logger, useWatermill bool) *CQRSFactory {
	return &CQRSFactory{
		logger: logger,
		useWatermill: useWatermill,
	}
}

// CreateEventStore creates an event store
func (f *CQRSFactory) CreateEventStore() store.EventStore {
	// Create an in-memory event store
	memoryStore := store.NewInMemoryEventStore(f.logger)
	
	// Create a batched event store
	batchStore := store.NewBatchEventStore(memoryStore, f.logger,
		store.WithBatchSize(100),
		store.WithFlushInterval(100),
	)
	
	return batchStore
}

// CreateEventBus creates an event bus
func (f *CQRSFactory) CreateEventBus(eventStore store.EventStore) (eventbus.EventBus, error) {
	if f.useWatermill {
		// Create a Watermill event bus
		config := DefaultWatermillEventBusConfig()
		bus, err := NewWatermillEventBus(eventStore, f.logger, config)
		if err != nil {
			return nil, err
		}
		
		// Start the event bus
		err = bus.Start()
		if err != nil {
			return nil, err
		}
		
		return bus, nil
	}
	
	// Create an asynchronous event bus
	return eventbus.NewAsyncEventBus(eventStore, f.logger, 10), nil
}

// CreateCommandBus creates a command bus
func (f *CQRSFactory) CreateCommandBus() command.Bus {
	// Create a default command bus
	return command.NewDefaultBus()
}

// CreateQueryBus creates a query bus
func (f *CQRSFactory) CreateQueryBus() query.Bus {
	// Create a default query bus
	return query.NewDefaultBus()
}

// CreateAggregateRepository creates an aggregate repository
func (f *CQRSFactory) CreateAggregateRepository(eventStore store.EventStore) aggregate.Repository {
	// Create an event-sourced repository
	return aggregate.NewEventSourcedRepository(eventStore, f.logger,
		aggregate.WithSnapshotFrequency(100),
	)
}

// CreateProjectionManager creates a projection manager
func (f *CQRSFactory) CreateProjectionManager(eventStore store.EventStore) projection.ProjectionManager {
	// Create a default projection manager
	return projection.NewDefaultProjectionManager(eventStore, f.logger)
}

// CreateEventSourcedCommandBus creates an event-sourced command bus
func (f *CQRSFactory) CreateEventSourcedCommandBus(eventBus eventbus.EventBus, aggregateRepo aggregate.Repository) *command.EventSourcedCommandBus {
	// Create an event-sourced command bus
	return command.NewEventSourcedCommandBus(eventBus, aggregateRepo, f.logger)
}

// CreateWatermillCQRSAdapter creates a Watermill CQRS adapter
func (f *CQRSFactory) CreateWatermillCQRSAdapter(eventStore store.EventStore, aggregateRepo aggregate.Repository) (*WatermillCQRSAdapter, error) {
	// Create a Watermill CQRS adapter
	config := DefaultWatermillCQRSConfig()
	adapter, err := NewWatermillCQRSAdapter(eventStore, aggregateRepo, f.logger, config)
	if err != nil {
		return nil, err
	}
	
	// Start the adapter
	err = adapter.Start()
	if err != nil {
		return nil, err
	}
	
	return adapter, nil
}

// CreateCQRSSystem creates a complete CQRS system
func (f *CQRSFactory) CreateCQRSSystem() (*CQRSSystem, error) {
	// Create the event store
	eventStore := f.CreateEventStore()
	
	// Create the aggregate repository
	aggregateRepo := f.CreateAggregateRepository(eventStore)
	
	// Create the projection manager
	projectionManager := f.CreateProjectionManager(eventStore)
	
	var eventBus eventbus.EventBus
	var commandBus command.Bus
	var queryBus query.Bus
	var eventSourcedCommandBus *command.EventSourcedCommandBus
	var watermillAdapter *WatermillCQRSAdapter
	
	if f.useWatermill {
		// Create the Watermill CQRS adapter
		adapter, err := f.CreateWatermillCQRSAdapter(eventStore, aggregateRepo)
		if err != nil {
			return nil, err
		}
		
		// Create the event bus adapter
		eventBus = adapter.CreateEventBusAdapter()
		
		// Create the command bus
		commandBus = f.CreateCommandBus()
		
		// Create the event-sourced command bus
		eventSourcedCommandBus = f.CreateEventSourcedCommandBus(eventBus, aggregateRepo)
		
		// Create the query bus
		queryBus = f.CreateQueryBus()
		
		// Set the Watermill adapter
		watermillAdapter = adapter
	} else {
		// Create the event bus
		var err error
		eventBus, err = f.CreateEventBus(eventStore)
		if err != nil {
			return nil, err
		}
		
		// Create the command bus
		commandBus = f.CreateCommandBus()
		
		// Create the event-sourced command bus
		eventSourcedCommandBus = f.CreateEventSourcedCommandBus(eventBus, aggregateRepo)
		
		// Create the query bus
		queryBus = f.CreateQueryBus()
	}
	
	// Create the CQRS system
	return &CQRSSystem{
		EventStore:            eventStore,
		EventBus:              eventBus,
		AggregateRepository:   aggregateRepo,
		ProjectionManager:     projectionManager,
		CommandBus:            commandBus,
		EventSourcedCommandBus: eventSourcedCommandBus,
		QueryBus:              queryBus,
		WatermillAdapter:      watermillAdapter,
		Logger:                f.logger,
		UseWatermill:          f.useWatermill,
	}, nil
}

// CQRSSystem represents a complete CQRS system
type CQRSSystem struct {
	EventStore            store.EventStore
	EventBus              eventbus.EventBus
	AggregateRepository   aggregate.Repository
	ProjectionManager     projection.ProjectionManager
	CommandBus            command.Bus
	EventSourcedCommandBus *command.EventSourcedCommandBus
	QueryBus              query.Bus
	WatermillAdapter      *WatermillCQRSAdapter
	Logger                *zap.Logger
	UseWatermill          bool
}

// Start starts the CQRS system
func (s *CQRSSystem) Start() error {
	// Start the event bus if it implements the Start method
	if starter, ok := s.EventBus.(interface{ Start() error }); ok {
		err := starter.Start()
		if err != nil {
			return err
		}
	}
	
	return nil
}

// Stop stops the CQRS system
func (s *CQRSSystem) Stop() error {
	// Stop the Watermill adapter if it exists
	if s.WatermillAdapter != nil {
		err := s.WatermillAdapter.Stop()
		if err != nil {
			return err
		}
	}
	
	// Stop the event bus if it implements the Stop method
	if stopper, ok := s.EventBus.(interface{ Stop() error }); ok {
		err := stopper.Stop()
		if err != nil {
			return err
		}
	}
	
	// Close the event store if it implements the Close method
	if closer, ok := s.EventStore.(interface{ Close() error }); ok {
		return closer.Close()
	}
	
	return nil
}

