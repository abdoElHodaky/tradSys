package timeframe

import (
	"errors"
	"math"

	"github.com/markcheno/go-talib"
)

// Common errors
var (
	ErrInsufficientData = errors.New("insufficient data for calculation")
)

// IndicatorType represents the type of indicator
type IndicatorType string

// Indicator types
const (
	IndicatorSMA  IndicatorType = "sma"
	IndicatorEMA  IndicatorType = "ema"
	IndicatorRSI  IndicatorType = "rsi"
	IndicatorMACD IndicatorType = "macd"
	IndicatorBB   IndicatorType = "bollinger_bands"
	IndicatorATR  IndicatorType = "atr"
)

// IndicatorResult represents the result of an indicator calculation
type IndicatorResult struct {
	// Type is the type of indicator
	Type IndicatorType

	// Symbol is the trading symbol
	Symbol string

	// Interval is the timeframe interval
	Interval TimeframeInterval

	// Timestamp is the timestamp of the calculation
	Timestamp int64

	// Values are the indicator values
	Values map[string]float64
}

// IndicatorCalculator calculates technical indicators
type IndicatorCalculator struct {
	// Aggregator is the timeframe aggregator
	aggregator *TimeframeAggregator
}

// NewIndicatorCalculator creates a new IndicatorCalculator
func NewIndicatorCalculator(aggregator *TimeframeAggregator) *IndicatorCalculator {
	return &IndicatorCalculator{
		aggregator: aggregator,
	}
}

// CalculateSMA calculates the Simple Moving Average
func (c *IndicatorCalculator) CalculateSMA(
	symbol string,
	interval TimeframeInterval,
	period int,
) (*IndicatorResult, error) {
	// Get historical candles
	candles := c.aggregator.GetHistoricalCandles(symbol, interval, period+1)
	if len(candles) < period {
		return nil, ErrInsufficientData
	}

	// Extract close prices
	closes := make([]float64, len(candles))
	for i, candle := range candles {
		closes[i] = candle.Close
	}

	// Calculate SMA
	sma := talib.Sma(closes, period)

	// Create result
	result := &IndicatorResult{
		Type:      IndicatorSMA,
		Symbol:    symbol,
		Interval:  interval,
		Timestamp: candles[len(candles)-1].Timestamp.Unix(),
		Values: map[string]float64{
			"sma": sma[len(sma)-1],
		},
	}

	return result, nil
}

// CalculateEMA calculates the Exponential Moving Average
func (c *IndicatorCalculator) CalculateEMA(
	symbol string,
	interval TimeframeInterval,
	period int,
) (*IndicatorResult, error) {
	// Get historical candles
	candles := c.aggregator.GetHistoricalCandles(symbol, interval, period*3) // Need more data for EMA
	if len(candles) < period {
		return nil, ErrInsufficientData
	}

	// Extract close prices
	closes := make([]float64, len(candles))
	for i, candle := range candles {
		closes[i] = candle.Close
	}

	// Calculate EMA
	ema := talib.Ema(closes, period)

	// Create result
	result := &IndicatorResult{
		Type:      IndicatorEMA,
		Symbol:    symbol,
		Interval:  interval,
		Timestamp: candles[len(candles)-1].Timestamp.Unix(),
		Values: map[string]float64{
			"ema": ema[len(ema)-1],
		},
	}

	return result, nil
}

// CalculateRSI calculates the Relative Strength Index
func (c *IndicatorCalculator) CalculateRSI(
	symbol string,
	interval TimeframeInterval,
	period int,
) (*IndicatorResult, error) {
	// Get historical candles
	candles := c.aggregator.GetHistoricalCandles(symbol, interval, period*3) // Need more data for RSI
	if len(candles) < period+1 {
		return nil, ErrInsufficientData
	}

	// Extract close prices
	closes := make([]float64, len(candles))
	for i, candle := range candles {
		closes[i] = candle.Close
	}

	// Calculate RSI
	rsi := talib.Rsi(closes, period)

	// Create result
	result := &IndicatorResult{
		Type:      IndicatorRSI,
		Symbol:    symbol,
		Interval:  interval,
		Timestamp: candles[len(candles)-1].Timestamp.Unix(),
		Values: map[string]float64{
			"rsi": rsi[len(rsi)-1],
		},
	}

	return result, nil
}

// CalculateMACD calculates the Moving Average Convergence Divergence
func (c *IndicatorCalculator) CalculateMACD(
	symbol string,
	interval TimeframeInterval,
	fastPeriod, slowPeriod, signalPeriod int,
) (*IndicatorResult, error) {
	// Get historical candles
	requiredPeriod := slowPeriod + signalPeriod
	candles := c.aggregator.GetHistoricalCandles(symbol, interval, requiredPeriod*3) // Need more data for MACD
	if len(candles) < requiredPeriod {
		return nil, ErrInsufficientData
	}

	// Extract close prices
	closes := make([]float64, len(candles))
	for i, candle := range candles {
		closes[i] = candle.Close
	}

	// Calculate MACD
	macd, signal, hist := talib.Macd(closes, fastPeriod, slowPeriod, signalPeriod)

	// Create result
	result := &IndicatorResult{
		Type:      IndicatorMACD,
		Symbol:    symbol,
		Interval:  interval,
		Timestamp: candles[len(candles)-1].Timestamp.Unix(),
		Values: map[string]float64{
			"macd":   macd[len(macd)-1],
			"signal": signal[len(signal)-1],
			"hist":   hist[len(hist)-1],
		},
	}

	return result, nil
}

// CalculateBollingerBands calculates Bollinger Bands
func (c *IndicatorCalculator) CalculateBollingerBands(
	symbol string,
	interval TimeframeInterval,
	period int,
	devUp, devDown float64,
) (*IndicatorResult, error) {
	// Get historical candles
	candles := c.aggregator.GetHistoricalCandles(symbol, interval, period*2)
	if len(candles) < period {
		return nil, ErrInsufficientData
	}

	// Extract close prices
	closes := make([]float64, len(candles))
	for i, candle := range candles {
		closes[i] = candle.Close
	}

	// Calculate Bollinger Bands
	upper, middle, lower := talib.BBands(closes, period, devUp, devDown, talib.SMA)

	// Create result
	result := &IndicatorResult{
		Type:      IndicatorBB,
		Symbol:    symbol,
		Interval:  interval,
		Timestamp: candles[len(candles)-1].Timestamp.Unix(),
		Values: map[string]float64{
			"upper":  upper[len(upper)-1],
			"middle": middle[len(middle)-1],
			"lower":  lower[len(lower)-1],
		},
	}

	return result, nil
}

// CalculateATR calculates the Average True Range
func (c *IndicatorCalculator) CalculateATR(
	symbol string,
	interval TimeframeInterval,
	period int,
) (*IndicatorResult, error) {
	// Get historical candles
	candles := c.aggregator.GetHistoricalCandles(symbol, interval, period*2)
	if len(candles) < period {
		return nil, ErrInsufficientData
	}

	// Extract high, low, close prices
	highs := make([]float64, len(candles))
	lows := make([]float64, len(candles))
	closes := make([]float64, len(candles))
	for i, candle := range candles {
		highs[i] = candle.High
		lows[i] = candle.Low
		closes[i] = candle.Close
	}

	// Calculate ATR
	atr := talib.Atr(highs, lows, closes, period)

	// Create result
	result := &IndicatorResult{
		Type:      IndicatorATR,
		Symbol:    symbol,
		Interval:  interval,
		Timestamp: candles[len(candles)-1].Timestamp.Unix(),
		Values: map[string]float64{
			"atr": atr[len(atr)-1],
		},
	}

	return result, nil
}

// CalculateZScore calculates the Z-Score
func (c *IndicatorCalculator) CalculateZScore(
	symbol string,
	interval TimeframeInterval,
	period int,
) (*IndicatorResult, error) {
	// Get historical candles
	candles := c.aggregator.GetHistoricalCandles(symbol, interval, period)
	if len(candles) < period {
		return nil, ErrInsufficientData
	}

	// Extract close prices
	closes := make([]float64, len(candles))
	for i, candle := range candles {
		closes[i] = candle.Close
	}

	// Calculate mean
	var sum float64
	for _, price := range closes {
		sum += price
	}
	mean := sum / float64(len(closes))

	// Calculate standard deviation
	var variance float64
	for _, price := range closes {
		variance += math.Pow(price-mean, 2)
	}
	stdDev := math.Sqrt(variance / float64(len(closes)))

	// Calculate Z-Score
	currentPrice := closes[len(closes)-1]
	zScore := (currentPrice - mean) / stdDev

	// Create result
	result := &IndicatorResult{
		Type:      "z_score",
		Symbol:    symbol,
		Interval:  interval,
		Timestamp: candles[len(candles)-1].Timestamp.Unix(),
		Values: map[string]float64{
			"z_score": zScore,
			"mean":    mean,
			"std_dev": stdDev,
		},
	}

	return result, nil
}

// CalculateMultipleTimeframeIndicator calculates an indicator across multiple timeframes
func (c *IndicatorCalculator) CalculateMultipleTimeframeIndicator(
	symbol string,
	intervals []TimeframeInterval,
	indicatorType IndicatorType,
	params map[string]interface{},
) (map[TimeframeInterval]*IndicatorResult, error) {
	results := make(map[TimeframeInterval]*IndicatorResult)

	for _, interval := range intervals {
		var result *IndicatorResult
		var err error

		switch indicatorType {
		case IndicatorSMA:
			period := params["period"].(int)
			result, err = c.CalculateSMA(symbol, interval, period)
		case IndicatorEMA:
			period := params["period"].(int)
			result, err = c.CalculateEMA(symbol, interval, period)
		case IndicatorRSI:
			period := params["period"].(int)
			result, err = c.CalculateRSI(symbol, interval, period)
		case IndicatorMACD:
			fastPeriod := params["fast_period"].(int)
			slowPeriod := params["slow_period"].(int)
			signalPeriod := params["signal_period"].(int)
			result, err = c.CalculateMACD(symbol, interval, fastPeriod, slowPeriod, signalPeriod)
		case IndicatorBB:
			period := params["period"].(int)
			devUp := params["dev_up"].(float64)
			devDown := params["dev_down"].(float64)
			result, err = c.CalculateBollingerBands(symbol, interval, period, devUp, devDown)
		case IndicatorATR:
			period := params["period"].(int)
			result, err = c.CalculateATR(symbol, interval, period)
		default:
			err = errors.New("unsupported indicator type")
		}

		if err != nil {
			// Skip this interval if there's an error
			continue
		}

		results[interval] = result
	}

	if len(results) == 0 {
		return nil, ErrInsufficientData
	}

	return results, nil
}

