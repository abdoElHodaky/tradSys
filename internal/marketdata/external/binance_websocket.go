package external

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// subscriptionKey generates a unique key for a subscription
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

			callback(orderBookData)

		} else if key[:5] == string(MarketDataTypeTrade) {
			var tradeUpdate struct {
				EventType    string `json:"e"`
				EventTime    int64  `json:"E"`
				Symbol       string `json:"s"`
				TradeId      int64  `json:"t"`
				Price        string `json:"p"`
				Quantity     string `json:"q"`
				BuyerOrderId int64  `json:"b"`
				SellerOrderId int64 `json:"a"`
				TradeTime    int64  `json:"T"`
				IsBuyerMaker bool   `json:"m"`
			}

			if err := json.Unmarshal(message, &tradeUpdate); err != nil {
				p.logger.Error("Failed to parse trade update",
					zap.Error(err),
					zap.String("message", string(message)))
				continue
			}

			price, _ := strconv.ParseFloat(tradeUpdate.Price, 64)
			quantity, _ := strconv.ParseFloat(tradeUpdate.Quantity, 64)

			side := "buy"
			if tradeUpdate.IsBuyerMaker {
				side = "sell"
			}

			tradeData := &TradeData{
				Symbol:    tradeUpdate.Symbol,
				Price:     price,
				Quantity:  quantity,
				Side:      side,
				Timestamp: time.Unix(0, tradeUpdate.TradeTime*int64(time.Millisecond)),
				TradeID:   fmt.Sprintf("%d", tradeUpdate.TradeId),
			}

			callback(tradeData)

		} else if key[:6] == string(MarketDataTypeTicker) {
			var tickerUpdate struct {
				EventType          string `json:"e"`
				EventTime          int64  `json:"E"`
				Symbol             string `json:"s"`
				PriceChange        string `json:"p"`
				PriceChangePercent string `json:"P"`
				WeightedAvgPrice   string `json:"w"`
				FirstTradePrice    string `json:"x"`
				LastPrice          string `json:"c"`
				LastQty            string `json:"Q"`
				BidPrice           string `json:"b"`
				BidQty             string `json:"B"`
				AskPrice           string `json:"a"`
				AskQty             string `json:"A"`
				OpenPrice          string `json:"o"`
				HighPrice          string `json:"h"`
				LowPrice           string `json:"l"`
				Volume             string `json:"v"`
				QuoteVolume        string `json:"q"`
				OpenTime           int64  `json:"O"`
				CloseTime          int64  `json:"C"`
				FirstTradeId       int64  `json:"F"`
				LastTradeId        int64  `json:"L"`
				TradeCount         int64  `json:"n"`
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
			changePercent, _ := strconv.ParseFloat(tickerUpdate.PriceChangePercent, 64)
			high, _ := strconv.ParseFloat(tickerUpdate.HighPrice, 64)
			low, _ := strconv.ParseFloat(tickerUpdate.LowPrice, 64)

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

			callback(tickerData)

		} else if key[:5] == string(MarketDataTypeOHLCV) {
			var klineUpdate struct {
				EventType string `json:"e"`
				EventTime int64  `json:"E"`
				Symbol    string `json:"s"`
				Kline     struct {
					StartTime                int64  `json:"t"`
					EndTime                  int64  `json:"T"`
					Symbol                   string `json:"s"`
					Interval                 string `json:"i"`
					FirstTradeId             int64  `json:"f"`
					LastTradeId              int64  `json:"L"`
					Open                     string `json:"o"`
					Close                    string `json:"c"`
					High                     string `json:"h"`
					Low                      string `json:"l"`
					Volume                   string `json:"v"`
					NumberOfTrades           int64  `json:"n"`
					IsClosed                 bool   `json:"x"`
					QuoteVolume              string `json:"q"`
					TakerBuyBaseAssetVolume  string `json:"V"`
					TakerBuyQuoteAssetVolume string `json:"Q"`
				} `json:"k"`
			}

			if err := json.Unmarshal(message, &klineUpdate); err != nil {
				p.logger.Error("Failed to parse kline update",
					zap.Error(err),
					zap.String("message", string(message)))
				continue
			}

			open, _ := strconv.ParseFloat(klineUpdate.Kline.Open, 64)
			high, _ := strconv.ParseFloat(klineUpdate.Kline.High, 64)
			low, _ := strconv.ParseFloat(klineUpdate.Kline.Low, 64)
			close, _ := strconv.ParseFloat(klineUpdate.Kline.Close, 64)
			volume, _ := strconv.ParseFloat(klineUpdate.Kline.Volume, 64)

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

// SubscribeTrade subscribes to trade updates
func (p *BinanceProvider) SubscribeTrade(ctx context.Context, symbol string, callback MarketDataCallback) error {
	key := p.subscriptionKey(MarketDataTypeTrade, symbol, "")
	streamName := fmt.Sprintf("%s@trade", symbol)
	return p.connectWebSocket(streamName, key, callback)
}

// UnsubscribeTrade unsubscribes from trade updates
func (p *BinanceProvider) UnsubscribeTrade(ctx context.Context, symbol string) error {
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

// SubscribeKline subscribes to kline/candlestick updates
func (p *BinanceProvider) SubscribeKline(ctx context.Context, symbol, interval string, callback MarketDataCallback) error {
	key := p.subscriptionKey(MarketDataTypeOHLCV, symbol, interval)
	streamName := fmt.Sprintf("%s@kline_%s", symbol, interval)
	return p.connectWebSocket(streamName, key, callback)
}

// UnsubscribeKline unsubscribes from kline/candlestick updates
func (p *BinanceProvider) UnsubscribeKline(ctx context.Context, symbol, interval string) error {
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

// SubscribeOHLCV subscribes to OHLCV updates (alias for SubscribeKline)
func (p *BinanceProvider) SubscribeOHLCV(ctx context.Context, symbol, interval string, callback MarketDataCallback) error {
	return p.SubscribeKline(ctx, symbol, interval, callback)
}

// UnsubscribeOHLCV unsubscribes from OHLCV updates (alias for UnsubscribeKline)
func (p *BinanceProvider) UnsubscribeOHLCV(ctx context.Context, symbol, interval string) error {
	return p.UnsubscribeKline(ctx, symbol, interval)
}
