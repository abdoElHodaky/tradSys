package external

import (
	"context"
	"fmt"
	"net/http"
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

// MarketDataCallback represents a callback function for market data
type MarketDataCallback func(data interface{}) error

// BinanceSymbol represents a Binance symbol
type BinanceSymbol struct {
	Symbol                     string   `json:"symbol"`
	Status                     string   `json:"status"`
	BaseAsset                  string   `json:"baseAsset"`
	BaseAssetPrecision         int      `json:"baseAssetPrecision"`
	QuoteAsset                 string   `json:"quoteAsset"`
	QuotePrecision             int      `json:"quotePrecision"`
	QuoteAssetPrecision        int      `json:"quoteAssetPrecision"`
	BaseCommissionPrecision    int      `json:"baseCommissionPrecision"`
	QuoteCommissionPrecision   int      `json:"quoteCommissionPrecision"`
	OrderTypes                 []string `json:"orderTypes"`
	IcebergAllowed             bool     `json:"icebergAllowed"`
	OcoAllowed                 bool     `json:"ocoAllowed"`
	QuoteOrderQtyMarketAllowed bool     `json:"quoteOrderQtyMarketAllowed"`
	AllowTrailingStop          bool     `json:"allowTrailingStop"`
	CancelReplaceAllowed       bool     `json:"cancelReplaceAllowed"`
	IsSpotTradingAllowed       bool     `json:"isSpotTradingAllowed"`
	IsMarginTradingAllowed     bool     `json:"isMarginTradingAllowed"`
	Filters                    []Filter `json:"filters"`
	Permissions                []string `json:"permissions"`
}

// Filter represents a Binance filter
type Filter struct {
	FilterType          string `json:"filterType"`
	MinPrice            string `json:"minPrice,omitempty"`
	MaxPrice            string `json:"maxPrice,omitempty"`
	TickSize            string `json:"tickSize,omitempty"`
	MultiplierUp        string `json:"multiplierUp,omitempty"`
	MultiplierDown      string `json:"multiplierDown,omitempty"`
	AvgPriceMins        int    `json:"avgPriceMins,omitempty"`
	MinQty              string `json:"minQty,omitempty"`
	MaxQty              string `json:"maxQty,omitempty"`
	StepSize            string `json:"stepSize,omitempty"`
	MinNotional         string `json:"minNotional,omitempty"`
	ApplyToMarket       bool   `json:"applyToMarket,omitempty"`
	Limit               int    `json:"limit,omitempty"`
	MaxNumOrders        int    `json:"maxNumOrders,omitempty"`
	MaxNumAlgoOrders    int    `json:"maxNumAlgoOrders,omitempty"`
	MaxNumIcebergOrders int    `json:"maxNumIcebergOrders,omitempty"`
	MaxPosition         string `json:"maxPosition,omitempty"`
}

// BinanceTicker represents a Binance ticker
type BinanceTicker struct {
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

// BinanceKline represents a Binance kline/candlestick
type BinanceKline struct {
	OpenTime                 int64  `json:"openTime"`
	Open                     string `json:"open"`
	High                     string `json:"high"`
	Low                      string `json:"low"`
	Close                    string `json:"close"`
	Volume                   string `json:"volume"`
	CloseTime                int64  `json:"closeTime"`
	QuoteAssetVolume         string `json:"quoteAssetVolume"`
	NumberOfTrades           int64  `json:"numberOfTrades"`
	TakerBuyBaseAssetVolume  string `json:"takerBuyBaseAssetVolume"`
	TakerBuyQuoteAssetVolume string `json:"takerBuyQuoteAssetVolume"`
}

// BinanceOrderBook represents a Binance order book
type BinanceOrderBook struct {
	LastUpdateId int64      `json:"lastUpdateId"`
	Bids         [][]string `json:"bids"`
	Asks         [][]string `json:"asks"`
}

// BinanceTrade represents a Binance trade
type BinanceTrade struct {
	Id           int64  `json:"id"`
	Price        string `json:"price"`
	Qty          string `json:"qty"`
	QuoteQty     string `json:"quoteQty"`
	Time         int64  `json:"time"`
	IsBuyerMaker bool   `json:"isBuyerMaker"`
	IsBestMatch  bool   `json:"isBestMatch"`
}

// BinanceAggTrade represents a Binance aggregate trade
type BinanceAggTrade struct {
	AggTradeId   int64  `json:"a"`
	Price        string `json:"p"`
	Quantity     string `json:"q"`
	FirstTradeId int64  `json:"f"`
	LastTradeId  int64  `json:"l"`
	Timestamp    int64  `json:"T"`
	IsBuyerMaker bool   `json:"m"`
	IsBestMatch  bool   `json:"M"`
}

// WebSocket Stream Types

// BinanceStreamTicker represents a ticker stream message
type BinanceStreamTicker struct {
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

// BinanceStreamKline represents a kline stream message
type BinanceStreamKline struct {
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

// BinanceStreamTrade represents a trade stream message
type BinanceStreamTrade struct {
	EventType         string `json:"e"`
	EventTime         int64  `json:"E"`
	Symbol            string `json:"s"`
	TradeId           int64  `json:"t"`
	Price             string `json:"p"`
	Quantity          string `json:"q"`
	BuyerOrderId      int64  `json:"b"`
	SellerOrderId     int64  `json:"a"`
	TradeTime         int64  `json:"T"`
	IsBuyerMaker      bool   `json:"m"`
	PlaceholderIgnore bool   `json:"M"`
}

// BinanceStreamDepth represents a depth stream message
type BinanceStreamDepth struct {
	EventType     string     `json:"e"`
	EventTime     int64      `json:"E"`
	Symbol        string     `json:"s"`
	FirstUpdateId int64      `json:"U"`
	FinalUpdateId int64      `json:"u"`
	Bids          [][]string `json:"b"`
	Asks          [][]string `json:"a"`
}

// BinanceStreamAggTrade represents an aggregate trade stream message
type BinanceStreamAggTrade struct {
	EventType    string `json:"e"`
	EventTime    int64  `json:"E"`
	Symbol       string `json:"s"`
	AggTradeId   int64  `json:"a"`
	Price        string `json:"p"`
	Quantity     string `json:"q"`
	FirstTradeId int64  `json:"f"`
	LastTradeId  int64  `json:"l"`
	TradeTime    int64  `json:"T"`
	IsBuyerMaker bool   `json:"m"`
	Ignore       bool   `json:"M"`
}

// BinanceError represents a Binance API error
type BinanceError struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

// Error implements the error interface
func (e BinanceError) Error() string {
	return fmt.Sprintf("Binance API error %d: %s", e.Code, e.Msg)
}

// BinanceExchangeInfo represents exchange information
type BinanceExchangeInfo struct {
	Timezone        string          `json:"timezone"`
	ServerTime      int64           `json:"serverTime"`
	RateLimits      []RateLimit     `json:"rateLimits"`
	ExchangeFilters []Filter        `json:"exchangeFilters"`
	Symbols         []BinanceSymbol `json:"symbols"`
}

// RateLimit represents a rate limit
type RateLimit struct {
	RateLimitType string `json:"rateLimitType"`
	Interval      string `json:"interval"`
	IntervalNum   int    `json:"intervalNum"`
	Limit         int    `json:"limit"`
}

// SubscriptionRequest represents a WebSocket subscription request
type SubscriptionRequest struct {
	Method string   `json:"method"`
	Params []string `json:"params"`
	Id     int      `json:"id"`
}

// SubscriptionResponse represents a WebSocket subscription response
type SubscriptionResponse struct {
	Result interface{} `json:"result"`
	Id     int         `json:"id"`
}

// StreamMessage represents a generic stream message
type StreamMessage struct {
	Stream string      `json:"stream"`
	Data   interface{} `json:"data"`
}

// Constants for Binance API
const (
	// API Endpoints
	ExchangeInfoEndpoint = "/api/v3/exchangeInfo"
	TickerEndpoint       = "/api/v3/ticker/24hr"
	KlinesEndpoint       = "/api/v3/klines"
	DepthEndpoint        = "/api/v3/depth"
	TradesEndpoint       = "/api/v3/trades"
	AggTradesEndpoint    = "/api/v3/aggTrades"
	
	// WebSocket Streams
	TickerStream    = "@ticker"
	KlineStream     = "@kline_"
	TradeStream     = "@trade"
	DepthStream     = "@depth"
	AggTradeStream  = "@aggTrade"
	
	// Intervals
	Interval1m  = "1m"
	Interval3m  = "3m"
	Interval5m  = "5m"
	Interval15m = "15m"
	Interval30m = "30m"
	Interval1h  = "1h"
	Interval2h  = "2h"
	Interval4h  = "4h"
	Interval6h  = "6h"
	Interval8h  = "8h"
	Interval12h = "12h"
	Interval1d  = "1d"
	Interval3d  = "3d"
	Interval1w  = "1w"
	Interval1M  = "1M"
)

// Configuration
type BinanceConfig struct {
	BaseURL         string        `yaml:"base_url" json:"base_url"`
	WebSocketURL    string        `yaml:"websocket_url" json:"websocket_url"`
	APIKey          string        `yaml:"api_key" json:"api_key"`
	APISecret       string        `yaml:"api_secret" json:"api_secret"`
	Timeout         time.Duration `yaml:"timeout" json:"timeout"`
	MaxConnections  int           `yaml:"max_connections" json:"max_connections"`
	ReconnectDelay  time.Duration `yaml:"reconnect_delay" json:"reconnect_delay"`
	PingInterval    time.Duration `yaml:"ping_interval" json:"ping_interval"`
	EnableTestnet   bool          `yaml:"enable_testnet" json:"enable_testnet"`
}
