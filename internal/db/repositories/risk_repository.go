package repositories

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// RiskRepositoryParams contains the parameters for creating a risk repository
type RiskRepositoryParams struct {
	fx.In

	Logger *zap.Logger
	DB     *sqlx.DB `optional:"true"`
}

// RiskRepository provides database operations for risk management
type RiskRepository struct {
	logger *zap.Logger
	db     *sqlx.DB
}

// Position represents a position in the database
type Position struct {
	ID           int64     `db:"id"`
	AccountID    string    `db:"account_id"`
	Symbol       string    `db:"symbol"`
	Quantity     float64   `db:"quantity"`
	AveragePrice float64   `db:"average_price"`
	UnrealizedPnl float64  `db:"unrealized_pnl"`
	RealizedPnl  float64   `db:"realized_pnl"`
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
}

// RiskLimit represents risk limits in the database
type RiskLimit struct {
	ID                 int64     `db:"id"`
	AccountID          string    `db:"account_id"`
	Symbol             string    `db:"symbol"`
	MaxPositionSize    float64   `db:"max_position_size"`
	MaxNotionalValue   float64   `db:"max_notional_value"`
	MaxLeverage        float64   `db:"max_leverage"`
	MaxDailyVolume     float64   `db:"max_daily_volume"`
	MaxDailyTrades     int       `db:"max_daily_trades"`
	MaxDrawdownPercent float64   `db:"max_drawdown_percent"`
	CreatedAt          time.Time `db:"created_at"`
	UpdatedAt          time.Time `db:"updated_at"`
}

// NewRiskRepository creates a new risk repository with fx dependency injection
func NewRiskRepository(p RiskRepositoryParams) *RiskRepository {
	return &RiskRepository{
		logger: p.Logger,
		db:     p.DB,
	}
}

// GetPosition retrieves a position by account ID and symbol
func (r *RiskRepository) GetPosition(ctx context.Context, accountID, symbol string) (*Position, error) {
	if r.db == nil {
		r.logger.Warn("Database connection not available")
		return nil, nil
	}

	query := `
		SELECT id, account_id, symbol, quantity, average_price, unrealized_pnl, realized_pnl, created_at, updated_at
		FROM positions
		WHERE account_id = $1 AND symbol = $2
	`

	var position Position
	err := r.db.GetContext(ctx, &position, query, accountID, symbol)
	if err != nil {
		r.logger.Error("Failed to get position",
			zap.String("account_id", accountID),
			zap.String("symbol", symbol),
			zap.Error(err))
		return nil, err
	}

	return &position, nil
}

// GetPositions retrieves positions by account ID
func (r *RiskRepository) GetPositions(ctx context.Context, accountID string) ([]*Position, error) {
	if r.db == nil {
		r.logger.Warn("Database connection not available")
		return nil, nil
	}

	query := `
		SELECT id, account_id, symbol, quantity, average_price, unrealized_pnl, realized_pnl, created_at, updated_at
		FROM positions
		WHERE account_id = $1
	`

	var positions []*Position
	err := r.db.SelectContext(ctx, &positions, query, accountID)
	if err != nil {
		r.logger.Error("Failed to get positions",
			zap.String("account_id", accountID),
			zap.Error(err))
		return nil, err
	}

	return positions, nil
}

// UpdatePosition updates a position
func (r *RiskRepository) UpdatePosition(ctx context.Context, position *Position) error {
	if r.db == nil {
		r.logger.Warn("Database connection not available")
		return nil
	}

	query := `
		UPDATE positions
		SET quantity = $1, average_price = $2, unrealized_pnl = $3, realized_pnl = $4, updated_at = $5
		WHERE account_id = $6 AND symbol = $7
	`

	position.UpdatedAt = time.Now()

	_, err := r.db.ExecContext(ctx, query,
		position.Quantity,
		position.AveragePrice,
		position.UnrealizedPnl,
		position.RealizedPnl,
		position.UpdatedAt,
		position.AccountID,
		position.Symbol,
	)

	if err != nil {
		r.logger.Error("Failed to update position",
			zap.String("account_id", position.AccountID),
			zap.String("symbol", position.Symbol),
			zap.Error(err))
		return err
	}

	return nil
}

// GetRiskLimits retrieves risk limits by account ID and symbol
func (r *RiskRepository) GetRiskLimits(ctx context.Context, accountID, symbol string) (*RiskLimit, error) {
	if r.db == nil {
		r.logger.Warn("Database connection not available")
		return nil, nil
	}

	query := `
		SELECT id, account_id, symbol, max_position_size, max_notional_value, max_leverage,
			max_daily_volume, max_daily_trades, max_drawdown_percent, created_at, updated_at
		FROM risk_limits
		WHERE account_id = $1 AND symbol = $2
	`

	var limits RiskLimit
	err := r.db.GetContext(ctx, &limits, query, accountID, symbol)
	if err != nil {
		r.logger.Error("Failed to get risk limits",
			zap.String("account_id", accountID),
			zap.String("symbol", symbol),
			zap.Error(err))
		return nil, err
	}

	return &limits, nil
}

// UpdateRiskLimits updates risk limits
func (r *RiskRepository) UpdateRiskLimits(ctx context.Context, limits *RiskLimit) error {
	if r.db == nil {
		r.logger.Warn("Database connection not available")
		return nil
	}

	query := `
		UPDATE risk_limits
		SET max_position_size = $1, max_notional_value = $2, max_leverage = $3,
			max_daily_volume = $4, max_daily_trades = $5, max_drawdown_percent = $6, updated_at = $7
		WHERE account_id = $8 AND symbol = $9
	`

	limits.UpdatedAt = time.Now()

	_, err := r.db.ExecContext(ctx, query,
		limits.MaxPositionSize,
		limits.MaxNotionalValue,
		limits.MaxLeverage,
		limits.MaxDailyVolume,
		limits.MaxDailyTrades,
		limits.MaxDrawdownPercent,
		limits.UpdatedAt,
		limits.AccountID,
		limits.Symbol,
	)

	if err != nil {
		r.logger.Error("Failed to update risk limits",
			zap.String("account_id", limits.AccountID),
			zap.String("symbol", limits.Symbol),
			zap.Error(err))
		return err
	}

	return nil
}

// RiskRepositoryModule provides the risk repository module for fx
var RiskRepositoryModule = fx.Options(
	fx.Provide(NewRiskRepository),
)

