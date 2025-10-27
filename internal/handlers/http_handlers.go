package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/services"
	"github.com/abdoElHodaky/tradSys/pkg/errors"
	"github.com/abdoElHodaky/tradSys/pkg/interfaces"
	"github.com/abdoElHodaky/tradSys/pkg/types"
	"github.com/abdoElHodaky/tradSys/pkg/utils"
)

// HTTPHandlers provides HTTP API handlers for the trading system
type HTTPHandlers struct {
	registry    *services.ServiceRegistry
	logger      interfaces.Logger
	metrics     interfaces.MetricsCollector
	performance *utils.PerformanceMonitor
}

// NewHTTPHandlers creates new HTTP handlers
func NewHTTPHandlers(
	registry *services.ServiceRegistry,
	logger interfaces.Logger,
	metrics interfaces.MetricsCollector,
) *HTTPHandlers {
	return &HTTPHandlers{
		registry:    registry,
		logger:      logger,
		metrics:     metrics,
		performance: utils.NewPerformanceMonitor(metrics, logger),
	}
}

// RegisterRoutes registers all HTTP routes
func (h *HTTPHandlers) RegisterRoutes(mux *http.ServeMux) {
	// Order endpoints
	mux.HandleFunc("/api/v1/orders", h.handleOrders)
	mux.HandleFunc("/api/v1/orders/", h.handleOrderByID)

	// Trade endpoints
	mux.HandleFunc("/api/v1/trades", h.handleTrades)
	mux.HandleFunc("/api/v1/trades/", h.handleTradeByID)

	// Market data endpoints
	mux.HandleFunc("/api/v1/market-data/", h.handleMarketData)
	mux.HandleFunc("/api/v1/ohlcv/", h.handleOHLCV)
	mux.HandleFunc("/api/v1/symbols", h.handleSymbols)

	// System endpoints
	mux.HandleFunc("/api/v1/health", h.handleHealth)
	mux.HandleFunc("/api/v1/metrics", h.handleMetrics)

	// WebSocket endpoint for real-time data
	mux.HandleFunc("/ws", h.handleWebSocket)
}

// Order Handlers

func (h *HTTPHandlers) handleOrders(w http.ResponseWriter, r *http.Request) {
	err := h.performance.TrackRequest(r.Context(), "handle_orders", func() error {
		switch r.Method {
		case http.MethodPost:
			return h.createOrder(w, r)
		case http.MethodGet:
			return h.listOrders(w, r)
		default:
			return h.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	})

	if err != nil {
		h.logger.Error("Request failed", "error", err, "path", r.URL.Path, "method", r.Method)
	}
}

func (h *HTTPHandlers) handleOrderByID(w http.ResponseWriter, r *http.Request) {
	err := h.performance.TrackRequest(r.Context(), "handle_order_by_id", func() error {
		orderID := extractIDFromPath(r.URL.Path, "/api/v1/orders/")
		if orderID == "" {
			return h.writeError(w, http.StatusBadRequest, "invalid order ID")
		}

		switch r.Method {
		case http.MethodGet:
			return h.getOrder(w, r, orderID)
		case http.MethodDelete:
			return h.cancelOrder(w, r, orderID)
		default:
			return h.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	})

	if err != nil {
		h.logger.Error("Request failed", "error", err, "path", r.URL.Path, "method", r.Method)
	}
}

func (h *HTTPHandlers) createOrder(w http.ResponseWriter, r *http.Request) error {
	var orderRequest CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&orderRequest); err != nil {
		return h.writeError(w, http.StatusBadRequest, "invalid JSON")
	}

	// Convert request to domain object
	order := &types.Order{
		ID:            generateOrderID(),
		ClientOrderID: orderRequest.ClientOrderID,
		UserID:        orderRequest.UserID,
		Symbol:        orderRequest.Symbol,
		Side:          types.OrderSide(orderRequest.Side),
		Type:          types.OrderType(orderRequest.Type),
		Price:         orderRequest.Price,
		Quantity:      orderRequest.Quantity,
		TimeInForce:   types.TimeInForce(orderRequest.TimeInForce),
		StopPrice:     orderRequest.StopPrice,
		Status:        types.OrderStatusPending,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// Set remaining quantity
	order.RemainingQuantity = order.Quantity

	// Process through matching engine
	matchingEngine := h.registry.GetMatchingEngine()
	if matchingEngine == nil {
		return h.writeError(w, http.StatusServiceUnavailable, "matching engine unavailable")
	}

	trades, err := matchingEngine.ProcessOrder(r.Context(), order)
	if err != nil {
		return h.writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to process order: %v", err))
	}

	// Create response
	response := CreateOrderResponse{
		Order:  convertOrderToResponse(order),
		Trades: convertTradesToResponse(trades),
	}

	return h.writeJSON(w, http.StatusCreated, response)
}

func (h *HTTPHandlers) getOrder(w http.ResponseWriter, r *http.Request, orderID string) error {
	orderService := h.registry.GetOrderService()
	if orderService == nil {
		return h.writeError(w, http.StatusServiceUnavailable, "order service unavailable")
	}

	order, err := orderService.GetOrder(r.Context(), orderID)
	if err != nil {
		if errors.Is(err, errors.ErrOrderNotFound) {
			return h.writeError(w, http.StatusNotFound, "order not found")
		}
		return h.writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get order: %v", err))
	}

	return h.writeJSON(w, http.StatusOK, convertOrderToResponse(order))
}

func (h *HTTPHandlers) listOrders(w http.ResponseWriter, r *http.Request) error {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		return h.writeError(w, http.StatusBadRequest, "user_id parameter required")
	}

	// Parse pagination parameters
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 1000 {
		limit = 100
	}

	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if offset < 0 {
		offset = 0
	}

	// Parse filters
	filters := &interfaces.OrderFilters{
		Symbol: r.URL.Query().Get("symbol"),
		Side:   types.OrderSide(r.URL.Query().Get("side")),
		Status: types.OrderStatus(r.URL.Query().Get("status")),
		Limit:  limit,
		Offset: offset,
	}

	orderService := h.registry.GetOrderService()
	if orderService == nil {
		return h.writeError(w, http.StatusServiceUnavailable, "order service unavailable")
	}

	orders, err := orderService.ListOrders(r.Context(), userID, filters)
	if err != nil {
		return h.writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to list orders: %v", err))
	}

	response := ListOrdersResponse{
		Orders: make([]OrderResponse, len(orders)),
		Total:  len(orders),
		Limit:  limit,
		Offset: offset,
	}

	for i, order := range orders {
		response.Orders[i] = convertOrderToResponse(order)
	}

	return h.writeJSON(w, http.StatusOK, response)
}

func (h *HTTPHandlers) cancelOrder(w http.ResponseWriter, r *http.Request, orderID string) error {
	matchingEngine := h.registry.GetMatchingEngine()
	if matchingEngine == nil {
		return h.writeError(w, http.StatusServiceUnavailable, "matching engine unavailable")
	}

	err := matchingEngine.CancelOrder(r.Context(), orderID)
	if err != nil {
		if errors.Is(err, errors.ErrOrderNotFound) {
			return h.writeError(w, http.StatusNotFound, "order not found")
		}
		return h.writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to cancel order: %v", err))
	}

	return h.writeJSON(w, http.StatusOK, map[string]string{"status": "cancelled"})
}

// Trade Handlers

func (h *HTTPHandlers) handleTrades(w http.ResponseWriter, r *http.Request) {
	err := h.performance.TrackRequest(r.Context(), "handle_trades", func() error {
		switch r.Method {
		case http.MethodGet:
			return h.listTrades(w, r)
		default:
			return h.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	})

	if err != nil {
		h.logger.Error("Request failed", "error", err, "path", r.URL.Path, "method", r.Method)
	}
}

func (h *HTTPHandlers) handleTradeByID(w http.ResponseWriter, r *http.Request) {
	err := h.performance.TrackRequest(r.Context(), "handle_trade_by_id", func() error {
		tradeID := extractIDFromPath(r.URL.Path, "/api/v1/trades/")
		if tradeID == "" {
			return h.writeError(w, http.StatusBadRequest, "invalid trade ID")
		}

		switch r.Method {
		case http.MethodGet:
			return h.getTrade(w, r, tradeID)
		default:
			return h.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	})

	if err != nil {
		h.logger.Error("Request failed", "error", err, "path", r.URL.Path, "method", r.Method)
	}
}

func (h *HTTPHandlers) listTrades(w http.ResponseWriter, r *http.Request) error {
	// Parse filters
	filters := &interfaces.TradeFilters{
		Symbol: r.URL.Query().Get("symbol"),
		UserID: r.URL.Query().Get("user_id"),
	}

	// Parse pagination
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	filters.Limit = limit

	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if offset < 0 {
		offset = 0
	}
	filters.Offset = offset

	tradeService := h.registry.GetTradeService()
	if tradeService == nil {
		return h.writeError(w, http.StatusServiceUnavailable, "trade service unavailable")
	}

	trades, err := tradeService.ListTrades(r.Context(), filters)
	if err != nil {
		return h.writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to list trades: %v", err))
	}

	response := ListTradesResponse{
		Trades: convertTradesToResponse(trades),
		Total:  len(trades),
		Limit:  limit,
		Offset: offset,
	}

	return h.writeJSON(w, http.StatusOK, response)
}

func (h *HTTPHandlers) getTrade(w http.ResponseWriter, r *http.Request, tradeID string) error {
	tradeService := h.registry.GetTradeService()
	if tradeService == nil {
		return h.writeError(w, http.StatusServiceUnavailable, "trade service unavailable")
	}

	trade, err := tradeService.GetTrade(r.Context(), tradeID)
	if err != nil {
		if errors.Is(err, errors.ErrOrderNotFound) {
			return h.writeError(w, http.StatusNotFound, "trade not found")
		}
		return h.writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get trade: %v", err))
	}

	return h.writeJSON(w, http.StatusOK, convertTradeToResponse(trade))
}

// Market Data Handlers

func (h *HTTPHandlers) handleMarketData(w http.ResponseWriter, r *http.Request) {
	err := h.performance.TrackRequest(r.Context(), "handle_market_data", func() error {
		symbol := extractIDFromPath(r.URL.Path, "/api/v1/market-data/")
		if symbol == "" {
			return h.writeError(w, http.StatusBadRequest, "invalid symbol")
		}

		switch r.Method {
		case http.MethodGet:
			return h.getMarketData(w, r, symbol)
		default:
			return h.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	})

	if err != nil {
		h.logger.Error("Request failed", "error", err, "path", r.URL.Path, "method", r.Method)
	}
}

func (h *HTTPHandlers) getMarketData(w http.ResponseWriter, r *http.Request, symbol string) error {
	marketDataService := h.registry.GetMarketDataService()
	if marketDataService == nil {
		return h.writeError(w, http.StatusServiceUnavailable, "market data service unavailable")
	}

	marketData, err := marketDataService.GetMarketData(r.Context(), symbol)
	if err != nil {
		if errors.Is(err, errors.ErrSymbolNotFound) {
			return h.writeError(w, http.StatusNotFound, "symbol not found")
		}
		return h.writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get market data: %v", err))
	}

	return h.writeJSON(w, http.StatusOK, marketData)
}

func (h *HTTPHandlers) handleOHLCV(w http.ResponseWriter, r *http.Request) {
	err := h.performance.TrackRequest(r.Context(), "handle_ohlcv", func() error {
		symbol := extractIDFromPath(r.URL.Path, "/api/v1/ohlcv/")
		if symbol == "" {
			return h.writeError(w, http.StatusBadRequest, "invalid symbol")
		}

		switch r.Method {
		case http.MethodGet:
			return h.getOHLCV(w, r, symbol)
		default:
			return h.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	})

	if err != nil {
		h.logger.Error("Request failed", "error", err, "path", r.URL.Path, "method", r.Method)
	}
}

func (h *HTTPHandlers) getOHLCV(w http.ResponseWriter, r *http.Request, symbol string) error {
	interval := r.URL.Query().Get("interval")
	if interval == "" {
		interval = "1m"
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 1000 {
		limit = 100
	}

	marketDataService := h.registry.GetMarketDataService()
	if marketDataService == nil {
		return h.writeError(w, http.StatusServiceUnavailable, "market data service unavailable")
	}

	ohlcvData, err := marketDataService.GetOHLCV(r.Context(), symbol, interval, limit)
	if err != nil {
		if errors.Is(err, errors.ErrSymbolNotFound) {
			return h.writeError(w, http.StatusNotFound, "symbol not found")
		}
		return h.writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get OHLCV data: %v", err))
	}

	return h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"symbol":   symbol,
		"interval": interval,
		"data":     ohlcvData,
	})
}

func (h *HTTPHandlers) handleSymbols(w http.ResponseWriter, r *http.Request) {
	err := h.performance.TrackRequest(r.Context(), "handle_symbols", func() error {
		switch r.Method {
		case http.MethodGet:
			return h.getSymbols(w, r)
		default:
			return h.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	})

	if err != nil {
		h.logger.Error("Request failed", "error", err, "path", r.URL.Path, "method", r.Method)
	}
}

func (h *HTTPHandlers) getSymbols(w http.ResponseWriter, r *http.Request) error {
	marketDataService := h.registry.GetMarketDataService()
	if marketDataService == nil {
		return h.writeError(w, http.StatusServiceUnavailable, "market data service unavailable")
	}

	symbols, err := marketDataService.GetSymbols(r.Context())
	if err != nil {
		return h.writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get symbols: %v", err))
	}

	return h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"symbols": symbols,
		"count":   len(symbols),
	})
}

// System Handlers

func (h *HTTPHandlers) handleHealth(w http.ResponseWriter, r *http.Request) {
	err := h.performance.TrackRequest(r.Context(), "handle_health", func() error {
		ctx := context.WithValue(r.Context(), "timestamp", time.Now())
		healthStatus := h.registry.GetHealthStatus(ctx)

		statusCode := http.StatusOK
		if healthStatus.Overall != interfaces.HealthStatusHealthy {
			statusCode = http.StatusServiceUnavailable
		}

		return h.writeJSON(w, statusCode, healthStatus)
	})

	if err != nil {
		h.logger.Error("Health check failed", "error", err)
	}
}

func (h *HTTPHandlers) handleMetrics(w http.ResponseWriter, r *http.Request) {
	err := h.performance.TrackRequest(r.Context(), "handle_metrics", func() error {
		stats := h.performance.GetStatistics()
		serviceStats := h.registry.GetServiceStatistics()

		response := map[string]interface{}{
			"performance": stats,
			"services":    serviceStats,
		}

		return h.writeJSON(w, http.StatusOK, response)
	})

	if err != nil {
		h.logger.Error("Metrics request failed", "error", err)
	}
}

// WebSocket Handler (placeholder)
func (h *HTTPHandlers) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// WebSocket implementation would go here
	// This is a placeholder for real-time data streaming
	h.writeError(w, http.StatusNotImplemented, "WebSocket not implemented yet")
}

// Helper functions

func (h *HTTPHandlers) writeJSON(w http.ResponseWriter, statusCode int, data interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	return json.NewEncoder(w).Encode(data)
}

func (h *HTTPHandlers) writeError(w http.ResponseWriter, statusCode int, message string) error {
	return h.writeJSON(w, statusCode, map[string]string{"error": message})
}

// Note: Functions and types moved to common.go to avoid duplication
