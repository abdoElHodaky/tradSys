// Package types provides exchange-related type definitions
package types

import (
	"fmt"
	"strings"
	"time"
)

// ExchangeType represents different exchanges
type ExchangeType string

const (
	EGX ExchangeType = "EGX" // Egyptian Exchange
	ADX ExchangeType = "ADX" // Abu Dhabi Exchange
)

// IsValid checks if the exchange type is valid
func (et ExchangeType) IsValid() bool {
	switch et {
	case EGX, ADX:
		return true
	default:
		return false
	}
}

// String returns the string representation of ExchangeType
func (et ExchangeType) String() string {
	return string(et)
}

// GetTimezone returns the timezone for the exchange
func (et ExchangeType) GetTimezone() string {
	switch et {
	case EGX:
		return "Africa/Cairo" // EET (Egypt Eastern Time)
	case ADX:
		return "Asia/Dubai"   // GST (Gulf Standard Time)
	default:
		return "UTC"
	}
}

// GetCurrency returns the primary currency for the exchange
func (et ExchangeType) GetCurrency() string {
	switch et {
	case EGX:
		return "EGP" // Egyptian Pound
	case ADX:
		return "AED" // UAE Dirham
	default:
		return "USD"
	}
}

// ParseExchangeType parses a string into ExchangeType
func ParseExchangeType(s string) (ExchangeType, error) {
	exchangeType := ExchangeType(strings.ToUpper(s))
	if !exchangeType.IsValid() {
		return "", fmt.Errorf("invalid exchange type: %s", s)
	}
	return exchangeType, nil
}

// TradingHours represents trading hours for an exchange
type TradingHours struct {
	Open     string `json:"open"`     // e.g., "10:00"
	Close    string `json:"close"`    // e.g., "14:30"
	Timezone string `json:"timezone"` // e.g., "EET"
}

// GetTradingHours returns trading hours for the exchange
func (et ExchangeType) GetTradingHours() *TradingHours {
	switch et {
	case EGX:
		return &TradingHours{
			Open:     "10:00",
			Close:    "14:30",
			Timezone: "EET",
		}
	case ADX:
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

// IsMarketOpen checks if the market is currently open
func (et ExchangeType) IsMarketOpen() bool {
	hours := et.GetTradingHours()
	location, err := time.LoadLocation(et.GetTimezone())
	if err != nil {
		return false
	}
	
	now := time.Now().In(location)
	
	// Validate time format
	var err error
	_, err = time.Parse("15:04", hours.Open)
	if err != nil {
		return false
	}
	
	_, err = time.Parse("15:04", hours.Close)
	if err != nil {
		return false
	}
	
	// Create today's open and close times
	today := now.Format("2006-01-02")
	todayOpen, _ := time.ParseInLocation("2006-01-02 15:04", today+" "+hours.Open, location)
	todayClose, _ := time.ParseInLocation("2006-01-02 15:04", today+" "+hours.Close, location)
	
	// Check if current time is between open and close
	return now.After(todayOpen) && now.Before(todayClose) && isWeekday(now)
}

// isWeekday checks if the given time is a weekday
func isWeekday(t time.Time) bool {
	weekday := t.Weekday()
	return weekday >= time.Monday && weekday <= time.Friday
}

// GetAllExchangeTypes returns all valid exchange types
func GetAllExchangeTypes() []ExchangeType {
	return []ExchangeType{EGX, ADX}
}
