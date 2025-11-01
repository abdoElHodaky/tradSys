// Package types provides type conversion utilities for TradSys
package types

import (
	"fmt"
	"strings"

	tradingTypes "github.com/abdoElHodaky/tradSys/internal/trading/types"
)

// ParseOrderSide converts string representation to OrderSide enum
func ParseOrderSide(side string) (tradingTypes.OrderSide, error) {
	switch strings.ToLower(strings.TrimSpace(side)) {
	case "buy":
		return tradingTypes.OrderSideBuy, nil
	case "sell":
		return tradingTypes.OrderSideSell, nil
	default:
		return "", fmt.Errorf("invalid order side: %s (must be 'buy' or 'sell')", side)
	}
}

// ParseOrderType converts string representation to OrderType enum
func ParseOrderType(orderType string) (tradingTypes.OrderType, error) {
	switch strings.ToLower(strings.TrimSpace(orderType)) {
	case "market":
		return tradingTypes.OrderTypeMarket, nil
	case "limit":
		return tradingTypes.OrderTypeLimit, nil
	case "stop":
		return tradingTypes.OrderTypeStop, nil
	case "stop_limit", "stop-limit":
		return tradingTypes.OrderTypeStopLimit, nil
	default:
		return "", fmt.Errorf("invalid order type: %s (must be 'market', 'limit', 'stop', or 'stop_limit')", orderType)
	}
}

// MustParseOrderSide is like ParseOrderSide but panics on error
func MustParseOrderSide(side string) tradingTypes.OrderSide {
	result, err := ParseOrderSide(side)
	if err != nil {
		panic(err)
	}
	return result
}

// MustParseOrderType is like ParseOrderType but panics on error
func MustParseOrderType(orderType string) tradingTypes.OrderType {
	result, err := ParseOrderType(orderType)
	if err != nil {
		panic(err)
	}
	return result
}

// OrderSideToString converts OrderSide enum to string representation
func OrderSideToString(side tradingTypes.OrderSide) string {
	switch side {
	case tradingTypes.OrderSideBuy:
		return "buy"
	case tradingTypes.OrderSideSell:
		return "sell"
	default:
		return "unknown"
	}
}

// OrderTypeToString converts OrderType enum to string representation
func OrderTypeToString(orderType tradingTypes.OrderType) string {
	switch orderType {
	case tradingTypes.OrderTypeMarket:
		return "market"
	case tradingTypes.OrderTypeLimit:
		return "limit"
	case tradingTypes.OrderTypeStop:
		return "stop"
	case tradingTypes.OrderTypeStopLimit:
		return "stop_limit"
	default:
		return "unknown"
	}
}
