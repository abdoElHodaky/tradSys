// Package assets provides asset handler registry and management for TradSys v3
package assets

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/pkg/interfaces"
	"github.com/abdoElHodaky/tradSys/pkg/types"
)

// HandlerRegistry manages asset-specific handlers
type HandlerRegistry struct {
	handlers map[types.AssetType]AssetHandler
	mu       sync.RWMutex
}

// AssetHandler defines the interface for asset-specific operations
type AssetHandler interface {
	ValidateOrder(ctx context.Context, order *interfaces.Order) error
	CalculateSettlement(ctx context.Context, order *interfaces.Order) (*Settlement, error)
	GetTradingHours(exchange types.ExchangeType) *TradingHours
	GetRiskParameters() *RiskParameters
	GetMinOrderSize() float64
	GetMaxOrderSize() float64
	GetPriceStep() float64
	IsMarketOrder(orderType interfaces.OrderType) bool
	CalculateFees(ctx context.Context, order *interfaces.Order) (*FeeCalculation, error)
}

// BaseAssetHandler provides common functionality for all asset handlers
type BaseAssetHandler struct {
	AssetType       types.AssetType
	MinOrderSize    float64
	MaxOrderSize    float64
	PriceStep       float64
	SettlementDays  int
	RiskParameters  *RiskParameters
	FeeStructure    *FeeStructure
}

// Settlement represents settlement information
type Settlement struct {
	OrderID         string    `json:"order_id"`
	SettlementDate  time.Time `json:"settlement_date"`
	SettlementType  string    `json:"settlement_type"`
	Currency        string    `json:"currency"`
	Amount          float64   `json:"amount"`
	Fees            float64   `json:"fees"`
	NetAmount       float64   `json:"net_amount"`
}

// TradingHours represents trading hours for an asset
type TradingHours struct {
	Open         string `json:"open"`
	Close        string `json:"close"`
	PreMarket    string `json:"pre_market,omitempty"`
	PostMarket   string `json:"post_market,omitempty"`
	Timezone     string `json:"timezone"`
	IsExtended   bool   `json:"is_extended"`
}

// RiskParameters represents risk management parameters
type RiskParameters struct {
	MaxPositionSize    float64 `json:"max_position_size"`
	MaxDailyVolume     float64 `json:"max_daily_volume"`
	VolatilityLimit    float64 `json:"volatility_limit"`
	LiquidityThreshold float64 `json:"liquidity_threshold"`
	MarginRequirement  float64 `json:"margin_requirement"`
}

// FeeStructure represents fee calculation structure
type FeeStructure struct {
	CommissionRate    float64            `json:"commission_rate"`
	MinCommission     float64            `json:"min_commission"`
	MaxCommission     float64            `json:"max_commission"`
	ExchangeFee       float64            `json:"exchange_fee"`
	RegulatoryFee     float64            `json:"regulatory_fee"`
	TierRates         map[string]float64 `json:"tier_rates"`
}

// FeeCalculation represents calculated fees
type FeeCalculation struct {
	Commission     float64 `json:"commission"`
	ExchangeFee    float64 `json:"exchange_fee"`
	RegulatoryFee  float64 `json:"regulatory_fee"`
	TotalFees      float64 `json:"total_fees"`
	NetAmount      float64 `json:"net_amount"`
}

// NewHandlerRegistry creates a new asset handler registry
func NewHandlerRegistry() *HandlerRegistry {
	registry := &HandlerRegistry{
		handlers: make(map[types.AssetType]AssetHandler),
	}
	
	// Register default handlers
	registry.registerDefaultHandlers()
	
	return registry
}

// RegisterHandler registers an asset handler
func (r *HandlerRegistry) RegisterHandler(assetType types.AssetType, handler AssetHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[assetType] = handler
}

// GetHandler retrieves an asset handler
func (r *HandlerRegistry) GetHandler(assetType types.AssetType) (AssetHandler, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	handler, exists := r.handlers[assetType]
	if !exists {
		return nil, fmt.Errorf("no handler registered for asset type: %s", assetType)
	}
	
	return handler, nil
}

// GetAllHandlers returns all registered handlers
func (r *HandlerRegistry) GetAllHandlers() map[types.AssetType]AssetHandler {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	result := make(map[types.AssetType]AssetHandler)
	for k, v := range r.handlers {
		result[k] = v
	}
	
	return result
}

// ValidateOrder validates an order using the appropriate asset handler
func (r *HandlerRegistry) ValidateOrder(ctx context.Context, order *interfaces.Order) error {
	handler, err := r.GetHandler(order.AssetType)
	if err != nil {
		return err
	}
	
	return handler.ValidateOrder(ctx, order)
}

// CalculateSettlement calculates settlement using the appropriate asset handler
func (r *HandlerRegistry) CalculateSettlement(ctx context.Context, order *interfaces.Order) (*Settlement, error) {
	handler, err := r.GetHandler(order.AssetType)
	if err != nil {
		return nil, err
	}
	
	return handler.CalculateSettlement(ctx, order)
}

// registerDefaultHandlers registers default handlers for all asset types
func (r *HandlerRegistry) registerDefaultHandlers() {
	// Traditional Assets
	r.RegisterHandler(types.STOCK, NewStockHandler())
	r.RegisterHandler(types.BOND, NewBondHandler())
	r.RegisterHandler(types.ETF, NewETFHandler())
	r.RegisterHandler(types.REIT, NewREITHandler())
	r.RegisterHandler(types.MUTUAL_FUND, NewMutualFundHandler())
	r.RegisterHandler(types.CRYPTO, NewCryptoHandler())
	r.RegisterHandler(types.FOREX, NewForexHandler())
	r.RegisterHandler(types.COMMODITY, NewCommodityHandler())
	
	// Islamic Assets
	r.RegisterHandler(types.SUKUK, NewSukukHandler())
	r.RegisterHandler(types.ISLAMIC_FUND, NewIslamicFundHandler())
	r.RegisterHandler(types.SHARIA_STOCK, NewShariaStockHandler())
	r.RegisterHandler(types.ISLAMIC_ETF, NewIslamicETFHandler())
	r.RegisterHandler(types.ISLAMIC_REIT, NewIslamicREITHandler())
	r.RegisterHandler(types.TAKAFUL, NewTakafulHandler())
}

// Base implementation methods
func (b *BaseAssetHandler) ValidateOrder(ctx context.Context, order *interfaces.Order) error {
	// Basic validations
	if order.Quantity < b.MinOrderSize {
		return fmt.Errorf("order quantity %f below minimum %f", order.Quantity, b.MinOrderSize)
	}
	
	if b.MaxOrderSize > 0 && order.Quantity > b.MaxOrderSize {
		return fmt.Errorf("order quantity %f exceeds maximum %f", order.Quantity, b.MaxOrderSize)
	}
	
	if order.Price <= 0 && order.Type != interfaces.OrderTypeMarket {
		return fmt.Errorf("price must be positive for non-market orders")
	}
	
	return nil
}

func (b *BaseAssetHandler) CalculateSettlement(ctx context.Context, order *interfaces.Order) (*Settlement, error) {
	settlementDate := time.Now().AddDate(0, 0, b.SettlementDays)
	amount := order.Price * order.Quantity
	
	fees, err := b.CalculateFees(ctx, order)
	if err != nil {
		return nil, err
	}
	
	netAmount := amount
	if order.Side == interfaces.OrderSideBuy {
		netAmount += fees.TotalFees
	} else {
		netAmount -= fees.TotalFees
	}
	
	return &Settlement{
		OrderID:        order.ID,
		SettlementDate: settlementDate,
		SettlementType: fmt.Sprintf("T+%d", b.SettlementDays),
		Currency:       "USD", // This would be determined by exchange
		Amount:         amount,
		Fees:           fees.TotalFees,
		NetAmount:      netAmount,
	}, nil
}

func (b *BaseAssetHandler) GetTradingHours(exchange types.ExchangeType) *TradingHours {
	// Default trading hours, can be overridden by specific handlers
	switch exchange {
	case types.EGX:
		return &TradingHours{
			Open:     "10:00",
			Close:    "14:30",
			Timezone: "EET",
		}
	case types.ADX:
		return &TradingHours{
			Open:     "10:00",
			Close:    "15:00",
			Timezone: "GST",
		}
	default:
		return &TradingHours{
			Open:     "09:30",
			Close:    "16:00",
			Timezone: "UTC",
		}
	}
}

func (b *BaseAssetHandler) GetRiskParameters() *RiskParameters {
	return b.RiskParameters
}

func (b *BaseAssetHandler) GetMinOrderSize() float64 {
	return b.MinOrderSize
}

func (b *BaseAssetHandler) GetMaxOrderSize() float64 {
	return b.MaxOrderSize
}

func (b *BaseAssetHandler) GetPriceStep() float64 {
	return b.PriceStep
}

func (b *BaseAssetHandler) IsMarketOrder(orderType interfaces.OrderType) bool {
	return orderType == interfaces.OrderTypeMarket
}

func (b *BaseAssetHandler) CalculateFees(ctx context.Context, order *interfaces.Order) (*FeeCalculation, error) {
	if b.FeeStructure == nil {
		return &FeeCalculation{}, nil
	}
	
	orderValue := order.Price * order.Quantity
	
	// Calculate commission
	commission := orderValue * b.FeeStructure.CommissionRate
	if commission < b.FeeStructure.MinCommission {
		commission = b.FeeStructure.MinCommission
	}
	if b.FeeStructure.MaxCommission > 0 && commission > b.FeeStructure.MaxCommission {
		commission = b.FeeStructure.MaxCommission
	}
	
	// Calculate other fees
	exchangeFee := orderValue * b.FeeStructure.ExchangeFee
	regulatoryFee := orderValue * b.FeeStructure.RegulatoryFee
	
	totalFees := commission + exchangeFee + regulatoryFee
	netAmount := orderValue
	
	if order.Side == interfaces.OrderSideBuy {
		netAmount += totalFees
	} else {
		netAmount -= totalFees
	}
	
	return &FeeCalculation{
		Commission:    commission,
		ExchangeFee:   exchangeFee,
		RegulatoryFee: regulatoryFee,
		TotalFees:     totalFees,
		NetAmount:     netAmount,
	}, nil
}

// Specific asset handlers

// StockHandler handles stock-specific operations
type StockHandler struct {
	BaseAssetHandler
}

func NewStockHandler() *StockHandler {
	return &StockHandler{
		BaseAssetHandler: BaseAssetHandler{
			AssetType:      types.STOCK,
			MinOrderSize:   1,
			MaxOrderSize:   1000000,
			PriceStep:      0.01,
			SettlementDays: 2, // T+2
			RiskParameters: &RiskParameters{
				MaxPositionSize:    100000,
				MaxDailyVolume:     1000000,
				VolatilityLimit:    0.20,
				LiquidityThreshold: 10000,
				MarginRequirement:  0.25,
			},
			FeeStructure: &FeeStructure{
				CommissionRate: 0.001, // 0.1%
				MinCommission:  5.0,
				MaxCommission:  100.0,
				ExchangeFee:    0.0001,
				RegulatoryFee:  0.0001,
			},
		},
	}
}

// SukukHandler handles Sukuk-specific operations
type SukukHandler struct {
	BaseAssetHandler
}

func NewSukukHandler() *SukukHandler {
	return &SukukHandler{
		BaseAssetHandler: BaseAssetHandler{
			AssetType:      types.SUKUK,
			MinOrderSize:   1000, // Higher minimum for Sukuk
			MaxOrderSize:   10000000,
			PriceStep:      0.01,
			SettlementDays: 1, // T+1 for bonds
			RiskParameters: &RiskParameters{
				MaxPositionSize:    1000000,
				MaxDailyVolume:     10000000,
				VolatilityLimit:    0.10,
				LiquidityThreshold: 100000,
				MarginRequirement:  0.10,
			},
			FeeStructure: &FeeStructure{
				CommissionRate: 0.0005, // 0.05% - lower for bonds
				MinCommission:  10.0,
				MaxCommission:  500.0,
				ExchangeFee:    0.00005,
				RegulatoryFee:  0.00005,
			},
		},
	}
}

func (s *SukukHandler) ValidateOrder(ctx context.Context, order *interfaces.Order) error {
	// Call base validation first
	if err := s.BaseAssetHandler.ValidateOrder(ctx, order); err != nil {
		return err
	}
	
	// Sukuk-specific validations
	if time.Now().Weekday() == time.Friday {
		return fmt.Errorf("Sukuk trading not allowed on Fridays")
	}
	
	return nil
}

// Factory functions for other handlers (simplified implementations)
func NewBondHandler() *BaseAssetHandler {
	return &BaseAssetHandler{
		AssetType:      types.BOND,
		MinOrderSize:   1000,
		MaxOrderSize:   10000000,
		PriceStep:      0.01,
		SettlementDays: 1,
		RiskParameters: &RiskParameters{
			MaxPositionSize:    1000000,
			MaxDailyVolume:     10000000,
			VolatilityLimit:    0.10,
			LiquidityThreshold: 100000,
			MarginRequirement:  0.10,
		},
		FeeStructure: &FeeStructure{
			CommissionRate: 0.0005,
			MinCommission:  10.0,
			MaxCommission:  500.0,
			ExchangeFee:    0.00005,
			RegulatoryFee:  0.00005,
		},
	}
}

func NewETFHandler() *BaseAssetHandler {
	return &BaseAssetHandler{
		AssetType:      types.ETF,
		MinOrderSize:   1,
		MaxOrderSize:   1000000,
		PriceStep:      0.01,
		SettlementDays: 2,
		RiskParameters: &RiskParameters{
			MaxPositionSize:    500000,
			MaxDailyVolume:     5000000,
			VolatilityLimit:    0.15,
			LiquidityThreshold: 50000,
			MarginRequirement:  0.20,
		},
		FeeStructure: &FeeStructure{
			CommissionRate: 0.0008,
			MinCommission:  5.0,
			MaxCommission:  100.0,
			ExchangeFee:    0.0001,
			RegulatoryFee:  0.0001,
		},
	}
}

func NewREITHandler() *BaseAssetHandler {
	return &BaseAssetHandler{
		AssetType:      types.REIT,
		MinOrderSize:   1,
		MaxOrderSize:   500000,
		PriceStep:      0.01,
		SettlementDays: 2,
		RiskParameters: &RiskParameters{
			MaxPositionSize:    200000,
			MaxDailyVolume:     2000000,
			VolatilityLimit:    0.18,
			LiquidityThreshold: 20000,
			MarginRequirement:  0.30,
		},
		FeeStructure: &FeeStructure{
			CommissionRate: 0.001,
			MinCommission:  5.0,
			MaxCommission:  150.0,
			ExchangeFee:    0.0001,
			RegulatoryFee:  0.0001,
		},
	}
}

func NewMutualFundHandler() *BaseAssetHandler {
	return &BaseAssetHandler{
		AssetType:      types.MUTUAL_FUND,
		MinOrderSize:   100,
		MaxOrderSize:   1000000,
		PriceStep:      0.01,
		SettlementDays: 1,
		RiskParameters: &RiskParameters{
			MaxPositionSize:    500000,
			MaxDailyVolume:     2000000,
			VolatilityLimit:    0.12,
			LiquidityThreshold: 10000,
			MarginRequirement:  0.15,
		},
		FeeStructure: &FeeStructure{
			CommissionRate: 0.0015,
			MinCommission:  10.0,
			MaxCommission:  200.0,
			ExchangeFee:    0.0002,
			RegulatoryFee:  0.0001,
		},
	}
}

func NewCryptoHandler() *BaseAssetHandler {
	return &BaseAssetHandler{
		AssetType:      types.CRYPTO,
		MinOrderSize:   0.001,
		MaxOrderSize:   1000,
		PriceStep:      0.01,
		SettlementDays: 0, // Instant settlement
		RiskParameters: &RiskParameters{
			MaxPositionSize:    100000,
			MaxDailyVolume:     500000,
			VolatilityLimit:    0.50,
			LiquidityThreshold: 5000,
			MarginRequirement:  0.50,
		},
		FeeStructure: &FeeStructure{
			CommissionRate: 0.002,
			MinCommission:  1.0,
			MaxCommission:  500.0,
			ExchangeFee:    0.0005,
			RegulatoryFee:  0.0001,
		},
	}
}

func NewForexHandler() *BaseAssetHandler {
	return &BaseAssetHandler{
		AssetType:      types.FOREX,
		MinOrderSize:   1000,
		MaxOrderSize:   10000000,
		PriceStep:      0.0001,
		SettlementDays: 2,
		RiskParameters: &RiskParameters{
			MaxPositionSize:    1000000,
			MaxDailyVolume:     50000000,
			VolatilityLimit:    0.30,
			LiquidityThreshold: 100000,
			MarginRequirement:  0.02,
		},
		FeeStructure: &FeeStructure{
			CommissionRate: 0.0001,
			MinCommission:  2.0,
			MaxCommission:  50.0,
			ExchangeFee:    0.00001,
			RegulatoryFee:  0.00001,
		},
	}
}

func NewCommodityHandler() *BaseAssetHandler {
	return &BaseAssetHandler{
		AssetType:      types.COMMODITY,
		MinOrderSize:   1,
		MaxOrderSize:   10000,
		PriceStep:      0.01,
		SettlementDays: 2,
		RiskParameters: &RiskParameters{
			MaxPositionSize:    100000,
			MaxDailyVolume:     1000000,
			VolatilityLimit:    0.25,
			LiquidityThreshold: 10000,
			MarginRequirement:  0.10,
		},
		FeeStructure: &FeeStructure{
			CommissionRate: 0.0012,
			MinCommission:  5.0,
			MaxCommission:  200.0,
			ExchangeFee:    0.0002,
			RegulatoryFee:  0.0001,
		},
	}
}

// Islamic asset handlers
func NewIslamicFundHandler() *BaseAssetHandler {
	return &BaseAssetHandler{
		AssetType:      types.ISLAMIC_FUND,
		MinOrderSize:   100,
		MaxOrderSize:   1000000,
		PriceStep:      0.01,
		SettlementDays: 1,
		RiskParameters: &RiskParameters{
			MaxPositionSize:    500000,
			MaxDailyVolume:     2000000,
			VolatilityLimit:    0.12,
			LiquidityThreshold: 10000,
			MarginRequirement:  0.15,
		},
		FeeStructure: &FeeStructure{
			CommissionRate: 0.0012, // Slightly lower for Islamic funds
			MinCommission:  8.0,
			MaxCommission:  180.0,
			ExchangeFee:    0.0002,
			RegulatoryFee:  0.0001,
		},
	}
}

func NewShariaStockHandler() *BaseAssetHandler {
	return &BaseAssetHandler{
		AssetType:      types.SHARIA_STOCK,
		MinOrderSize:   1,
		MaxOrderSize:   1000000,
		PriceStep:      0.01,
		SettlementDays: 2,
		RiskParameters: &RiskParameters{
			MaxPositionSize:    100000,
			MaxDailyVolume:     1000000,
			VolatilityLimit:    0.20,
			LiquidityThreshold: 10000,
			MarginRequirement:  0.25,
		},
		FeeStructure: &FeeStructure{
			CommissionRate: 0.0009, // Slightly lower for Sharia stocks
			MinCommission:  4.0,
			MaxCommission:  90.0,
			ExchangeFee:    0.0001,
			RegulatoryFee:  0.0001,
		},
	}
}

func NewIslamicETFHandler() *BaseAssetHandler {
	return &BaseAssetHandler{
		AssetType:      types.ISLAMIC_ETF,
		MinOrderSize:   1,
		MaxOrderSize:   1000000,
		PriceStep:      0.01,
		SettlementDays: 2,
		RiskParameters: &RiskParameters{
			MaxPositionSize:    500000,
			MaxDailyVolume:     5000000,
			VolatilityLimit:    0.15,
			LiquidityThreshold: 50000,
			MarginRequirement:  0.20,
		},
		FeeStructure: &FeeStructure{
			CommissionRate: 0.0007,
			MinCommission:  4.0,
			MaxCommission:  90.0,
			ExchangeFee:    0.0001,
			RegulatoryFee:  0.0001,
		},
	}
}

func NewIslamicREITHandler() *BaseAssetHandler {
	return &BaseAssetHandler{
		AssetType:      types.ISLAMIC_REIT,
		MinOrderSize:   1,
		MaxOrderSize:   500000,
		PriceStep:      0.01,
		SettlementDays: 2,
		RiskParameters: &RiskParameters{
			MaxPositionSize:    200000,
			MaxDailyVolume:     2000000,
			VolatilityLimit:    0.18,
			LiquidityThreshold: 20000,
			MarginRequirement:  0.30,
		},
		FeeStructure: &FeeStructure{
			CommissionRate: 0.0009,
			MinCommission:  4.0,
			MaxCommission:  130.0,
			ExchangeFee:    0.0001,
			RegulatoryFee:  0.0001,
		},
	}
}

func NewTakafulHandler() *BaseAssetHandler {
	return &BaseAssetHandler{
		AssetType:      types.TAKAFUL,
		MinOrderSize:   1,
		MaxOrderSize:   100000,
		PriceStep:      0.01,
		SettlementDays: 1,
		RiskParameters: &RiskParameters{
			MaxPositionSize:    50000,
			MaxDailyVolume:     500000,
			VolatilityLimit:    0.15,
			LiquidityThreshold: 5000,
			MarginRequirement:  0.20,
		},
		FeeStructure: &FeeStructure{
			CommissionRate: 0.0015,
			MinCommission:  5.0,
			MaxCommission:  100.0,
			ExchangeFee:    0.0002,
			RegulatoryFee:  0.0001,
		},
	}
}

// DefaultRegistry is the global asset handler registry
var DefaultRegistry = NewHandlerRegistry()

// GetDefaultRegistry returns the default registry instance
func GetDefaultRegistry() *HandlerRegistry {
	return DefaultRegistry
}
