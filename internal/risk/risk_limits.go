package risk

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/patrickmn/go-cache"
	"go.uber.org/zap"
)

// RiskLimitsManager manages risk limits for users and symbols
type RiskLimitsManager struct {
	logger *zap.Logger
	mu     sync.RWMutex

	// Risk limits storage
	riskLimits   map[string][]*RiskLimit // userID -> limits
	symbolLimits map[string][]*RiskLimit // symbol -> limits
	globalLimits []*RiskLimit            // global limits

	// Cache for performance
	riskLimitCache *cache.Cache

	// Batch processing
	batchChan chan RiskLimitOperation
	ctx       context.Context
	cancel    context.CancelFunc
}

// RiskLimitOperation represents a batch operation on risk limits
type RiskLimitOperation struct {
	OpType   string
	UserID   string
	Symbol   string
	Limit    *RiskLimit
	ResultCh chan RiskLimitOperationResult
}

// RiskLimitOperationResult represents the result of a risk limit operation
type RiskLimitOperationResult struct {
	Success bool
	Error   error
	Data    interface{}
}

// NewRiskLimitsManager creates a new risk limits manager
func NewRiskLimitsManager(logger *zap.Logger) *RiskLimitsManager {
	ctx, cancel := context.WithCancel(context.Background())

	manager := &RiskLimitsManager{
		logger:         logger,
		riskLimits:     make(map[string][]*RiskLimit),
		symbolLimits:   make(map[string][]*RiskLimit),
		globalLimits:   make([]*RiskLimit, 0),
		riskLimitCache: cache.New(5*time.Minute, 10*time.Minute),
		batchChan:      make(chan RiskLimitOperation, 1000),
		ctx:            ctx,
		cancel:         cancel,
	}

	// Start batch processor
	go manager.processBatchOperations()

	return manager
}

// AddRiskLimit adds a risk limit
func (rlm *RiskLimitsManager) AddRiskLimit(ctx context.Context, limit *RiskLimit) (*RiskLimit, error) {
	if limit == nil {
		return nil, errors.New("limit cannot be nil")
	}

	// Generate ID if not provided
	if limit.ID == "" {
		limit.ID = uuid.New().String()
	}

	// Set timestamps
	now := time.Now()
	if limit.CreatedAt.IsZero() {
		limit.CreatedAt = now
	}
	limit.UpdatedAt = now
	limit.Enabled = true // Enable by default

	// Use batch processing for better performance
	resultCh := make(chan RiskLimitOperationResult, 1)
	operation := RiskLimitOperation{
		OpType:   "add",
		UserID:   limit.UserID,
		Symbol:   limit.Symbol,
		Limit:    limit,
		ResultCh: resultCh,
	}

	select {
	case rlm.batchChan <- operation:
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	// Wait for result
	select {
	case result := <-resultCh:
		if !result.Success {
			return nil, result.Error
		}
		return result.Data.(*RiskLimit), nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// UpdateRiskLimit updates a risk limit
func (rlm *RiskLimitsManager) UpdateRiskLimit(ctx context.Context, limit *RiskLimit) (*RiskLimit, error) {
	if limit == nil || limit.ID == "" {
		return nil, errors.New("limit and limit ID cannot be empty")
	}

	// Set update timestamp
	limit.UpdatedAt = time.Now()

	// Use batch processing
	resultCh := make(chan RiskLimitOperationResult, 1)
	operation := RiskLimitOperation{
		OpType:   "update",
		UserID:   limit.UserID,
		Symbol:   limit.Symbol,
		Limit:    limit,
		ResultCh: resultCh,
	}

	select {
	case rlm.batchChan <- operation:
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	// Wait for result
	select {
	case result := <-resultCh:
		if !result.Success {
			return nil, result.Error
		}
		return result.Data.(*RiskLimit), nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// DeleteRiskLimit deletes a risk limit
func (rlm *RiskLimitsManager) DeleteRiskLimit(ctx context.Context, userID, limitID string) error {
	if userID == "" || limitID == "" {
		return errors.New("userID and limitID cannot be empty")
	}

	// Use batch processing
	resultCh := make(chan RiskLimitOperationResult, 1)
	operation := RiskLimitOperation{
		OpType:   "delete",
		UserID:   userID,
		Limit:    &RiskLimit{ID: limitID},
		ResultCh: resultCh,
	}

	select {
	case rlm.batchChan <- operation:
	case <-ctx.Done():
		return ctx.Err()
	}

	// Wait for result
	select {
	case result := <-resultCh:
		if !result.Success {
			return result.Error
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// GetRiskLimits gets all risk limits for a user
func (rlm *RiskLimitsManager) GetRiskLimits(ctx context.Context, userID string) ([]*RiskLimit, error) {
	if userID == "" {
		return nil, errors.New("userID cannot be empty")
	}

	// Check cache first
	cacheKey := "user_limits:" + userID
	if cached, found := rlm.riskLimitCache.Get(cacheKey); found {
		return cached.([]*RiskLimit), nil
	}

	rlm.mu.RLock()
	userLimits, exists := rlm.riskLimits[userID]
	rlm.mu.RUnlock()

	if !exists {
		return []*RiskLimit{}, nil
	}

	// Create a copy to avoid race conditions
	limits := make([]*RiskLimit, len(userLimits))
	copy(limits, userLimits)

	// Cache the result
	rlm.riskLimitCache.Set(cacheKey, limits, cache.DefaultExpiration)

	return limits, nil
}

// GetRiskLimitsBySymbol gets all risk limits for a symbol
func (rlm *RiskLimitsManager) GetRiskLimitsBySymbol(ctx context.Context, symbol string) ([]*RiskLimit, error) {
	if symbol == "" {
		return nil, errors.New("symbol cannot be empty")
	}

	// Check cache first
	cacheKey := "symbol_limits:" + symbol
	if cached, found := rlm.riskLimitCache.Get(cacheKey); found {
		return cached.([]*RiskLimit), nil
	}

	rlm.mu.RLock()
	symbolLimits, exists := rlm.symbolLimits[symbol]
	rlm.mu.RUnlock()

	if !exists {
		return []*RiskLimit{}, nil
	}

	// Create a copy to avoid race conditions
	limits := make([]*RiskLimit, len(symbolLimits))
	copy(limits, symbolLimits)

	// Cache the result
	rlm.riskLimitCache.Set(cacheKey, limits, cache.DefaultExpiration)

	return limits, nil
}

// GetGlobalRiskLimits gets all global risk limits
func (rlm *RiskLimitsManager) GetGlobalRiskLimits(ctx context.Context) ([]*RiskLimit, error) {
	// Check cache first
	cacheKey := "global_limits"
	if cached, found := rlm.riskLimitCache.Get(cacheKey); found {
		return cached.([]*RiskLimit), nil
	}

	rlm.mu.RLock()
	limits := make([]*RiskLimit, len(rlm.globalLimits))
	copy(limits, rlm.globalLimits)
	rlm.mu.RUnlock()

	// Cache the result
	rlm.riskLimitCache.Set(cacheKey, limits, cache.DefaultExpiration)

	return limits, nil
}

// CheckRiskLimit checks if an operation violates risk limits
func (rlm *RiskLimitsManager) CheckRiskLimit(ctx context.Context, userID, symbol string, quantity, price float64, side string) (bool, string, error) {
	// Get applicable limits
	userLimits, err := rlm.GetRiskLimits(ctx, userID)
	if err != nil {
		return false, "", err
	}

	symbolLimits, err := rlm.GetRiskLimitsBySymbol(ctx, symbol)
	if err != nil {
		return false, "", err
	}

	globalLimits, err := rlm.GetGlobalRiskLimits(ctx)
	if err != nil {
		return false, "", err
	}

	// Combine all applicable limits
	allLimits := make([]*RiskLimit, 0)
	allLimits = append(allLimits, userLimits...)
	allLimits = append(allLimits, symbolLimits...)
	allLimits = append(allLimits, globalLimits...)

	// Check each limit
	for _, limit := range allLimits {
		if !limit.IsEnabled() {
			continue
		}

		// Check if limit applies to this symbol
		if limit.Symbol != "" && limit.Symbol != symbol {
			continue
		}

		// Check specific limit types
		violated, reason := rlm.checkSpecificLimit(limit, userID, symbol, quantity, price, side)
		if violated {
			return false, reason, nil
		}
	}

	return true, "", nil
}

// EnableRiskLimit enables a risk limit
func (rlm *RiskLimitsManager) EnableRiskLimit(ctx context.Context, userID, limitID string) error {
	return rlm.setRiskLimitStatus(ctx, userID, limitID, true)
}

// DisableRiskLimit disables a risk limit
func (rlm *RiskLimitsManager) DisableRiskLimit(ctx context.Context, userID, limitID string) error {
	return rlm.setRiskLimitStatus(ctx, userID, limitID, false)
}

// GetRiskLimitStats gets statistics about risk limits
func (rlm *RiskLimitsManager) GetRiskLimitStats(ctx context.Context) (map[string]interface{}, error) {
	rlm.mu.RLock()
	defer rlm.mu.RUnlock()

	stats := map[string]interface{}{
		"total_users":     len(rlm.riskLimits),
		"total_symbols":   len(rlm.symbolLimits),
		"global_limits":   len(rlm.globalLimits),
		"cache_items":     rlm.riskLimitCache.ItemCount(),
		"limits_by_type":  make(map[string]int),
		"enabled_limits":  0,
		"disabled_limits": 0,
	}

	limitsByType := stats["limits_by_type"].(map[string]int)
	enabledCount := 0
	disabledCount := 0

	// Count user limits
	for _, userLimits := range rlm.riskLimits {
		for _, limit := range userLimits {
			limitsByType[string(limit.Type)]++
			if limit.Enabled {
				enabledCount++
			} else {
				disabledCount++
			}
		}
	}

	// Count symbol limits
	for _, symbolLimits := range rlm.symbolLimits {
		for _, limit := range symbolLimits {
			limitsByType[string(limit.Type)]++
			if limit.Enabled {
				enabledCount++
			} else {
				disabledCount++
			}
		}
	}

	// Count global limits
	for _, limit := range rlm.globalLimits {
		limitsByType[string(limit.Type)]++
		if limit.Enabled {
			enabledCount++
		} else {
			disabledCount++
		}
	}

	stats["enabled_limits"] = enabledCount
	stats["disabled_limits"] = disabledCount

	return stats, nil
}

// Stop stops the risk limits manager
func (rlm *RiskLimitsManager) Stop() {
	rlm.cancel()
}

// processBatchOperations processes batch operations for risk limits
func (rlm *RiskLimitsManager) processBatchOperations() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	batch := make([]RiskLimitOperation, 0, 100)

	for {
		select {
		case <-rlm.ctx.Done():
			return
		case op := <-rlm.batchChan:
			batch = append(batch, op)

			// Process batch when it's full or on ticker
			if len(batch) >= 100 {
				rlm.processBatch(batch)
				batch = batch[:0]
			}
		case <-ticker.C:
			if len(batch) > 0 {
				rlm.processBatch(batch)
				batch = batch[:0]
			}
		}
	}
}

// processBatch processes a batch of operations
func (rlm *RiskLimitsManager) processBatch(batch []RiskLimitOperation) {
	// Group operations by type
	addOps := make([]RiskLimitOperation, 0)
	updateOps := make([]RiskLimitOperation, 0)
	deleteOps := make([]RiskLimitOperation, 0)

	for _, op := range batch {
		switch op.OpType {
		case "add":
			addOps = append(addOps, op)
		case "update":
			updateOps = append(updateOps, op)
		case "delete":
			deleteOps = append(deleteOps, op)
		}
	}

	// Process each type
	if len(addOps) > 0 {
		rlm.processAddBatch(addOps)
	}
	if len(updateOps) > 0 {
		rlm.processUpdateBatch(updateOps)
	}
	if len(deleteOps) > 0 {
		rlm.processDeleteBatch(deleteOps)
	}
}

// processAddBatch processes a batch of add operations
func (rlm *RiskLimitsManager) processAddBatch(ops []RiskLimitOperation) {
	rlm.mu.Lock()
	defer rlm.mu.Unlock()

	for _, op := range ops {
		limit := op.Limit

		// Add to user limits
		if limit.UserID != "" {
			if _, exists := rlm.riskLimits[limit.UserID]; !exists {
				rlm.riskLimits[limit.UserID] = make([]*RiskLimit, 0)
			}
			rlm.riskLimits[limit.UserID] = append(rlm.riskLimits[limit.UserID], limit)
		}

		// Add to symbol limits
		if limit.Symbol != "" {
			if _, exists := rlm.symbolLimits[limit.Symbol]; !exists {
				rlm.symbolLimits[limit.Symbol] = make([]*RiskLimit, 0)
			}
			rlm.symbolLimits[limit.Symbol] = append(rlm.symbolLimits[limit.Symbol], limit)
		}

		// Add to global limits if no user or symbol specified
		if limit.UserID == "" && limit.Symbol == "" {
			rlm.globalLimits = append(rlm.globalLimits, limit)
		}

		// Add to cache
		rlm.riskLimitCache.Set(limit.UserID+":"+limit.ID, limit, cache.DefaultExpiration)

		// Invalidate related caches
		rlm.invalidateCache(limit.UserID, limit.Symbol)

		// Send result
		op.ResultCh <- RiskLimitOperationResult{
			Success: true,
			Data:    limit,
		}
	}
}

// processUpdateBatch processes a batch of update operations
func (rlm *RiskLimitsManager) processUpdateBatch(ops []RiskLimitOperation) {
	rlm.mu.Lock()
	defer rlm.mu.Unlock()

	for _, op := range ops {
		limit := op.Limit
		found := false

		// Update in user limits
		if userLimits, exists := rlm.riskLimits[limit.UserID]; exists {
			for i, existingLimit := range userLimits {
				if existingLimit.ID == limit.ID {
					rlm.riskLimits[limit.UserID][i] = limit
					found = true
					break
				}
			}
		}

		// Update in symbol limits
		if symbolLimits, exists := rlm.symbolLimits[limit.Symbol]; exists {
			for i, existingLimit := range symbolLimits {
				if existingLimit.ID == limit.ID {
					rlm.symbolLimits[limit.Symbol][i] = limit
					found = true
					break
				}
			}
		}

		// Update in global limits
		for i, existingLimit := range rlm.globalLimits {
			if existingLimit.ID == limit.ID {
				rlm.globalLimits[i] = limit
				found = true
				break
			}
		}

		if found {
			// Update cache
			rlm.riskLimitCache.Set(limit.UserID+":"+limit.ID, limit, cache.DefaultExpiration)

			// Invalidate related caches
			rlm.invalidateCache(limit.UserID, limit.Symbol)

			op.ResultCh <- RiskLimitOperationResult{
				Success: true,
				Data:    limit,
			}
		} else {
			op.ResultCh <- RiskLimitOperationResult{
				Success: false,
				Error:   errors.New("risk limit not found"),
			}
		}
	}
}

// processDeleteBatch processes a batch of delete operations
func (rlm *RiskLimitsManager) processDeleteBatch(ops []RiskLimitOperation) {
	rlm.mu.Lock()
	defer rlm.mu.Unlock()

	for _, op := range ops {
		limitID := op.Limit.ID
		userID := op.UserID
		found := false

		// Delete from user limits
		if userLimits, exists := rlm.riskLimits[userID]; exists {
			for i, limit := range userLimits {
				if limit.ID == limitID {
					rlm.riskLimits[userID] = append(userLimits[:i], userLimits[i+1:]...)
					found = true
					break
				}
			}
		}

		if found {
			// Remove from cache
			rlm.riskLimitCache.Delete(userID + ":" + limitID)

			// Invalidate related caches
			rlm.invalidateCache(userID, "")

			op.ResultCh <- RiskLimitOperationResult{
				Success: true,
			}
		} else {
			op.ResultCh <- RiskLimitOperationResult{
				Success: false,
				Error:   errors.New("risk limit not found"),
			}
		}
	}
}

// setRiskLimitStatus sets the enabled status of a risk limit
func (rlm *RiskLimitsManager) setRiskLimitStatus(ctx context.Context, userID, limitID string, enabled bool) error {
	rlm.mu.Lock()
	defer rlm.mu.Unlock()

	// Find and update the limit
	if userLimits, exists := rlm.riskLimits[userID]; exists {
		for _, limit := range userLimits {
			if limit.ID == limitID {
				limit.Enabled = enabled
				limit.UpdatedAt = time.Now()

				// Update cache
				rlm.riskLimitCache.Set(userID+":"+limitID, limit, cache.DefaultExpiration)

				// Invalidate related caches
				rlm.invalidateCache(userID, limit.Symbol)

				return nil
			}
		}
	}

	return errors.New("risk limit not found")
}

// checkSpecificLimit checks a specific limit type
func (rlm *RiskLimitsManager) checkSpecificLimit(limit *RiskLimit, userID, symbol string, quantity, price float64, side string) (bool, string) {
	switch limit.Type {
	case RiskLimitTypeOrderSize:
		orderValue := quantity * price
		if orderValue > limit.Value {
			return true, "Order size limit exceeded"
		}
	case RiskLimitTypePosition:
		// This would require current position data
		// For now, just check the order quantity
		if quantity > limit.Value {
			return true, "Position limit would be exceeded"
		}
	case RiskLimitTypeExposure:
		// This would require current exposure calculation
		// For now, just check the order value
		orderValue := quantity * price
		if orderValue > limit.Value {
			return true, "Exposure limit would be exceeded"
		}
	}

	return false, ""
}

// invalidateCache invalidates related cache entries
func (rlm *RiskLimitsManager) invalidateCache(userID, symbol string) {
	if userID != "" {
		rlm.riskLimitCache.Delete("user_limits:" + userID)
	}
	if symbol != "" {
		rlm.riskLimitCache.Delete("symbol_limits:" + symbol)
	}
	rlm.riskLimitCache.Delete("global_limits")
}
