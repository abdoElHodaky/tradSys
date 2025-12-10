// Package types provides type conversion utilities for TradSys
package types

import (
	"fmt"
	"strings"
)

// ParseOrderSide converts string representation to OrderSide enum
func ParseOrderSide(side string) (OrderSide, error) {
	switch strings.ToLower(strings.TrimSpace(side)) {
	case "buy":
		return OrderSideBuy, nil
	case "sell":
		return OrderSideSell, nil
	default:
		return "", fmt.Errorf("invalid order side: %s (must be 'buy' or 'sell')", side)
	}
}

// ParseOrderType converts string representation to OrderType enum
func ParseOrderType(orderType string) (OrderType, error) {
	switch strings.ToLower(strings.TrimSpace(orderType)) {
	case "market":
		return OrderTypeMarket, nil
	case "limit":
		return OrderTypeLimit, nil
	case "stop":
		return OrderTypeStop, nil
	case "stop_limit", "stop-limit":
		return OrderTypeStopLimit, nil
	default:
		return "", fmt.Errorf("invalid order type: %s (must be 'market', 'limit', 'stop', or 'stop_limit')", orderType)
	}
}

// MustParseOrderSide is like ParseOrderSide but panics on error
func MustParseOrderSide(side string) OrderSide {
	result, err := ParseOrderSide(side)
	if err != nil {
		panic(err)
	}
	return result
}

// MustParseOrderType is like ParseOrderType but panics on error
func MustParseOrderType(orderType string) OrderType {
	result, err := ParseOrderType(orderType)
	if err != nil {
		panic(err)
	}
	return result
}

// OrderSideToString converts OrderSide enum to string representation
func OrderSideToString(side OrderSide) string {
	switch side {
	case OrderSideBuy:
		return "buy"
	case OrderSideSell:
		return "sell"
	default:
		return "unknown"
	}
}

// OrderTypeToString converts OrderType enum to string representation
func OrderTypeToString(orderType OrderType) string {
	switch orderType {
	case OrderTypeMarket:
		return "market"
	case OrderTypeLimit:
		return "limit"
	case OrderTypeStop:
		return "stop"
	case OrderTypeStopLimit:
		return "stop_limit"
	default:
		return "unknown"
	}
}
