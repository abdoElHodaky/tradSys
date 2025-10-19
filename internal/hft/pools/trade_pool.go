package pools

import (
	"sync"
	"time"
)

// Trade represents a trade execution
type Trade struct {
	ID          string    `json:"id"`
	Symbol      string    `json:"symbol"`
	Price       float64   `json:"price"`
	Quantity    float64   `json:"quantity"`
	BuyOrderID  string    `json:"buy_order_id"`
	SellOrderID string    `json:"sell_order_id"`
	Timestamp   time.Time `json:"timestamp"`
	TakerSide   string    `json:"taker_side"`
	MakerSide   string    `json:"maker_side"`
	TakerFee    float64   `json:"taker_fee"`
	MakerFee    float64   `json:"maker_fee"`
}

// Reset resets the Trade to zero values
func (t *Trade) Reset() {
	t.ID = ""
	t.Symbol = ""
	t.Price = 0
	t.Quantity = 0
	t.BuyOrderID = ""
	t.SellOrderID = ""
	t.Timestamp = time.Time{}
	t.TakerSide = ""
	t.MakerSide = ""
	t.TakerFee = 0
	t.MakerFee = 0
}

// TradePool manages a pool of Trade objects to reduce GC pressure
type TradePool struct {
	pool sync.Pool
	size int
}

// NewTradePool creates a new trade pool with specified initial size
func NewTradePool(initialSize int) *TradePool {
	tp := &TradePool{
		size: initialSize,
		pool: sync.Pool{
			New: func() interface{} {
				return &Trade{}
			},
		},
	}
	
	// Pre-populate the pool
	for i := 0; i < initialSize; i++ {
		tp.pool.Put(&Trade{})
	}
	
	return tp
}

// Get retrieves a Trade from the pool
func (p *TradePool) Get() *Trade {
	trade := p.pool.Get().(*Trade)
	trade.Reset()
	return trade
}

// Put returns a Trade to the pool
func (p *TradePool) Put(trade *Trade) {
	if trade != nil {
		trade.Reset()
		p.pool.Put(trade)
	}
}

// Global trade pool instance
var globalTradePool = NewTradePool(1000)

// GetTradeFromPool retrieves a Trade from the global pool
func GetTradeFromPool() *Trade {
	return globalTradePool.Get()
}

// PutTradeToPool returns a Trade to the global pool
func PutTradeToPool(trade *Trade) {
	globalTradePool.Put(trade)
}

// TradeNotification represents a pooled trade notification
type TradeNotification struct {
	TradeID     string    `json:"trade_id"`
	Symbol      string    `json:"symbol"`
	Price       float64   `json:"price"`
	Quantity    float64   `json:"quantity"`
	Side        string    `json:"side"`
	Timestamp   time.Time `json:"timestamp"`
	OrderID     string    `json:"order_id"`
	UserID      string    `json:"user_id"`
	Commission  float64   `json:"commission"`
	NetAmount   float64   `json:"net_amount"`
}

// Reset resets the TradeNotification to zero values
func (tn *TradeNotification) Reset() {
	tn.TradeID = ""
	tn.Symbol = ""
	tn.Price = 0
	tn.Quantity = 0
	tn.Side = ""
	tn.Timestamp = time.Time{}
	tn.OrderID = ""
	tn.UserID = ""
	tn.Commission = 0
	tn.NetAmount = 0
}

// TradeNotificationPool manages a pool of TradeNotification objects
type TradeNotificationPool struct {
	pool sync.Pool
}

// NewTradeNotificationPool creates a new trade notification pool
func NewTradeNotificationPool() *TradeNotificationPool {
	return &TradeNotificationPool{
		pool: sync.Pool{
			New: func() interface{} {
				return &TradeNotification{}
			},
		},
	}
}

// Get retrieves a TradeNotification from the pool
func (p *TradeNotificationPool) Get() *TradeNotification {
	notification := p.pool.Get().(*TradeNotification)
	notification.Reset()
	return notification
}

// Put returns a TradeNotification to the pool
func (p *TradeNotificationPool) Put(notification *TradeNotification) {
	if notification != nil {
		notification.Reset()
		p.pool.Put(notification)
	}
}

// Global trade notification pool
var globalTradeNotificationPool = NewTradeNotificationPool()

// GetTradeNotificationFromPool retrieves a TradeNotification from the global pool
func GetTradeNotificationFromPool() *TradeNotification {
	return globalTradeNotificationPool.Get()
}

// PutTradeNotificationToPool returns a TradeNotification to the global pool
func PutTradeNotificationToPool(notification *TradeNotification) {
	globalTradeNotificationPool.Put(notification)
}

// TradeHistory represents a pooled trade history entry
type TradeHistory struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	Symbol      string    `json:"symbol"`
	Side        string    `json:"side"`
	Price       float64   `json:"price"`
	Quantity    float64   `json:"quantity"`
	Commission  float64   `json:"commission"`
	NetAmount   float64   `json:"net_amount"`
	OrderID     string    `json:"order_id"`
	TradeID     string    `json:"trade_id"`
	Timestamp   time.Time `json:"timestamp"`
	SettledAt   *time.Time `json:"settled_at,omitempty"`
}

// Reset resets the TradeHistory to zero values
func (th *TradeHistory) Reset() {
	th.ID = ""
	th.UserID = ""
	th.Symbol = ""
	th.Side = ""
	th.Price = 0
	th.Quantity = 0
	th.Commission = 0
	th.NetAmount = 0
	th.OrderID = ""
	th.TradeID = ""
	th.Timestamp = time.Time{}
	th.SettledAt = nil
}

// TradeHistoryPool manages a pool of TradeHistory objects
type TradeHistoryPool struct {
	pool sync.Pool
}

// NewTradeHistoryPool creates a new trade history pool
func NewTradeHistoryPool() *TradeHistoryPool {
	return &TradeHistoryPool{
		pool: sync.Pool{
			New: func() interface{} {
				return &TradeHistory{}
			},
		},
	}
}

// Get retrieves a TradeHistory from the pool
func (p *TradeHistoryPool) Get() *TradeHistory {
	history := p.pool.Get().(*TradeHistory)
	history.Reset()
	return history
}

// Put returns a TradeHistory to the pool
func (p *TradeHistoryPool) Put(history *TradeHistory) {
	if history != nil {
		history.Reset()
		p.pool.Put(history)
	}
}

// Global trade history pool
var globalTradeHistoryPool = NewTradeHistoryPool()

// GetTradeHistoryFromPool retrieves a TradeHistory from the global pool
func GetTradeHistoryFromPool() *TradeHistory {
	return globalTradeHistoryPool.Get()
}

// PutTradeHistoryToPool returns a TradeHistory to the global pool
func PutTradeHistoryToPool(history *TradeHistory) {
	globalTradeHistoryPool.Put(history)
}

// BatchTradeProcessor represents a batch processor for trades
type BatchTradeProcessor struct {
	trades   []*Trade
	capacity int
	mu       sync.Mutex
}

// NewBatchTradeProcessor creates a new batch trade processor
func NewBatchTradeProcessor(capacity int) *BatchTradeProcessor {
	return &BatchTradeProcessor{
		trades:   make([]*Trade, 0, capacity),
		capacity: capacity,
	}
}

// Add adds a trade to the batch
func (btp *BatchTradeProcessor) Add(trade *Trade) bool {
	btp.mu.Lock()
	defer btp.mu.Unlock()
	
	if len(btp.trades) >= btp.capacity {
		return false // Batch is full
	}
	
	btp.trades = append(btp.trades, trade)
	return true
}

// Flush returns all trades in the batch and resets it
func (btp *BatchTradeProcessor) Flush() []*Trade {
	btp.mu.Lock()
	defer btp.mu.Unlock()
	
	if len(btp.trades) == 0 {
		return nil
	}
	
	// Create a copy of the trades
	result := make([]*Trade, len(btp.trades))
	copy(result, btp.trades)
	
	// Reset the batch
	btp.trades = btp.trades[:0]
	
	return result
}

// Size returns the current size of the batch
func (btp *BatchTradeProcessor) Size() int {
	btp.mu.Lock()
	defer btp.mu.Unlock()
	return len(btp.trades)
}

// IsFull returns whether the batch is full
func (btp *BatchTradeProcessor) IsFull() bool {
	btp.mu.Lock()
	defer btp.mu.Unlock()
	return len(btp.trades) >= btp.capacity
}

// TradeMetrics represents trade execution metrics
type TradeMetrics struct {
	TotalTrades       uint64    `json:"total_trades"`
	TotalVolume       float64   `json:"total_volume"`
	TotalValue        float64   `json:"total_value"`
	AveragePrice      float64   `json:"average_price"`
	LastTradeTime     time.Time `json:"last_trade_time"`
	TradesPerSecond   float64   `json:"trades_per_second"`
	VolumePerSecond   float64   `json:"volume_per_second"`
	ValuePerSecond    float64   `json:"value_per_second"`
	PeakTradesPerSec  float64   `json:"peak_trades_per_second"`
	PeakVolumePerSec  float64   `json:"peak_volume_per_second"`
}

// Reset resets the TradeMetrics to zero values
func (tm *TradeMetrics) Reset() {
	tm.TotalTrades = 0
	tm.TotalVolume = 0
	tm.TotalValue = 0
	tm.AveragePrice = 0
	tm.LastTradeTime = time.Time{}
	tm.TradesPerSecond = 0
	tm.VolumePerSecond = 0
	tm.ValuePerSecond = 0
	tm.PeakTradesPerSec = 0
	tm.PeakVolumePerSec = 0
}

// Update updates the metrics with a new trade
func (tm *TradeMetrics) Update(trade *Trade) {
	tm.TotalTrades++
	tm.TotalVolume += trade.Quantity
	tm.TotalValue += trade.Price * trade.Quantity
	
	if tm.TotalTrades > 0 {
		tm.AveragePrice = tm.TotalValue / tm.TotalVolume
	}
	
	tm.LastTradeTime = trade.Timestamp
	
	// Calculate rates (simplified - would need time window tracking for accuracy)
	if !tm.LastTradeTime.IsZero() {
		duration := time.Since(tm.LastTradeTime).Seconds()
		if duration > 0 {
			currentTPS := 1.0 / duration
			currentVPS := trade.Quantity / duration
			
			if currentTPS > tm.PeakTradesPerSec {
				tm.PeakTradesPerSec = currentTPS
			}
			if currentVPS > tm.PeakVolumePerSec {
				tm.PeakVolumePerSec = currentVPS
			}
		}
	}
}

// TradeMetricsPool manages a pool of TradeMetrics objects
type TradeMetricsPool struct {
	pool sync.Pool
}

// NewTradeMetricsPool creates a new trade metrics pool
func NewTradeMetricsPool() *TradeMetricsPool {
	return &TradeMetricsPool{
		pool: sync.Pool{
			New: func() interface{} {
				return &TradeMetrics{}
			},
		},
	}
}

// Get retrieves a TradeMetrics from the pool
func (p *TradeMetricsPool) Get() *TradeMetrics {
	metrics := p.pool.Get().(*TradeMetrics)
	metrics.Reset()
	return metrics
}

// Put returns a TradeMetrics to the pool
func (p *TradeMetricsPool) Put(metrics *TradeMetrics) {
	if metrics != nil {
		metrics.Reset()
		p.pool.Put(metrics)
	}
}

// Global trade metrics pool
var globalTradeMetricsPool = NewTradeMetricsPool()

// GetTradeMetricsFromPool retrieves a TradeMetrics from the global pool
func GetTradeMetricsFromPool() *TradeMetrics {
	return globalTradeMetricsPool.Get()
}

// PutTradeMetricsToPool returns a TradeMetrics to the global pool
func PutTradeMetricsToPool(metrics *TradeMetrics) {
	globalTradeMetricsPool.Put(metrics)
}

