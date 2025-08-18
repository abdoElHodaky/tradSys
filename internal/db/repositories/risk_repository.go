package repositories

import (
	"context"
	"errors"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/db/models"
	"github.com/abdoElHodaky/tradSys/internal/db/query"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RiskRepository handles database operations for risk management
type RiskRepository struct {
	db        *gorm.DB
	logger    *zap.Logger
	optimizer *query.Optimizer
}

// NewRiskRepository creates a new risk repository
func NewRiskRepository(db *gorm.DB, logger *zap.Logger) *RiskRepository {
	repo := &RiskRepository{
		db:        db,
		logger:    logger,
		optimizer: query.NewOptimizer(db, logger),
	}
	
	return repo
}

// GetRiskLimit retrieves risk limits for an account and symbol
func (r *RiskRepository) GetRiskLimit(ctx context.Context, accountID, symbol string) (*models.RiskLimit, error) {
	var limit models.RiskLimit
	
	builder := query.NewBuilder(r.db, r.logger).
		Table("risk_limits").
		Where("account_id = ?", accountID)
	
	if symbol != "" {
		builder.Where("symbol = ?", symbol)
	} else {
		// Get default risk limit (empty symbol)
		builder.Where("symbol = ''")
	}
	
	err := builder.First(&limit)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Return default risk limits if not found
			return &models.RiskLimit{
				AccountID:       accountID,
				Symbol:          symbol,
				MaxPosition:     1000,
				MaxOrderSize:    100,
				MaxDailyLoss:    1000,
				CurrentDailyLoss: 0,
				Active:          true,
			}, nil
		}
		
		r.logger.Error("Failed to get risk limit",
			zap.Error(err),
			zap.String("account_id", accountID),
			zap.String("symbol", symbol))
		return nil, err
	}
	
	return &limit, nil
}

// UpdateRiskLimit updates a risk limit
func (r *RiskRepository) UpdateRiskLimit(ctx context.Context, limit *models.RiskLimit) error {
	result := r.db.WithContext(ctx).Save(limit)
	if result.Error != nil {
		r.logger.Error("Failed to update risk limit", 
			zap.Error(result.Error),
			zap.String("account_id", limit.AccountID),
			zap.String("symbol", limit.Symbol))
		return result.Error
	}
	return nil
}

// CreateRiskCheck records a risk check
func (r *RiskRepository) CreateRiskCheck(ctx context.Context, check *models.RiskCheck) error {
	result := r.db.WithContext(ctx).Create(check)
	if result.Error != nil {
		r.logger.Error("Failed to create risk check", 
			zap.Error(result.Error),
			zap.String("order_id", check.OrderID))
		return result.Error
	}
	return nil
}

// GetCircuitBreaker retrieves circuit breaker status for a symbol
func (r *RiskRepository) GetCircuitBreaker(ctx context.Context, symbol string) (*models.CircuitBreaker, error) {
	var breaker models.CircuitBreaker
	
	builder := query.NewBuilder(r.db, r.logger).
		Table("circuit_breakers").
		Where("symbol = ?", symbol)
	
	err := builder.First(&breaker)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Return a new circuit breaker if not found
			return &models.CircuitBreaker{
				Symbol:    symbol,
				Triggered: false,
			}, nil
		}
		
		r.logger.Error("Failed to get circuit breaker",
			zap.Error(err),
			zap.String("symbol", symbol))
		return nil, err
	}
	
	return &breaker, nil
}

// UpdateCircuitBreaker updates a circuit breaker
func (r *RiskRepository) UpdateCircuitBreaker(ctx context.Context, breaker *models.CircuitBreaker) error {
	// Use a transaction to ensure atomicity
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return tx.Error
	}
	
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	
	// Try to update existing circuit breaker
	result := tx.Model(&models.CircuitBreaker{}).
		Where("symbol = ?", breaker.Symbol).
		Updates(map[string]interface{}{
			"triggered":     breaker.Triggered,
			"reason":        breaker.Reason,
			"trigger_time":  breaker.TriggerTime,
			"reset_time":    breaker.ResetTime,
			"updated_at":    time.Now(),
		})
	
	// If no record was updated, create a new one
	if result.RowsAffected == 0 {
		if err := tx.Create(breaker).Error; err != nil {
			tx.Rollback()
			r.logger.Error("Failed to create circuit breaker", 
				zap.Error(err),
				zap.String("symbol", breaker.Symbol))
			return err
		}
	} else if result.Error != nil {
		tx.Rollback()
		r.logger.Error("Failed to update circuit breaker", 
			zap.Error(result.Error),
			zap.String("symbol", breaker.Symbol))
		return result.Error
	}
	
	return tx.Commit().Error
}

// GetRiskChecks retrieves risk checks for an order
func (r *RiskRepository) GetRiskChecks(ctx context.Context, orderID string) ([]*models.RiskCheck, error) {
	var checks []*models.RiskCheck
	
	builder := query.NewBuilder(r.db, r.logger).
		Table("risk_checks").
		Where("order_id = ?", orderID).
		OrderBy("check_time DESC")
	
	err := builder.Execute(&checks)
	if err != nil {
		r.logger.Error("Failed to get risk checks",
			zap.Error(err),
			zap.String("order_id", orderID))
		return nil, err
	}
	
	return checks, nil
}

