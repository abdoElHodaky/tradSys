package websocket

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/trading/market_data"
	"go.uber.org/zap"
)

// MarketDataHandler handles WebSocket connections for market data
type MarketDataHandler struct {
	hub                *Hub
	logger             *zap.Logger
	marketDataService  *market_data.MarketDataService
	subscriptions      map[string]map[string]bool // Map of client ID to subscribed symbols
	subscriptionsMu    sync.RWMutex
	throttleInterval   time.Duration
	lastUpdateTime     map[string]time.Time // Map of symbol to last update time
	lastUpdateTimeMu   sync.RWMutex
}

// MarketDataMessage represents a market data message
type MarketDataMessage struct {
	Action string   `json:"action"`
	Symbol string   `json:"symbol,omitempty"`
	Symbols []string `json:"symbols,omitempty"`
}

// MarketDataUpdate represents a market data update
type MarketDataUpdate struct {
	Symbol    string    `json:"symbol"`
	Price     float64   `json:"price"`
	Bid       float64   `json:"bid"`
	Ask       float64   `json:"ask"`
	Volume    float64   `json:"volume"`
	Timestamp time.Time `json:"timestamp"`
}

// NewMarketDataHandler creates a new market data handler
func NewMarketDataHandler(
	hub *Hub,
	logger *zap.Logger,
	marketDataService *market_data.MarketDataService,
) *MarketDataHandler {
	handler := &MarketDataHandler{
		hub:               hub,
		logger:            logger,
		marketDataService: marketDataService,
		subscriptions:     make(map[string]map[string]bool),
		throttleInterval:  100 * time.Millisecond,
		lastUpdateTime:    make(map[string]time.Time),
	}

	// Register message handlers
	hub.RegisterMessageHandler("marketdata.subscribe", handler.handleSubscribe)
	hub.RegisterMessageHandler("marketdata.unsubscribe", handler.handleUnsubscribe)
	hub.RegisterMessageHandler("marketdata.get", handler.handleGet)

	// Start the market data update goroutine
	go handler.processMarketDataUpdates()

	return handler
}

// handleSubscribe handles a market data subscription request
func (h *MarketDataHandler) handleSubscribe(client *Client, msg *Message) {
	// Parse the subscription request
	var request MarketDataMessage
	if err := json.Unmarshal(msg.Data, &request); err != nil {
		h.logger.Error("Failed to parse market data subscription request",
			zap.Error(err),
			zap.String("client_id", client.ID))
		return
	}

	h.logger.Debug("Market data subscription request",
		zap.String("client_id", client.ID),
		zap.String("symbol", request.Symbol),
		zap.Strings("symbols", request.Symbols))

	// Handle single symbol subscription
	if request.Symbol != "" {
		h.subscribeToSymbol(client.ID, request.Symbol)
	}

	// Handle multiple symbol subscription
	for _, symbol := range request.Symbols {
		h.subscribeToSymbol(client.ID, symbol)
	}

	// Send confirmation
	response := Message{
		Type: "marketdata.subscribed",
		Data: json.RawMessage(`{"status":"success"}`),
	}
	client.SendMessage(&response)
}

// handleUnsubscribe handles a market data unsubscription request
func (h *MarketDataHandler) handleUnsubscribe(client *Client, msg *Message) {
	// Parse the unsubscription request
	var request MarketDataMessage
	if err := json.Unmarshal(msg.Data, &request); err != nil {
		h.logger.Error("Failed to parse market data unsubscription request",
			zap.Error(err),
			zap.String("client_id", client.ID))
		return
	}

	h.logger.Debug("Market data unsubscription request",
		zap.String("client_id", client.ID),
		zap.String("symbol", request.Symbol),
		zap.Strings("symbols", request.Symbols))

	// Handle single symbol unsubscription
	if request.Symbol != "" {
		h.unsubscribeFromSymbol(client.ID, request.Symbol)
	}

	// Handle multiple symbol unsubscription
	for _, symbol := range request.Symbols {
		h.unsubscribeFromSymbol(client.ID, symbol)
	}

	// Send confirmation
	response := Message{
		Type: "marketdata.unsubscribed",
		Data: json.RawMessage(`{"status":"success"}`),
	}
	client.SendMessage(&response)
}

// handleGet handles a market data get request
func (h *MarketDataHandler) handleGet(client *Client, msg *Message) {
	// Parse the get request
	var request MarketDataMessage
	if err := json.Unmarshal(msg.Data, &request); err != nil {
		h.logger.Error("Failed to parse market data get request",
			zap.Error(err),
			zap.String("client_id", client.ID))
		return
	}

	h.logger.Debug("Market data get request",
		zap.String("client_id", client.ID),
		zap.String("symbol", request.Symbol))

	// Get the market data
	if request.Symbol == "" {
		h.logger.Error("Missing symbol in market data get request",
			zap.String("client_id", client.ID))
		return
	}

	// Get the latest market data
	data, err := h.marketDataService.GetLatestMarketData(request.Symbol)
	if err != nil {
		h.logger.Error("Failed to get market data",
			zap.Error(err),
			zap.String("client_id", client.ID),
			zap.String("symbol", request.Symbol))
		return
	}

	// Create the update
	update := MarketDataUpdate{
		Symbol:    data.Symbol,
		Price:     data.Price,
		Bid:       data.Bid,
		Ask:       data.Ask,
		Volume:    data.Volume,
		Timestamp: data.Timestamp,
	}

	// Send the update
	response := Message{
		Type: "marketdata.update",
		Data: h.serializeUpdate(update),
	}
	client.SendMessage(&response)
}

// subscribeToSymbol subscribes a client to a symbol
func (h *MarketDataHandler) subscribeToSymbol(clientID, symbol string) {
	h.subscriptionsMu.Lock()
	defer h.subscriptionsMu.Unlock()

	// Initialize the client's subscriptions if needed
	if _, ok := h.subscriptions[clientID]; !ok {
		h.subscriptions[clientID] = make(map[string]bool)
	}

	// Subscribe to the symbol
	h.subscriptions[clientID][symbol] = true

	// Subscribe to the market data service if this is the first client
	isFirstSubscriber := true
	for cid, symbols := range h.subscriptions {
		if cid != clientID && symbols[symbol] {
			isFirstSubscriber = false
			break
		}
	}

	if isFirstSubscriber {
		h.marketDataService.Subscribe(symbol)
	}
}

// unsubscribeFromSymbol unsubscribes a client from a symbol
func (h *MarketDataHandler) unsubscribeFromSymbol(clientID, symbol string) {
	h.subscriptionsMu.Lock()
	defer h.subscriptionsMu.Unlock()

	// Check if the client is subscribed
	if _, ok := h.subscriptions[clientID]; !ok {
		return
	}

	// Unsubscribe from the symbol
	delete(h.subscriptions[clientID], symbol)

	// Remove the client if they have no more subscriptions
	if len(h.subscriptions[clientID]) == 0 {
		delete(h.subscriptions, clientID)
	}

	// Check if there are any other clients subscribed to this symbol
	hasOtherSubscribers := false
	for _, symbols := range h.subscriptions {
		if symbols[symbol] {
			hasOtherSubscribers = true
			break
		}
	}

	// Unsubscribe from the market data service if there are no more clients
	if !hasOtherSubscribers {
		h.marketDataService.Unsubscribe(symbol)
	}
}

// processMarketDataUpdates processes market data updates
func (h *MarketDataHandler) processMarketDataUpdates() {
	// Subscribe to market data updates
	updates := h.marketDataService.GetUpdateChannel()

	for update := range updates {
		// Check if we should throttle this update
		if h.shouldThrottleUpdate(update.Symbol) {
			continue
		}

		// Update the last update time
		h.updateLastUpdateTime(update.Symbol)

		// Create the update
		wsUpdate := MarketDataUpdate{
			Symbol:    update.Symbol,
			Price:     update.Price,
			Bid:       update.Bid,
			Ask:       update.Ask,
			Volume:    update.Volume,
			Timestamp: update.Timestamp,
		}

		// Get clients subscribed to this symbol
		clients := h.getSubscribedClients(update.Symbol)

		// Send the update to subscribed clients
		if len(clients) > 0 {
			message := Message{
				Type: "marketdata.update",
				Data: h.serializeUpdate(wsUpdate),
			}

			for _, clientID := range clients {
				h.hub.SendToClient(clientID, &message)
			}
		}
	}
}

// shouldThrottleUpdate checks if an update should be throttled
func (h *MarketDataHandler) shouldThrottleUpdate(symbol string) bool {
	h.lastUpdateTimeMu.RLock()
	lastUpdate, ok := h.lastUpdateTime[symbol]
	h.lastUpdateTimeMu.RUnlock()

	if !ok {
		return false
	}

	return time.Since(lastUpdate) < h.throttleInterval
}

// updateLastUpdateTime updates the last update time for a symbol
func (h *MarketDataHandler) updateLastUpdateTime(symbol string) {
	h.lastUpdateTimeMu.Lock()
	h.lastUpdateTime[symbol] = time.Now()
	h.lastUpdateTimeMu.Unlock()
}

// getSubscribedClients gets the clients subscribed to a symbol
func (h *MarketDataHandler) getSubscribedClients(symbol string) []string {
	h.subscriptionsMu.RLock()
	defer h.subscriptionsMu.RUnlock()

	var clients []string
	for clientID, symbols := range h.subscriptions {
		if symbols[symbol] {
			clients = append(clients, clientID)
		}
	}

	return clients
}

// serializeUpdate serializes a market data update
func (h *MarketDataHandler) serializeUpdate(update MarketDataUpdate) json.RawMessage {
	data, err := json.Marshal(update)
	if err != nil {
		h.logger.Error("Failed to serialize market data update", zap.Error(err))
		return json.RawMessage("{}")
	}

	return data
}

// GetSubscribedSymbols gets the symbols a client is subscribed to
func (h *MarketDataHandler) GetSubscribedSymbols(clientID string) []string {
	h.subscriptionsMu.RLock()
	defer h.subscriptionsMu.RUnlock()

	symbols, ok := h.subscriptions[clientID]
	if !ok {
		return nil
	}

	result := make([]string, 0, len(symbols))
	for symbol := range symbols {
		result = append(result, symbol)
	}

	return result
}

// GetSubscribedClients gets the clients subscribed to a symbol
func (h *MarketDataHandler) GetSubscribedClients(symbol string) []string {
	return h.getSubscribedClients(symbol)
}

// SetThrottleInterval sets the throttle interval
func (h *MarketDataHandler) SetThrottleInterval(interval time.Duration) {
	h.throttleInterval = interval
}

