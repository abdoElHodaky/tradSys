package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// SettlementServiceImpl implements the SettlementService interface
type SettlementServiceImpl struct {
	settlements map[string]*Settlement
}

// NewSettlementService creates a new settlement service instance
func NewSettlementService() SettlementService {
	return &SettlementServiceImpl{
		settlements: make(map[string]*Settlement),
	}
}

// ProcessSettlement processes a trade settlement
func (s *SettlementServiceImpl) ProcessSettlement(ctx context.Context, trade *Trade) (*Settlement, error) {
	if trade == nil {
		return nil, fmt.Errorf("trade cannot be nil")
	}

	settlement := &Settlement{
		ID:        uuid.New().String(),
		TradeID:   trade.ID,
		Status:    "pending",
		Amount:    trade.Quantity * trade.Price,
		Currency:  "USD", // Default currency
		CreatedAt: time.Now(),
	}

	// Simulate settlement processing
	if err := s.processSettlementLogic(settlement); err != nil {
		settlement.Status = "failed"
		return settlement, fmt.Errorf("settlement processing failed: %w", err)
	}

	// Store settlement
	s.settlements[settlement.ID] = settlement

	return settlement, nil
}

// GetSettlement retrieves a settlement by ID
func (s *SettlementServiceImpl) GetSettlement(ctx context.Context, id string) (*Settlement, error) {
	settlement, exists := s.settlements[id]
	if !exists {
		return nil, fmt.Errorf("settlement not found: %s", id)
	}

	return settlement, nil
}

// ListSettlements retrieves settlements based on filter criteria
func (s *SettlementServiceImpl) ListSettlements(ctx context.Context, filter *SettlementFilter) ([]*Settlement, error) {
	var result []*Settlement

	for _, settlement := range s.settlements {
		if s.matchesFilter(settlement, filter) {
			result = append(result, settlement)
		}
	}

	// Apply pagination
	if filter != nil {
		start := filter.Offset
		if start > len(result) {
			start = len(result)
		}

		end := start + filter.Limit
		if filter.Limit == 0 || end > len(result) {
			end = len(result)
		}

		if start < end {
			result = result[start:end]
		} else {
			result = []*Settlement{}
		}
	}

	return result, nil
}

// ProcessBatchSettlement processes multiple trade settlements
func (s *SettlementServiceImpl) ProcessBatchSettlement(ctx context.Context, trades []*Trade) ([]*Settlement, error) {
	var settlements []*Settlement
	var errors []error

	for _, trade := range trades {
		settlement, err := s.ProcessSettlement(ctx, trade)
		if err != nil {
			errors = append(errors, err)
			continue
		}
		settlements = append(settlements, settlement)
	}

	if len(errors) > 0 {
		return settlements, fmt.Errorf("batch settlement completed with %d errors", len(errors))
	}

	return settlements, nil
}

// GetPendingSettlements retrieves all pending settlements
func (s *SettlementServiceImpl) GetPendingSettlements(ctx context.Context) ([]*Settlement, error) {
	var pending []*Settlement

	for _, settlement := range s.settlements {
		if settlement.Status == "pending" {
			pending = append(pending, settlement)
		}
	}

	return pending, nil
}

// processSettlementLogic simulates the actual settlement processing
func (s *SettlementServiceImpl) processSettlementLogic(settlement *Settlement) error {
	// Simulate processing time
	time.Sleep(10 * time.Millisecond)

	// Simulate success/failure (95% success rate)
	if time.Now().UnixNano()%100 < 95 {
		settlement.Status = "completed"
		now := time.Now()
		settlement.ProcessedAt = &now
		return nil
	}

	return fmt.Errorf("settlement processing failed for trade %s", settlement.TradeID)
}

// matchesFilter checks if a settlement matches the given filter
func (s *SettlementServiceImpl) matchesFilter(settlement *Settlement, filter *SettlementFilter) bool {
	if filter == nil {
		return true
	}

	if filter.TradeID != nil && settlement.TradeID != *filter.TradeID {
		return false
	}
	if filter.Status != nil && settlement.Status != *filter.Status {
		return false
	}
	if filter.Currency != nil && settlement.Currency != *filter.Currency {
		return false
	}
	if filter.From != nil && settlement.CreatedAt.Before(*filter.From) {
		return false
	}
	if filter.To != nil && settlement.CreatedAt.After(*filter.To) {
		return false
	}

	return true
}

// SettlementProcessor is an alias for SettlementServiceImpl to maintain compatibility
type SettlementProcessor = SettlementServiceImpl

// NewSettlementProcessor creates a new settlement processor (alias for service)
func NewSettlementProcessor() *SettlementProcessor {
	return &SettlementProcessor{
		settlements: make(map[string]*Settlement),
	}
}
