package repositories

import (
	"context"
	"errors"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/db/models"
	"github.com/abdoElHodaky/tradSys/internal/db/queries"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// StrategyRepository handles database operations for trading strategies
type StrategyRepository struct {
	db        *gorm.DB
	logger    *zap.Logger
	optimizer *query.Optimizer
}

// NewStrategyRepository creates a new strategy repository
func NewStrategyRepository(db *gorm.DB, logger *zap.Logger) *StrategyRepository {
	repo := &StrategyRepository{
		db:        db,
		logger:    logger,
		optimizer: query.NewOptimizer(db, logger),
	}
	
	return repo
}

// GetStrategy retrieves a strategy by name
func (r *StrategyRepository) GetStrategy(ctx context.Context, name string) (*models.Strategy, error) {
	var strategy models.Strategy
	
	builder := query.NewBuilder(r.db, r.logger).
		Table("strategies").
		Where("name = ?", name)
	
	err := builder.First(&strategy)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		
		r.logger.Error("Failed to get strategy",
			zap.Error(err),
			zap.String("name", name))
		return nil, err
	}
	
	return &strategy, nil
}

// GetAllStrategies retrieves all strategies
func (r *StrategyRepository) GetAllStrategies(ctx context.Context) ([]*models.Strategy, error) {
	var strategies []*models.Strategy
	
	builder := query.NewBuilder(r.db, r.logger).
		Table("strategies").
		OrderBy("name ASC")
	
	err := builder.Execute(&strategies)
	if err != nil {
		r.logger.Error("Failed to get all strategies", zap.Error(err))
		return nil, err
	}
	
	return strategies, nil
}

// CreateStrategy creates a new strategy
func (r *StrategyRepository) CreateStrategy(ctx context.Context, strategy *models.Strategy) error {
	result := r.db.WithContext(ctx).Create(strategy)
	if result.Error != nil {
		r.logger.Error("Failed to create strategy", 
			zap.Error(result.Error),
			zap.String("name", strategy.Name))
		return result.Error
	}
	return nil
}

// UpdateStrategy updates an existing strategy
func (r *StrategyRepository) UpdateStrategy(ctx context.Context, strategy *models.Strategy) error {
	result := r.db.WithContext(ctx).Save(strategy)
	if result.Error != nil {
		r.logger.Error("Failed to update strategy", 
			zap.Error(result.Error),
			zap.String("name", strategy.Name))
		return result.Error
	}
	return nil
}

// DeleteStrategy deletes a strategy
func (r *StrategyRepository) DeleteStrategy(ctx context.Context, name string) error {
	result := r.db.WithContext(ctx).Where("name = ?", name).Delete(&models.Strategy{})
	if result.Error != nil {
		r.logger.Error("Failed to delete strategy", 
			zap.Error(result.Error),
			zap.String("name", name))
		return result.Error
	}
	return nil
}

// CreateStrategyExecution creates a new strategy execution
func (r *StrategyRepository) CreateStrategyExecution(ctx context.Context, execution *models.StrategyExecution) error {
	result := r.db.WithContext(ctx).Create(execution)
	if result.Error != nil {
		r.logger.Error("Failed to create strategy execution", 
			zap.Error(result.Error),
			zap.Uint("strategy_id", execution.StrategyID))
		return result.Error
	}
	return nil
}

// UpdateStrategyExecution updates a strategy execution
func (r *StrategyRepository) UpdateStrategyExecution(ctx context.Context, execution *models.StrategyExecution) error {
	result := r.db.WithContext(ctx).Save(execution)
	if result.Error != nil {
		r.logger.Error("Failed to update strategy execution", 
			zap.Error(result.Error),
			zap.Uint("id", execution.ID))
		return result.Error
	}
	return nil
}

// GetStrategyExecutions retrieves executions for a strategy
func (r *StrategyRepository) GetStrategyExecutions(ctx context.Context, strategyID uint) ([]*models.StrategyExecution, error) {
	var executions []*models.StrategyExecution
	
	builder := query.NewBuilder(r.db, r.logger).
		Table("strategy_executions").
		Where("strategy_id = ?", strategyID).
		OrderBy("start_time DESC")
	
	err := builder.Execute(&executions)
	if err != nil {
		r.logger.Error("Failed to get strategy executions",
			zap.Error(err),
			zap.Uint("strategy_id", strategyID))
		return nil, err
	}
	
	return executions, nil
}

// CreateSignal creates a new trading signal
func (r *StrategyRepository) CreateSignal(ctx context.Context, signal *models.Signal) error {
	result := r.db.WithContext(ctx).Create(signal)
	if result.Error != nil {
		r.logger.Error("Failed to create signal", 
			zap.Error(result.Error),
			zap.Uint("strategy_id", signal.StrategyID),
			zap.String("symbol", signal.Symbol))
		return result.Error
	}
	return nil
}

// GetActiveSignals retrieves active signals for a symbol
func (r *StrategyRepository) GetActiveSignals(ctx context.Context, symbol string) ([]*models.Signal, error) {
	var signals []*models.Signal
	
	builder := query.NewBuilder(r.db, r.logger).
		Table("signals").
		Where("symbol = ?", symbol).
		Where("executed = ?", false).
		Where("expires_at > ?", time.Now()).
		OrderBy("generated_at DESC")
	
	err := builder.Execute(&signals)
	if err != nil {
		r.logger.Error("Failed to get active signals",
			zap.Error(err),
			zap.String("symbol", symbol))
		return nil, err
	}
	
	return signals, nil
}

// UpdateSignal updates a signal
func (r *StrategyRepository) UpdateSignal(ctx context.Context, signal *models.Signal) error {
	result := r.db.WithContext(ctx).Save(signal)
	if result.Error != nil {
		r.logger.Error("Failed to update signal", 
			zap.Error(result.Error),
			zap.Uint("id", signal.ID))
		return result.Error
	}
	return nil
}

