package indicators

import (
	"github.com/abdoElHodaky/tradSys/internal/marketdata"
)

// IndicatorParams contains parameters for a technical indicator
type IndicatorParams struct {
	// Period is the period for the indicator
	Period int
	
	// Source is the source for the indicator (e.g., "close", "open", "high", "low")
	Source string
	
	// Alpha is the alpha parameter for some indicators
	Alpha float64
	
	// CustomParams is a map of custom parameters
	CustomParams map[string]interface{}
}

// DefaultIndicatorParams returns the default indicator parameters
func DefaultIndicatorParams() IndicatorParams {
	return IndicatorParams{
		Period: 14,
		Source: "close",
		Alpha:  0.2,
		CustomParams: make(map[string]interface{}),
	}
}

// Indicator defines the interface for a technical indicator
type Indicator interface {
	// Calculate calculates the indicator values for a series of candles
	Calculate(candles []*marketdata.Candle) ([]float64, error)
	
	// GetName returns the name of the indicator
	GetName() string
	
	// GetParams returns the parameters of the indicator
	GetParams() IndicatorParams
	
	// GetDescription returns a description of the indicator
	GetDescription() string
}

// BaseIndicator provides a base implementation for indicators
type BaseIndicator struct {
	name        string
	params      IndicatorParams
	description string
}

// NewBaseIndicator creates a new base indicator
func NewBaseIndicator(name string, params IndicatorParams, description string) *BaseIndicator {
	return &BaseIndicator{
		name:        name,
		params:      params,
		description: description,
	}
}

// GetName returns the name of the indicator
func (i *BaseIndicator) GetName() string {
	return i.name
}

// GetParams returns the parameters of the indicator
func (i *BaseIndicator) GetParams() IndicatorParams {
	return i.params
}

// GetDescription returns a description of the indicator
func (i *BaseIndicator) GetDescription() string {
	return i.description
}

// GetSourceValues extracts the source values from candles
func (i *BaseIndicator) GetSourceValues(candles []*marketdata.Candle) []float64 {
	values := make([]float64, len(candles))
	
	for j, candle := range candles {
		switch i.params.Source {
		case "open":
			values[j] = candle.Open
		case "high":
			values[j] = candle.High
		case "low":
			values[j] = candle.Low
		case "close":
			values[j] = candle.Close
		case "volume":
			values[j] = candle.Volume
		case "hl2":
			values[j] = (candle.High + candle.Low) / 2
		case "hlc3":
			values[j] = (candle.High + candle.Low + candle.Close) / 3
		case "ohlc4":
			values[j] = (candle.Open + candle.High + candle.Low + candle.Close) / 4
		default:
			values[j] = candle.Close
		}
	}
	
	return values
}

