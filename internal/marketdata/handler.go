package marketdata

import (
	"context"

	"github.com/abdoElHodaky/tradSys/proto/marketdata"
	"go.uber.org/zap"
)

// Handler implements the MarketDataService interface with stub methods
type Handler struct {
	logger *zap.Logger
}

// NewHandler creates a new market data handler
func NewHandler(logger *zap.Logger) *Handler {
	return &Handler{
		logger: logger,
	}
}

// GetMarketData implements the MarketDataService.GetMarketData method
func (h *Handler) GetMarketData(ctx context.Context, req *marketdata.MarketDataRequest) (*marketdata.MarketDataResponse, error) {
	h.logger.Info("GetMarketData called", zap.String("symbol", req.Symbol))

	return &marketdata.MarketDataResponse{
		Symbol:    req.Symbol,
		Price:     50000.0,
		Volume:    1000.0,
		High:      51000.0,
		Low:       49000.0,
		Open:      49500.0,
		Close:     50000.0,
		Timestamp: 1625097600000,
	}, nil
}

// StreamMarketData implements the MarketDataService.StreamMarketData method
func (h *Handler) StreamMarketData(req *marketdata.MarketDataRequest, stream marketdata.MarketDataService_StreamMarketDataServer) error {
	h.logger.Info("StreamMarketData called", zap.String("symbol", req.Symbol))

	// Send a sample market data update
	data := &marketdata.MarketDataResponse{
		Symbol:    req.Symbol,
		Price:     50000.0,
		Volume:    1000.0,
		High:      51000.0,
		Low:       49000.0,
		Open:      49500.0,
		Close:     50000.0,
		Timestamp: 1625097600000,
	}

	return stream.Send(data)
}

// GetHistoricalData implements the MarketDataService.GetHistoricalData method
func (h *Handler) GetHistoricalData(ctx context.Context, req *marketdata.HistoricalDataRequest) (*marketdata.HistoricalDataResponse, error) {
	h.logger.Info("GetHistoricalData called", 
		zap.String("symbol", req.Symbol),
		zap.String("interval", req.Interval))

	return &marketdata.HistoricalDataResponse{
		Symbol: req.Symbol,
		Interval: req.Interval,
		Data: []*marketdata.MarketDataResponse{
			{
				Symbol:    req.Symbol,
				Price:     50000.0,
				Volume:    1000.0,
				High:      51000.0,
				Low:       49000.0,
				Open:      49500.0,
				Close:     50000.0,
				Timestamp: 1625097600000,
			},
		},
	}, nil
}

// GetSymbols implements the MarketDataService.GetSymbols method
func (h *Handler) GetSymbols(ctx context.Context, req *marketdata.SymbolsRequest) (*marketdata.SymbolsResponse, error) {
	h.logger.Info("GetSymbols called")

	return &marketdata.SymbolsResponse{
		Symbols: []*marketdata.Symbol{
			{
				Name:          "BTC-USD",
				BaseCurrency:  "BTC",
				QuoteCurrency: "USD",
			},
			{
				Name:          "ETH-USD",
				BaseCurrency:  "ETH",
				QuoteCurrency: "USD",
			},
			{
				Name:          "ADA-USD",
				BaseCurrency:  "ADA",
				QuoteCurrency: "USD",
			},
		},
	}, nil
}
