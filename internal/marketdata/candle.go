package marketdata

import (
	"time"
)

// Candle represents a candlestick in market data
type Candle struct {
	Symbol    string    `json:"symbol"`
	Open      float64   `json:"open"`
	High      float64   `json:"high"`
	Low       float64   `json:"low"`
	Close     float64   `json:"close"`
	Volume    float64   `json:"volume"`
	Timestamp time.Time `json:"timestamp"`
	Interval  string    `json:"interval"`
}

// NewCandle creates a new candle
func NewCandle(symbol string, open, high, low, close, volume float64, timestamp time.Time, interval string) *Candle {
	return &Candle{
		Symbol:    symbol,
		Open:      open,
		High:      high,
		Low:       low,
		Close:     close,
		Volume:    volume,
		Timestamp: timestamp,
		Interval:  interval,
	}
}

