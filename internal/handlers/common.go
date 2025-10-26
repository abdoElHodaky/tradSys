package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/abdoElHodaky/tradSys/pkg/types"
)

// Constants for pagination and validation.
const (
	defaultPageSize = 100
	maxPageSize     = 1000
)

// Common HTTP utility functions.

// writeJSON writes a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, statusCode int, data interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	return json.NewEncoder(w).Encode(data)
}

// writeError writes an error response with the given status code and message.
func writeError(w http.ResponseWriter, statusCode int, message string) error {
	return writeJSON(w, statusCode, map[string]string{"error": message})
}

// extractIDFromPath extracts an ID from a URL path.
func extractIDFromPath(path, prefix string) string {
	if len(path) <= len(prefix) {
		return ""
	}
	id := path[len(prefix):]
	// Remove any trailing slashes or query parameters
	if idx := strings.Index(id, "/"); idx != -1 {
		id = id[:idx]
	}
	if idx := strings.Index(id, "?"); idx != -1 {
		id = id[:idx]
	}
	return id
}

// generateOrderID generates a unique order ID.
func generateOrderID() string {
	return fmt.Sprintf("order_%d", time.Now().UnixNano())
}

// generateTradeID generates a unique trade ID.
func generateTradeID() string {
	return fmt.Sprintf("trade_%d", time.Now().UnixNano())
}

// Conversion functions between domain objects and API responses.

// convertOrderToResponse converts a domain Order to an API OrderResponse.
func convertOrderToResponse(order *types.Order) OrderResponse {
	return OrderResponse{
		ID:                order.ID,
		ClientOrderID:     order.ClientOrderID,
		UserID:            order.UserID,
		Symbol:            order.Symbol,
		Side:              string(order.Side),
		Type:              string(order.Type),
		Price:             order.Price,
		Quantity:          order.Quantity,
		FilledQuantity:    order.FilledQuantity,
		RemainingQuantity: order.RemainingQuantity,
		Status:            string(order.Status),
		TimeInForce:       string(order.TimeInForce),
		StopPrice:         order.StopPrice,
		CreatedAt:         order.CreatedAt,
		UpdatedAt:         order.UpdatedAt,
		ExpiresAt:         order.ExpiresAt,
	}
}

// convertTradeToResponse converts a domain Trade to an API TradeResponse.
func convertTradeToResponse(trade *types.Trade) TradeResponse {
	return TradeResponse{
		ID:           trade.ID,
		Symbol:       trade.Symbol,
		BuyOrderID:   trade.BuyOrderID,
		SellOrderID:  trade.SellOrderID,
		Price:        trade.Price,
		Quantity:     trade.Quantity,
		Value:        trade.Value,
		Timestamp:    trade.Timestamp,
		BuyUserID:    trade.BuyUserID,
		SellUserID:   trade.SellUserID,
		TakerSide:    string(trade.TakerSide),
		MakerOrderID: trade.MakerOrderID,
		TakerOrderID: trade.TakerOrderID,
	}
}

// convertTradesToResponse converts a slice of domain Trades to API TradeResponses.
func convertTradesToResponse(trades []*types.Trade) []TradeResponse {
	responses := make([]TradeResponse, len(trades))
	for i, trade := range trades {
		responses[i] = convertTradeToResponse(trade)
	}
	return responses
}

// TradeResponse represents a trade in API responses.
type TradeResponse struct {
	ID           string    `json:"id"`
	Symbol       string    `json:"symbol"`
	BuyOrderID   string    `json:"buy_order_id"`
	SellOrderID  string    `json:"sell_order_id"`
	Price        float64   `json:"price"`
	Quantity     float64   `json:"quantity"`
	Value        float64   `json:"value"`
	Timestamp    time.Time `json:"timestamp"`
	BuyUserID    string    `json:"buy_user_id"`
	SellUserID   string    `json:"sell_user_id"`
	TakerSide    string    `json:"taker_side"`
	MakerOrderID string    `json:"maker_order_id"`
	TakerOrderID string    `json:"taker_order_id"`
}

// ListTradesResponse represents a list of trades in API responses.
type ListTradesResponse struct {
	Trades []TradeResponse `json:"trades"`
	Total  int             `json:"total"`
	Limit  int             `json:"limit"`
	Offset int             `json:"offset"`
}

// HealthResponse represents a health check response.
type HealthResponse struct {
	Status    string                 `json:"status"`
	Timestamp time.Time              `json:"timestamp"`
	Services  map[string]interface{} `json:"services,omitempty"`
	Version   string                 `json:"version,omitempty"`
}

// MetricsResponse represents a metrics response.
type MetricsResponse struct {
	Performance map[string]interface{} `json:"performance"`
	Services    map[string]interface{} `json:"services"`
	Timestamp   time.Time              `json:"timestamp"`
}

// ErrorResponse represents an error response.
type ErrorResponse struct {
	Error     string    `json:"error"`
	Code      string    `json:"code,omitempty"`
	Details   string    `json:"details,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// PaginationParams represents common pagination parameters.
type PaginationParams struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

// ValidatePaginationParams validates and normalizes pagination parameters.
func ValidatePaginationParams(limit, offset int) PaginationParams {
	if limit <= 0 || limit > maxPageSize {
		limit = defaultPageSize
	}
	if offset < 0 {
		offset = 0
	}
	return PaginationParams{
		Limit:  limit,
		Offset: offset,
	}
}

// RequestValidator provides common request validation functions.
type RequestValidator struct{}

// NewRequestValidator creates a new request validator.
func NewRequestValidator() *RequestValidator {
	return &RequestValidator{}
}

// ValidateRequired checks if a string field is not empty.
func (v *RequestValidator) ValidateRequired(field, name string) error {
	if strings.TrimSpace(field) == "" {
		return fmt.Errorf("%s is required", name)
	}
	return nil
}

// ValidatePositive checks if a numeric field is positive.
func (v *RequestValidator) ValidatePositive(value float64, name string) error {
	if value <= 0 {
		return fmt.Errorf("%s must be positive", name)
	}
	return nil
}

// ValidateEnum checks if a string value is in the allowed enum values.
func (v *RequestValidator) ValidateEnum(value string, allowed []string, name string) error {
	for _, a := range allowed {
		if value == a {
			return nil
		}
	}
	return fmt.Errorf("%s must be one of: %s", name, strings.Join(allowed, ", "))
}

// Common validation constants.
var (
	ValidOrderSides = []string{
		string(types.OrderSideBuy),
		string(types.OrderSideSell),
	}
	
	ValidOrderTypes = []string{
		string(types.OrderTypeMarket),
		string(types.OrderTypeLimit),
		string(types.OrderTypeStopLoss),
		string(types.OrderTypeStopLossLimit),
		string(types.OrderTypeTakeProfit),
		string(types.OrderTypeTakeProfitLimit),
	}
	
	ValidTimeInForce = []string{
		string(types.TimeInForceGTC),
		string(types.TimeInForceIOC),
		string(types.TimeInForceFOK),
	}
)
