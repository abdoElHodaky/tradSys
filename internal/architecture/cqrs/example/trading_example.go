package example

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/architecture/cqrs/command"
	"github.com/abdoElHodaky/tradSys/internal/architecture/cqrs/integration"
	"github.com/abdoElHodaky/tradSys/internal/architecture/cqrs/query"
	"github.com/abdoElHodaky/tradSys/internal/eventsourcing"
	"github.com/abdoElHodaky/tradSys/internal/eventsourcing/aggregate"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// This file provides an example of how to use the CQRS and event sourcing system
// for a high-frequency trading application.

// TradingAccount represents a trading account aggregate
type TradingAccount struct {
	*aggregate.BaseAggregate
	Balance     float64
	Positions   map[string]float64 // symbol -> quantity
	LastUpdated time.Time
}

// NewTradingAccount creates a new trading account
func NewTradingAccount(id string) *TradingAccount {
	return &TradingAccount{
		BaseAggregate: aggregate.NewBaseAggregate(id, "TradingAccount"),
		Positions:     make(map[string]float64),
		LastUpdated:   time.Now(),
	}
}

// ApplyEvent applies an event to the trading account
func (a *TradingAccount) ApplyEvent(event *eventsourcing.Event) error {
	switch e := event.Data.(type) {
	case *TradingAccountCreated:
		a.Balance = e.InitialBalance
		a.LastUpdated = time.Now()
		
	case *FundsDeposited:
		a.Balance += e.Amount
		a.LastUpdated = time.Now()
		
	case *FundsWithdrawn:
		a.Balance -= e.Amount
		a.LastUpdated = time.Now()
		
	case *PositionOpened:
		a.Positions[e.Symbol] += e.Quantity
		a.Balance -= e.Price * e.Quantity
		a.LastUpdated = time.Now()
		
	case *PositionClosed:
		a.Positions[e.Symbol] -= e.Quantity
		a.Balance += e.Price * e.Quantity
		a.LastUpdated = time.Now()
	}
	
	return nil
}

// CreateSnapshot creates a snapshot of the trading account
func (a *TradingAccount) CreateSnapshot() (interface{}, error) {
	return &TradingAccountSnapshot{
		ID:          a.ID,
		Balance:     a.Balance,
		Positions:   a.Positions,
		LastUpdated: a.LastUpdated,
		Version:     a.Version,
	}, nil
}

// ApplySnapshot applies a snapshot to the trading account
func (a *TradingAccount) ApplySnapshot(snapshot interface{}) error {
	s, ok := snapshot.(*TradingAccountSnapshot)
	if !ok {
		return fmt.Errorf("invalid snapshot type: %T", snapshot)
	}
	
	a.ID = s.ID
	a.Balance = s.Balance
	a.Positions = s.Positions
	a.LastUpdated = s.LastUpdated
	a.Version = s.Version
	
	return nil
}

// TradingAccountSnapshot represents a snapshot of a trading account
type TradingAccountSnapshot struct {
	ID          string
	Balance     float64
	Positions   map[string]float64
	LastUpdated time.Time
	Version     int
}

// Events
type TradingAccountCreated struct {
	InitialBalance float64
}

type FundsDeposited struct {
	Amount float64
}

type FundsWithdrawn struct {
	Amount float64
}

type PositionOpened struct {
	Symbol   string
	Quantity float64
	Price    float64
}

type PositionClosed struct {
	Symbol   string
	Quantity float64
	Price    float64
}

// Commands
type CreateTradingAccount struct {
	command.BaseCommand
	InitialBalance float64
}

func (c CreateTradingAccount) CommandName() string {
	return "CreateTradingAccount"
}

type DepositFunds struct {
	command.BaseCommand
	Amount float64
}

func (c DepositFunds) CommandName() string {
	return "DepositFunds"
}

type WithdrawFunds struct {
	command.BaseCommand
	Amount float64
}

func (c WithdrawFunds) CommandName() string {
	return "WithdrawFunds"
}

type OpenPosition struct {
	command.BaseCommand
	Symbol   string
	Quantity float64
	Price    float64
}

func (c OpenPosition) CommandName() string {
	return "OpenPosition"
}

type ClosePosition struct {
	command.BaseCommand
	Symbol   string
	Quantity float64
	Price    float64
}

func (c ClosePosition) CommandName() string {
	return "ClosePosition"
}

// Queries
type GetTradingAccount struct {
	AccountID string
}

func (q GetTradingAccount) QueryName() string {
	return "GetTradingAccount"
}

type GetTradingAccountBalance struct {
	AccountID string
}

func (q GetTradingAccountBalance) QueryName() string {
	return "GetTradingAccountBalance"
}

type GetTradingAccountPositions struct {
	AccountID string
}

func (q GetTradingAccountPositions) QueryName() string {
	return "GetTradingAccountPositions"
}

// TradingAccountCommandHandler handles commands for trading accounts
type TradingAccountCommandHandler struct {
	aggregateHandler *command.AggregateCommandHandler
}

// NewTradingAccountCommandHandler creates a new trading account command handler
func NewTradingAccountCommandHandler(aggregateRepo aggregate.Repository, logger *zap.Logger) *TradingAccountCommandHandler {
	return &TradingAccountCommandHandler{
		aggregateHandler: command.NewAggregateCommandHandler("TradingAccount", aggregateRepo, logger),
	}
}

// Handle handles a command
func (h *TradingAccountCommandHandler) Handle(ctx context.Context, cmd command.Command) ([]*eventsourcing.Event, error) {
	switch c := cmd.(type) {
	case CreateTradingAccount:
		return h.handleCreateTradingAccount(ctx, c)
	case DepositFunds:
		return h.handleDepositFunds(ctx, c)
	case WithdrawFunds:
		return h.handleWithdrawFunds(ctx, c)
	case OpenPosition:
		return h.handleOpenPosition(ctx, c)
	case ClosePosition:
		return h.handleClosePosition(ctx, c)
	default:
		return nil, fmt.Errorf("unknown command type: %T", cmd)
	}
}

func (h *TradingAccountCommandHandler) handleCreateTradingAccount(ctx context.Context, cmd CreateTradingAccount) ([]*eventsourcing.Event, error) {
	return h.aggregateHandler.HandleCreate(ctx, cmd, func(cmd command.Command) (aggregate.Aggregate, error) {
		c := cmd.(CreateTradingAccount)
		
		// Generate a new ID if not provided
		id := c.AggregateID
		if id == "" {
			id = uuid.New().String()
		}
		
		// Create a new trading account
		account := NewTradingAccount(id)
		
		// Add the created event
		account.AddEvent("TradingAccountCreated", map[string]interface{}{
			"initial_balance": c.InitialBalance,
		}, nil)
		
		return account, nil
	})
}

func (h *TradingAccountCommandHandler) handleDepositFunds(ctx context.Context, cmd DepositFunds) ([]*eventsourcing.Event, error) {
	return h.aggregateHandler.HandleUpdate(ctx, cmd, cmd.AggregateID, func(agg aggregate.Aggregate, cmd command.Command) error {
		account := agg.(*TradingAccount)
		c := cmd.(DepositFunds)
		
		// Validate the command
		if c.Amount <= 0 {
			return fmt.Errorf("deposit amount must be positive")
		}
		
		// Add the event
		account.AddEvent("FundsDeposited", map[string]interface{}{
			"amount": c.Amount,
		}, nil)
		
		return nil
	})
}

func (h *TradingAccountCommandHandler) handleWithdrawFunds(ctx context.Context, cmd WithdrawFunds) ([]*eventsourcing.Event, error) {
	return h.aggregateHandler.HandleUpdate(ctx, cmd, cmd.AggregateID, func(agg aggregate.Aggregate, cmd command.Command) error {
		account := agg.(*TradingAccount)
		c := cmd.(WithdrawFunds)
		
		// Validate the command
		if c.Amount <= 0 {
			return fmt.Errorf("withdrawal amount must be positive")
		}
		
		if account.Balance < c.Amount {
			return fmt.Errorf("insufficient funds")
		}
		
		// Add the event
		account.AddEvent("FundsWithdrawn", map[string]interface{}{
			"amount": c.Amount,
		}, nil)
		
		return nil
	})
}

func (h *TradingAccountCommandHandler) handleOpenPosition(ctx context.Context, cmd OpenPosition) ([]*eventsourcing.Event, error) {
	return h.aggregateHandler.HandleUpdate(ctx, cmd, cmd.AggregateID, func(agg aggregate.Aggregate, cmd command.Command) error {
		account := agg.(*TradingAccount)
		c := cmd.(OpenPosition)
		
		// Validate the command
		if c.Quantity <= 0 {
			return fmt.Errorf("quantity must be positive")
		}
		
		if c.Price <= 0 {
			return fmt.Errorf("price must be positive")
		}
		
		cost := c.Price * c.Quantity
		if account.Balance < cost {
			return fmt.Errorf("insufficient funds")
		}
		
		// Add the event
		account.AddEvent("PositionOpened", map[string]interface{}{
			"symbol":   c.Symbol,
			"quantity": c.Quantity,
			"price":    c.Price,
		}, nil)
		
		return nil
	})
}

func (h *TradingAccountCommandHandler) handleClosePosition(ctx context.Context, cmd ClosePosition) ([]*eventsourcing.Event, error) {
	return h.aggregateHandler.HandleUpdate(ctx, cmd, cmd.AggregateID, func(agg aggregate.Aggregate, cmd command.Command) error {
		account := agg.(*TradingAccount)
		c := cmd.(ClosePosition)
		
		// Validate the command
		if c.Quantity <= 0 {
			return fmt.Errorf("quantity must be positive")
		}
		
		if c.Price <= 0 {
			return fmt.Errorf("price must be positive")
		}
		
		currentPosition, exists := account.Positions[c.Symbol]
		if !exists || currentPosition < c.Quantity {
			return fmt.Errorf("insufficient position")
		}
		
		// Add the event
		account.AddEvent("PositionClosed", map[string]interface{}{
			"symbol":   c.Symbol,
			"quantity": c.Quantity,
			"price":    c.Price,
		}, nil)
		
		return nil
	})
}

// TradingAccountQueryHandler handles queries for trading accounts
type TradingAccountQueryHandler struct {
	aggregateRepo aggregate.Repository
	logger        *zap.Logger
}

// NewTradingAccountQueryHandler creates a new trading account query handler
func NewTradingAccountQueryHandler(aggregateRepo aggregate.Repository, logger *zap.Logger) *TradingAccountQueryHandler {
	return &TradingAccountQueryHandler{
		aggregateRepo: aggregateRepo,
		logger:        logger,
	}
}

// HandleGetTradingAccount handles the GetTradingAccount query
func (h *TradingAccountQueryHandler) HandleGetTradingAccount(ctx context.Context, q GetTradingAccount) (*TradingAccount, error) {
	// Create a new trading account
	account := NewTradingAccount(q.AccountID)
	
	// Load the account
	err := h.aggregateRepo.Load(ctx, q.AccountID, account)
	if err != nil {
		return nil, err
	}
	
	return account, nil
}

// HandleGetTradingAccountBalance handles the GetTradingAccountBalance query
func (h *TradingAccountQueryHandler) HandleGetTradingAccountBalance(ctx context.Context, q GetTradingAccountBalance) (float64, error) {
	// Get the account
	account, err := h.HandleGetTradingAccount(ctx, GetTradingAccount{AccountID: q.AccountID})
	if err != nil {
		return 0, err
	}
	
	return account.Balance, nil
}

// HandleGetTradingAccountPositions handles the GetTradingAccountPositions query
func (h *TradingAccountQueryHandler) HandleGetTradingAccountPositions(ctx context.Context, q GetTradingAccountPositions) (map[string]float64, error) {
	// Get the account
	account, err := h.HandleGetTradingAccount(ctx, GetTradingAccount{AccountID: q.AccountID})
	if err != nil {
		return nil, err
	}
	
	return account.Positions, nil
}

// SetupTradingExample sets up the trading example
func SetupTradingExample(useWatermill bool) (*integration.CQRSSystem, error) {
	// Create a logger
	logger, err := zap.NewDevelopment()
	if err != nil {
		return nil, err
	}
	
	// Create a CQRS factory
	factory := integration.NewCQRSFactory(logger, useWatermill)
	
	// Create a CQRS system
	system, err := factory.CreateCQRSSystem()
	if err != nil {
		return nil, err
	}
	
	// Create a trading account command handler
	tradingAccountCommandHandler := NewTradingAccountCommandHandler(system.AggregateRepository, logger)
	
	// Register command handlers
	system.EventSourcedCommandBus.RegisterFunc(
		reflect.TypeOf(CreateTradingAccount{}),
		tradingAccountCommandHandler.Handle,
	)
	system.EventSourcedCommandBus.RegisterFunc(
		reflect.TypeOf(DepositFunds{}),
		tradingAccountCommandHandler.Handle,
	)
	system.EventSourcedCommandBus.RegisterFunc(
		reflect.TypeOf(WithdrawFunds{}),
		tradingAccountCommandHandler.Handle,
	)
	system.EventSourcedCommandBus.RegisterFunc(
		reflect.TypeOf(OpenPosition{}),
		tradingAccountCommandHandler.Handle,
	)
	system.EventSourcedCommandBus.RegisterFunc(
		reflect.TypeOf(ClosePosition{}),
		tradingAccountCommandHandler.Handle,
	)
	
	// Create a trading account query handler
	tradingAccountQueryHandler := NewTradingAccountQueryHandler(system.AggregateRepository, logger)
	
	// Register query handlers
	system.QueryBus.Register(
		reflect.TypeOf(GetTradingAccount{}),
		query.HandlerFunc[*TradingAccount](func(ctx context.Context, q query.Query) (*TradingAccount, error) {
			return tradingAccountQueryHandler.HandleGetTradingAccount(ctx, q.(GetTradingAccount))
		}),
	)
	system.QueryBus.Register(
		reflect.TypeOf(GetTradingAccountBalance{}),
		query.HandlerFunc[float64](func(ctx context.Context, q query.Query) (float64, error) {
			return tradingAccountQueryHandler.HandleGetTradingAccountBalance(ctx, q.(GetTradingAccountBalance))
		}),
	)
	system.QueryBus.Register(
		reflect.TypeOf(GetTradingAccountPositions{}),
		query.HandlerFunc[map[string]float64](func(ctx context.Context, q query.Query) (map[string]float64, error) {
			return tradingAccountQueryHandler.HandleGetTradingAccountPositions(ctx, q.(GetTradingAccountPositions))
		}),
	)
	
	// Start the system
	err = system.Start()
	if err != nil {
		return nil, err
	}
	
	return system, nil
}

