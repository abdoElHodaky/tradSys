package connectivity

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/trading/types"
	"go.uber.org/zap"
)

// UnifiedExchangeConnector provides unified connectivity to multiple exchanges
type UnifiedExchangeConnector struct {
	config      *ConnectivityConfig
	logger      *zap.Logger
	exchanges   map[string]ExchangeAdapter
	marketData  *MarketDataManager
	orderRouter *OrderRouter
	metrics     *ConnectivityMetrics
	isRunning   int32
	stopChannel chan struct{}
	mu          sync.RWMutex
}

// ConnectivityConfig contains configuration for exchange connectivity
type ConnectivityConfig struct {
	EnabledExchanges    []string      `json:"enabled_exchanges"`
	MarketDataEnabled   bool          `json:"market_data_enabled"`
	OrderRoutingEnabled bool          `json:"order_routing_enabled"`
	MaxLatency          time.Duration `json:"max_latency"`
	ReconnectInterval   time.Duration `json:"reconnect_interval"`
	HeartbeatInterval   time.Duration `json:"heartbeat_interval"`
	BufferSize          int           `json:"buffer_size"`
}

// ConnectivityMetrics tracks connectivity performance
type ConnectivityMetrics struct {
	TotalMessages      int64         `json:"total_messages"`
	MarketDataMessages int64         `json:"market_data_messages"`
	OrderMessages      int64         `json:"order_messages"`
	AverageLatency     time.Duration `json:"average_latency"`
	MaxLatency         time.Duration `json:"max_latency"`
	ConnectionErrors   int64         `json:"connection_errors"`
	ReconnectCount     int64         `json:"reconnect_count"`
	LastUpdateTime     time.Time     `json:"last_update_time"`
}

// ExchangeAdapter defines the interface for exchange adapters
type ExchangeAdapter interface {
	Connect(ctx context.Context) error
	Disconnect() error
	IsConnected() bool
	SubmitOrder(order *types.Order) (*OrderResponse, error)
	CancelOrder(orderID string) error
	GetOrderStatus(orderID string) (*OrderStatus, error)
	SubscribeMarketData(symbols []string) error
	UnsubscribeMarketData(symbols []string) error
	GetExchangeInfo() *ExchangeInfo
}

// MarketDataManager manages market data feeds from multiple exchanges
type MarketDataManager struct {
	subscribers map[string][]MarketDataHandler
	feedBuffer  chan *MarketDataMessage
	logger      *zap.Logger
	mu          sync.RWMutex
}

// OrderRouter routes orders to appropriate exchanges
type OrderRouter struct {
	routingRules map[string]string // symbol -> exchange
	exchanges    map[string]ExchangeAdapter
	logger       *zap.Logger
	mu           sync.RWMutex
}

// MarketDataHandler defines the interface for market data handlers
type MarketDataHandler interface {
	HandleMarketData(data *MarketDataMessage) error
}

// MarketDataMessage represents a market data message
type MarketDataMessage struct {
	Exchange  string         `json:"exchange"`
	Symbol    string         `json:"symbol"`
	Type      MarketDataType `json:"type"`
	Price     float64        `json:"price"`
	Quantity  float64        `json:"quantity"`
	Timestamp time.Time      `json:"timestamp"`
	Sequence  int64          `json:"sequence"`
	BidPrice  float64        `json:"bid_price,omitempty"`
	AskPrice  float64        `json:"ask_price,omitempty"`
	BidSize   float64        `json:"bid_size,omitempty"`
	AskSize   float64        `json:"ask_size,omitempty"`
	LastPrice float64        `json:"last_price,omitempty"`
	Volume    float64        `json:"volume,omitempty"`
}

// MarketDataType defines types of market data
type MarketDataType string

const (
	MarketDataTypeTrade     MarketDataType = "trade"
	MarketDataTypeQuote     MarketDataType = "quote"
	MarketDataTypeOrderBook MarketDataType = "orderbook"
	MarketDataTypeTicker    MarketDataType = "ticker"
)

// OrderResponse represents a response from order submission
type OrderResponse struct {
	OrderID    string        `json:"order_id"`
	ExchangeID string        `json:"exchange_id"`
	Status     string        `json:"status"`
	Message    string        `json:"message"`
	Timestamp  time.Time     `json:"timestamp"`
	Latency    time.Duration `json:"latency"`
}

// OrderStatus represents the status of an order
type OrderStatus struct {
	OrderID      string    `json:"order_id"`
	ExchangeID   string    `json:"exchange_id"`
	Status       string    `json:"status"`
	FilledQty    float64   `json:"filled_qty"`
	RemainingQty float64   `json:"remaining_qty"`
	AveragePrice float64   `json:"average_price"`
	LastUpdate   time.Time `json:"last_update"`
}

// ExchangeInfo contains information about an exchange
type ExchangeInfo struct {
	Name             string             `json:"name"`
	Status           string             `json:"status"`
	TradingFees      map[string]float64 `json:"trading_fees"`
	MinOrderSize     map[string]float64 `json:"min_order_size"`
	MaxOrderSize     map[string]float64 `json:"max_order_size"`
	SupportedSymbols []string           `json:"supported_symbols"`
	Capabilities     []string           `json:"capabilities"`
}

// NewUnifiedExchangeConnector creates a new unified exchange connector
func NewUnifiedExchangeConnector(config *ConnectivityConfig, logger *zap.Logger) *UnifiedExchangeConnector {
	connector := &UnifiedExchangeConnector{
		config:      config,
		logger:      logger,
		exchanges:   make(map[string]ExchangeAdapter),
		metrics:     &ConnectivityMetrics{LastUpdateTime: time.Now()},
		stopChannel: make(chan struct{}),
	}

	// Initialize market data manager
	connector.marketData = &MarketDataManager{
		subscribers: make(map[string][]MarketDataHandler),
		feedBuffer:  make(chan *MarketDataMessage, config.BufferSize),
		logger:      logger.Named("market_data"),
	}

	// Initialize order router
	connector.orderRouter = &OrderRouter{
		routingRules: make(map[string]string),
		exchanges:    connector.exchanges,
		logger:       logger.Named("order_router"),
	}

	return connector
}

// Start starts the unified exchange connector
func (c *UnifiedExchangeConnector) Start(ctx context.Context) error {
	if !atomic.CompareAndSwapInt32(&c.isRunning, 0, 1) {
		return fmt.Errorf("exchange connector is already running")
	}

	c.logger.Info("Starting unified exchange connector",
		zap.Any("config", c.config))

	// Connect to enabled exchanges
	for _, exchangeName := range c.config.EnabledExchanges {
		if adapter, exists := c.exchanges[exchangeName]; exists {
			if err := adapter.Connect(ctx); err != nil {
				c.logger.Error("Failed to connect to exchange",
					zap.String("exchange", exchangeName),
					zap.Error(err))
				atomic.AddInt64(&c.metrics.ConnectionErrors, 1)
			} else {
				c.logger.Info("Connected to exchange",
					zap.String("exchange", exchangeName))
			}
		}
	}

	// Start market data processing if enabled
	if c.config.MarketDataEnabled {
		go c.processMarketData(ctx)
	}

	// Start connection monitoring
	go c.monitorConnections(ctx)

	c.logger.Info("Unified exchange connector started successfully")
	return nil
}

// Stop stops the unified exchange connector
func (c *UnifiedExchangeConnector) Stop() error {
	if !atomic.CompareAndSwapInt32(&c.isRunning, 1, 0) {
		return fmt.Errorf("exchange connector is not running")
	}

	c.logger.Info("Stopping unified exchange connector")

	// Disconnect from all exchanges
	c.mu.RLock()
	for name, adapter := range c.exchanges {
		if err := adapter.Disconnect(); err != nil {
			c.logger.Error("Failed to disconnect from exchange",
				zap.String("exchange", name),
				zap.Error(err))
		}
	}
	c.mu.RUnlock()

	close(c.stopChannel)
	c.logger.Info("Unified exchange connector stopped")
	return nil
}

// RegisterExchange registers an exchange adapter
func (c *UnifiedExchangeConnector) RegisterExchange(name string, adapter ExchangeAdapter) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.exchanges[name] = adapter
	c.orderRouter.exchanges[name] = adapter
	c.logger.Info("Registered exchange adapter", zap.String("exchange", name))
}

// SubmitOrder submits an order through the appropriate exchange
func (c *UnifiedExchangeConnector) SubmitOrder(order *types.Order) (*OrderResponse, error) {
	startTime := time.Now()
	defer func() {
		latency := time.Since(startTime)
		c.updateMetrics(latency)
		atomic.AddInt64(&c.metrics.OrderMessages, 1)
	}()

	if !c.config.OrderRoutingEnabled {
		return nil, fmt.Errorf("order routing is disabled")
	}

	return c.orderRouter.RouteOrder(order)
}

// SubscribeMarketData subscribes to market data for symbols
func (c *UnifiedExchangeConnector) SubscribeMarketData(symbols []string, handler MarketDataHandler) error {
	if !c.config.MarketDataEnabled {
		return fmt.Errorf("market data is disabled")
	}

	return c.marketData.Subscribe(symbols, handler)
}

// processMarketData processes incoming market data messages
func (c *UnifiedExchangeConnector) processMarketData(ctx context.Context) {
	for {
		select {
		case message := <-c.marketData.feedBuffer:
			c.marketData.processMessage(message)
			atomic.AddInt64(&c.metrics.MarketDataMessages, 1)
		case <-ctx.Done():
			return
		case <-c.stopChannel:
			return
		}
	}
}

// monitorConnections monitors exchange connections and handles reconnections
func (c *UnifiedExchangeConnector) monitorConnections(ctx context.Context) {
	ticker := time.NewTicker(c.config.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.checkConnections()
		case <-ctx.Done():
			return
		case <-c.stopChannel:
			return
		}
	}
}

// checkConnections checks the status of all exchange connections
func (c *UnifiedExchangeConnector) checkConnections() {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for name, adapter := range c.exchanges {
		if !adapter.IsConnected() {
			c.logger.Warn("Exchange connection lost", zap.String("exchange", name))
			atomic.AddInt64(&c.metrics.ConnectionErrors, 1)

			// Attempt reconnection
			go c.reconnectExchange(name, adapter)
		}
	}
}

// reconnectExchange attempts to reconnect to a disconnected exchange
func (c *UnifiedExchangeConnector) reconnectExchange(name string, adapter ExchangeAdapter) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	c.logger.Info("Attempting to reconnect to exchange", zap.String("exchange", name))

	if err := adapter.Connect(ctx); err != nil {
		c.logger.Error("Failed to reconnect to exchange",
			zap.String("exchange", name),
			zap.Error(err))
		atomic.AddInt64(&c.metrics.ConnectionErrors, 1)
	} else {
		c.logger.Info("Successfully reconnected to exchange", zap.String("exchange", name))
		atomic.AddInt64(&c.metrics.ReconnectCount, 1)
	}
}

// updateMetrics updates connectivity metrics
func (c *UnifiedExchangeConnector) updateMetrics(latency time.Duration) {
	if latency > c.metrics.MaxLatency {
		c.metrics.MaxLatency = latency
	}

	// Simple moving average
	c.metrics.AverageLatency = (c.metrics.AverageLatency + latency) / 2
	c.metrics.LastUpdateTime = time.Now()
	atomic.AddInt64(&c.metrics.TotalMessages, 1)
}

// GetMetrics returns current connectivity metrics
func (c *UnifiedExchangeConnector) GetMetrics() *ConnectivityMetrics {
	return c.metrics
}

// GetExchangeStatus returns the status of all exchanges
func (c *UnifiedExchangeConnector) GetExchangeStatus() map[string]bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	status := make(map[string]bool)
	for name, adapter := range c.exchanges {
		status[name] = adapter.IsConnected()
	}
	return status
}

// RouteOrder routes an order to the appropriate exchange
func (r *OrderRouter) RouteOrder(order *types.Order) (*OrderResponse, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Determine target exchange (simplified routing logic)
	exchangeName := r.routingRules[order.Symbol]
	if exchangeName == "" {
		// Default to first available exchange
		for name, adapter := range r.exchanges {
			if adapter.IsConnected() {
				exchangeName = name
				break
			}
		}
	}

	if exchangeName == "" {
		return nil, fmt.Errorf("no available exchange for symbol %s", order.Symbol)
	}

	adapter, exists := r.exchanges[exchangeName]
	if !exists {
		return nil, fmt.Errorf("exchange %s not found", exchangeName)
	}

	if !adapter.IsConnected() {
		return nil, fmt.Errorf("exchange %s is not connected", exchangeName)
	}

	return adapter.SubmitOrder(order)
}

// SetRoutingRule sets a routing rule for a symbol
func (r *OrderRouter) SetRoutingRule(symbol, exchange string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.routingRules[symbol] = exchange
}

// Subscribe subscribes to market data for symbols
func (m *MarketDataManager) Subscribe(symbols []string, handler MarketDataHandler) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, symbol := range symbols {
		m.subscribers[symbol] = append(m.subscribers[symbol], handler)
	}

	m.logger.Info("Subscribed to market data",
		zap.Strings("symbols", symbols),
		zap.Int("handler_count", len(symbols)))

	return nil
}

// processMessage processes a market data message
func (m *MarketDataManager) processMessage(message *MarketDataMessage) {
	m.mu.RLock()
	handlers := m.subscribers[message.Symbol]
	m.mu.RUnlock()

	for _, handler := range handlers {
		if err := handler.HandleMarketData(message); err != nil {
			m.logger.Error("Market data handler failed",
				zap.String("symbol", message.Symbol),
				zap.Error(err))
		}
	}
}

// PublishMarketData publishes market data to subscribers
func (m *MarketDataManager) PublishMarketData(message *MarketDataMessage) {
	select {
	case m.feedBuffer <- message:
		// Successfully queued
	default:
		m.logger.Warn("Market data buffer full, dropping message",
			zap.String("symbol", message.Symbol),
			zap.String("type", string(message.Type)))
	}
}
