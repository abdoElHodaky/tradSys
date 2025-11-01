// 🎯 **ADX Service Processors**
// Generated using TradSys Code Splitting Standards
//
// This file contains validation methods, helper functions, and processing logic
// for the Abu Dhabi Exchange (ADX) Service. These functions handle Islamic compliance
// validation, market timing calculations, and various utility operations.
//
// Performance Requirements: Standard latency, Islamic compliance processing
// File size limit: 410 lines

package exchanges

import (
	"fmt"
	"strings"
	"time"
)

// validateOrder validates an order for ADX submission
func (adx *ADXService) validateOrder(order *Order) error {
	if order == nil {
		return fmt.Errorf("order cannot be nil")
	}

	// Validate basic order fields
	if order.Symbol == "" {
		return fmt.Errorf("order symbol cannot be empty")
	}

	if order.Quantity <= 0 {
		return fmt.Errorf("order quantity must be positive")
	}

	if order.Price < 0 {
		return fmt.Errorf("order price cannot be negative")
	}

	// Validate asset type
	if !adx.isValidAssetType(order.AssetType) {
		return fmt.Errorf("invalid asset type: %v", order.AssetType)
	}

	// Validate symbol format for ADX
	if !adx.isValidADXSymbol(order.Symbol) {
		return fmt.Errorf("invalid ADX symbol format: %s", order.Symbol)
	}

	// Check market hours
	if !adx.IsMarketOpen() {
		return fmt.Errorf("market is currently closed")
	}

	// Validate order size limits
	if err := adx.validateOrderSize(order); err != nil {
		return fmt.Errorf("order size validation failed: %w", err)
	}

	return nil
}

// validateOrderSize validates order size against ADX limits
func (adx *ADXService) validateOrderSize(order *Order) error {
	// Get asset-specific limits
	limits := adx.getAssetLimits(order.AssetType)

	if limits == nil {
		return fmt.Errorf("no limits defined for asset type: %v", order.AssetType)
	}

	// Check minimum order size
	if order.Quantity < limits.MinOrderSize {
		return fmt.Errorf("order quantity %f below minimum %f", order.Quantity, limits.MinOrderSize)
	}

	// Check maximum order size
	if order.Quantity > limits.MaxOrderSize {
		return fmt.Errorf("order quantity %f exceeds maximum %f", order.Quantity, limits.MaxOrderSize)
	}

	// Check minimum order value
	orderValue := order.Quantity * order.Price
	if orderValue < limits.MinOrderValue {
		return fmt.Errorf("order value %f below minimum %f", orderValue, limits.MinOrderValue)
	}

	return nil
}

// isValidADXSymbol validates ADX symbol format
func (adx *ADXService) isValidADXSymbol(symbol string) bool {
	if len(symbol) < 2 || len(symbol) > 10 {
		return false
	}

	// ADX symbols are typically alphanumeric
	for _, char := range symbol {
		if !((char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9')) {
			return false
		}
	}

	// Check if symbol exists in ADX listings
	return adx.isSymbolListed(symbol)
}

// isSymbolListed checks if symbol is listed on ADX
func (adx *ADXService) isSymbolListed(symbol string) bool {
	// This would typically check against a database or cache
	// For now, we'll use a simple validation
	adxSymbols := []string{
		"ADCB", "FAB", "ADNOC", "ALDAR", "TAQA", "ETISALAT",
		"DANA", "AGTHIA", "ARKAN", "ADPORTS", "MULTIPLY",
	}

	for _, adxSymbol := range adxSymbols {
		if strings.EqualFold(symbol, adxSymbol) {
			return true
		}
	}

	return false
}

// isValidAssetType validates asset type for ADX
func (adx *ADXService) isValidAssetType(assetType AssetType) bool {
	validTypes := []AssetType{
		AssetTypeEquity,
		AssetTypeSukuk,
		AssetTypeIslamicFund,
		AssetTypeIslamicREIT,
		AssetTypeIslamicInstrument,
	}

	for _, validType := range validTypes {
		if assetType == validType {
			return true
		}
	}

	return false
}

// isIslamicAsset checks if an asset type is Islamic
func (adx *ADXService) isIslamicAsset(assetType AssetType) bool {
	islamicAssets := []AssetType{
		AssetTypeIslamicInstrument,
		AssetTypeSukuk,
		AssetTypeIslamicFund,
		AssetTypeIslamicREIT,
	}

	for _, islamic := range islamicAssets {
		if islamic == assetType {
			return true
		}
	}
	return false
}

// getAssetLimits returns trading limits for an asset type
func (adx *ADXService) getAssetLimits(assetType AssetType) *AssetLimits {
	limits := map[AssetType]*AssetLimits{
		AssetTypeEquity: {
			MinOrderSize:  1,
			MaxOrderSize:  1000000,
			MinOrderValue: 1000,     // AED 1,000
			MaxOrderValue: 50000000, // AED 50 million
		},
		AssetTypeSukuk: {
			MinOrderSize:  1000,     // AED 1,000 face value
			MaxOrderSize:  10000000, // AED 10 million
			MinOrderValue: 1000,
			MaxOrderValue: 100000000, // AED 100 million
		},
		AssetTypeIslamicFund: {
			MinOrderSize:  100,     // 100 units
			MaxOrderSize:  1000000, // 1 million units
			MinOrderValue: 1000,
			MaxOrderValue: 10000000, // AED 10 million
		},
	}

	return limits[assetType]
}

// isMarketOpen checks if the ADX market is currently open
func (adx *ADXService) isMarketOpen(now time.Time) bool {
	// Convert to UAE timezone
	uaeTime := now.In(adx.tradingHours.Timezone)

	// Check if it's a weekend (Friday-Saturday in UAE)
	weekday := uaeTime.Weekday()
	if weekday == time.Friday || weekday == time.Saturday {
		return false
	}

	// Check if it's a holiday
	for _, holiday := range adx.tradingHours.Holidays {
		if uaeTime.Format("2006-01-02") == holiday.Format("2006-01-02") {
			return false
		}
	}

	// Check trading hours (typically 10:00 AM - 3:00 PM UAE time)
	hour := uaeTime.Hour()
	minute := uaeTime.Minute()
	currentMinutes := hour*60 + minute

	openMinutes := 10 * 60  // 10:00 AM
	closeMinutes := 15 * 60 // 3:00 PM

	return currentMinutes >= openMinutes && currentMinutes < closeMinutes
}

// getCurrentSession returns the current trading session
func (adx *ADXService) getCurrentSession(now time.Time) *TradingSession {
	uaeTime := now.In(adx.tradingHours.Timezone)

	for _, session := range adx.tradingHours.TradingSessions {
		if uaeTime.After(session.StartTime) && uaeTime.Before(session.EndTime) {
			return &session
		}
	}

	return nil
}

// getNextMarketOpen returns the next market opening time
func (adx *ADXService) getNextMarketOpen(now time.Time) time.Time {
	uaeTime := now.In(adx.tradingHours.Timezone)

	// If market is currently open, return tomorrow's opening
	if adx.isMarketOpen(uaeTime) {
		return adx.getNextBusinessDay(uaeTime).Add(10 * time.Hour) // 10:00 AM
	}

	// If it's the same day but before opening, return today's opening
	if uaeTime.Hour() < 10 {
		return time.Date(uaeTime.Year(), uaeTime.Month(), uaeTime.Day(), 10, 0, 0, 0, adx.tradingHours.Timezone)
	}

	// Otherwise, return next business day opening
	return adx.getNextBusinessDay(uaeTime).Add(10 * time.Hour)
}

// getNextMarketClose returns the next market closing time
func (adx *ADXService) getNextMarketClose(now time.Time) time.Time {
	uaeTime := now.In(adx.tradingHours.Timezone)

	// If market is currently open, return today's closing
	if adx.isMarketOpen(uaeTime) {
		return time.Date(uaeTime.Year(), uaeTime.Month(), uaeTime.Day(), 15, 0, 0, 0, adx.tradingHours.Timezone)
	}

	// Otherwise, return next business day closing
	return adx.getNextBusinessDay(uaeTime).Add(15 * time.Hour)
}

// getNextBusinessDay returns the next business day (excluding weekends and holidays)
func (adx *ADXService) getNextBusinessDay(from time.Time) time.Time {
	next := from.AddDate(0, 0, 1)

	for {
		weekday := next.Weekday()
		if weekday != time.Friday && weekday != time.Saturday {
			// Check if it's not a holiday
			isHoliday := false
			for _, holiday := range adx.tradingHours.Holidays {
				if next.Format("2006-01-02") == holiday.Format("2006-01-02") {
					isHoliday = true
					break
				}
			}

			if !isHoliday {
				return next
			}
		}

		next = next.AddDate(0, 0, 1)
	}
}

// calculateTradingFees calculates trading fees for ADX orders
func (adx *ADXService) calculateTradingFees(order *Order) (*TradingFees, error) {
	if order == nil {
		return nil, fmt.Errorf("order cannot be nil")
	}

	orderValue := order.Quantity * order.Price

	fees := &TradingFees{
		Currency: "AED",
	}

	// Base commission (0.15% for equities, 0.10% for Sukuk)
	switch order.AssetType {
	case AssetTypeEquity:
		fees.Commission = orderValue * 0.0015
	case AssetTypeSukuk:
		fees.Commission = orderValue * 0.001
	case AssetTypeIslamicFund:
		fees.Commission = orderValue * 0.002
	default:
		fees.Commission = orderValue * 0.0015
	}

	// Minimum commission
	minCommission := 10.0 // AED 10
	if fees.Commission < minCommission {
		fees.Commission = minCommission
	}

	// Market fees (0.005%)
	fees.MarketFee = orderValue * 0.00005

	// Clearing fees (0.002%)
	fees.ClearingFee = orderValue * 0.00002

	// VAT (5% on fees)
	totalFeesBeforeVAT := fees.Commission + fees.MarketFee + fees.ClearingFee
	fees.VAT = totalFeesBeforeVAT * 0.05

	// Total fees
	fees.TotalFees = fees.Commission + fees.MarketFee + fees.ClearingFee + fees.VAT

	return fees, nil
}

// formatADXSymbol formats a symbol according to ADX conventions
func (adx *ADXService) formatADXSymbol(symbol string) string {
	// Convert to uppercase and remove spaces
	formatted := strings.ToUpper(strings.ReplaceAll(symbol, " ", ""))

	// Ensure it meets ADX format requirements
	if len(formatted) > 10 {
		formatted = formatted[:10]
	}

	return formatted
}

// validateIslamicCompliance validates Islamic compliance for an order
func (adx *ADXService) validateIslamicCompliance(order *Order) error {
	if !adx.isIslamicAsset(order.AssetType) {
		return nil // Not an Islamic asset, no validation needed
	}

	// Check if symbol is Sharia compliant
	if !adx.islamicCompliance.IsCompliant(order.Symbol) {
		return fmt.Errorf("symbol %s is not Sharia compliant", order.Symbol)
	}

	// Check for prohibited activities
	if err := adx.islamicCompliance.CheckProhibitedActivities(order); err != nil {
		return fmt.Errorf("prohibited activity detected: %w", err)
	}

	// Validate against Sharia rules
	for _, rule := range adx.islamicCompliance.GetApplicableRules(order.AssetType) {
		if !rule.Validator(order) {
			return fmt.Errorf("Sharia rule violation: %s", rule.Description)
		}
	}

	return nil
}

// generateOrderID generates a unique order ID for ADX
func (adx *ADXService) generateOrderID() string {
	timestamp := time.Now().Unix()
	return fmt.Sprintf("ADX%d", timestamp)
}

// logOrderActivity logs order activity for audit purposes
func (adx *ADXService) logOrderActivity(order *Order, activity string, details map[string]interface{}) {
	logEntry := map[string]interface{}{
		"timestamp":  time.Now(),
		"exchange":   "ADX",
		"order_id":   order.OrderID,
		"symbol":     order.Symbol,
		"activity":   activity,
		"asset_type": order.AssetType,
		"islamic":    adx.isIslamicAsset(order.AssetType),
		"details":    details,
	}

	// Log to audit trail
	adx.islamicCompliance.auditTrail.LogActivity(logEntry)
}
