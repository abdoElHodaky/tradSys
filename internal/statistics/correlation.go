package statistics

import (
	"errors"
	"math"
)

// CalculateCorrelation calculates Pearson correlation coefficient
func CalculateCorrelation(x, y []float64) (float64, error) {
	if len(x) != len(y) || len(x) < 2 {
		return 0, errors.New("input slices must have same length and at least 2 elements")
	}
	
	// Calculate means
	var sumX, sumY float64
	for i := 0; i < len(x); i++ {
		sumX += x[i]
		sumY += y[i]
	}
	meanX := sumX / float64(len(x))
	meanY := sumY / float64(len(y))
	
	// Calculate correlation coefficient
	var numerator, denominatorX, denominatorY float64
	for i := 0; i < len(x); i++ {
		xDiff := x[i] - meanX
		yDiff := y[i] - meanY
		numerator += xDiff * yDiff
		denominatorX += xDiff * xDiff
		denominatorY += yDiff * yDiff
	}
	
	// Check for division by zero
	if denominatorX == 0 || denominatorY == 0 {
		return 0, errors.New("standard deviation is zero")
	}
	
	return numerator / math.Sqrt(denominatorX*denominatorY), nil
}

// CalculateZScore calculates the z-score of the current spread
func CalculateZScore(spread, mean, stdDev float64) float64 {
	if stdDev == 0 {
		return 0
	}
	return (spread - mean) / stdDev
}

// CalculateSpread calculates the spread between two price series
func CalculateSpread(prices1, prices2 []float64, ratio float64) ([]float64, error) {
	if len(prices1) != len(prices2) {
		return nil, errors.New("price series must have the same length")
	}
	
	spread := make([]float64, len(prices1))
	for i := 0; i < len(prices1); i++ {
		spread[i] = prices1[i] - (ratio * prices2[i])
	}
	return spread, nil
}

// CalculateMean calculates the mean of a slice of float64
func CalculateMean(data []float64) (float64, error) {
	if len(data) == 0 {
		return 0, errors.New("empty data slice")
	}
	
	var sum float64
	for _, v := range data {
		sum += v
	}
	return sum / float64(len(data)), nil
}

// CalculateStdDev calculates the standard deviation of a slice of float64
func CalculateStdDev(data []float64, mean float64) (float64, error) {
	if len(data) < 2 {
		return 0, errors.New("need at least two data points")
	}
	
	var sumSquaredDiff float64
	for _, v := range data {
		diff := v - mean
		sumSquaredDiff += diff * diff
	}
	
	variance := sumSquaredDiff / float64(len(data)-1)
	return math.Sqrt(variance), nil
}

// EstimateHalfLife estimates the half-life of mean reversion using Ornstein-Uhlenbeck process
func EstimateHalfLife(spread []float64) (int, error) {
	if len(spread) < 3 {
		return 0, errors.New("need at least three data points")
	}
	
	// Calculate lagged spread and differences
	y := make([]float64, len(spread)-1)
	x := make([]float64, len(spread)-1)
	
	for i := 0; i < len(spread)-1; i++ {
		y[i] = spread[i+1] - spread[i]
		x[i] = spread[i]
	}
	
	// Perform linear regression to estimate lambda
	var sumX, sumY, sumXY, sumXX float64
	for i := 0; i < len(x); i++ {
		sumX += x[i]
		sumY += y[i]
		sumXY += x[i] * y[i]
		sumXX += x[i] * x[i]
	}
	
	n := float64(len(x))
	lambda := (n*sumXY - sumX*sumY) / (n*sumXX - sumX*sumX)
	
	// Calculate half-life
	if lambda >= 0 {
		return 0, errors.New("process is not mean-reverting")
	}
	
	halfLife := math.Log(2) / math.Abs(lambda)
	return int(math.Round(halfLife)), nil
}
