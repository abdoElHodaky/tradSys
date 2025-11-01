// Package assets provides asset handler registry and management for TradSys v3
package assets

import (
	"context"
	"time"

	"github.com/abdoElHodaky/tradSys/pkg/interfaces"
	"github.com/abdoElHodaky/tradSys/pkg/types"
)

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
	AssetType      types.AssetType
	MinOrderSize   float64
	MaxOrderSize   float64
	PriceStep      float64
	SettlementDays int
	RiskParameters *RiskParameters
	FeeStructure   *FeeStructure
}

// Settlement represents settlement information
type Settlement struct {
	OrderID        string    `json:"order_id"`
	SettlementDate time.Time `json:"settlement_date"`
	SettlementType string    `json:"settlement_type"`
	Currency       string    `json:"currency"`
	Amount         float64   `json:"amount"`
	Fees           float64   `json:"fees"`
	NetAmount      float64   `json:"net_amount"`
}

// TradingHours represents trading hours for an exchange
type TradingHours struct {
	Open     string `json:"open"`
	Close    string `json:"close"`
	Timezone string `json:"timezone"`
}

// RiskParameters defines risk limits for an asset type
type RiskParameters struct {
	MaxPositionSize    float64 `json:"max_position_size"`
	MaxDailyVolume     float64 `json:"max_daily_volume"`
	VolatilityLimit    float64 `json:"volatility_limit"`
	LiquidityThreshold float64 `json:"liquidity_threshold"`
	MarginRequirement  float64 `json:"margin_requirement"`
}

// FeeStructure defines fee calculation parameters
type FeeStructure struct {
	CommissionRate float64 `json:"commission_rate"`
	MinCommission  float64 `json:"min_commission"`
	MaxCommission  float64 `json:"max_commission"`
	ExchangeFee    float64 `json:"exchange_fee"`
	RegulatoryFee  float64 `json:"regulatory_fee"`
}

// FeeCalculation represents calculated fees for an order
type FeeCalculation struct {
	Commission    float64 `json:"commission"`
	ExchangeFee   float64 `json:"exchange_fee"`
	RegulatoryFee float64 `json:"regulatory_fee"`
	TotalFees     float64 `json:"total_fees"`
	NetAmount     float64 `json:"net_amount"`
}
