// Package services provides unified trading services for TradSys v3
package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/pkg/interfaces"
	"github.com/abdoElHodaky/tradSys/pkg/types"
	"github.com/abdoElHodaky/tradSys/services/assets"
	"github.com/abdoElHodaky/tradSys/services/exchange"
	"github.com/abdoElHodaky/tradSys/services/licensing"
)

// OrderManager provides unified order management across all exchanges
type OrderManager struct {
	exchangeFactory  *exchange.Factory
	assetRegistry    *assets.HandlerRegistry
	licenseValidator *licensing.Validator
	orderStore       OrderStore
	riskManager      *RiskManager
	config           *OrderManagerConfig
	mu               sync.RWMutex
}

// OrderManagerConfig holds configuration for the order manager
type OrderManagerConfig struct {
	MaxOrdersPerUser    int
	MaxOrderValue       float64
	EnableRiskChecks    bool
	EnableLicenseChecks bool
	OrderTimeout        time.Duration
	RetryAttempts       int
}

// OrderStore interface for persisting orders
type OrderStore interface {
	SaveOrder(ctx context.Context, order *interfaces.Order) error
	GetOrder(ctx context.Context, orderID string) (*interfaces.Order, error)
	UpdateOrder(ctx context.Context, order *interfaces.Order) error
	GetUserOrders(ctx context.Context, userID string, limit int) ([]*interfaces.Order, error)
	GetOrdersByStatus(ctx context.Context, status string, limit int) ([]*interfaces.Order, error)
}

// OrderRequest represents a request to place an order
type OrderRequest struct {
	UserID      string                 `json:"user_id"`
	Symbol      string                 `json:"symbol"`
	AssetType   types.AssetType        `json:"asset_type"`
	Exchange    types.ExchangeType     `json:"exchange"`
	Side        interfaces.OrderSide   `json:"side"`
	Type        interfaces.OrderType   `json:"type"`
	Quantity    float64                `json:"quantity"`
	Price       float64                `json:"price"`
	TimeInForce interfaces.TimeInForce `json:"time_in_force"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// OrderResult represents the result of an order operation
type OrderResult struct {
	Order        *interfaces.Order           `json:"order"`
	Response     *interfaces.OrderResponse   `json:"response"`
	Settlement   *assets.Settlement          `json:"settlement,omitempty"`
	Fees         *assets.FeeCalculation      `json:"fees,omitempty"`
	RiskCheck    *RiskCheckResult            `json:"risk_check,omitempty"`
	LicenseCheck *licensing.ValidationResult `json:"license_check,omitempty"`
}

// NewOrderManager creates a new order manager
func NewOrderManager(
	exchangeFactory *exchange.Factory,
	assetRegistry *assets.HandlerRegistry,
	licenseValidator *licensing.Validator,
	orderStore OrderStore,
	riskManager *RiskManager,
	config *OrderManagerConfig,
) *OrderManager {
	if config == nil {
		config = GetDefaultOrderManagerConfig()
	}

	return &OrderManager{
		exchangeFactory:  exchangeFactory,
		assetRegistry:    assetRegistry,
		licenseValidator: licenseValidator,
		orderStore:       orderStore,
		riskManager:      riskManager,
		config:           config,
	}
}

// PlaceOrder places a new order with comprehensive validation
func (om *OrderManager) PlaceOrder(ctx context.Context, request *OrderRequest) (*OrderResult, error) {
	// Create order from request
	order := &interfaces.Order{
		ID:          generateOrderID(),
		Symbol:      request.Symbol,
		AssetType:   request.AssetType,
		Exchange:    request.Exchange,
		Side:        request.Side,
		Type:        request.Type,
		Quantity:    request.Quantity,
		Price:       request.Price,
		TimeInForce: request.TimeInForce,
		UserID:      request.UserID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	result := &OrderResult{
		Order: order,
	}

	// 1. License validation
	if om.config.EnableLicenseChecks && om.licenseValidator != nil {
		licenseResult, err := om.validateLicense(ctx, request)
		if err != nil {
			return result, fmt.Errorf("license validation failed: %w", err)
		}
		result.LicenseCheck = licenseResult

		if !licenseResult.Valid {
			return result, fmt.Errorf("license validation failed: %s", licenseResult.Reason)
		}
	}

	// 2. Risk management checks
	if om.config.EnableRiskChecks && om.riskManager != nil {
		riskResult, err := om.riskManager.ValidateOrder(ctx, order)
		if err != nil {
			return result, fmt.Errorf("risk validation failed: %w", err)
		}
		result.RiskCheck = riskResult

		if !riskResult.Approved {
			return result, fmt.Errorf("risk validation failed: %s", riskResult.Reason)
		}
	}

	// 3. Asset-specific validation
	if err := om.assetRegistry.ValidateOrder(ctx, order); err != nil {
		return result, fmt.Errorf("asset validation failed: %w", err)
	}

	// 4. Calculate fees and settlement
	settlement, err := om.assetRegistry.CalculateSettlement(ctx, order)
	if err != nil {
		return result, fmt.Errorf("settlement calculation failed: %w", err)
	}
	result.Settlement = settlement

	// 5. Get exchange client
	exchangeClient, err := om.exchangeFactory.GetExchange(request.Exchange)
	if err != nil {
		return result, fmt.Errorf("exchange not available: %w", err)
	}

	// 6. Place order on exchange
	response, err := exchangeClient.PlaceOrder(ctx, order)
	if err != nil {
		return result, fmt.Errorf("exchange order failed: %w", err)
	}
	result.Response = response

	// 7. Save order to store
	if err := om.orderStore.SaveOrder(ctx, order); err != nil {
		// Log error but don't fail the order
		fmt.Printf("Failed to save order to store: %v\n", err)
	}

	// 8. Record usage for licensing
	if om.config.EnableLicenseChecks && om.licenseValidator != nil {
		if err := om.licenseValidator.RecordUsage(ctx, request.UserID, "orders_per_day", 1); err != nil {
			// Log error but don't fail the order
			fmt.Printf("Failed to record usage: %v\n", err)
		}
	}

	return result, nil
}

// CancelOrder cancels an existing order
func (om *OrderManager) CancelOrder(ctx context.Context, userID, orderID string) error {
	// Get order from store
	order, err := om.orderStore.GetOrder(ctx, orderID)
	if err != nil {
		return fmt.Errorf("order not found: %w", err)
	}

	// Verify user ownership
	if order.UserID != userID {
		return fmt.Errorf("unauthorized: order belongs to different user")
	}

	// Get exchange client
	exchangeClient, err := om.exchangeFactory.GetExchange(order.Exchange)
	if err != nil {
		return fmt.Errorf("exchange not available: %w", err)
	}

	// Cancel order on exchange
	if err := exchangeClient.CancelOrder(ctx, orderID); err != nil {
		return fmt.Errorf("exchange cancel failed: %w", err)
	}

	// Update order status
	order.UpdatedAt = time.Now()
	if err := om.orderStore.UpdateOrder(ctx, order); err != nil {
		// Log error but don't fail the cancellation
		fmt.Printf("Failed to update order in store: %v\n", err)
	}

	return nil
}

// GetOrderStatus retrieves the status of an order
func (om *OrderManager) GetOrderStatus(ctx context.Context, userID, orderID string) (*interfaces.OrderStatus, error) {
	// Get order from store
	order, err := om.orderStore.GetOrder(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("order not found: %w", err)
	}

	// Verify user ownership
	if order.UserID != userID {
		return nil, fmt.Errorf("unauthorized: order belongs to different user")
	}

	// Get exchange client
	exchangeClient, err := om.exchangeFactory.GetExchange(order.Exchange)
	if err != nil {
		return nil, fmt.Errorf("exchange not available: %w", err)
	}

	// Get status from exchange
	return exchangeClient.GetOrderStatus(ctx, orderID)
}

// GetUserOrders retrieves orders for a user
func (om *OrderManager) GetUserOrders(ctx context.Context, userID string, limit int) ([]*interfaces.Order, error) {
	if limit <= 0 || limit > 1000 {
		limit = 100 // Default limit
	}

	return om.orderStore.GetUserOrders(ctx, userID, limit)
}

// GetOrdersByStatus retrieves orders by status
func (om *OrderManager) GetOrdersByStatus(ctx context.Context, status string, limit int) ([]*interfaces.Order, error) {
	if limit <= 0 || limit > 1000 {
		limit = 100 // Default limit
	}

	return om.orderStore.GetOrdersByStatus(ctx, status, limit)
}

// validateLicense validates user license for trading
func (om *OrderManager) validateLicense(ctx context.Context, request *OrderRequest) (*licensing.ValidationResult, error) {
	// Check basic trading feature
	result, err := om.licenseValidator.ValidateFeature(ctx, request.UserID, licensing.BASIC_TRADING)
	if err != nil {
		return nil, err
	}

	if !result.Valid {
		return result, nil
	}

	// Check exchange-specific access
	var exchangeFeature licensing.LicenseFeature
	switch request.Exchange {
	case types.EGX:
		exchangeFeature = licensing.EGX_ACCESS
	case types.ADX:
		exchangeFeature = licensing.ADX_ACCESS
	default:
		return result, nil // No specific exchange feature required
	}

	return om.licenseValidator.ValidateFeature(ctx, request.UserID, exchangeFeature)
}

// validateOrderLimits validates order against user limits
func (om *OrderManager) validateOrderLimits(ctx context.Context, request *OrderRequest) error {
	// Check maximum order value
	orderValue := request.Price * request.Quantity
	if orderValue > om.config.MaxOrderValue {
		return fmt.Errorf("order value %f exceeds maximum %f", orderValue, om.config.MaxOrderValue)
	}

	// Check maximum orders per user
	userOrders, err := om.orderStore.GetUserOrders(ctx, request.UserID, om.config.MaxOrdersPerUser+1)
	if err != nil {
		return fmt.Errorf("failed to check user order count: %w", err)
	}

	if len(userOrders) >= om.config.MaxOrdersPerUser {
		return fmt.Errorf("user has reached maximum order limit of %d", om.config.MaxOrdersPerUser)
	}

	return nil
}

// generateOrderID generates a unique order ID
func generateOrderID() string {
	return fmt.Sprintf("ORD_%d_%d", time.Now().UnixNano(), time.Now().Unix())
}

// GetDefaultOrderManagerConfig returns default configuration
func GetDefaultOrderManagerConfig() *OrderManagerConfig {
	return &OrderManagerConfig{
		MaxOrdersPerUser:    1000,
		MaxOrderValue:       1000000, // $1M
		EnableRiskChecks:    true,
		EnableLicenseChecks: true,
		OrderTimeout:        30 * time.Second,
		RetryAttempts:       3,
	}
}

// OrderStatistics represents order statistics
type OrderStatistics struct {
	TotalOrders      int64   `json:"total_orders"`
	ActiveOrders     int64   `json:"active_orders"`
	CompletedOrders  int64   `json:"completed_orders"`
	CancelledOrders  int64   `json:"cancelled_orders"`
	TotalVolume      float64 `json:"total_volume"`
	AverageOrderSize float64 `json:"average_order_size"`
}

// GetOrderStatistics returns order statistics for a user
func (om *OrderManager) GetOrderStatistics(ctx context.Context, userID string) (*OrderStatistics, error) {
	// This would typically query the database for statistics
	// For now, return a simplified implementation

	orders, err := om.orderStore.GetUserOrders(ctx, userID, 10000) // Get many orders for stats
	if err != nil {
		return nil, fmt.Errorf("failed to get user orders: %w", err)
	}

	stats := &OrderStatistics{
		TotalOrders: int64(len(orders)),
	}

	var totalVolume float64
	for _, order := range orders {
		orderValue := order.Price * order.Quantity
		totalVolume += orderValue

		// This would typically check actual order status from exchange
		// For now, using simplified logic
		stats.CompletedOrders++
	}

	stats.TotalVolume = totalVolume
	if stats.TotalOrders > 0 {
		stats.AverageOrderSize = totalVolume / float64(stats.TotalOrders)
	}

	return stats, nil
}

// BatchPlaceOrders places multiple orders in a batch
func (om *OrderManager) BatchPlaceOrders(ctx context.Context, requests []*OrderRequest) ([]*OrderResult, error) {
	if len(requests) == 0 {
		return nil, fmt.Errorf("no orders to place")
	}

	if len(requests) > 100 {
		return nil, fmt.Errorf("batch size too large, maximum 100 orders")
	}

	results := make([]*OrderResult, len(requests))

	// Process orders concurrently with limited concurrency
	semaphore := make(chan struct{}, 10) // Limit to 10 concurrent orders
	var wg sync.WaitGroup

	for i, request := range requests {
		wg.Add(1)
		go func(index int, req *OrderRequest) {
			defer wg.Done()

			semaphore <- struct{}{}        // Acquire semaphore
			defer func() { <-semaphore }() // Release semaphore

			result, err := om.PlaceOrder(ctx, req)
			if err != nil {
				result = &OrderResult{
					Order: &interfaces.Order{
						UserID:    req.UserID,
						Symbol:    req.Symbol,
						AssetType: req.AssetType,
						Exchange:  req.Exchange,
					},
				}
				// Store error in metadata or handle appropriately
			}
			results[index] = result
		}(i, request)
	}

	wg.Wait()
	return results, nil
}
