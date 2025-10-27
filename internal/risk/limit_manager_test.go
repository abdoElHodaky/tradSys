package risk

import (
	"context"
	"fmt"
	"testing"
	"time"

	"go.uber.org/zap/zaptest"
)

func TestLimitManager_NewLimitManager(t *testing.T) {
	logger := zaptest.NewLogger(t)
	lm := NewLimitManager(logger)

	if lm == nil {
		t.Fatal("Expected limit manager to be created")
	}

	if lm.RiskLimits == nil {
		t.Error("Expected risk limits map to be initialized")
	}

	if lm.RiskLimitCache == nil {
		t.Error("Expected risk limit cache to be initialized")
	}
}

func TestLimitManager_AddRiskLimit(t *testing.T) {
	logger := zaptest.NewLogger(t)
	lm := NewLimitManager(logger)
	ctx := context.Background()

	limit := &RiskLimit{
		ID:     "limit1",
		UserID: "user123",
		Symbol: "BTCUSD",
		Type:   RiskLimitTypeMaxOrderSize,
		Value:  1000.0,
	}

	// Test adding risk limit
	result, err := lm.AddRiskLimit(ctx, limit)
	if err != nil {
		t.Errorf("Expected no error adding risk limit, got %v", err)
	}

	if result == nil {
		t.Fatal("Expected risk limit to be returned")
	}

	if result.ID != limit.ID {
		t.Errorf("Expected limit ID %s, got %s", limit.ID, result.ID)
	}

	if !result.Enabled {
		t.Error("Expected limit to be enabled by default")
	}

	if result.CreatedAt.IsZero() {
		t.Error("Expected CreatedAt to be set")
	}

	if result.UpdatedAt.IsZero() {
		t.Error("Expected UpdatedAt to be set")
	}

	// Verify limit is stored
	userLimits := lm.RiskLimits[limit.UserID]
	if len(userLimits) != 1 {
		t.Errorf("Expected 1 limit for user, got %d", len(userLimits))
	}

	if userLimits[0].ID != limit.ID {
		t.Error("Expected stored limit to match added limit")
	}
}

func TestLimitManager_GetRiskLimits(t *testing.T) {
	logger := zaptest.NewLogger(t)
	lm := NewLimitManager(logger)
	ctx := context.Background()

	userID := "user123"

	// Test getting limits for user with no limits
	limits, err := lm.GetRiskLimits(ctx, userID)
	if err != nil {
		t.Errorf("Expected no error getting limits, got %v", err)
	}

	if len(limits) != 0 {
		t.Errorf("Expected 0 limits, got %d", len(limits))
	}

	// Add some limits
	limit1 := &RiskLimit{
		ID:     "limit1",
		UserID: userID,
		Symbol: "BTCUSD",
		Type:   RiskLimitTypeMaxOrderSize,
		Value:  1000.0,
	}

	limit2 := &RiskLimit{
		ID:     "limit2",
		UserID: userID,
		Symbol: "ETHUSD",
		Type:   RiskLimitTypeMaxPosition,
		Value:  5000.0,
	}

	_, err = lm.AddRiskLimit(ctx, limit1)
	if err != nil {
		t.Fatalf("Failed to add limit1: %v", err)
	}

	_, err = lm.AddRiskLimit(ctx, limit2)
	if err != nil {
		t.Fatalf("Failed to add limit2: %v", err)
	}

	// Test getting all limits
	limits, err = lm.GetRiskLimits(ctx, userID)
	if err != nil {
		t.Errorf("Expected no error getting limits, got %v", err)
	}

	if len(limits) != 2 {
		t.Errorf("Expected 2 limits, got %d", len(limits))
	}

	// Verify limits
	limitMap := make(map[string]*RiskLimit)
	for _, limit := range limits {
		limitMap[limit.ID] = limit
	}

	if _, exists := limitMap["limit1"]; !exists {
		t.Error("Expected limit1 to be returned")
	}

	if _, exists := limitMap["limit2"]; !exists {
		t.Error("Expected limit2 to be returned")
	}
}

func TestLimitManager_UpdateRiskLimit(t *testing.T) {
	logger := zaptest.NewLogger(t)
	lm := NewLimitManager(logger)
	ctx := context.Background()

	// Add a limit first
	limit := &RiskLimit{
		ID:     "limit1",
		UserID: "user123",
		Symbol: "BTCUSD",
		Type:   RiskLimitTypeMaxOrderSize,
		Value:  1000.0,
	}

	_, err := lm.AddRiskLimit(ctx, limit)
	if err != nil {
		t.Fatalf("Failed to add limit: %v", err)
	}

	// Update the limit
	updatedLimit := &RiskLimit{
		ID:     "limit1",
		UserID: "user123",
		Symbol: "BTCUSD",
		Type:   RiskLimitTypeMaxOrderSize,
		Value:  2000.0, // Changed value
	}

	result, err := lm.UpdateRiskLimit(ctx, updatedLimit)
	if err != nil {
		t.Errorf("Expected no error updating limit, got %v", err)
	}

	if result == nil {
		t.Fatal("Expected updated limit to be returned")
	}

	if result.Value != 2000.0 {
		t.Errorf("Expected updated value 2000.0, got %f", result.Value)
	}

	if result.UpdatedAt.IsZero() {
		t.Error("Expected UpdatedAt to be updated")
	}

	// Verify limit is updated in storage
	limits, err := lm.GetRiskLimits(ctx, "user123")
	if err != nil {
		t.Fatalf("Failed to get limits: %v", err)
	}

	if len(limits) != 1 {
		t.Fatalf("Expected 1 limit, got %d", len(limits))
	}

	if limits[0].Value != 2000.0 {
		t.Errorf("Expected stored value 2000.0, got %f", limits[0].Value)
	}
}

func TestLimitManager_DeleteRiskLimit(t *testing.T) {
	logger := zaptest.NewLogger(t)
	lm := NewLimitManager(logger)
	ctx := context.Background()

	// Add a limit first
	limit := &RiskLimit{
		ID:     "limit1",
		UserID: "user123",
		Symbol: "BTCUSD",
		Type:   RiskLimitTypeMaxOrderSize,
		Value:  1000.0,
	}

	_, err := lm.AddRiskLimit(ctx, limit)
	if err != nil {
		t.Fatalf("Failed to add limit: %v", err)
	}

	// Delete the limit
	err = lm.DeleteRiskLimit(ctx, "user123", "limit1")
	if err != nil {
		t.Errorf("Expected no error deleting limit, got %v", err)
	}

	// Verify limit is deleted
	limits, err := lm.GetRiskLimits(ctx, "user123")
	if err != nil {
		t.Errorf("Expected no error getting limits, got %v", err)
	}

	if len(limits) != 0 {
		t.Errorf("Expected 0 limits after deletion, got %d", len(limits))
	}

	// Test deleting non-existent limit
	err = lm.DeleteRiskLimit(ctx, "user123", "nonexistent")
	if err == nil {
		t.Error("Expected error deleting non-existent limit")
	}
}

func TestLimitManager_CheckRiskLimit(t *testing.T) {
	logger := zaptest.NewLogger(t)
	lm := NewLimitManager(logger)
	ctx := context.Background()

	userID := "user123"
	symbol := "BTCUSD"

	// Add a max order size limit
	limit := &RiskLimit{
		ID:     "limit1",
		UserID: userID,
		Symbol: symbol,
		Type:   RiskLimitTypeMaxOrderSize,
		Value:  1000.0,
	}

	_, err := lm.AddRiskLimit(ctx, limit)
	if err != nil {
		t.Fatalf("Failed to add limit: %v", err)
	}

	// Test order within limit
	approved, reason, err := lm.CheckRiskLimit(ctx, userID, symbol, 500.0, "buy")
	if err != nil {
		t.Errorf("Expected no error checking limit, got %v", err)
	}

	if !approved {
		t.Error("Expected order to be approved")
	}

	if reason != "" {
		t.Errorf("Expected no reason for approved order, got %s", reason)
	}

	// Test order exceeding limit
	approved, reason, err = lm.CheckRiskLimit(ctx, userID, symbol, 1500.0, "buy")
	if err != nil {
		t.Errorf("Expected no error checking limit, got %v", err)
	}

	if approved {
		t.Error("Expected order to be rejected")
	}

	if reason == "" {
		t.Error("Expected reason for rejected order")
	}
}

func TestLimitManager_ConcurrentAccess(t *testing.T) {
	logger := zaptest.NewLogger(t)
	lm := NewLimitManager(logger)
	ctx := context.Background()

	userID := "user123"
	numGoroutines := 10
	done := make(chan bool, numGoroutines)

	// Test concurrent limit additions
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer func() { done <- true }()

			limit := &RiskLimit{
				ID:     fmt.Sprintf("limit%d", id),
				UserID: userID,
				Symbol: "BTCUSD",
				Type:   RiskLimitTypeMaxOrderSize,
				Value:  float64(1000 + id*100),
			}

			_, err := lm.AddRiskLimit(ctx, limit)
			if err != nil {
				t.Errorf("Failed to add limit %d: %v", id, err)
			}
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify all limits were added
	limits, err := lm.GetRiskLimits(ctx, userID)
	if err != nil {
		t.Errorf("Failed to get limits: %v", err)
	}

	if len(limits) != numGoroutines {
		t.Errorf("Expected %d limits, got %d", numGoroutines, len(limits))
	}
}

func TestLimitManager_CacheIntegration(t *testing.T) {
	logger := zaptest.NewLogger(t)
	lm := NewLimitManager(logger)
	ctx := context.Background()

	userID := "user123"

	// Add a limit
	limit := &RiskLimit{
		ID:     "limit1",
		UserID: userID,
		Symbol: "BTCUSD",
		Type:   RiskLimitTypeMaxOrderSize,
		Value:  1000.0,
	}

	_, err := lm.AddRiskLimit(ctx, limit)
	if err != nil {
		t.Fatalf("Failed to add limit: %v", err)
	}

	// First call should populate cache
	limits1, err := lm.GetRiskLimits(ctx, userID)
	if err != nil {
		t.Errorf("Expected no error getting limits, got %v", err)
	}

	// Second call should use cache
	limits2, err := lm.GetRiskLimits(ctx, userID)
	if err != nil {
		t.Errorf("Expected no error getting limits, got %v", err)
	}

	// Verify results are consistent
	if len(limits1) != len(limits2) {
		t.Error("Cache returned different number of limits")
	}

	if len(limits1) > 0 && len(limits2) > 0 {
		if limits1[0].ID != limits2[0].ID {
			t.Error("Cache returned different limit data")
		}
	}
}

func BenchmarkLimitManager_AddRiskLimit(b *testing.B) {
	logger := zaptest.NewLogger(b)
	lm := NewLimitManager(logger)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		limit := &RiskLimit{
			ID:     fmt.Sprintf("limit%d", i),
			UserID: "user123",
			Symbol: "BTCUSD",
			Type:   RiskLimitTypeMaxOrderSize,
			Value:  1000.0,
		}

		_, err := lm.AddRiskLimit(ctx, limit)
		if err != nil {
			b.Errorf("Failed to add limit: %v", err)
		}
	}
}

func BenchmarkLimitManager_GetRiskLimits(b *testing.B) {
	logger := zaptest.NewLogger(b)
	lm := NewLimitManager(logger)
	ctx := context.Background()

	userID := "user123"

	// Add some limits
	for i := 0; i < 10; i++ {
		limit := &RiskLimit{
			ID:     fmt.Sprintf("limit%d", i),
			UserID: userID,
			Symbol: "BTCUSD",
			Type:   RiskLimitTypeMaxOrderSize,
			Value:  1000.0,
		}

		_, err := lm.AddRiskLimit(ctx, limit)
		if err != nil {
			b.Fatalf("Failed to add limit: %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := lm.GetRiskLimits(ctx, userID)
		if err != nil {
			b.Errorf("Failed to get limits: %v", err)
		}
	}
}

func BenchmarkLimitManager_CheckRiskLimit(b *testing.B) {
	logger := zaptest.NewLogger(b)
	lm := NewLimitManager(logger)
	ctx := context.Background()

	userID := "user123"
	symbol := "BTCUSD"

	// Add a limit
	limit := &RiskLimit{
		ID:     "limit1",
		UserID: userID,
		Symbol: symbol,
		Type:   RiskLimitTypeMaxOrderSize,
		Value:  1000.0,
	}

	_, err := lm.AddRiskLimit(ctx, limit)
	if err != nil {
		b.Fatalf("Failed to add limit: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := lm.CheckRiskLimit(ctx, userID, symbol, 500.0, "buy")
		if err != nil {
			b.Errorf("Failed to check limit: %v", err)
		}
	}
}
