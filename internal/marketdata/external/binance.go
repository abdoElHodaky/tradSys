package external

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// BinanceProvider represents a Binance market data provider
type BinanceProvider struct {
	// BaseURL is the base URL for the Binance API
	BaseURL string
	// WebSocketURL is the WebSocket URL for the Binance API
	WebSocketURL string
	// APIKey is the API key for the Binance API
	APIKey string
	// APISecret is the API secret for the Binance API
	APISecret string
	// HTTPClient is the HTTP client
	HTTPClient *http.Client
	// WebSocketConnections is a map of subscription key to WebSocket connection
	WebSocketConnections map[string]*websocket.Conn
	// Callbacks is a map of subscription key to callback function
	Callbacks map[string]MarketDataCallback
	// Logger
	logger *zap.Logger
	// Mutex for thread safety
	mu sync.RWMutex
	// Context
	ctx context.Context
	// Cancel function
	cancel context.CancelFunc
}

// NewBinanceProvider creates a new Binance market data provider
func NewBinanceProvider(apiKey, apiSecret string, logger *zap.Logger) *BinanceProvider {
	ctx, cancel := context.WithCancel(context.Background())

	return &BinanceProvider{
		BaseURL:              "https://api.binance.com",
		WebSocketURL:         "wss://stream.binance.com:9443/ws",
		APIKey:               apiKey,
		APISecret:            apiSecret,
		HTTPClient:           &http.Client{Timeout: 10 * time.Second},
		WebSocketConnections: make(map[string]*websocket.Conn),
		Callbacks:            make(map[string]MarketDataCallback),
		logger:               logger,
		ctx:                  ctx,
		cancel:               cancel,
	}
}

// Name returns the name of the provider
func (p *BinanceProvider) Name() string {
	return "binance"
}

// Connect connects to the provider
func (p *BinanceProvider) Connect(ctx context.Context) error {
	// Binance REST API doesn't require a persistent connection
	// WebSocket connections are established on subscription
	return nil
}

// Disconnect disconnects from the provider
func (p *BinanceProvider) Disconnect(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Close all WebSocket connections
	for key, conn := range p.WebSocketConnections {
		if err := conn.Close(); err != nil {
			p.logger.Error("Failed to close WebSocket connection",
				zap.Error(err),
				zap.String("key", key))
		}
	}

	// Clear maps
	p.WebSocketConnections = make(map[string]*websocket.Conn)
	p.Callbacks = make(map[string]MarketDataCallback)

	// Cancel context
	p.cancel()

	return nil
}

// subscriptionKey generates a subscription key
func (p *BinanceProvider) subscriptionKey(dataType MarketDataType, symbol string, interval string) string {
	if dataType == MarketDataTypeOHLCV {
		return fmt.Sprintf("%s:%s:%s", dataType, symbol, interval)
	}
	return fmt.Sprintf("%s:%s", dataType, symbol)
}

// connectWebSocket connects to a WebSocket stream
func (p *BinanceProvider) connectWebSocket(streamName string, key string, callback MarketDataCallback) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Check if already connected
	if _, exists := p.WebSocketConnections[key]; exists {
		return nil
	}

	// Connect to WebSocket
	url := fmt.Sprintf("%s/%s", p.WebSocketURL, streamName)
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		p.logger.Error("Failed to connect to WebSocket",
			zap.Error(err),
			zap.String("url", url))
		return err
	}

	// Store connection and callback
	p.WebSocketConnections[key] = conn
	p.Callbacks[key] = callback

	// Start message handler
	go p.handleWebSocketMessages(conn, key)

	return nil
}

// handleWebSocketMessages handles WebSocket messages
func (p *BinanceProvider) handleWebSocketMessages(conn *websocket.Conn, key string) {
	defer func() {
		p.mu.Lock()
		delete(p.WebSocketConnections, key)
		delete(p.Callbacks, key)
		p.mu.Unlock()

		conn.Close()
	}()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			p.logger.Error("WebSocket read error",
				zap.Error(err),
				zap.String("key", key))
			return
		}

		// Get callback
		p.mu.RLock()
		callback, exists := p.Callbacks[key]
		p.mu.RUnlock()

		if !exists {
			continue
		}

		// Parse message based on subscription type
		if key[:9] == string(MarketDataTypeOrderBook) {
			var orderBookUpdate struct {
				LastUpdateID int64      `json:"lastUpdateId"`
				Bids         [][]string `json:"bids"`
				Asks         [][]string `json:"asks"`
				Symbol       string     `json:"s"`
				EventTime    int64      `json:"E"`
			}

			if err := json.Unmarshal(message, &orderBookUpdate); err != nil {
				p.logger.Error("Failed to parse order book update",
					zap.Error(err),
					zap.String("message", string(message)))
				continue
			}

			// Convert string arrays to float arrays
			bids := make([][]float64, len(orderBookUpdate.Bids))
			for i, bid := range orderBookUpdate.Bids {
				price, _ := strconv.ParseFloat(bid[0], 64)
				quantity, _ := strconv.ParseFloat(bid[1], 64)
				bids[i] = []float64{price, quantity}
			}

			asks := make([][]float64, len(orderBookUpdate.Asks))
			for i, ask := range orderBookUpdate.Asks {
				price, _ := strconv.ParseFloat(ask[0], 64)
				quantity, _ := strconv.ParseFloat(ask[1], 64)
				asks[i] = []float64{price, quantity}
			}

			// Create order book data
			orderBookData := &OrderBookData{
				Symbol:    orderBookUpdate.Symbol,
				Bids:      bids,
				Asks:      asks,
				Timestamp: time.Unix(0, orderBookUpdate.EventTime*int64(time.Millisecond)),
			}

			// Call callback
			callback(orderBookData)
		} else if key[:5] == string(MarketDataTypeTrade) {
			var tradeUpdate struct {
				EventType      string `json:"e"`
				EventTime      int64  `json:"E"`
				Symbol         string `json:"s"`
				TradeID        int64  `json:"t"`
				Price          string `json:"p"`
				Quantity       string `json:"q"`
				BuyerOrderID   int64  `json:"b"`
				SellerOrderID  int64  `json:"a"`
				TradeTime      int64  `json:"T"`
				IsBuyerMaker   bool   `json:"m"`
				IsRegularTrade bool   `json:"M"`
			}

			if err := json.Unmarshal(message, &tradeUpdate); err != nil {
				p.logger.Error("Failed to parse trade update",
					zap.Error(err),
					zap.String("message", string(message)))
				continue
			}

			price, _ := strconv.ParseFloat(tradeUpdate.Price, 64)
			quantity, _ := strconv.ParseFloat(tradeUpdate.Quantity, 64)

			// Create trade data
			tradeData := &TradeData{
				Symbol:    tradeUpdate.Symbol,
				Price:     price,
				Quantity:  quantity,
				Side:      "sell",
				Timestamp: time.Unix(0, tradeUpdate.TradeTime*int64(time.Millisecond)),
				TradeID:   strconv.FormatInt(tradeUpdate.TradeID, 10),
			}

			if tradeUpdate.IsBuyerMaker {
				tradeData.Side = "buy"
			}

			// Call callback
			callback(tradeData)
		} else if key[:6] == string(MarketDataTypeTicker) {
			var tickerUpdate struct {
				EventType    string `json:"e"`
				EventTime    int64  `json:"E"`
				Symbol       string `json:"s"`
				PriceChange  string `json:"p"`
				PriceChangeP string `json:"P"`
				WeightedAvg  string `json:"w"`
				PrevCloseP   string `json:"x"`
				LastPrice    string `json:"c"`
				LastQty      string `json:"Q"`
				BidPrice     string `json:"b"`
				BidQty       string `json:"B"`
				AskPrice     string `json:"a"`
				AskQty       string `json:"A"`
				OpenPrice    string `json:"o"`
				HighPrice    string `json:"h"`
				LowPrice     string `json:"l"`
				Volume       string `json:"v"`
				QuoteVolume  string `json:"q"`
				OpenTime     int64  `json:"O"`
				CloseTime    int64  `json:"C"`
				FirstTradeID int64  `json:"F"`
				LastTradeID  int64  `json:"L"`
				TradeCount   int64  `json:"n"`
			}

			if err := json.Unmarshal(message, &tickerUpdate); err != nil {
				p.logger.Error("Failed to parse ticker update",
					zap.Error(err),
					zap.String("message", string(message)))
				continue
			}

			price, _ := strconv.ParseFloat(tickerUpdate.LastPrice, 64)
			volume, _ := strconv.ParseFloat(tickerUpdate.Volume, 64)
			change, _ := strconv.ParseFloat(tickerUpdate.PriceChange, 64)
			changePercent, _ := strconv.ParseFloat(tickerUpdate.PriceChangeP, 64)
			high, _ := strconv.ParseFloat(tickerUpdate.HighPrice, 64)
			low, _ := strconv.ParseFloat(tickerUpdate.LowPrice, 64)

			// Create ticker data
			tickerData := &TickerData{
				Symbol:        tickerUpdate.Symbol,
				Price:         price,
				Volume:        volume,
				Change:        change,
				ChangePercent: changePercent,
				High:          high,
				Low:           low,
				Timestamp:     time.Unix(0, tickerUpdate.EventTime*int64(time.Millisecond)),
			}

			// Call callback
			callback(tickerData)
		} else if key[:5] == string(MarketDataTypeOHLCV) {
			var klineUpdate struct {
				EventType string `json:"e"`
				EventTime int64  `json:"E"`
				Symbol    string `json:"s"`
				Kline     struct {
					StartTime           int64  `json:"t"`
					CloseTime           int64  `json:"T"`
					Symbol              string `json:"s"`
					Interval            string `json:"i"`
					FirstTradeID        int64  `json:"f"`
					LastTradeID         int64  `json:"L"`
					OpenPrice           string `json:"o"`
					ClosePrice          string `json:"c"`
					HighPrice           string `json:"h"`
					LowPrice            string `json:"l"`
					BaseAssetVolume     string `json:"v"`
					NumberOfTrades      int64  `json:"n"`
					IsClosed            bool   `json:"x"`
					QuoteAssetVolume    string `json:"q"`
					TakerBuyBaseVolume  string `json:"V"`
					TakerBuyQuoteVolume string `json:"Q"`
				} `json:"k"`
			}

			if err := json.Unmarshal(message, &klineUpdate); err != nil {
				p.logger.Error("Failed to parse kline update",
					zap.Error(err),
					zap.String("message", string(message)))
				continue
			}

			open, _ := strconv.ParseFloat(klineUpdate.Kline.OpenPrice, 64)
			high, _ := strconv.ParseFloat(klineUpdate.Kline.HighPrice, 64)
			low, _ := strconv.ParseFloat(klineUpdate.Kline.LowPrice, 64)
			close, _ := strconv.ParseFloat(klineUpdate.Kline.ClosePrice, 64)
			volume, _ := strconv.ParseFloat(klineUpdate.Kline.BaseAssetVolume, 64)

			// Create OHLCV data
			ohlcvData := &OHLCVData{
				Symbol:    klineUpdate.Symbol,
				Interval:  klineUpdate.Kline.Interval,
				Open:      open,
				High:      high,
				Low:       low,
				Close:     close,
				Volume:    volume,
				Timestamp: time.Unix(0, klineUpdate.Kline.StartTime*int64(time.Millisecond)),
			}

			// Call callback
			callback(ohlcvData)
		}
	}
}

// SubscribeOrderBook subscribes to order book updates
func (p *BinanceProvider) SubscribeOrderBook(ctx context.Context, symbol string, callback MarketDataCallback) error {
	key := p.subscriptionKey(MarketDataTypeOrderBook, symbol, "")
	streamName := fmt.Sprintf("%s@depth", symbol)
	return p.connectWebSocket(streamName, key, callback)
}

// UnsubscribeOrderBook unsubscribes from order book updates
func (p *BinanceProvider) UnsubscribeOrderBook(ctx context.Context, symbol string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	key := p.subscriptionKey(MarketDataTypeOrderBook, symbol, "")
	conn, exists := p.WebSocketConnections[key]
	if !exists {
		return nil
	}

	if err := conn.Close(); err != nil {
		p.logger.Error("Failed to close WebSocket connection",
			zap.Error(err),
			zap.String("key", key))
		return err
	}

	delete(p.WebSocketConnections, key)
	delete(p.Callbacks, key)

	return nil
}

// SubscribeTrades subscribes to trade updates
func (p *BinanceProvider) SubscribeTrades(ctx context.Context, symbol string, callback MarketDataCallback) error {
	key := p.subscriptionKey(MarketDataTypeTrade, symbol, "")
	streamName := fmt.Sprintf("%s@trade", symbol)
	return p.connectWebSocket(streamName, key, callback)
}

// UnsubscribeTrades unsubscribes from trade updates
func (p *BinanceProvider) UnsubscribeTrades(ctx context.Context, symbol string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	key := p.subscriptionKey(MarketDataTypeTrade, symbol, "")
	conn, exists := p.WebSocketConnections[key]
	if !exists {
		return nil
	}

	if err := conn.Close(); err != nil {
		p.logger.Error("Failed to close WebSocket connection",
			zap.Error(err),
			zap.String("key", key))
		return err
	}

	delete(p.WebSocketConnections, key)
	delete(p.Callbacks, key)

	return nil
}

// SubscribeTicker subscribes to ticker updates
func (p *BinanceProvider) SubscribeTicker(ctx context.Context, symbol string, callback MarketDataCallback) error {
	key := p.subscriptionKey(MarketDataTypeTicker, symbol, "")
	streamName := fmt.Sprintf("%s@ticker", symbol)
	return p.connectWebSocket(streamName, key, callback)
}

// UnsubscribeTicker unsubscribes from ticker updates
func (p *BinanceProvider) UnsubscribeTicker(ctx context.Context, symbol string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	key := p.subscriptionKey(MarketDataTypeTicker, symbol, "")
	conn, exists := p.WebSocketConnections[key]
	if !exists {
		return nil
	}

	if err := conn.Close(); err != nil {
		p.logger.Error("Failed to close WebSocket connection",
			zap.Error(err),
			zap.String("key", key))
		return err
	}

	delete(p.WebSocketConnections, key)
	delete(p.Callbacks, key)

	return nil
}

// SubscribeOHLCV subscribes to OHLCV updates
func (p *BinanceProvider) SubscribeOHLCV(ctx context.Context, symbol, interval string, callback MarketDataCallback) error {
	key := p.subscriptionKey(MarketDataTypeOHLCV, symbol, interval)
	streamName := fmt.Sprintf("%s@kline_%s", symbol, interval)
	return p.connectWebSocket(streamName, key, callback)
}

// UnsubscribeOHLCV unsubscribes from OHLCV updates
func (p *BinanceProvider) UnsubscribeOHLCV(ctx context.Context, symbol, interval string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	key := p.subscriptionKey(MarketDataTypeOHLCV, symbol, interval)
	conn, exists := p.WebSocketConnections[key]
	if !exists {
		return nil
	}

	if err := conn.Close(); err != nil {
		p.logger.Error("Failed to close WebSocket connection",
			zap.Error(err),
			zap.String("key", key))
		return err
	}

	delete(p.WebSocketConnections, key)
	delete(p.Callbacks, key)

	return nil
}

// GetOrderBook gets the order book
func (p *BinanceProvider) GetOrderBook(ctx context.Context, symbol string) (*OrderBookData, error) {
	url := fmt.Sprintf("%s/api/v3/depth?symbol=%s&limit=100", p.BaseURL, symbol)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var response struct {
		LastUpdateID int64      `json:"lastUpdateId"`
		Bids         [][]string `json:"bids"`
		Asks         [][]string `json:"asks"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	// Convert string arrays to float arrays
	bids := make([][]float64, len(response.Bids))
	for i, bid := range response.Bids {
		price, _ := strconv.ParseFloat(bid[0], 64)
		quantity, _ := strconv.ParseFloat(bid[1], 64)
		bids[i] = []float64{price, quantity}
	}

	asks := make([][]float64, len(response.Asks))
	for i, ask := range response.Asks {
		price, _ := strconv.ParseFloat(ask[0], 64)
		quantity, _ := strconv.ParseFloat(ask[1], 64)
		asks[i] = []float64{price, quantity}
	}

	return &OrderBookData{
		Symbol:    symbol,
		Bids:      bids,
		Asks:      asks,
		Timestamp: time.Now(),
	}, nil
}

// GetTrades gets trades
func (p *BinanceProvider) GetTrades(ctx context.Context, symbol string, limit int) ([]TradeData, error) {
	url := fmt.Sprintf("%s/api/v3/trades?symbol=%s&limit=%d", p.BaseURL, symbol, limit)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var response []struct {
		ID           int64  `json:"id"`
		Price        string `json:"price"`
		Qty          string `json:"qty"`
		QuoteQty     string `json:"quoteQty"`
		Time         int64  `json:"time"`
		IsBuyerMaker bool   `json:"isBuyerMaker"`
		IsBestMatch  bool   `json:"isBestMatch"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	trades := make([]TradeData, len(response))
	for i, trade := range response {
		price, _ := strconv.ParseFloat(trade.Price, 64)
		quantity, _ := strconv.ParseFloat(trade.Qty, 64)

		side := "sell"
		if trade.IsBuyerMaker {
			side = "buy"
		}

		trades[i] = TradeData{
			Symbol:    symbol,
			Price:     price,
			Quantity:  quantity,
			Side:      side,
			Timestamp: time.Unix(0, trade.Time*int64(time.Millisecond)),
			TradeID:   strconv.FormatInt(trade.ID, 10),
		}
	}

	return trades, nil
}

// GetTicker gets the ticker
func (p *BinanceProvider) GetTicker(ctx context.Context, symbol string) (*TickerData, error) {
	url := fmt.Sprintf("%s/api/v3/ticker/24hr?symbol=%s", p.BaseURL, symbol)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var response struct {
		Symbol             string `json:"symbol"`
		PriceChange        string `json:"priceChange"`
		PriceChangePercent string `json:"priceChangePercent"`
		WeightedAvgPrice   string `json:"weightedAvgPrice"`
		PrevClosePrice     string `json:"prevClosePrice"`
		LastPrice          string `json:"lastPrice"`
		LastQty            string `json:"lastQty"`
		BidPrice           string `json:"bidPrice"`
		BidQty             string `json:"bidQty"`
		AskPrice           string `json:"askPrice"`
		AskQty             string `json:"askQty"`
		OpenPrice          string `json:"openPrice"`
		HighPrice          string `json:"highPrice"`
		LowPrice           string `json:"lowPrice"`
		Volume             string `json:"volume"`
		QuoteVolume        string `json:"quoteVolume"`
		OpenTime           int64  `json:"openTime"`
		CloseTime          int64  `json:"closeTime"`
		FirstId            int64  `json:"firstId"`
		LastId             int64  `json:"lastId"`
		Count              int64  `json:"count"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	price, _ := strconv.ParseFloat(response.LastPrice, 64)
	volume, _ := strconv.ParseFloat(response.Volume, 64)
	change, _ := strconv.ParseFloat(response.PriceChange, 64)
	changePercent, _ := strconv.ParseFloat(response.PriceChangePercent, 64)
	high, _ := strconv.ParseFloat(response.HighPrice, 64)
	low, _ := strconv.ParseFloat(response.LowPrice, 64)

	return &TickerData{
		Symbol:        response.Symbol,
		Price:         price,
		Volume:        volume,
		Change:        change,
		ChangePercent: changePercent,
		High:          high,
		Low:           low,
		Timestamp:     time.Unix(0, response.CloseTime*int64(time.Millisecond)),
	}, nil
}

// GetOHLCV gets OHLCV data
func (p *BinanceProvider) GetOHLCV(ctx context.Context, symbol, interval string, limit int) ([]OHLCVData, error) {
	url := fmt.Sprintf("%s/api/v3/klines?symbol=%s&interval=%s&limit=%d", p.BaseURL, symbol, interval, limit)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var response [][]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	ohlcvData := make([]OHLCVData, len(response))
	for i, kline := range response {
		openTime := int64(kline[0].(float64))
		open, _ := strconv.ParseFloat(kline[1].(string), 64)
		high, _ := strconv.ParseFloat(kline[2].(string), 64)
		low, _ := strconv.ParseFloat(kline[3].(string), 64)
		close, _ := strconv.ParseFloat(kline[4].(string), 64)
		volume, _ := strconv.ParseFloat(kline[5].(string), 64)

		ohlcvData[i] = OHLCVData{
			Symbol:    symbol,
			Interval:  interval,
			Open:      open,
			High:      high,
			Low:       low,
			Close:     close,
			Volume:    volume,
			Timestamp: time.Unix(0, openTime*int64(time.Millisecond)),
		}
	}

	return ohlcvData, nil
}
