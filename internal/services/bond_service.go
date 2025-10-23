package services

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/db/models"
	"github.com/abdoElHodaky/tradSys/internal/trading/types"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// BondService handles bond-specific operations and calculations
type BondService struct {
	db           *gorm.DB
	assetService *AssetService
	logger       *zap.Logger
}

// NewBondService creates a new bond service instance
func NewBondService(db *gorm.DB, assetService *AssetService, logger *zap.Logger) *BondService {
	return &BondService{
		db:           db,
		assetService: assetService,
		logger:       logger,
	}
}

// BondMetrics represents comprehensive bond performance and risk metrics
type BondMetrics struct {
	Symbol           string    `json:"symbol"`
	FaceValue        float64   `json:"face_value"`
	CouponRate       float64   `json:"coupon_rate"`
	YieldToMaturity  float64   `json:"yield_to_maturity"`
	CurrentYield     float64   `json:"current_yield"`
	Duration         float64   `json:"duration"`
	ModifiedDuration float64   `json:"modified_duration"`
	Convexity        float64   `json:"convexity"`
	CreditRating     string    `json:"credit_rating"`
	MaturityDate     time.Time `json:"maturity_date"`
	IssueDate        time.Time `json:"issue_date"`
	CallableDate     *time.Time `json:"callable_date,omitempty"`
	AccruedInterest  float64   `json:"accrued_interest"`
	DaysToMaturity   int       `json:"days_to_maturity"`
	YearsToMaturity  float64   `json:"years_to_maturity"`
	PriceVolatility  float64   `json:"price_volatility"`
	CreditSpread     float64   `json:"credit_spread"`
	OptionAdjustedSpread float64 `json:"option_adjusted_spread"`
}

// YieldCurvePoint represents a point on the yield curve
type YieldCurvePoint struct {
	Maturity string  `json:"maturity"`
	Yield    float64 `json:"yield"`
	Duration float64 `json:"duration"`
}

// CreditRiskAssessment represents bond credit risk analysis
type CreditRiskAssessment struct {
	CurrentRating    string    `json:"current_rating"`
	PreviousRating   string    `json:"previous_rating"`
	RatingDate       time.Time `json:"rating_date"`
	ProbabilityOfDefault float64 `json:"probability_of_default"`
	LossGivenDefault float64   `json:"loss_given_default"`
	CreditSpread     float64   `json:"credit_spread"`
	RatingOutlook    string    `json:"rating_outlook"`
	RatingAgency     string    `json:"rating_agency"`
}

// CashFlow represents a bond cash flow
type CashFlow struct {
	Date        time.Time `json:"date"`
	Amount      float64   `json:"amount"`
	Type        string    `json:"type"` // "coupon", "principal", "call"
	PresentValue float64  `json:"present_value"`
}

// CreateBond creates a new bond with initial metadata
func (s *BondService) CreateBond(symbol, issuer string, faceValue, couponRate float64, 
	maturityDate time.Time, creditRating string) error {
	s.logger.Info("Creating new bond", zap.String("symbol", symbol))

	attributesMap := map[string]interface{}{
		"issuer":         issuer,
		"face_value":     faceValue,
		"coupon_rate":    couponRate,
		"maturity_date":  maturityDate,
		"credit_rating":  creditRating,
		"issue_date":     time.Now(),
		"bond_type":      "corporate",
		"currency":       "USD",
		"payment_frequency": 2, // Semi-annual
	}

	attributesJSON, err := json.Marshal(attributesMap)
	if err != nil {
		return fmt.Errorf("failed to marshal bond attributes: %w", err)
	}

	// Convert JSON bytes to AssetAttributes map
	var attributes models.AssetAttributes
	if err := json.Unmarshal(attributesJSON, &attributes); err != nil {
		return fmt.Errorf("failed to parse bond attributes: %w", err)
	}

	asset := &models.AssetMetadata{
		Symbol:     symbol,
		AssetType:  types.AssetTypeBond,
		Sector:     "fixed_income",
		Attributes: attributes,
		IsActive:   true,
	}

	if err := s.db.Create(asset).Error; err != nil {
		return fmt.Errorf("failed to create bond asset: %w", err)
	}

	s.logger.Info("Bond created successfully", zap.String("symbol", symbol))
	return nil
}

// GetBondMetrics retrieves comprehensive bond metrics
func (s *BondService) GetBondMetrics(symbol string) (*BondMetrics, error) {
	s.logger.Debug("Retrieving bond metrics", zap.String("symbol", symbol))

	asset, err := s.assetService.GetAssetBySymbol(context.Background(), symbol)
	if err != nil {
		return nil, fmt.Errorf("failed to get bond asset: %w", err)
	}

	if asset.AssetType != types.AssetTypeBond {
		return nil, fmt.Errorf("asset %s is not a bond", symbol)
	}

	attributes := map[string]interface{}(asset.Attributes)

	pricing, err := s.assetService.GetCurrentPricing(context.Background(), symbol)
	if err != nil {
		s.logger.Warn("Failed to get current pricing", zap.String("symbol", symbol), zap.Error(err))
		pricing = &models.AssetPricing{Price: 100.0} // Par value default
	}

	metrics := &BondMetrics{
		Symbol:      symbol,
		FaceValue:   s.getFloatAttribute(attributes, "face_value"),
		CouponRate:  s.getFloatAttribute(attributes, "coupon_rate"),
		CreditRating: s.getStringAttribute(attributes, "credit_rating"),
	}

	// Parse dates
	if maturityStr := s.getStringAttribute(attributes, "maturity_date"); maturityStr != "" {
		if parsed, err := time.Parse(time.RFC3339, maturityStr); err == nil {
			metrics.MaturityDate = parsed
		}
	}

	if issueStr := s.getStringAttribute(attributes, "issue_date"); issueStr != "" {
		if parsed, err := time.Parse(time.RFC3339, issueStr); err == nil {
			metrics.IssueDate = parsed
		}
	}

	// Calculate derived metrics
	s.calculateBondMetrics(metrics, pricing.Price)

	return metrics, nil
}

// CalculateYieldToMaturity calculates YTM for a bond
func (s *BondService) CalculateYieldToMaturity(faceValue, currentPrice, couponRate float64, 
	yearsToMaturity float64, paymentsPerYear int) float64 {
	
	if yearsToMaturity <= 0 || currentPrice <= 0 {
		return 0.0
	}

	// Approximate YTM using Newton-Raphson method
	ytm := couponRate / 100.0 // Initial guess
	
	for i := 0; i < 100; i++ { // Max iterations
		pv := s.calculatePresentValue(faceValue, couponRate, ytm, yearsToMaturity, paymentsPerYear)
		derivative := s.calculatePVDerivative(faceValue, couponRate, ytm, yearsToMaturity, paymentsPerYear)
		
		if math.Abs(derivative) < 1e-10 {
			break
		}
		
		newYTM := ytm - (pv-currentPrice)/derivative
		
		if math.Abs(newYTM-ytm) < 1e-8 {
			break
		}
		
		ytm = newYTM
	}

	return ytm * 100 // Convert to percentage
}

// CalculateDuration calculates modified duration for a bond
func (s *BondService) CalculateDuration(faceValue, couponRate, ytm float64, 
	yearsToMaturity float64, paymentsPerYear int) (float64, float64) {
	
	if yearsToMaturity <= 0 || ytm <= 0 {
		return 0.0, 0.0
	}

	ytmDecimal := ytm / 100.0
	couponPayment := (couponRate / 100.0) * faceValue / float64(paymentsPerYear)
	
	var macaulayDuration float64
	var presentValue float64
	
	// Calculate Macaulay Duration
	for i := 1; i <= int(yearsToMaturity*float64(paymentsPerYear)); i++ {
		period := float64(i)
		discountFactor := math.Pow(1+ytmDecimal/float64(paymentsPerYear), -period)
		
		var cashFlow float64
		if i == int(yearsToMaturity*float64(paymentsPerYear)) {
			cashFlow = couponPayment + faceValue // Final payment includes principal
		} else {
			cashFlow = couponPayment
		}
		
		pv := cashFlow * discountFactor
		weightedTime := pv * (period / float64(paymentsPerYear))
		
		macaulayDuration += weightedTime
		presentValue += pv
	}
	
	if presentValue > 0 {
		macaulayDuration /= presentValue
	}
	
	// Calculate Modified Duration
	modifiedDuration := macaulayDuration / (1 + ytmDecimal/float64(paymentsPerYear))
	
	return macaulayDuration, modifiedDuration
}

// GetYieldCurve retrieves yield curve data for bond analysis
func (s *BondService) GetYieldCurve(curveType string) ([]YieldCurvePoint, error) {
	s.logger.Debug("Retrieving yield curve", zap.String("curve_type", curveType))

	// In a real implementation, this would fetch from market data providers
	// For now, return sample yield curve data
	yieldCurve := []YieldCurvePoint{
		{Maturity: "1M", Yield: 0.25, Duration: 0.08},
		{Maturity: "3M", Yield: 0.35, Duration: 0.25},
		{Maturity: "6M", Yield: 0.45, Duration: 0.49},
		{Maturity: "1Y", Yield: 0.65, Duration: 0.98},
		{Maturity: "2Y", Yield: 0.95, Duration: 1.94},
		{Maturity: "3Y", Yield: 1.25, Duration: 2.87},
		{Maturity: "5Y", Yield: 1.75, Duration: 4.65},
		{Maturity: "7Y", Yield: 2.15, Duration: 6.32},
		{Maturity: "10Y", Yield: 2.45, Duration: 8.87},
		{Maturity: "20Y", Yield: 2.85, Duration: 15.23},
		{Maturity: "30Y", Yield: 3.05, Duration: 19.45},
	}

	return yieldCurve, nil
}

// AssessCreditRisk performs credit risk analysis for a bond
func (s *BondService) AssessCreditRisk(symbol string) (*CreditRiskAssessment, error) {
	s.logger.Debug("Assessing credit risk", zap.String("symbol", symbol))

	asset, err := s.assetService.GetAssetBySymbol(context.Background(), symbol)
	if err != nil {
		return nil, fmt.Errorf("failed to get bond asset: %w", err)
	}

	attributes := map[string]interface{}(asset.Attributes)

	currentRating := s.getStringAttribute(attributes, "credit_rating")
	
	// Calculate risk metrics based on rating
	assessment := &CreditRiskAssessment{
		CurrentRating:    currentRating,
		RatingDate:       time.Now(),
		RatingAgency:     "S&P",
		RatingOutlook:    "Stable",
	}

	// Map rating to risk metrics
	switch currentRating {
	case "AAA":
		assessment.ProbabilityOfDefault = 0.02
		assessment.LossGivenDefault = 0.40
		assessment.CreditSpread = 0.15
	case "AA":
		assessment.ProbabilityOfDefault = 0.05
		assessment.LossGivenDefault = 0.40
		assessment.CreditSpread = 0.25
	case "A":
		assessment.ProbabilityOfDefault = 0.15
		assessment.LossGivenDefault = 0.40
		assessment.CreditSpread = 0.50
	case "BBB":
		assessment.ProbabilityOfDefault = 0.35
		assessment.LossGivenDefault = 0.40
		assessment.CreditSpread = 1.00
	case "BB":
		assessment.ProbabilityOfDefault = 1.20
		assessment.LossGivenDefault = 0.40
		assessment.CreditSpread = 2.50
	case "B":
		assessment.ProbabilityOfDefault = 4.50
		assessment.LossGivenDefault = 0.40
		assessment.CreditSpread = 5.00
	default:
		assessment.ProbabilityOfDefault = 2.00
		assessment.LossGivenDefault = 0.40
		assessment.CreditSpread = 3.00
	}

	return assessment, nil
}

// ProjectCashFlows calculates future cash flows for a bond
func (s *BondService) ProjectCashFlows(symbol string, discountRate float64) ([]CashFlow, error) {
	s.logger.Debug("Projecting cash flows", zap.String("symbol", symbol))

	metrics, err := s.GetBondMetrics(symbol)
	if err != nil {
		return nil, fmt.Errorf("failed to get bond metrics: %w", err)
	}

	var cashFlows []CashFlow
	paymentsPerYear := 2 // Semi-annual
	couponPayment := (metrics.CouponRate / 100.0) * metrics.FaceValue / float64(paymentsPerYear)
	
	// Calculate payment dates
	currentDate := time.Now()
	paymentDate := s.getNextPaymentDate(currentDate, paymentsPerYear)
	
	for paymentDate.Before(metrics.MaturityDate) || paymentDate.Equal(metrics.MaturityDate) {
		var amount float64
		var flowType string
		
		if paymentDate.Equal(metrics.MaturityDate) || 
		   paymentDate.After(metrics.MaturityDate.AddDate(0, 0, -1)) {
			// Final payment includes principal
			amount = couponPayment + metrics.FaceValue
			flowType = "principal"
		} else {
			amount = couponPayment
			flowType = "coupon"
		}
		
		// Calculate present value
		yearsToPayment := paymentDate.Sub(currentDate).Hours() / (24 * 365.25)
		presentValue := amount / math.Pow(1+discountRate/100.0, yearsToPayment)
		
		cashFlows = append(cashFlows, CashFlow{
			Date:         paymentDate,
			Amount:       amount,
			Type:         flowType,
			PresentValue: presentValue,
		})
		
		// Move to next payment date
		paymentDate = paymentDate.AddDate(0, 12/paymentsPerYear, 0)
	}

	return cashFlows, nil
}

// Helper methods

func (s *BondService) calculateBondMetrics(metrics *BondMetrics, currentPrice float64) {
	now := time.Now()
	
	// Calculate time to maturity
	if !metrics.MaturityDate.IsZero() {
		duration := metrics.MaturityDate.Sub(now)
		metrics.DaysToMaturity = int(duration.Hours() / 24)
		metrics.YearsToMaturity = duration.Hours() / (24 * 365.25)
	}
	
	// Calculate current yield
	if currentPrice > 0 && metrics.CouponRate > 0 {
		annualCoupon := (metrics.CouponRate / 100.0) * metrics.FaceValue
		metrics.CurrentYield = (annualCoupon / currentPrice) * 100
	}
	
	// Calculate YTM
	if metrics.YearsToMaturity > 0 && currentPrice > 0 {
		metrics.YieldToMaturity = s.CalculateYieldToMaturity(
			metrics.FaceValue, currentPrice, metrics.CouponRate, 
			metrics.YearsToMaturity, 2)
	}
	
	// Calculate duration
	if metrics.YieldToMaturity > 0 {
		macaulay, modified := s.CalculateDuration(
			metrics.FaceValue, metrics.CouponRate, metrics.YieldToMaturity,
			metrics.YearsToMaturity, 2)
		metrics.Duration = macaulay
		metrics.ModifiedDuration = modified
	}
	
	// Calculate convexity (simplified)
	if metrics.ModifiedDuration > 0 {
		metrics.Convexity = metrics.ModifiedDuration * 1.2 // Approximation
	}
	
	// Calculate accrued interest
	metrics.AccruedInterest = s.calculateAccruedInterest(metrics)
}

func (s *BondService) calculatePresentValue(faceValue, couponRate, ytm, yearsToMaturity float64, paymentsPerYear int) float64 {
	couponPayment := (couponRate / 100.0) * faceValue / float64(paymentsPerYear)
	totalPayments := int(yearsToMaturity * float64(paymentsPerYear))
	
	var pv float64
	
	// Present value of coupon payments
	for i := 1; i <= totalPayments; i++ {
		discountFactor := math.Pow(1+ytm/float64(paymentsPerYear), -float64(i))
		pv += couponPayment * discountFactor
	}
	
	// Present value of principal
	principalPV := faceValue * math.Pow(1+ytm/float64(paymentsPerYear), -float64(totalPayments))
	pv += principalPV
	
	return pv
}

func (s *BondService) calculatePVDerivative(faceValue, couponRate, ytm, yearsToMaturity float64, paymentsPerYear int) float64 {
	couponPayment := (couponRate / 100.0) * faceValue / float64(paymentsPerYear)
	totalPayments := int(yearsToMaturity * float64(paymentsPerYear))
	
	var derivative float64
	
	// Derivative of coupon payments
	for i := 1; i <= totalPayments; i++ {
		period := float64(i)
		discountFactor := math.Pow(1+ytm/float64(paymentsPerYear), -period-1)
		derivative -= couponPayment * period * discountFactor / float64(paymentsPerYear)
	}
	
	// Derivative of principal
	period := float64(totalPayments)
	principalDerivative := -faceValue * period * math.Pow(1+ytm/float64(paymentsPerYear), -period-1) / float64(paymentsPerYear)
	derivative += principalDerivative
	
	return derivative
}

func (s *BondService) calculateAccruedInterest(metrics *BondMetrics) float64 {
	if metrics.CouponRate <= 0 || metrics.FaceValue <= 0 {
		return 0.0
	}
	
	// Simplified calculation - assumes semi-annual payments
	annualCoupon := (metrics.CouponRate / 100.0) * metrics.FaceValue
	semiAnnualCoupon := annualCoupon / 2.0
	
	// Days since last payment (simplified - assumes 30 days ago)
	daysSinceLastPayment := 30.0
	daysInPeriod := 182.5 // Semi-annual period
	
	return semiAnnualCoupon * (daysSinceLastPayment / daysInPeriod)
}

func (s *BondService) getNextPaymentDate(currentDate time.Time, paymentsPerYear int) time.Time {
	// Simplified - assumes payments on 15th of month
	year := currentDate.Year()
	month := currentDate.Month()
	
	if paymentsPerYear == 2 {
		// Semi-annual: June 15 and December 15
		if month <= 6 {
			return time.Date(year, 6, 15, 0, 0, 0, 0, time.UTC)
		} else {
			return time.Date(year, 12, 15, 0, 0, 0, 0, time.UTC)
		}
	}
	
	// Default to next month 15th
	nextMonth := month + 1
	nextYear := year
	if nextMonth > 12 {
		nextMonth = 1
		nextYear++
	}
	
	return time.Date(nextYear, nextMonth, 15, 0, 0, 0, 0, time.UTC)
}

// Helper methods for attribute parsing
func (s *BondService) getStringAttribute(attributes map[string]interface{}, key string) string {
	if val, ok := attributes[key].(string); ok {
		return val
	}
	return ""
}

func (s *BondService) getFloatAttribute(attributes map[string]interface{}, key string) float64 {
	if val, ok := attributes[key].(float64); ok {
		return val
	}
	return 0.0
}
