package services

import (
	"context"
	"fmt"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/db/models"
	"github.com/abdoElHodaky/tradSys/internal/trading/types"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// AssetService provides asset-related operations
type AssetService struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewAssetService creates a new asset service
func NewAssetService(db *gorm.DB, logger *zap.Logger) *AssetService {
	return &AssetService{
		db:     db,
		logger: logger,
	}
}

// GetAssetMetadata retrieves metadata for a specific asset
func (s *AssetService) GetAssetMetadata(ctx context.Context, symbol string) (*models.AssetMetadata, error) {
	var metadata models.AssetMetadata
	err := s.db.WithContext(ctx).Where("symbol = ? AND is_active = ?", symbol, true).First(&metadata).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("asset metadata not found for symbol: %s", symbol)
		}
		return nil, fmt.Errorf("failed to get asset metadata: %w", err)
	}
	return &metadata, nil
}

// CreateAssetMetadata creates new asset metadata
func (s *AssetService) CreateAssetMetadata(ctx context.Context, metadata *models.AssetMetadata) error {
	if err := s.validateAssetMetadata(metadata); err != nil {
		return fmt.Errorf("invalid asset metadata: %w", err)
	}

	err := s.db.WithContext(ctx).Create(metadata).Error
	if err != nil {
		return fmt.Errorf("failed to create asset metadata: %w", err)
	}

	s.logger.Info("Created asset metadata", 
		zap.String("symbol", metadata.Symbol),
		zap.String("asset_type", string(metadata.AssetType)))
	return nil
}

// UpdateAssetMetadata updates existing asset metadata
func (s *AssetService) UpdateAssetMetadata(ctx context.Context, symbol string, updates *models.AssetMetadata) error {
	if err := s.validateAssetMetadata(updates); err != nil {
		return fmt.Errorf("invalid asset metadata: %w", err)
	}

	result := s.db.WithContext(ctx).Model(&models.AssetMetadata{}).
		Where("symbol = ?", symbol).
		Updates(updates)
	
	if result.Error != nil {
		return fmt.Errorf("failed to update asset metadata: %w", result.Error)
	}
	
	if result.RowsAffected == 0 {
		return fmt.Errorf("asset metadata not found for symbol: %s", symbol)
	}

	s.logger.Info("Updated asset metadata", zap.String("symbol", symbol))
	return nil
}

// GetAssetConfiguration retrieves configuration for an asset type
func (s *AssetService) GetAssetConfiguration(ctx context.Context, assetType types.AssetType) (*models.AssetConfiguration, error) {
	var config models.AssetConfiguration
	err := s.db.WithContext(ctx).Where("asset_type = ?", assetType).First(&config).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("asset configuration not found for type: %s", assetType)
		}
		return nil, fmt.Errorf("failed to get asset configuration: %w", err)
	}
	return &config, nil
}

// GetAssetsByType retrieves all assets of a specific type
func (s *AssetService) GetAssetsByType(ctx context.Context, assetType types.AssetType) ([]*models.AssetMetadata, error) {
	var assets []*models.AssetMetadata
	err := s.db.WithContext(ctx).
		Where("asset_type = ? AND is_active = ?", assetType, true).
		Find(&assets).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get assets by type: %w", err)
	}
	return assets, nil
}

// GetAssetPricing retrieves latest pricing for an asset
func (s *AssetService) GetAssetPricing(ctx context.Context, symbol string) (*models.AssetPricing, error) {
	var pricing models.AssetPricing
	err := s.db.WithContext(ctx).
		Where("symbol = ?", symbol).
		Order("timestamp DESC").
		First(&pricing).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("pricing not found for symbol: %s", symbol)
		}
		return nil, fmt.Errorf("failed to get asset pricing: %w", err)
	}
	return &pricing, nil
}

// UpdateAssetPricing updates or creates pricing information
func (s *AssetService) UpdateAssetPricing(ctx context.Context, pricing *models.AssetPricing) error {
	if pricing.Symbol == "" || pricing.Price <= 0 {
		return fmt.Errorf("invalid pricing data: symbol and price are required")
	}

	pricing.Timestamp = time.Now()
	err := s.db.WithContext(ctx).Create(pricing).Error
	if err != nil {
		return fmt.Errorf("failed to update asset pricing: %w", err)
	}

	return nil
}

// GetAssetDividends retrieves dividend information for an asset
func (s *AssetService) GetAssetDividends(ctx context.Context, symbol string, limit int) ([]*models.AssetDividend, error) {
	var dividends []*models.AssetDividend
	query := s.db.WithContext(ctx).Where("symbol = ?", symbol).Order("ex_date DESC")
	
	if limit > 0 {
		query = query.Limit(limit)
	}
	
	err := query.Find(&dividends).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get asset dividends: %w", err)
	}
	return dividends, nil
}

// CreateAssetDividend creates a new dividend record
func (s *AssetService) CreateAssetDividend(ctx context.Context, dividend *models.AssetDividend) error {
	if dividend.Symbol == "" || dividend.Amount <= 0 {
		return fmt.Errorf("invalid dividend data: symbol and amount are required")
	}

	err := s.db.WithContext(ctx).Create(dividend).Error
	if err != nil {
		return fmt.Errorf("failed to create asset dividend: %w", err)
	}

	s.logger.Info("Created asset dividend", 
		zap.String("symbol", dividend.Symbol),
		zap.Float64("amount", dividend.Amount),
		zap.Time("ex_date", dividend.ExDate))
	return nil
}

// IsAssetTradeable checks if an asset type is enabled for trading
func (s *AssetService) IsAssetTradeable(ctx context.Context, assetType types.AssetType) (bool, error) {
	config, err := s.GetAssetConfiguration(ctx, assetType)
	if err != nil {
		return false, err
	}
	return config.TradingEnabled, nil
}

// GetTradingLimits returns trading limits for an asset type
func (s *AssetService) GetTradingLimits(ctx context.Context, assetType types.AssetType) (minSize, maxSize float64, err error) {
	config, err := s.GetAssetConfiguration(ctx, assetType)
	if err != nil {
		return 0, 0, err
	}
	return config.MinOrderSize, config.MaxOrderSize, nil
}

// ValidateOrderForAsset validates an order against asset-specific rules
func (s *AssetService) ValidateOrderForAsset(ctx context.Context, symbol string, assetType types.AssetType, quantity, price float64) error {
	// Get asset configuration
	config, err := s.GetAssetConfiguration(ctx, assetType)
	if err != nil {
		return fmt.Errorf("failed to get asset configuration: %w", err)
	}

	// Check if trading is enabled
	if !config.TradingEnabled {
		return fmt.Errorf("trading is disabled for asset type: %s", assetType)
	}

	// Check order size limits
	orderValue := quantity * price
	if orderValue < config.MinOrderSize {
		return fmt.Errorf("order size %.2f is below minimum %.2f for asset type %s", 
			orderValue, config.MinOrderSize, assetType)
	}
	if orderValue > config.MaxOrderSize {
		return fmt.Errorf("order size %.2f exceeds maximum %.2f for asset type %s", 
			orderValue, config.MaxOrderSize, assetType)
	}

	// Check quantity increment
	if config.QuantityIncrement > 0 {
		remainder := quantity - (float64(int(quantity/config.QuantityIncrement)) * config.QuantityIncrement)
		if remainder > 0.0001 { // Small tolerance for floating point precision
			return fmt.Errorf("quantity %.8f does not match increment %.8f for asset type %s", 
				quantity, config.QuantityIncrement, assetType)
		}
	}

	// Check price increment
	if config.PriceIncrement > 0 {
		remainder := price - (float64(int(price/config.PriceIncrement)) * config.PriceIncrement)
		if remainder > 0.0001 { // Small tolerance for floating point precision
			return fmt.Errorf("price %.8f does not match increment %.8f for asset type %s", 
				price, config.PriceIncrement, assetType)
		}
	}

	return nil
}

// GetAssetInfo returns comprehensive information about an asset
func (s *AssetService) GetAssetInfo(ctx context.Context, symbol string) (*types.AssetInfo, error) {
	metadata, err := s.GetAssetMetadata(ctx, symbol)
	if err != nil {
		return nil, err
	}

	return &types.AssetInfo{
		Symbol: metadata.Symbol,
		Type:   metadata.AssetType,
		Metadata: types.AssetMetadata{
			Sector:     metadata.Sector,
			Industry:   metadata.Industry,
			Country:    metadata.Country,
			Currency:   metadata.Currency,
			Exchange:   metadata.Exchange,
			Attributes: map[string]interface{}(metadata.Attributes),
		},
	}, nil
}

// validateAssetMetadata validates asset metadata before creation/update
func (s *AssetService) validateAssetMetadata(metadata *models.AssetMetadata) error {
	if metadata.Symbol == "" {
		return fmt.Errorf("symbol is required")
	}
	if !metadata.AssetType.IsValid() {
		return fmt.Errorf("invalid asset type: %s", metadata.AssetType)
	}
	return nil
}

// ListAssets returns a paginated list of assets
func (s *AssetService) ListAssets(ctx context.Context, offset, limit int, assetType *types.AssetType) ([]*models.AssetMetadata, int64, error) {
	var assets []*models.AssetMetadata
	var total int64

	query := s.db.WithContext(ctx).Model(&models.AssetMetadata{}).Where("is_active = ?", true)
	
	if assetType != nil {
		query = query.Where("asset_type = ?", *assetType)
	}

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count assets: %w", err)
	}

	// Get paginated results
	if err := query.Offset(offset).Limit(limit).Find(&assets).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list assets: %w", err)
	}

	return assets, total, nil
}

// GetAssetBySymbol retrieves asset metadata by symbol (alias for GetAssetMetadata)
func (s *AssetService) GetAssetBySymbol(ctx context.Context, symbol string) (*models.AssetMetadata, error) {
	return s.GetAssetMetadata(ctx, symbol)
}

// GetCurrentPricing retrieves current pricing for an asset (alias for GetAssetPricing)
func (s *AssetService) GetCurrentPricing(ctx context.Context, symbol string) (*models.AssetPricing, error) {
	return s.GetAssetPricing(ctx, symbol)
}
