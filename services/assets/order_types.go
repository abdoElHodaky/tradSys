// Package assets provides type aliases for trading and asset types
// Following KISS: Use type aliases from pkg for shared types
package assets

import (
	"github.com/abdoElhodaky/tradSys/pkg/interfaces"
	"github.com/abdoElhodaky/tradSys/pkg/types"
)

// Type aliases for KISS compliance - eliminate duplicates
type (
	// Order aliases from pkg/interfaces for polymorphic operations
	Order        = interfaces.Order
	OrderSide    = interfaces.OrderSide
	OrderType    = interfaces.OrderType
	TimeInForce  = interfaces.TimeInForce
	OrderStatus  = interfaces.OrderStatus
	OrderResponse = interfaces.OrderResponse
	
	// Asset aliases from pkg/types
	AssetType      = types.AssetType
	AssetInfo      = types.AssetInfo
	Asset          = types.Asset
	
	// Export local types
	AssetFilter    = AssetFilter
	AssetData      = AssetData
)

// AssetStatus for local use
type AssetStatus string

const (
	AssetStatusActive    AssetStatus = "active"
	AssetStatusInactive  AssetStatus = "inactive"
	AssetStatusPending   AssetStatus = "pending"
	AssetStatusSuspended AssetStatus = "suspended"
)

// AssetFilter provides filtering capabilities
type AssetFilter struct {
	Symbol     string
	Name       string
	AssetType  AssetType
	Exchange   string
	Region     string
	MinPrice   float64
	MaxPrice   float64
}

// AssetData holds asset information
type AssetData struct {
	Symbol      string
	Name        string
	AssetType   AssetType
	Exchange    string
	Region      string
	Currency    string
	Description string
}
