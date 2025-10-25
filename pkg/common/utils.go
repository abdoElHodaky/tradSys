// Package common provides common utilities for TradSys v3
package common

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

// GenerateID generates a unique identifier
func GenerateID(prefix string) string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return fmt.Sprintf("%s_%s_%d", prefix, hex.EncodeToString(bytes), time.Now().Unix())
}

// GenerateOrderID generates a unique order ID
func GenerateOrderID() string {
	return GenerateID("ORD")
}

// GenerateTradeID generates a unique trade ID
func GenerateTradeID() string {
	return GenerateID("TRD")
}

// GenerateUserID generates a unique user ID
func GenerateUserID() string {
	return GenerateID("USR")
}

// ValidateSymbol validates a trading symbol format
func ValidateSymbol(symbol string) error {
	if symbol == "" {
		return fmt.Errorf("symbol cannot be empty")
	}
	
	if len(symbol) < 2 || len(symbol) > 10 {
		return fmt.Errorf("symbol must be between 2 and 10 characters")
	}
	
	// Symbol should contain only alphanumeric characters and dots
	for _, char := range symbol {
		if !((char >= 'A' && char <= 'Z') || 
			 (char >= 'a' && char <= 'z') || 
			 (char >= '0' && char <= '9') || 
			 char == '.') {
			return fmt.Errorf("symbol contains invalid character: %c", char)
		}
	}
	
	return nil
}

// NormalizeSymbol normalizes a trading symbol to uppercase
func NormalizeSymbol(symbol string) string {
	return strings.ToUpper(strings.TrimSpace(symbol))
}

// ValidatePrice validates a price value
func ValidatePrice(price float64) error {
	if price <= 0 {
		return fmt.Errorf("price must be positive, got: %f", price)
	}
	
	if price > 1000000 {
		return fmt.Errorf("price too high, got: %f", price)
	}
	
	return nil
}

// ValidateQuantity validates a quantity value
func ValidateQuantity(quantity float64) error {
	if quantity <= 0 {
		return fmt.Errorf("quantity must be positive, got: %f", quantity)
	}
	
	if quantity > 1000000 {
		return fmt.Errorf("quantity too high, got: %f", quantity)
	}
	
	return nil
}

// RoundPrice rounds a price to the appropriate number of decimal places
func RoundPrice(price float64, decimals int) float64 {
	multiplier := 1.0
	for i := 0; i < decimals; i++ {
		multiplier *= 10
	}
	return float64(int(price*multiplier+0.5)) / multiplier
}

// CalculatePercentageChange calculates percentage change between two values
func CalculatePercentageChange(oldValue, newValue float64) float64 {
	if oldValue == 0 {
		return 0
	}
	return ((newValue - oldValue) / oldValue) * 100
}

// IsWeekday checks if the given time is a weekday
func IsWeekday(t time.Time) bool {
	weekday := t.Weekday()
	return weekday >= time.Monday && weekday <= time.Friday
}

// GetNextTradingDay returns the next trading day (skips weekends)
func GetNextTradingDay(t time.Time) time.Time {
	next := t.AddDate(0, 0, 1)
	for !IsWeekday(next) {
		next = next.AddDate(0, 0, 1)
	}
	return next
}

// GetPreviousTradingDay returns the previous trading day (skips weekends)
func GetPreviousTradingDay(t time.Time) time.Time {
	prev := t.AddDate(0, 0, -1)
	for !IsWeekday(prev) {
		prev = prev.AddDate(0, 0, -1)
	}
	return prev
}

// FormatCurrency formats a currency value with the appropriate symbol
func FormatCurrency(amount float64, currency string) string {
	switch currency {
	case "EGP":
		return fmt.Sprintf("%.2f EGP", amount)
	case "AED":
		return fmt.Sprintf("%.2f AED", amount)
	case "USD":
		return fmt.Sprintf("$%.2f", amount)
	case "EUR":
		return fmt.Sprintf("€%.2f", amount)
	default:
		return fmt.Sprintf("%.2f %s", amount, currency)
	}
}

// SafeDivide performs division with zero check
func SafeDivide(numerator, denominator float64) float64 {
	if denominator == 0 {
		return 0
	}
	return numerator / denominator
}

// MinFloat64 returns the minimum of two float64 values
func MinFloat64(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// MaxFloat64 returns the maximum of two float64 values
func MaxFloat64(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// ClampFloat64 clamps a value between min and max
func ClampFloat64(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
