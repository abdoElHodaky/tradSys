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

// NewHandlerRegistry creates a new handler registry with default handlers
func NewHandlerRegistry() *HandlerRegistry {
	registry := &HandlerRegistry{
		handlers: make(map[types.AssetType]AssetHandler),
	}
	
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

// DefaultRegistry is the global asset handler registry
var DefaultRegistry = NewHandlerRegistry()

// GetDefaultRegistry returns the default registry instance
func GetDefaultRegistry() *HandlerRegistry {
	return DefaultRegistry
}
