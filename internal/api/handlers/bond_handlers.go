package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/services"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// BondHandlers handles bond-specific API endpoints
type BondHandlers struct {
	bondService *services.BondService
	logger      *zap.Logger
}

// NewBondHandlers creates a new bond handlers instance
func NewBondHandlers(bondService *services.BondService, logger *zap.Logger) *BondHandlers {
	return &BondHandlers{
		bondService: bondService,
		logger:      logger,
	}
}

// CreateBondRequest represents the request body for creating a bond
type CreateBondRequest struct {
	Symbol       string    `json:"symbol" binding:"required"`
	Issuer       string    `json:"issuer" binding:"required"`
	FaceValue    float64   `json:"face_value" binding:"required"`
	CouponRate   float64   `json:"coupon_rate" binding:"required"`
	MaturityDate time.Time `json:"maturity_date" binding:"required"`
	CreditRating string    `json:"credit_rating" binding:"required"`
}

// UpdateCreditRatingRequest represents the request body for updating credit rating
type UpdateCreditRatingRequest struct {
	NewRating     string `json:"new_rating" binding:"required"`
	RatingAgency  string `json:"rating_agency"`
	RatingOutlook string `json:"rating_outlook"`
	EffectiveDate time.Time `json:"effective_date"`
}

// CreateBond creates a new bond
// @Summary Create bond
// @Description Create a new bond with initial metadata
// @Tags Bond
// @Accept json
// @Produce json
// @Param request body CreateBondRequest true "Bond creation request"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/bonds [post]
func (h *BondHandlers) CreateBond(c *gin.Context) {
	var req CreateBondRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid bond creation request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.bondService.CreateBond(req.Symbol, req.Issuer, req.FaceValue, 
		req.CouponRate, req.MaturityDate, req.CreditRating)
	if err != nil {
		h.logger.Error("Failed to create bond", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.logger.Info("Bond created successfully", zap.String("symbol", req.Symbol))
	c.JSON(http.StatusCreated, gin.H{
		"message": "Bond created successfully",
		"symbol":  req.Symbol,
	})
}

// GetBondMetrics retrieves comprehensive bond metrics
// @Summary Get bond metrics
// @Description Get comprehensive metrics for a bond
// @Tags Bond
// @Produce json
// @Param symbol path string true "Bond symbol"
// @Success 200 {object} services.BondMetrics
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/bonds/{symbol}/metrics [get]
func (h *BondHandlers) GetBondMetrics(c *gin.Context) {
	symbol := c.Param("symbol")
	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Symbol is required"})
		return
	}

	metrics, err := h.bondService.GetBondMetrics(symbol)
	if err != nil {
		h.logger.Error("Failed to get bond metrics", zap.String("symbol", symbol), zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, metrics)
}

// GetYieldCurve retrieves yield curve data
// @Summary Get yield curve
// @Description Get yield curve data for bond analysis
// @Tags Bond
// @Produce json
// @Param curve_type query string false "Yield curve type" default("treasury")
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/bonds/yield-curve [get]
func (h *BondHandlers) GetYieldCurve(c *gin.Context) {
	curveType := c.DefaultQuery("curve_type", "treasury")

	yieldCurve, err := h.bondService.GetYieldCurve(curveType)
	if err != nil {
		h.logger.Error("Failed to get yield curve", zap.String("curve_type", curveType), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"curve_type":   curveType,
		"yield_curve":  yieldCurve,
		"retrieved_at": time.Now(),
	})
}

// AssessCreditRisk performs credit risk analysis for a bond
// @Summary Assess credit risk
// @Description Perform credit risk analysis for a bond
// @Tags Bond
// @Produce json
// @Param symbol path string true "Bond symbol"
// @Success 200 {object} services.CreditRiskAssessment
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/bonds/{symbol}/credit-risk [get]
func (h *BondHandlers) AssessCreditRisk(c *gin.Context) {
	symbol := c.Param("symbol")
	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Symbol is required"})
		return
	}

	assessment, err := h.bondService.AssessCreditRisk(symbol)
	if err != nil {
		h.logger.Error("Failed to assess credit risk", zap.String("symbol", symbol), zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, assessment)
}

// GetDurationConvexity calculates duration and convexity for a bond
// @Summary Get duration and convexity
// @Description Calculate duration and convexity metrics for a bond
// @Tags Bond
// @Produce json
// @Param symbol path string true "Bond symbol"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/bonds/{symbol}/duration [get]
func (h *BondHandlers) GetDurationConvexity(c *gin.Context) {
	symbol := c.Param("symbol")
	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Symbol is required"})
		return
	}

	metrics, err := h.bondService.GetBondMetrics(symbol)
	if err != nil {
		h.logger.Error("Failed to get bond metrics for duration", zap.String("symbol", symbol), zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"symbol":            symbol,
		"duration":          metrics.Duration,
		"modified_duration": metrics.ModifiedDuration,
		"convexity":         metrics.Convexity,
		"price_volatility":  metrics.PriceVolatility,
		"calculated_at":     time.Now(),
	})
}

// ProjectCashFlows calculates future cash flows for a bond
// @Summary Project cash flows
// @Description Calculate future cash flows for a bond
// @Tags Bond
// @Produce json
// @Param symbol path string true "Bond symbol"
// @Param discount_rate query number false "Discount rate for present value calculation" default(3.0)
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/bonds/{symbol}/cash-flows [get]
func (h *BondHandlers) ProjectCashFlows(c *gin.Context) {
	symbol := c.Param("symbol")
	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Symbol is required"})
		return
	}

	discountRateStr := c.DefaultQuery("discount_rate", "3.0")
	discountRate, err := strconv.ParseFloat(discountRateStr, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid discount rate"})
		return
	}

	cashFlows, err := h.bondService.ProjectCashFlows(symbol, discountRate)
	if err != nil {
		h.logger.Error("Failed to project cash flows", zap.String("symbol", symbol), zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	// Calculate total present value
	var totalPV float64
	for _, cf := range cashFlows {
		totalPV += cf.PresentValue
	}

	c.JSON(http.StatusOK, gin.H{
		"symbol":              symbol,
		"discount_rate":       discountRate,
		"cash_flows":          cashFlows,
		"total_present_value": totalPV,
		"cash_flows_count":    len(cashFlows),
		"projected_at":        time.Now(),
	})
}

// UpdateCreditRating updates the credit rating for a bond
// @Summary Update credit rating
// @Description Update credit rating for a bond
// @Tags Bond
// @Accept json
// @Produce json
// @Param symbol path string true "Bond symbol"
// @Param request body UpdateCreditRatingRequest true "Credit rating update request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/bonds/{symbol}/rating-change [post]
func (h *BondHandlers) UpdateCreditRating(c *gin.Context) {
	symbol := c.Param("symbol")
	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Symbol is required"})
		return
	}

	var req UpdateCreditRatingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid credit rating update request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// In a real implementation, this would update the bond's credit rating
	h.logger.Info("Credit rating update processed", 
		zap.String("symbol", symbol),
		zap.String("new_rating", req.NewRating))

	c.JSON(http.StatusOK, gin.H{
		"message":        "Credit rating updated successfully",
		"symbol":         symbol,
		"new_rating":     req.NewRating,
		"rating_agency":  req.RatingAgency,
		"rating_outlook": req.RatingOutlook,
		"effective_date": req.EffectiveDate,
		"updated_at":     time.Now(),
	})
}

// CalculateYTM calculates yield to maturity for a bond
// @Summary Calculate YTM
// @Description Calculate yield to maturity for a bond
// @Tags Bond
// @Produce json
// @Param face_value query number true "Face value of the bond"
// @Param current_price query number true "Current market price"
// @Param coupon_rate query number true "Annual coupon rate (percentage)"
// @Param years_to_maturity query number true "Years to maturity"
// @Param payments_per_year query int false "Payments per year" default(2)
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/bonds/calculate-ytm [get]
func (h *BondHandlers) CalculateYTM(c *gin.Context) {
	// Parse parameters
	faceValueStr := c.Query("face_value")
	if faceValueStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Face value is required"})
		return
	}
	faceValue, err := strconv.ParseFloat(faceValueStr, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid face value"})
		return
	}

	currentPriceStr := c.Query("current_price")
	if currentPriceStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Current price is required"})
		return
	}
	currentPrice, err := strconv.ParseFloat(currentPriceStr, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid current price"})
		return
	}

	couponRateStr := c.Query("coupon_rate")
	if couponRateStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Coupon rate is required"})
		return
	}
	couponRate, err := strconv.ParseFloat(couponRateStr, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid coupon rate"})
		return
	}

	yearsToMaturityStr := c.Query("years_to_maturity")
	if yearsToMaturityStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Years to maturity is required"})
		return
	}
	yearsToMaturity, err := strconv.ParseFloat(yearsToMaturityStr, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid years to maturity"})
		return
	}

	paymentsPerYearStr := c.DefaultQuery("payments_per_year", "2")
	paymentsPerYear, err := strconv.Atoi(paymentsPerYearStr)
	if err != nil || paymentsPerYear <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payments per year"})
		return
	}

	// Calculate YTM
	ytm := h.bondService.CalculateYieldToMaturity(faceValue, currentPrice, couponRate, yearsToMaturity, paymentsPerYear)

	// Calculate current yield for comparison
	currentYield := (couponRate * faceValue / 100.0) / currentPrice * 100.0

	c.JSON(http.StatusOK, gin.H{
		"face_value":        faceValue,
		"current_price":     currentPrice,
		"coupon_rate":       couponRate,
		"years_to_maturity": yearsToMaturity,
		"payments_per_year": paymentsPerYear,
		"yield_to_maturity": ytm,
		"current_yield":     currentYield,
		"calculated_at":     time.Now(),
	})
}

// CalculateDuration calculates duration for a bond
// @Summary Calculate duration
// @Description Calculate Macaulay and modified duration for a bond
// @Tags Bond
// @Produce json
// @Param face_value query number true "Face value of the bond"
// @Param coupon_rate query number true "Annual coupon rate (percentage)"
// @Param ytm query number true "Yield to maturity (percentage)"
// @Param years_to_maturity query number true "Years to maturity"
// @Param payments_per_year query int false "Payments per year" default(2)
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/bonds/calculate-duration [get]
func (h *BondHandlers) CalculateDuration(c *gin.Context) {
	// Parse parameters
	faceValueStr := c.Query("face_value")
	if faceValueStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Face value is required"})
		return
	}
	faceValue, err := strconv.ParseFloat(faceValueStr, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid face value"})
		return
	}

	couponRateStr := c.Query("coupon_rate")
	if couponRateStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Coupon rate is required"})
		return
	}
	couponRate, err := strconv.ParseFloat(couponRateStr, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid coupon rate"})
		return
	}

	ytmStr := c.Query("ytm")
	if ytmStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "YTM is required"})
		return
	}
	ytm, err := strconv.ParseFloat(ytmStr, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid YTM"})
		return
	}

	yearsToMaturityStr := c.Query("years_to_maturity")
	if yearsToMaturityStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Years to maturity is required"})
		return
	}
	yearsToMaturity, err := strconv.ParseFloat(yearsToMaturityStr, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid years to maturity"})
		return
	}

	paymentsPerYearStr := c.DefaultQuery("payments_per_year", "2")
	paymentsPerYear, err := strconv.Atoi(paymentsPerYearStr)
	if err != nil || paymentsPerYear <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payments per year"})
		return
	}

	// Calculate duration
	macaulayDuration, modifiedDuration := h.bondService.CalculateDuration(
		faceValue, couponRate, ytm, yearsToMaturity, paymentsPerYear)

	c.JSON(http.StatusOK, gin.H{
		"face_value":         faceValue,
		"coupon_rate":        couponRate,
		"ytm":                ytm,
		"years_to_maturity":  yearsToMaturity,
		"payments_per_year":  paymentsPerYear,
		"macaulay_duration":  macaulayDuration,
		"modified_duration":  modifiedDuration,
		"calculated_at":      time.Now(),
	})
}

// GetBondsByRating retrieves bonds by credit rating
// @Summary Get bonds by rating
// @Description Get bonds filtered by credit rating
// @Tags Bond
// @Produce json
// @Param rating query string true "Credit rating (e.g., AAA, AA, A, BBB)"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/bonds/by-rating [get]
func (h *BondHandlers) GetBondsByRating(c *gin.Context) {
	rating := c.Query("rating")
	if rating == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Credit rating is required"})
		return
	}

	// In a real implementation, this would query the database for bonds with the specified rating
	h.logger.Info("Retrieving bonds by rating", zap.String("rating", rating))

	// Mock response
	c.JSON(http.StatusOK, gin.H{
		"rating":       rating,
		"bonds":        []string{}, // Would contain actual bond data
		"count":        0,
		"retrieved_at": time.Now(),
	})
}

// GetMaturitySchedule retrieves bonds by maturity date range
// @Summary Get maturity schedule
// @Description Get bonds maturing within specified date range
// @Tags Bond
// @Produce json
// @Param start_date query string true "Start date (YYYY-MM-DD)"
// @Param end_date query string true "End date (YYYY-MM-DD)"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/bonds/maturity-schedule [get]
func (h *BondHandlers) GetMaturitySchedule(c *gin.Context) {
	startDateStr := c.Query("start_date")
	if startDateStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Start date is required"})
		return
	}

	endDateStr := c.Query("end_date")
	if endDateStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "End date is required"})
		return
	}

	startDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid start date format (use YYYY-MM-DD)"})
		return
	}

	endDate, err := time.Parse("2006-01-02", endDateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid end date format (use YYYY-MM-DD)"})
		return
	}

	if endDate.Before(startDate) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "End date must be after start date"})
		return
	}

	// In a real implementation, this would query bonds maturing in the date range
	h.logger.Info("Retrieving maturity schedule", 
		zap.String("start_date", startDateStr),
		zap.String("end_date", endDateStr))

	// Mock response
	c.JSON(http.StatusOK, gin.H{
		"start_date":   startDate,
		"end_date":     endDate,
		"bonds":        []string{}, // Would contain actual bond data
		"count":        0,
		"retrieved_at": time.Now(),
	})
}
