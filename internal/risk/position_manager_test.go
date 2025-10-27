package risk

import (
	"context"
	"testing"
	"time"

	riskengine "github.com/abdoElHodaky/tradSys/internal/risk/engine"
	"go.uber.org/zap/zaptest"
)

func TestPositionManager_NewPositionManager(t *testing.T) {
	logger := zaptest.NewLogger(t)
	pm := NewPositionManager(logger)

	if pm == nil {
		t.Fatal("Expected position manager to be created")
	}

	if pm.Positions == nil {
		t.Error("Expected positions map to be initialized")
	}

	if pm.PositionCache == nil {
		t.Error("Expected position cache to be initialized")
	}
}

func TestPositionManager_UpdatePosition(t *testing.T) {
	logger := zaptest.NewLogger(t)
	pm := NewPositionManager(logger)

	userID := "user123"
	symbol := "BTCUSD"
	quantity := 100.0
	price := 50000.0

	// Test creating new position
	pm.UpdatePosition(userID, symbol, quantity, price)

	position, exists := pm.Positions[userID][symbol]
	if !exists {
		t.Fatal("Expected position to be created")
	}

	if position.Quantity != quantity {
		t.Errorf("Expected quantity %f, got %f", quantity, position.Quantity)
	}

	if position.AvgPrice != price {
		t.Errorf("Expected average price %f, got %f", price, position.AvgPrice)
	}

	// Test adding to existing position
	additionalQuantity := 50.0
	newPrice := 51000.0
	pm.UpdatePosition(userID, symbol, additionalQuantity, newPrice)

	expectedQuantity := quantity + additionalQuantity
	expectedAvgPrice := ((quantity * price) + (additionalQuantity * newPrice)) / expectedQuantity

	if position.Quantity != expectedQuantity {
		t.Errorf("Expected quantity %f, got %f", expectedQuantity, position.Quantity)
	}

	if position.AvgPrice != expectedAvgPrice {
		t.Errorf("Expected average price %f, got %f", expectedAvgPrice, position.AvgPrice)
	}

	// Test reducing position
	reductionQuantity := -30.0
	pm.UpdatePosition(userID, symbol, reductionQuantity, newPrice)

	expectedQuantity = expectedQuantity + reductionQuantity
	if position.Quantity != expectedQuantity {
		t.Errorf("Expected quantity %f, got %f", expectedQuantity, position.Quantity)
	}

	// Test closing position
	pm.UpdatePosition(userID, symbol, -expectedQuantity, newPrice)

	if position.Quantity != 0 {
		t.Errorf("Expected quantity to be 0, got %f", position.Quantity)
	}

	if position.AvgPrice != 0 {
		t.Errorf("Expected average price to be 0, got %f", position.AvgPrice)
	}
}

func TestPositionManager_GetPosition(t *testing.T) {
	logger := zaptest.NewLogger(t)
	pm := NewPositionManager(logger)
	ctx := context.Background()

	userID := "user123"
	symbol := "BTCUSD"

	// Test getting non-existent position
	position, err := pm.GetPosition(ctx, userID, symbol)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if position.Quantity != 0 {
		t.Errorf("Expected zero position quantity, got %f", position.Quantity)
	}

	// Create a position
	quantity := 100.0
	price := 50000.0
	pm.UpdatePosition(userID, symbol, quantity, price)

	// Test getting existing position
	position, err = pm.GetPosition(ctx, userID, symbol)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if position.Quantity != quantity {
		t.Errorf("Expected quantity %f, got %f", quantity, position.Quantity)
	}

	if position.AvgPrice != price {
		t.Errorf("Expected average price %f, got %f", price, position.AvgPrice)
	}
}

func TestPositionManager_GetPositions(t *testing.T) {
	logger := zaptest.NewLogger(t)
	pm := NewPositionManager(logger)
	ctx := context.Background()

	userID := "user123"

	// Test getting positions for user with no positions
	positions, err := pm.GetPositions(ctx, userID)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(positions) != 0 {
		t.Errorf("Expected 0 positions, got %d", len(positions))
	}

	// Create multiple positions
	symbols := []string{"BTCUSD", "ETHUSD", "ADAUSD"}
	for i, symbol := range symbols {
		pm.UpdatePosition(userID, symbol, float64(100*(i+1)), float64(1000*(i+1)))
	}

	// Test getting all positions
	positions, err = pm.GetPositions(ctx, userID)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(positions) != len(symbols) {
		t.Errorf("Expected %d positions, got %d", len(symbols), len(positions))
	}

	// Verify positions
	symbolMap := make(map[string]bool)
	for _, symbol := range symbols {
		symbolMap[symbol] = true
	}

	for _, position := range positions {
		if !symbolMap[position.Symbol] {
			t.Errorf("Unexpected symbol in positions: %s", position.Symbol)
		}
	}
}

func TestPositionManager_UpdateUnrealizedPnL(t *testing.T) {
	logger := zaptest.NewLogger(t)
	pm := NewPositionManager(logger)

	userID := "user123"
	symbol := "BTCUSD"
	quantity := 100.0
	avgPrice := 50000.0
	currentPrice := 52000.0

	// Create a position
	pm.UpdatePosition(userID, symbol, quantity, avgPrice)

	// Update unrealized P&L
	pm.UpdateUnrealizedPnL(symbol, currentPrice)

	position := pm.Positions[userID][symbol]
	expectedPnL := (currentPrice - avgPrice) * quantity

	if position.UnrealizedPnL != expectedPnL {
		t.Errorf("Expected unrealized P&L %f, got %f", expectedPnL, position.UnrealizedPnL)
	}

	// Test with negative P&L
	lowerPrice := 48000.0
	pm.UpdateUnrealizedPnL(symbol, lowerPrice)

	expectedPnL = (lowerPrice - avgPrice) * quantity
	if position.UnrealizedPnL != expectedPnL {
		t.Errorf("Expected unrealized P&L %f, got %f", expectedPnL, position.UnrealizedPnL)
	}
}

func TestPositionManager_ConcurrentAccess(t *testing.T) {
	logger := zaptest.NewLogger(t)
	pm := NewPositionManager(logger)

	userID := "user123"
	symbol := "BTCUSD"
	numGoroutines := 100
	done := make(chan bool, numGoroutines)

	// Test concurrent position updates
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			pm.UpdatePosition(userID, symbol, 1.0, 50000.0)
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify final position
	position := pm.Positions[userID][symbol]
	expectedQuantity := float64(numGoroutines)

	if position.Quantity != expectedQuantity {
		t.Errorf("Expected quantity %f, got %f", expectedQuantity, position.Quantity)
	}
}

func TestPositionManager_CacheIntegration(t *testing.T) {
	logger := zaptest.NewLogger(t)
	pm := NewPositionManager(logger)
	ctx := context.Background()

	userID := "user123"
	symbol := "BTCUSD"
	quantity := 100.0
	price := 50000.0

	// Create a position
	pm.UpdatePosition(userID, symbol, quantity, price)

	// First call should populate cache
	position1, err := pm.GetPosition(ctx, userID, symbol)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Second call should use cache
	position2, err := pm.GetPosition(ctx, userID, symbol)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Verify positions are equal
	if position1.Quantity != position2.Quantity {
		t.Error("Cache returned different position data")
	}

	if position1.AvgPrice != position2.AvgPrice {
		t.Error("Cache returned different position data")
	}
}

func BenchmarkPositionManager_UpdatePosition(b *testing.B) {
	logger := zaptest.NewLogger(b)
	pm := NewPositionManager(logger)

	userID := "user123"
	symbol := "BTCUSD"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pm.UpdatePosition(userID, symbol, 1.0, 50000.0)
	}
}

func BenchmarkPositionManager_GetPosition(b *testing.B) {
	logger := zaptest.NewLogger(b)
	pm := NewPositionManager(logger)
	ctx := context.Background()

	userID := "user123"
	symbol := "BTCUSD"

	// Create a position
	pm.UpdatePosition(userID, symbol, 100.0, 50000.0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := pm.GetPosition(ctx, userID, symbol)
		if err != nil {
			b.Errorf("Unexpected error: %v", err)
		}
	}
}

func BenchmarkPositionManager_ConcurrentUpdates(b *testing.B) {
	logger := zaptest.NewLogger(b)
	pm := NewPositionManager(logger)

	userID := "user123"
	symbol := "BTCUSD"

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			pm.UpdatePosition(userID, symbol, 1.0, 50000.0)
		}
	})
}
