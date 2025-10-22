package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// StrategyServiceImpl implements the StrategyService interface
type StrategyServiceImpl struct {
	strategies map[string]*Strategy
}

// NewStrategyService creates a new strategy service instance
func NewStrategyService() StrategyService {
	return &StrategyServiceImpl{
		strategies: make(map[string]*Strategy),
	}
}

// CreateStrategy creates a new trading strategy
func (s *StrategyServiceImpl) CreateStrategy(ctx context.Context, strategy *Strategy) (*Strategy, error) {
	if strategy == nil {
		return nil, fmt.Errorf("strategy cannot be nil")
	}

	// Generate ID if not provided
	if strategy.ID == "" {
		strategy.ID = uuid.New().String()
	}

	// Set timestamps
	now := time.Now()
	strategy.CreatedAt = now
	strategy.UpdatedAt = now
	strategy.Status = "inactive"

	// Validate strategy
	if err := s.validateStrategy(strategy); err != nil {
		return nil, fmt.Errorf("strategy validation failed: %w", err)
	}

	// Store strategy
	s.strategies[strategy.ID] = strategy

	return strategy, nil
}

// UpdateStrategy updates an existing strategy
func (s *StrategyServiceImpl) UpdateStrategy(ctx context.Context, id string, updates *StrategyUpdate) (*Strategy, error) {
	strategy, exists := s.strategies[id]
	if !exists {
		return nil, fmt.Errorf("strategy not found: %s", id)
	}

	// Check if strategy can be updated
	if strategy.Status == "running" {
		return nil, fmt.Errorf("cannot update running strategy: %s", id)
	}

	// Apply updates
	if updates.Name != nil {
		strategy.Name = *updates.Name
	}
	if updates.Description != nil {
		strategy.Description = *updates.Description
	}
	if updates.Parameters != nil {
		strategy.Parameters = *updates.Parameters
	}

	strategy.UpdatedAt = time.Now()

	// Validate updated strategy
	if err := s.validateStrategy(strategy); err != nil {
		return nil, fmt.Errorf("updated strategy validation failed: %w", err)
	}

	return strategy, nil
}

// DeleteStrategy deletes a strategy
func (s *StrategyServiceImpl) DeleteStrategy(ctx context.Context, id string) error {
	strategy, exists := s.strategies[id]
	if !exists {
		return fmt.Errorf("strategy not found: %s", id)
	}

	// Check if strategy can be deleted
	if strategy.Status == "running" {
		return fmt.Errorf("cannot delete running strategy: %s", id)
	}

	delete(s.strategies, id)
	return nil
}

// GetStrategy retrieves a strategy by ID
func (s *StrategyServiceImpl) GetStrategy(ctx context.Context, id string) (*Strategy, error) {
	strategy, exists := s.strategies[id]
	if !exists {
		return nil, fmt.Errorf("strategy not found: %s", id)
	}

	return strategy, nil
}

// ListStrategies retrieves strategies based on filter criteria
func (s *StrategyServiceImpl) ListStrategies(ctx context.Context, filter *StrategyFilter) ([]*Strategy, error) {
	var result []*Strategy

	for _, strategy := range s.strategies {
		if s.matchesFilter(strategy, filter) {
			result = append(result, strategy)
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
			result = []*Strategy{}
		}
	}

	return result, nil
}

// StartStrategy starts a strategy execution
func (s *StrategyServiceImpl) StartStrategy(ctx context.Context, id string) error {
	strategy, exists := s.strategies[id]
	if !exists {
		return fmt.Errorf("strategy not found: %s", id)
	}

	if strategy.Status == "running" {
		return fmt.Errorf("strategy is already running: %s", id)
	}

	// Validate strategy before starting
	if err := s.validateStrategy(strategy); err != nil {
		return fmt.Errorf("cannot start invalid strategy: %w", err)
	}

	strategy.Status = "running"
	strategy.UpdatedAt = time.Now()

	// In a real implementation, this would start the actual strategy execution
	// For now, we just update the status

	return nil
}

// StopStrategy stops a strategy execution
func (s *StrategyServiceImpl) StopStrategy(ctx context.Context, id string) error {
	strategy, exists := s.strategies[id]
	if !exists {
		return fmt.Errorf("strategy not found: %s", id)
	}

	if strategy.Status != "running" {
		return fmt.Errorf("strategy is not running: %s", id)
	}

	strategy.Status = "stopped"
	strategy.UpdatedAt = time.Now()

	// In a real implementation, this would stop the actual strategy execution
	// For now, we just update the status

	return nil
}

// GetStrategyStatus retrieves the current status of a strategy
func (s *StrategyServiceImpl) GetStrategyStatus(ctx context.Context, id string) (*StrategyStatus, error) {
	strategy, exists := s.strategies[id]
	if !exists {
		return nil, fmt.Errorf("strategy not found: %s", id)
	}

	status := &StrategyStatus{
		ID:           strategy.ID,
		Status:       strategy.Status,
		LastUpdated:  strategy.UpdatedAt,
		OrdersPlaced: 0,    // Would be tracked in real implementation
		ProfitLoss:   0.0,  // Would be calculated in real implementation
	}

	// Calculate running time if strategy is running
	if strategy.Status == "running" {
		status.RunningTime = time.Since(strategy.UpdatedAt)
	}

	return status, nil
}

// validateStrategy validates strategy parameters
func (s *StrategyServiceImpl) validateStrategy(strategy *Strategy) error {
	if strategy.Name == "" {
		return fmt.Errorf("strategy name is required")
	}
	if strategy.Type == "" {
		return fmt.Errorf("strategy type is required")
	}

	// Validate strategy type
	validTypes := []string{"momentum", "mean_reversion", "arbitrage", "market_making", "trend_following"}
	isValidType := false
	for _, validType := range validTypes {
		if strategy.Type == validType {
			isValidType = true
			break
		}
	}
	if !isValidType {
		return fmt.Errorf("invalid strategy type: %s", strategy.Type)
	}

	return nil
}

// matchesFilter checks if a strategy matches the given filter
func (s *StrategyServiceImpl) matchesFilter(strategy *Strategy, filter *StrategyFilter) bool {
	if filter == nil {
		return true
	}

	if filter.Type != nil && strategy.Type != *filter.Type {
		return false
	}
	if filter.Status != nil && strategy.Status != *filter.Status {
		return false
	}

	return true
}
