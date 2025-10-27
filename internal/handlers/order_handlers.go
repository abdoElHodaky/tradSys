package handlers

import (
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

// OrderHandlers handles order-related HTTP requests.
type OrderHandlers struct {
	registry    *services.ServiceRegistry
	logger      interfaces.Logger
	metrics     interfaces.MetricsCollector
	performance *utils.PerformanceMonitor
}

// NewOrderHandlers creates new order handlers.
func NewOrderHandlers(
	registry *services.ServiceRegistry,
	logger interfaces.Logger,
	metrics interfaces.MetricsCollector,
) *OrderHandlers {
	return &OrderHandlers{
		registry:    registry,
		logger:      logger,
		metrics:     metrics,
		performance: utils.NewPerformanceMonitor(metrics, logger),
	}
}

// HandleOrders handles order collection endpoints.
func (h *OrderHandlers) HandleOrders(w http.ResponseWriter, r *http.Request) {
	err := h.performance.TrackRequest(r.Context(), "handle_orders", func() error {
		switch r.Method {
		case http.MethodPost:
			return h.createOrder(w, r)
		case http.MethodGet:
			return h.listOrders(w, r)
		default:
			return writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	})
	
	if err != nil {
		h.logger.Error("Request failed", "error", err, "path", r.URL.Path, "method", r.Method)
	}
}

// HandleOrderByID handles individual order endpoints.
func (h *OrderHandlers) HandleOrderByID(w http.ResponseWriter, r *http.Request) {
	err := h.performance.TrackRequest(r.Context(), "handle_order_by_id", func() error {
		orderID := extractIDFromPath(r.URL.Path, "/api/v1/orders/")
		if orderID == "" {
			return writeError(w, http.StatusBadRequest, "invalid order ID")
		}

		switch r.Method {
		case http.MethodGet:
			return h.getOrder(w, r, orderID)
		case http.MethodDelete:
			return h.cancelOrder(w, r, orderID)
		default:
			return writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	})
	
	if err != nil {
		h.logger.Error("Request failed", "error", err, "path", r.URL.Path, "method", r.Method)
	}
}

func (h *OrderHandlers) createOrder(w http.ResponseWriter, r *http.Request) error {
	var orderRequest CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&orderRequest); err != nil {
		return writeError(w, http.StatusBadRequest, "invalid JSON")
	}

	// Validate request
	if err := h.validateOrderRequest(&orderRequest); err != nil {
		return writeError(w, http.StatusBadRequest, err.Error())
	}

	// Convert request to domain object
	order := &types.Order{
		ID:                generateOrderID(),
		ClientOrderID:     orderRequest.ClientOrderID,
		UserID:            orderRequest.UserID,
		Symbol:            orderRequest.Symbol,
		Side:              types.OrderSide(orderRequest.Side),
		Type:              types.OrderType(orderRequest.Type),
		Price:             orderRequest.Price,
		Quantity:          orderRequest.Quantity,
		RemainingQuantity: orderRequest.Quantity,
		TimeInForce:       types.TimeInForce(orderRequest.TimeInForce),
		StopPrice:         orderRequest.StopPrice,
		Status:            types.OrderStatusPending,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	// Process through matching engine
	matchingEngine := h.registry.GetMatchingEngine()
	if matchingEngine == nil {
		return writeError(w, http.StatusServiceUnavailable, "matching engine unavailable")
	}

	trades, err := matchingEngine.ProcessOrder(r.Context(), order)
	if err != nil {
		return writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to process order: %v", err))
	}

	// Create response
	response := CreateOrderResponse{
		Order:  convertOrderToResponse(order),
		Trades: convertTradesToResponse(trades),
	}

	return writeJSON(w, http.StatusCreated, response)
}

func (h *OrderHandlers) getOrder(w http.ResponseWriter, r *http.Request, orderID string) error {
	orderService := h.registry.GetOrderService()
	if orderService == nil {
		return writeError(w, http.StatusServiceUnavailable, "order service unavailable")
	}

	order, err := orderService.GetOrder(r.Context(), orderID)
	if err != nil {
		if errors.Is(err, errors.ErrOrderNotFound) {
			return writeError(w, http.StatusNotFound, "order not found")
		}
		return writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get order: %v", err))
	}

	return writeJSON(w, http.StatusOK, convertOrderToResponse(order))
}

func (h *OrderHandlers) listOrders(w http.ResponseWriter, r *http.Request) error {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		return writeError(w, http.StatusBadRequest, "user_id parameter required")
	}

	// Parse pagination parameters
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > maxPageSize {
		limit = defaultPageSize
	}

	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if offset < 0 {
		offset = 0
	}

	// Parse filters
	filters := &interfaces.OrderFilters{
		Symbol:  r.URL.Query().Get("symbol"),
		Side:    types.OrderSide(r.URL.Query().Get("side")),
		Status:  types.OrderStatus(r.URL.Query().Get("status")),
		Limit:   limit,
		Offset:  offset,
	}

	orderService := h.registry.GetOrderService()
	if orderService == nil {
		return writeError(w, http.StatusServiceUnavailable, "order service unavailable")
	}

	orders, err := orderService.ListOrders(r.Context(), userID, filters)
	if err != nil {
		return writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to list orders: %v", err))
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

	return writeJSON(w, http.StatusOK, response)
}

func (h *OrderHandlers) cancelOrder(w http.ResponseWriter, r *http.Request, orderID string) error {
	matchingEngine := h.registry.GetMatchingEngine()
	if matchingEngine == nil {
		return writeError(w, http.StatusServiceUnavailable, "matching engine unavailable")
	}

	err := matchingEngine.CancelOrder(r.Context(), orderID)
	if err != nil {
		if errors.Is(err, errors.ErrOrderNotFound) {
			return writeError(w, http.StatusNotFound, "order not found")
		}
		return writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to cancel order: %v", err))
	}

	return writeJSON(w, http.StatusOK, map[string]string{"status": "cancelled"})
}

func (h *OrderHandlers) validateOrderRequest(req *CreateOrderRequest) error {
	if req.UserID == "" {
		return fmt.Errorf("user_id is required")
	}
	if req.Symbol == "" {
		return fmt.Errorf("symbol is required")
	}
	if req.Side == "" {
		return fmt.Errorf("side is required")
	}
	if req.Type == "" {
		return fmt.Errorf("type is required")
	}
	if req.Quantity <= 0 {
		return fmt.Errorf("quantity must be positive")
	}
	if req.Type == string(types.OrderTypeLimit) && req.Price <= 0 {
		return fmt.Errorf("price must be positive for limit orders")
	}
	return nil
}

// Note: Request/Response types moved to common.go to avoid duplication
