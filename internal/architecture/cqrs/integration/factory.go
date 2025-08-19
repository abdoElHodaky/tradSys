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
}

// NewCQRSFactory creates a new CQRS factory
func NewCQRSFactory(logger *zap.Logger) *CQRSFactory {
	return &CQRSFactory{
		logger: logger,
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
func (f *CQRSFactory) CreateEventBus(eventStore store.EventStore) eventbus.EventBus {
	// Create an asynchronous event bus
	return eventbus.NewAsyncEventBus(eventStore, f.logger, 10)
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

// CreateCQRSSystem creates a complete CQRS system
func (f *CQRSFactory) CreateCQRSSystem() *CQRSSystem {
	// Create the event store
	eventStore := f.CreateEventStore()
	
	// Create the event bus
	eventBus := f.CreateEventBus(eventStore)
	
	// Create the aggregate repository
	aggregateRepo := f.CreateAggregateRepository(eventStore)
	
	// Create the projection manager
	projectionManager := f.CreateProjectionManager(eventStore)
	
	// Create the command bus
	commandBus := f.CreateCommandBus()
	
	// Create the event-sourced command bus
	eventSourcedCommandBus := f.CreateEventSourcedCommandBus(eventBus, aggregateRepo)
	
	// Create the query bus
	queryBus := f.CreateQueryBus()
	
	// Create the CQRS system
	return &CQRSSystem{
		EventStore:            eventStore,
		EventBus:              eventBus,
		AggregateRepository:   aggregateRepo,
		ProjectionManager:     projectionManager,
		CommandBus:            commandBus,
		EventSourcedCommandBus: eventSourcedCommandBus,
		QueryBus:              queryBus,
		Logger:                f.logger,
	}
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
	Logger                *zap.Logger
}

// Start starts the CQRS system
func (s *CQRSSystem) Start() error {
	// Start the event bus
	if starter, ok := s.EventBus.(interface{ Start() }); ok {
		starter.Start()
	}
	
	return nil
}

// Stop stops the CQRS system
func (s *CQRSSystem) Stop() error {
	// Stop the event bus
	if stopper, ok := s.EventBus.(interface{ Stop() }); ok {
		stopper.Stop()
	}
	
	// Close the event store
	if closer, ok := s.EventStore.(interface{ Close() error }); ok {
		return closer.Close()
	}
	
	return nil
}

