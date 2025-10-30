package external

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// NewBinanceProvider creates a new Binance market data provider
func NewBinanceProvider(apiKey, apiSecret string, logger *zap.Logger) *BinanceProvider {
	ctx, cancel := context.WithCancel(context.Background())

	return &BinanceProvider{
		BaseURL:              "https://api.binance.com",
		WebSocketURL:         "wss://stream.binance.com:9443/ws",
		APIKey:               apiKey,
		APISecret:            apiSecret,
		HTTPClient:           &http.Client{Timeout: 30 * time.Second},
		WebSocketConnections: make(map[string]*websocket.Conn),
		Callbacks:            make(map[string]MarketDataCallback),
		logger:               logger,
		ctx:                  ctx,
		cancel:               cancel,
	}
}

// NewBinanceProviderWithConfig creates a new Binance provider with custom configuration
func NewBinanceProviderWithConfig(config *BinanceConfig, logger *zap.Logger) *BinanceProvider {
	ctx, cancel := context.WithCancel(context.Background())

	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "https://api.binance.com"
		if config.EnableTestnet {
			baseURL = "https://testnet.binance.vision"
		}
	}

	wsURL := config.WebSocketURL
	if wsURL == "" {
		wsURL = "wss://stream.binance.com:9443/ws"
		if config.EnableTestnet {
			wsURL = "wss://testnet.binance.vision/ws"
		}
	}

	timeout := config.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &BinanceProvider{
		BaseURL:              baseURL,
		WebSocketURL:         wsURL,
		APIKey:               config.APIKey,
		APISecret:            config.APISecret,
		HTTPClient:           &http.Client{Timeout: timeout},
		WebSocketConnections: make(map[string]*websocket.Conn),
		Callbacks:            make(map[string]MarketDataCallback),
		logger:               logger,
		ctx:                  ctx,
		cancel:               cancel,
	}
}

// GetExchangeInfo retrieves exchange information
func (p *BinanceProvider) GetExchangeInfo(ctx context.Context) (*BinanceExchangeInfo, error) {
	url := p.BaseURL + ExchangeInfoEndpoint
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := p.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var binanceErr BinanceError
		if err := json.Unmarshal(body, &binanceErr); err == nil {
			return nil, binanceErr
		}
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var exchangeInfo BinanceExchangeInfo
	if err := json.Unmarshal(body, &exchangeInfo); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &exchangeInfo, nil
}

// GetTicker retrieves 24hr ticker price change statistics
func (p *BinanceProvider) GetTicker(ctx context.Context, symbol string) (*BinanceTicker, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}

	url := p.BaseURL + TickerEndpoint
	if len(params) > 0 {
		url += "?" + params.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := p.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var binanceErr BinanceError
		if err := json.Unmarshal(body, &binanceErr); err == nil {
			return nil, binanceErr
		}
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Handle both single ticker and array response
	if symbol != "" {
		var ticker BinanceTicker
		if err := json.Unmarshal(body, &ticker); err != nil {
			return nil, fmt.Errorf("failed to unmarshal ticker response: %w", err)
		}
		return &ticker, nil
	} else {
		var tickers []BinanceTicker
		if err := json.Unmarshal(body, &tickers); err != nil {
			return nil, fmt.Errorf("failed to unmarshal tickers response: %w", err)
		}
		if len(tickers) > 0 {
			return &tickers[0], nil
		}
		return nil, fmt.Errorf("no tickers returned")
	}
}

// GetAllTickers retrieves all 24hr ticker price change statistics
func (p *BinanceProvider) GetAllTickers(ctx context.Context) ([]BinanceTicker, error) {
	url := p.BaseURL + TickerEndpoint

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := p.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var binanceErr BinanceError
		if err := json.Unmarshal(body, &binanceErr); err == nil {
			return nil, binanceErr
		}
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tickers []BinanceTicker
	if err := json.Unmarshal(body, &tickers); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return tickers, nil
}

// GetKlines retrieves kline/candlestick data
func (p *BinanceProvider) GetKlines(ctx context.Context, symbol, interval string, limit int, startTime, endTime *time.Time) ([]BinanceKline, error) {
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("interval", interval)
	
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	
	if startTime != nil {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	
	if endTime != nil {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}

	url := p.BaseURL + KlinesEndpoint + "?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := p.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var binanceErr BinanceError
		if err := json.Unmarshal(body, &binanceErr); err == nil {
			return nil, binanceErr
		}
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse raw kline data
	var rawKlines [][]interface{}
	if err := json.Unmarshal(body, &rawKlines); err != nil {
		return nil, fmt.Errorf("failed to unmarshal klines response: %w", err)
	}

	klines := make([]BinanceKline, len(rawKlines))
	for i, rawKline := range rawKlines {
		if len(rawKline) < 11 {
			continue
		}

		klines[i] = BinanceKline{
			OpenTime:                 int64(rawKline[0].(float64)),
			Open:                     rawKline[1].(string),
			High:                     rawKline[2].(string),
			Low:                      rawKline[3].(string),
			Close:                    rawKline[4].(string),
			Volume:                   rawKline[5].(string),
			CloseTime:                int64(rawKline[6].(float64)),
			QuoteAssetVolume:         rawKline[7].(string),
			NumberOfTrades:           int64(rawKline[8].(float64)),
			TakerBuyBaseAssetVolume:  rawKline[9].(string),
			TakerBuyQuoteAssetVolume: rawKline[10].(string),
		}
	}

	return klines, nil
}

// GetOrderBook retrieves order book data
func (p *BinanceProvider) GetOrderBook(ctx context.Context, symbol string, limit int) (*BinanceOrderBook, error) {
	params := url.Values{}
	params.Set("symbol", symbol)
	
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}

	url := p.BaseURL + DepthEndpoint + "?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := p.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var binanceErr BinanceError
		if err := json.Unmarshal(body, &binanceErr); err == nil {
			return nil, binanceErr
		}
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var orderBook BinanceOrderBook
	if err := json.Unmarshal(body, &orderBook); err != nil {
		return nil, fmt.Errorf("failed to unmarshal order book response: %w", err)
	}

	return &orderBook, nil
}

// GetTrades retrieves recent trades
func (p *BinanceProvider) GetTrades(ctx context.Context, symbol string, limit int) ([]BinanceTrade, error) {
	params := url.Values{}
	params.Set("symbol", symbol)
	
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}

	url := p.BaseURL + TradesEndpoint + "?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := p.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var binanceErr BinanceError
		if err := json.Unmarshal(body, &binanceErr); err == nil {
			return nil, binanceErr
		}
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var trades []BinanceTrade
	if err := json.Unmarshal(body, &trades); err != nil {
		return nil, fmt.Errorf("failed to unmarshal trades response: %w", err)
	}

	return trades, nil
}

// GetAggTrades retrieves aggregate trades
func (p *BinanceProvider) GetAggTrades(ctx context.Context, symbol string, limit int, startTime, endTime *time.Time) ([]BinanceAggTrade, error) {
	params := url.Values{}
	params.Set("symbol", symbol)
	
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	
	if startTime != nil {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	
	if endTime != nil {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}

	url := p.BaseURL + AggTradesEndpoint + "?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := p.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var binanceErr BinanceError
		if err := json.Unmarshal(body, &binanceErr); err == nil {
			return nil, binanceErr
		}
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var aggTrades []BinanceAggTrade
	if err := json.Unmarshal(body, &aggTrades); err != nil {
		return nil, fmt.Errorf("failed to unmarshal aggregate trades response: %w", err)
	}

	return aggTrades, nil
}

// Close closes all WebSocket connections and cancels the context
func (p *BinanceProvider) Close() error {
	p.cancel()

	p.mu.Lock()
	defer p.mu.Unlock()

	for key, conn := range p.WebSocketConnections {
		conn.Close()
		delete(p.WebSocketConnections, key)
		delete(p.Callbacks, key)
	}

	return nil
}

// IsConnected checks if the provider has active connections
func (p *BinanceProvider) IsConnected() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	return len(p.WebSocketConnections) > 0
}

// GetActiveSubscriptions returns the list of active subscription keys
func (p *BinanceProvider) GetActiveSubscriptions() []string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	keys := make([]string, 0, len(p.WebSocketConnections))
	for key := range p.WebSocketConnections {
		keys = append(keys, key)
	}
	
	return keys
}
