package statistics

import (
	"errors"
	"math"
)

// ADF critical values at 5% significance level
var adfCriticalValues = map[string]float64{
	"1%":  -3.43,
	"5%":  -2.86,
	"10%": -2.57,
}

// EngleGrangerTest performs the Engle-Granger cointegration test
// Returns: test statistic, whether the series are cointegrated, error
func EngleGrangerTest(x, y []float64) (float64, bool, error) {
	if len(x) != len(y) || len(x) < 10 {
		return 0, false, errors.New("input slices must have same length and at least 10 elements")
	}

	// Step 1: Perform linear regression y = β*x + c
	beta, alpha, err := linearRegression(x, y)
	if err != nil {
		return 0, false, err
	}

	// Step 2: Calculate residuals
	residuals := make([]float64, len(x))
	for i := 0; i < len(x); i++ {
		residuals[i] = y[i] - (beta*x[i] + alpha)
	}

	// Step 3: Perform Augmented Dickey-Fuller test on residuals
	adfStat, err := augmentedDickeyFuller(residuals, 1) // lag=1
	if err != nil {
		return 0, false, err
	}

	// Step 4: Compare test statistic with critical values
	isCointegrated := adfStat < adfCriticalValues["5%"]

	return adfStat, isCointegrated, nil
}

// linearRegression performs simple linear regression
// Returns: slope (beta), intercept (alpha), error
func linearRegression(x, y []float64) (float64, float64, error) {
	if len(x) != len(y) || len(x) < 2 {
		return 0, 0, errors.New("input slices must have same length and at least 2 elements")
	}

	// Calculate means
	var sumX, sumY float64
	for i := 0; i < len(x); i++ {
		sumX += x[i]
		sumY += y[i]
	}
	meanX := sumX / float64(len(x))
	meanY := sumY / float64(len(y))

	// Calculate beta (slope)
	var numerator, denominator float64
	for i := 0; i < len(x); i++ {
		xDiff := x[i] - meanX
		yDiff := y[i] - meanY
		numerator += xDiff * yDiff
		denominator += xDiff * xDiff
	}

	if denominator == 0 {
		return 0, 0, errors.New("division by zero in regression")
	}

	beta := numerator / denominator

	// Calculate alpha (intercept)
	alpha := meanY - beta*meanX

	return beta, alpha, nil
}

// augmentedDickeyFuller performs the Augmented Dickey-Fuller test
// Returns: test statistic, error
func augmentedDickeyFuller(series []float64, lag int) (float64, error) {
	if len(series) < lag+3 {
		return 0, errors.New("series too short for specified lag")
	}

	// Create lagged series and differences
	n := len(series) - lag - 1
	y := make([]float64, n)         // Δy_t
	x1 := make([]float64, n)        // y_{t-1}
	xDiff := make([][]float64, lag) // Δy_{t-i}

	for i := 0; i < lag; i++ {
		xDiff[i] = make([]float64, n)
	}

	for t := 0; t < n; t++ {
		y[t] = series[t+lag+1] - series[t+lag]
		x1[t] = series[t+lag]

		for i := 0; i < lag; i++ {
			xDiff[i][t] = series[t+lag-i] - series[t+lag-i-1]
		}
	}

	// Perform regression: Δy_t = ρ*y_{t-1} + β_1*Δy_{t-1} + ... + β_p*Δy_{t-p} + ε_t
	// We're primarily interested in ρ (rho)

	// Create design matrix X
	X := make([][]float64, n)
	for i := 0; i < n; i++ {
		X[i] = make([]float64, lag+1)
		X[i][0] = x1[i] // y_{t-1}

		for j := 0; j < lag; j++ {
			X[i][j+1] = xDiff[j][i] // Δy_{t-j-1}
		}
	}

	// Perform OLS regression
	beta, err := multipleRegression(X, y)
	if err != nil {
		return 0, err
	}

	// Calculate standard error of rho
	residuals := make([]float64, n)
	for i := 0; i < n; i++ {
		predicted := 0.0
		for j := 0; j < len(beta); j++ {
			predicted += beta[j] * X[i][j]
		}
		residuals[i] = y[i] - predicted
	}

	// Calculate residual variance
	var sumSquaredResiduals float64
	for _, r := range residuals {
		sumSquaredResiduals += r * r
	}
	residualVariance := sumSquaredResiduals / float64(n-lag-1)

	// Calculate X'X matrix
	XtX := make([][]float64, lag+1)
	for i := 0; i < lag+1; i++ {
		XtX[i] = make([]float64, lag+1)
		for j := 0; j < lag+1; j++ {
			for k := 0; k < n; k++ {
				XtX[i][j] += X[k][i] * X[k][j]
			}
		}
	}

	// Calculate inverse of X'X (simplified approach)
	// For a proper implementation, use a linear algebra library
	// This is a simplification for the first element only
	var sumX1Squared float64
	for _, val := range x1 {
		sumX1Squared += val * val
	}

	// Standard error of rho (beta[0])
	seRho := math.Sqrt(residualVariance / sumX1Squared)

	// Calculate test statistic
	tStat := beta[0] / seRho

	return tStat, nil
}

// multipleRegression performs multiple linear regression
// X is a matrix where each row is an observation and each column is a variable
// y is the dependent variable
// Returns: coefficients, error
func multipleRegression(X [][]float64, y []float64) ([]float64, error) {
	if len(X) != len(y) {
		return nil, errors.New("X and y must have the same number of observations")
	}

	n := len(X)    // Number of observations
	p := len(X[0]) // Number of variables

	// Calculate X'X
	XtX := make([][]float64, p)
	for i := 0; i < p; i++ {
		XtX[i] = make([]float64, p)
		for j := 0; j < p; j++ {
			for k := 0; k < n; k++ {
				XtX[i][j] += X[k][i] * X[k][j]
			}
		}
	}

	// Calculate X'y
	Xty := make([]float64, p)
	for i := 0; i < p; i++ {
		for k := 0; k < n; k++ {
			Xty[i] += X[k][i] * y[k]
		}
	}

	// Solve (X'X)β = X'y for β
	// This is a simplified approach using Gaussian elimination
	// For a proper implementation, use a linear algebra library

	// Augment XtX with Xty
	aug := make([][]float64, p)
	for i := 0; i < p; i++ {
		aug[i] = make([]float64, p+1)
		for j := 0; j < p; j++ {
			aug[i][j] = XtX[i][j]
		}
		aug[i][p] = Xty[i]
	}

	// Gaussian elimination
	for i := 0; i < p-1; i++ {
		// Find pivot
		maxRow := i
		for k := i + 1; k < p; k++ {
			if math.Abs(aug[k][i]) > math.Abs(aug[maxRow][i]) {
				maxRow = k
			}
		}

		// Swap rows
		aug[i], aug[maxRow] = aug[maxRow], aug[i]

		// Eliminate
		for k := i + 1; k < p; k++ {
			factor := aug[k][i] / aug[i][i]
			for j := i; j <= p; j++ {
				aug[k][j] -= factor * aug[i][j]
			}
		}
	}

	// Back substitution
	beta := make([]float64, p)
	for i := p - 1; i >= 0; i-- {
		beta[i] = aug[i][p]
		for j := i + 1; j < p; j++ {
			beta[i] -= aug[i][j] * beta[j]
		}
		beta[i] /= aug[i][i]
	}

	return beta, nil
}

// CalculateOptimalHedgeRatio calculates the optimal hedge ratio between two price series
func CalculateOptimalHedgeRatio(x, y []float64) (float64, error) {
	beta, _, err := linearRegression(x, y)
	if err != nil {
		return 0, err
	}
	return beta, nil
}
