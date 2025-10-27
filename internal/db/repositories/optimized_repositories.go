package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Entity Types

// Order represents a trading order entity
type Order struct {
	ID                string    `db:"id" json:"id"`
	Symbol            string    `db:"symbol" json:"symbol"`
	Side              string    `db:"side" json:"side"`
	Type              string    `db:"type" json:"type"`
	Quantity          float64   `db:"quantity" json:"quantity"`
	Price             float64   `db:"price" json:"price"`
	RemainingQuantity float64   `db:"remaining_quantity" json:"remaining_quantity"`
	Status            string    `db:"status" json:"status"`
	UserID            string    `db:"user_id" json:"user_id"`
	CreatedAt         time.Time `db:"created_at" json:"created_at"`
	UpdatedAt         time.Time `db:"updated_at" json:"updated_at"`
}

// Trade represents a completed trade entity
type Trade struct {
	ID          string    `db:"id" json:"id"`
	Symbol      string    `db:"symbol" json:"symbol"`
	Price       float64   `db:"price" json:"price"`
	Quantity    float64   `db:"quantity" json:"quantity"`
	Side        string    `db:"side" json:"side"`
	BuyOrderID  string    `db:"buy_order_id" json:"buy_order_id"`
	SellOrderID string    `db:"sell_order_id" json:"sell_order_id"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
}

// Position represents a trading position entity
type Position struct {
	ID            string    `db:"id" json:"id"`
	Symbol        string    `db:"symbol" json:"symbol"`
	UserID        string    `db:"user_id" json:"user_id"`
	Quantity      float64   `db:"quantity" json:"quantity"`
	AveragePrice  float64   `db:"average_price" json:"average_price"`
	MarketPrice   float64   `db:"market_price" json:"market_price"`
	UnrealizedPnL float64   `db:"unrealized_pnl" json:"unrealized_pnl"`
	RealizedPnL   float64   `db:"realized_pnl" json:"realized_pnl"`
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time `db:"updated_at" json:"updated_at"`
}

// User represents a user entity
type User struct {
	ID        string    `db:"id" json:"id"`
	Username  string    `db:"username" json:"username"`
	Email     string    `db:"email" json:"email"`
	Status    string    `db:"status" json:"status"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// MarketData represents market data entity
type MarketData struct {
	ID        string    `db:"id" json:"id"`
	Symbol    string    `db:"symbol" json:"symbol"`
	Price     float64   `db:"price" json:"price"`
	Volume    float64   `db:"volume" json:"volume"`
	Timestamp time.Time `db:"timestamp" json:"timestamp"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

// Strategy represents a trading strategy entity
type Strategy struct {
	ID          string    `db:"id" json:"id"`
	Name        string    `db:"name" json:"name"`
	Description string    `db:"description" json:"description"`
	UserID      string    `db:"user_id" json:"user_id"`
	Status      string    `db:"status" json:"status"`
	Config      string    `db:"config" json:"config"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
}

// RiskMetric represents a risk metric entity
type RiskMetric struct {
	ID                string    `db:"id" json:"id"`
	UserID            string    `db:"user_id" json:"user_id"`
	Symbol            string    `db:"symbol" json:"symbol"`
	VaR               float64   `db:"var" json:"var"`
	ExpectedShortfall float64   `db:"expected_shortfall" json:"expected_shortfall"`
	MaxDrawdown       float64   `db:"max_drawdown" json:"max_drawdown"`
	Timestamp         time.Time `db:"timestamp" json:"timestamp"`
	CreatedAt         time.Time `db:"created_at" json:"created_at"`
}

// Pair represents a trading pair entity
type Pair struct {
	ID          string    `db:"id" json:"id"`
	Symbol      string    `db:"symbol" json:"symbol"`
	BaseAsset   string    `db:"base_asset" json:"base_asset"`
	QuoteAsset  string    `db:"quote_asset" json:"quote_asset"`
	Status      string    `db:"status" json:"status"`
	TickSize    float64   `db:"tick_size" json:"tick_size"`
	MinQuantity float64   `db:"min_quantity" json:"min_quantity"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
}

// Optimized Repository Implementations

// OrderRepository provides optimized order operations
type OrderRepository struct {
	*OptimizedRepository[Order]
}

// NewOrderRepository creates a new optimized order repository
func NewOrderRepository(db *gorm.DB, logger *zap.Logger) *OrderRepository {
	sqlDB, _ := db.DB()
	return &OrderRepository{
		OptimizedRepository: NewOptimizedRepository[Order](sqlDB, logger, "orders"),
	}
}

// FindByStatus finds orders by status
func (r *OrderRepository) FindByStatus(ctx context.Context, status string, limit int) ([]*Order, error) {
	return r.FindByField(ctx, "status", status, limit)
}

// FindByUserID finds orders by user ID
func (r *OrderRepository) FindByUserID(ctx context.Context, userID string, limit int) ([]*Order, error) {
	return r.FindByField(ctx, "user_id", userID, limit)
}

// TradeRepository provides optimized trade data access
type TradeRepository struct {
	*OptimizedRepository[Trade]
}

// NewTradeRepository creates a new trade repository
func NewTradeRepository(db *gorm.DB, logger *zap.Logger) *TradeRepository {
	sqlDB, _ := db.DB()
	return &TradeRepository{
		OptimizedRepository: NewOptimizedRepository[Trade](sqlDB, logger, "trades"),
	}
}

// FindBySymbol finds trades by symbol
func (r *TradeRepository) FindBySymbol(ctx context.Context, symbol string, limit int) ([]*Trade, error) {
	return r.FindByField(ctx, "symbol", symbol, limit)
}

// FindByOrderID finds trades by order ID
func (r *TradeRepository) FindByOrderID(ctx context.Context, orderID string) ([]*Trade, error) {
	query := `SELECT * FROM trades WHERE buy_order_id = $1 OR sell_order_id = $1 ORDER BY created_at DESC`

	rows, err := r.GetDB().QueryContext(ctx, query, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trades []*Trade
	for rows.Next() {
		trade := new(Trade)
		err := r.scanRowIntoEntity(rows, trade)
		if err != nil {
			continue
		}
		trades = append(trades, trade)
	}

	return trades, nil
}

// PositionRepository provides optimized position operations
type PositionRepository struct {
	*OptimizedRepository[Position]
}

// NewPositionRepository creates a new optimized position repository
func NewPositionRepository(db *gorm.DB, logger *zap.Logger) *PositionRepository {
	sqlDB, _ := db.DB()
	return &PositionRepository{
		OptimizedRepository: NewOptimizedRepository[Position](sqlDB, logger, "positions"),
	}
}

// FindBySymbol finds positions by symbol
func (r *PositionRepository) FindBySymbol(ctx context.Context, symbol string) (*Position, error) {
	positions, err := r.FindByField(ctx, "symbol", symbol, 1)
	if err != nil {
		return nil, err
	}
	if len(positions) == 0 {
		return nil, nil
	}
	return positions[0], nil
}

// FindByUserID finds positions by user ID
func (r *PositionRepository) FindByUserID(ctx context.Context, userID string) ([]*Position, error) {
	return r.FindByField(ctx, "user_id", userID, 100)
}

// UserRepository provides optimized user data access
type UserRepository struct {
	*OptimizedRepository[User]
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *gorm.DB, logger *zap.Logger) *UserRepository {
	sqlDB, _ := db.DB()
	return &UserRepository{
		OptimizedRepository: NewOptimizedRepository[User](sqlDB, logger, "users"),
	}
}

// FindByUsername finds user by username
func (r *UserRepository) FindByUsername(ctx context.Context, username string) (*User, error) {
	users, err := r.FindByField(ctx, "username", username, 1)
	if err != nil {
		return nil, err
	}
	if len(users) == 0 {
		return nil, nil
	}
	return users[0], nil
}

// FindByEmail finds user by email
func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*User, error) {
	users, err := r.FindByField(ctx, "email", email, 1)
	if err != nil {
		return nil, err
	}
	if len(users) == 0 {
		return nil, nil
	}
	return users[0], nil
}

// MarketDataRepository provides optimized market data operations
type MarketDataRepository struct {
	*OptimizedRepository[MarketData]
}

// NewMarketDataRepository creates a new optimized market data repository
func NewMarketDataRepository(db *gorm.DB, logger *zap.Logger) *MarketDataRepository {
	sqlDB, _ := db.DB()
	return &MarketDataRepository{
		OptimizedRepository: NewOptimizedRepository[MarketData](sqlDB, logger, "market_data"),
	}
}

// FindBySymbol finds market data by symbol
func (r *MarketDataRepository) FindBySymbol(ctx context.Context, symbol string, limit int) ([]*MarketData, error) {
	return r.FindByField(ctx, "symbol", symbol, limit)
}

// FindLatestBySymbol finds the latest market data for a symbol
func (r *MarketDataRepository) FindLatestBySymbol(ctx context.Context, symbol string) (*MarketData, error) {
	query := `SELECT * FROM market_data WHERE symbol = $1 ORDER BY timestamp DESC LIMIT 1`

	row := r.GetDB().QueryRowContext(ctx, query, symbol)

	marketData := new(MarketData)
	err := r.scanRowIntoEntity(row, marketData)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return marketData, nil
}

// StrategyRepository provides optimized strategy data access
type StrategyRepository struct {
	*OptimizedRepository[Strategy]
}

// NewStrategyRepository creates a new strategy repository
func NewStrategyRepository(db *gorm.DB, logger *zap.Logger) *StrategyRepository {
	sqlDB, _ := db.DB()
	return &StrategyRepository{
		OptimizedRepository: NewOptimizedRepository[Strategy](sqlDB, logger, "strategies"),
	}
}

// FindByUserID finds strategies by user ID
func (r *StrategyRepository) FindByUserID(ctx context.Context, userID string) ([]*Strategy, error) {
	return r.FindByField(ctx, "user_id", userID, 100)
}

// FindByStatus finds strategies by status
func (r *StrategyRepository) FindByStatus(ctx context.Context, status string, limit int) ([]*Strategy, error) {
	return r.FindByField(ctx, "status", status, limit)
}

// RiskRepository provides optimized risk operations
type RiskRepository struct {
	*OptimizedRepository[RiskMetric]
}

// NewRiskRepository creates a new optimized risk repository
func NewRiskRepository(db *gorm.DB, logger *zap.Logger) *RiskRepository {
	sqlDB, _ := db.DB()
	return &RiskRepository{
		OptimizedRepository: NewOptimizedRepository[RiskMetric](sqlDB, logger, "risk_metrics"),
	}
}

// FindByUserID finds risk metrics by user ID
func (r *RiskRepository) FindByUserID(ctx context.Context, userID string, limit int) ([]*RiskMetric, error) {
	return r.FindByField(ctx, "user_id", userID, limit)
}

// FindBySymbol finds risk metrics by symbol
func (r *RiskRepository) FindBySymbol(ctx context.Context, symbol string, limit int) ([]*RiskMetric, error) {
	return r.FindByField(ctx, "symbol", symbol, limit)
}

// PairRepository provides optimized pair operations
type PairRepository struct {
	*OptimizedRepository[Pair]
}

// NewPairRepository creates a new optimized pair repository
func NewPairRepository(db *gorm.DB, logger *zap.Logger) *PairRepository {
	sqlDB, _ := db.DB()
	return &PairRepository{
		OptimizedRepository: NewOptimizedRepository[Pair](sqlDB, logger, "pairs"),
	}
}

// FindBySymbol finds pair by symbol
func (r *PairRepository) FindBySymbol(ctx context.Context, symbol string) (*Pair, error) {
	pairs, err := r.FindByField(ctx, "symbol", symbol, 1)
	if err != nil {
		return nil, err
	}
	if len(pairs) == 0 {
		return nil, nil
	}
	return pairs[0], nil
}

// FindByStatus finds pairs by status
func (r *PairRepository) FindByStatus(ctx context.Context, status string, limit int) ([]*Pair, error) {
	return r.FindByField(ctx, "status", status, limit)
}

// Count returns the total number of pairs
func (r *PairRepository) Count(ctx context.Context) (int64, error) {
	return r.OptimizedRepository.Count(ctx)
}

// Repository Manager

// RepositoryManager manages all repositories
type RepositoryManager struct {
	Order      *OrderRepository
	Trade      *TradeRepository
	Position   *PositionRepository
	User       *UserRepository
	MarketData *MarketDataRepository
	Strategy   *StrategyRepository
	Risk       *RiskRepository
	Pair       *PairRepository
}

// NewRepositoryManager creates a new repository manager
func NewRepositoryManager(db *gorm.DB, logger *zap.Logger) *RepositoryManager {
	return &RepositoryManager{
		Order:      NewOrderRepository(db, logger),
		Trade:      NewTradeRepository(db, logger),
		Position:   NewPositionRepository(db, logger),
		User:       NewUserRepository(db, logger),
		MarketData: NewMarketDataRepository(db, logger),
		Strategy:   NewStrategyRepository(db, logger),
		Risk:       NewRiskRepository(db, logger),
		Pair:       NewPairRepository(db, logger),
	}
}

// Health check for all repositories
func (rm *RepositoryManager) HealthCheck(ctx context.Context) error {
	// Simple health check by counting records in each table
	repositories := map[string]interface {
		Count(context.Context) (int64, error)
	}{
		"orders":       rm.Order,
		"trades":       rm.Trade,
		"positions":    rm.Position,
		"users":        rm.User,
		"market_data":  rm.MarketData,
		"strategies":   rm.Strategy,
		"risk_metrics": rm.Risk,
		"pairs":        rm.Pair,
	}

	for name, repo := range repositories {
		_, err := repo.Count(ctx)
		if err != nil {
			return fmt.Errorf("health check failed for %s: %w", name, err)
		}
	}

	return nil
}
