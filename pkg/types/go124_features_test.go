package types

import (
	"context"
	"testing"
	"time"
)

// TestGenericTypeAliases demonstrates Go 1.24 generic type aliases
func TestGenericTypeAliases(t *testing.T) {
	// Test StringAttributes
	attrs := make(StringAttributes)
	attrs["key1"] = "value1"
	attrs["key2"] = 42
	attrs["key3"] = true

	if len(attrs) != 3 {
		t.Errorf("Expected 3 attributes, got %d", len(attrs))
	}

	// Test Set operations
	symbolSet := make(SymbolSet)
	symbolSet["AAPL"] = struct{}{}
	symbolSet["GOOGL"] = struct{}{}
	symbolSet["MSFT"] = struct{}{}

	if len(symbolSet) != 3 {
		t.Errorf("Expected 3 symbols, got %d", len(symbolSet))
	}

	// Check if symbol exists
	if _, exists := symbolSet["AAPL"]; !exists {
		t.Error("Expected AAPL to exist in symbol set")
	}
}

// TestResultType demonstrates the Result type with error handling
func TestResultType(t *testing.T) {
	// Test successful result
	successResult := NewResult("success value")
	if !successResult.IsSuccess() {
		t.Error("Expected result to be successful")
	}
	if successResult.IsError() {
		t.Error("Expected result not to be error")
	}
	if successResult.Unwrap() != "success value" {
		t.Errorf("Expected 'success value', got %v", successResult.Unwrap())
	}

	// Test error result
	errorResult := NewResultWithError[string](NewError("test_error", "test error message"))
	if errorResult.IsSuccess() {
		t.Error("Expected result to be error")
	}
	if !errorResult.IsError() {
		t.Error("Expected result to be error")
	}
	if errorResult.UnwrapOr("default") != "default" {
		t.Errorf("Expected 'default', got %v", errorResult.UnwrapOr("default"))
	}
}

// TestOptionType demonstrates the Option type
func TestOptionType(t *testing.T) {
	// Test Some option
	someOption := Some("test value")
	if !someOption.IsSome() {
		t.Error("Expected option to have value")
	}
	if someOption.IsNone() {
		t.Error("Expected option not to be none")
	}
	if someOption.Unwrap() != "test value" {
		t.Errorf("Expected 'test value', got %v", someOption.Unwrap())
	}

	// Test None option
	noneOption := None[string]()
	if noneOption.IsSome() {
		t.Error("Expected option to be none")
	}
	if !noneOption.IsNone() {
		t.Error("Expected option to be none")
	}
	if noneOption.UnwrapOr("default") != "default" {
		t.Errorf("Expected 'default', got %v", noneOption.UnwrapOr("default"))
	}

	// Test Map operation
	mappedOption := someOption.Map(func(s string) string {
		return s + " mapped"
	})
	if mappedOption.Unwrap() != "test value mapped" {
		t.Errorf("Expected 'test value mapped', got %v", mappedOption.Unwrap())
	}
}

// TestGenericHandlers demonstrates generic handler types
func TestGenericHandlers(t *testing.T) {
	// Test EventHandler
	var eventHandler EventHandler[string] = func(ctx context.Context, event string) error {
		if event == "error" {
			return NewError("handler_error", "Handler error")
		}
		return nil
	}

	// Test successful event handling
	err := eventHandler(context.Background(), "success")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Test error event handling
	err = eventHandler(context.Background(), "error")
	if err == nil {
		t.Error("Expected error, got nil")
	}

	// Test QueryHandler
	var queryHandler QueryHandler[string, int] = func(ctx context.Context, query string) (int, error) {
		if query == "length" {
			return len(query), nil
		}
		return 0, NewError("unknown_query", "Unknown query")
	}

	// Test successful query
	result, err := queryHandler(context.Background(), "length")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if result != 6 {
		t.Errorf("Expected 6, got %d", result)
	}

	// Test error query
	_, err = queryHandler(context.Background(), "unknown")
	if err == nil {
		t.Error("Expected error, got nil")
	}
}

// TestTradingError demonstrates enhanced error handling
func TestTradingError(t *testing.T) {
	// Test basic error
	err := NewError("validation_failed", "Order validation failed")
	if err.Code != "validation_failed" {
		t.Errorf("Expected 'validation_failed', got %s", err.Code)
	}
	if err.Message != "Order validation failed" {
		t.Errorf("Expected 'Order validation failed', got %s", err.Message)
	}

	// Test error with details
	err = err.WithDetail("order_id", "12345").WithDetail("symbol", "AAPL")
	if err.Details["order_id"] != "12345" {
		t.Errorf("Expected '12345', got %v", err.Details["order_id"])
	}
	if err.Details["symbol"] != "AAPL" {
		t.Errorf("Expected 'AAPL', got %v", err.Details["symbol"])
	}

	// Test error string representation
	errorStr := err.Error()
	expectedStr := "[validation_failed] Order validation failed"
	if errorStr != expectedStr {
		t.Errorf("Expected '%s', got '%s'", expectedStr, errorStr)
	}
}

// TestHealthStatus demonstrates health status functionality
func TestHealthStatus(t *testing.T) {
	details := make(Metadata)
	details["uptime"] = 3600
	details["connections"] = 10

	health := HealthStatus{
		Status:    "healthy",
		Message:   "Service is running normally",
		Timestamp: time.Now(),
		Details:   details,
	}

	if health.Status != "healthy" {
		t.Errorf("Expected 'healthy', got %s", health.Status)
	}
	if health.Details["uptime"] != 3600 {
		t.Errorf("Expected 3600, got %v", health.Details["uptime"])
	}
}

// BenchmarkGenericTypes benchmarks generic type operations
func BenchmarkGenericTypes(b *testing.B) {
	b.Run("StringAttributes", func(b *testing.B) {
		attrs := make(StringAttributes)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			attrs["key"] = "value"
			_ = attrs["key"]
		}
	})

	b.Run("SymbolSet", func(b *testing.B) {
		symbolSet := make(SymbolSet)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			symbolSet["AAPL"] = struct{}{}
			_, _ = symbolSet["AAPL"]
		}
	})

	b.Run("ResultType", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			result := NewResult("test")
			_ = result.IsSuccess()
			_ = result.Unwrap()
		}
	})

	b.Run("OptionType", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			option := Some("test")
			_ = option.IsSome()
			_ = option.Unwrap()
		}
	})
}

// TestOrderWithGenericAttributes demonstrates enhanced Order type
func TestOrderWithGenericAttributes(t *testing.T) {
	order := Order{
		ID:       "order-123",
		Symbol:   "AAPL",
		Side:     OrderSideBuy,
		Type:     OrderTypeLimit,
		Price:    150.0,
		Quantity: 100.0,
		Status:   OrderStatusPending,
		Attributes: OrderAttributes{
			"priority":     "high",
			"algo_params":  map[string]interface{}{"twap": true},
			"client_info":  "institutional",
		},
		Metadata: Metadata{
			"source":      "api",
			"version":     "v2",
			"created_by":  "user-456",
		},
	}

	// Test attributes access
	if order.Attributes["priority"] != "high" {
		t.Errorf("Expected 'high', got %v", order.Attributes["priority"])
	}

	// Test metadata access
	if order.Metadata["source"] != "api" {
		t.Errorf("Expected 'api', got %v", order.Metadata["source"])
	}

	// Test that attributes and metadata are properly typed
	if len(order.Attributes) != 3 {
		t.Errorf("Expected 3 attributes, got %d", len(order.Attributes))
	}
	if len(order.Metadata) != 3 {
		t.Errorf("Expected 3 metadata entries, got %d", len(order.Metadata))
	}
}
