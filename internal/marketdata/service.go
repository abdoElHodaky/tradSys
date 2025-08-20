package marketdata

import (
	"context"
	"errors"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Common errors
var (
	ErrInvalidCommand = errors.New("invalid command")
	ErrInvalidQuery   = errors.New("invalid query")
	ErrSourceNotFound = errors.New("market data source not found")
)

// MarketData represents market data for a symbol
type MarketData struct {
	Symbol    string
	Price     float64
	Volume    float64
	Timestamp time.Time
}

// Service provides market data functionality
type Service struct {
	logger  *zap.Logger
	sources map[string]DataSource
	mu      sync.RWMutex
}

// DataSource represents a market data source
type DataSource interface {
	GetData(ctx context.Context, symbol string, timeRange string) ([]MarketData, error)
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

// NewService creates a new market data service
func NewService(params ServiceParams) *Service {
	return &Service{
		logger:  params.Logger,
		sources: make(map[string]DataSource),
	}
}

// AddMarketDataSource adds a market data source
func (s *Service) AddMarketDataSource(ctx context.Context, source string, config map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Check if the source already exists
	if _, exists := s.sources[source]; exists {
		s.logger.Info("Market data source already exists", zap.String("source", source))
		return nil
	}
	
	// Create a new data source based on the source type
	var dataSource DataSource
	switch source {
	case "websocket":
		dataSource = NewWebSocketDataSource(config, s.logger)
	case "rest":
		dataSource = NewRESTDataSource(config, s.logger)
	default:
		s.logger.Error("Unknown market data source type", zap.String("source", source))
		return errors.New("unknown market data source type")
	}
	
	// Start the data source
	if err := dataSource.Start(ctx); err != nil {
		s.logger.Error("Failed to start market data source", zap.String("source", source), zap.Error(err))
		return err
	}
	
	// Add the data source to the map
	s.sources[source] = dataSource
	s.logger.Info("Added market data source", zap.String("source", source))
	
	return nil
}

// RemoveMarketDataSource removes a market data source
func (s *Service) RemoveMarketDataSource(ctx context.Context, source string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Check if the source exists
	dataSource, exists := s.sources[source]
	if !exists {
		s.logger.Warn("Market data source not found", zap.String("source", source))
		return ErrSourceNotFound
	}
	
	// Stop the data source
	if err := dataSource.Stop(ctx); err != nil {
		s.logger.Error("Failed to stop market data source", zap.String("source", source), zap.Error(err))
		return err
	}
	
	// Remove the data source from the map
	delete(s.sources, source)
	s.logger.Info("Removed market data source", zap.String("source", source))
	
	return nil
}

// GetMarketData gets market data for a symbol
func (s *Service) GetMarketData(ctx context.Context, symbol string, timeRange string) ([]MarketData, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	// Check if there are any sources
	if len(s.sources) == 0 {
		s.logger.Warn("No market data sources available")
		return nil, errors.New("no market data sources available")
	}
	
	// Get data from all sources
	var allData []MarketData
	for sourceName, source := range s.sources {
		data, err := source.GetData(ctx, symbol, timeRange)
		if err != nil {
			s.logger.Warn("Failed to get market data from source", 
				zap.String("source", sourceName), 
				zap.String("symbol", symbol), 
				zap.Error(err))
			continue
		}
		
		allData = append(allData, data...)
	}
	
	// Sort data by timestamp (newest first)
	// In a real implementation, we would sort the data here
	
	return allData, nil
}

// WebSocketDataSource implements DataSource using WebSockets
type WebSocketDataSource struct {
	config map[string]interface{}
	logger *zap.Logger
	// Add WebSocket connection and other fields
}

// NewWebSocketDataSource creates a new WebSocket data source
func NewWebSocketDataSource(config map[string]interface{}, logger *zap.Logger) *WebSocketDataSource {
	return &WebSocketDataSource{
		config: config,
		logger: logger,
	}
}

// GetData gets market data from the WebSocket data source
func (ds *WebSocketDataSource) GetData(ctx context.Context, symbol string, timeRange string) ([]MarketData, error) {
	// In a real implementation, this would get data from the WebSocket connection
	// For now, return some dummy data
	return []MarketData{
		{
			Symbol:    symbol,
			Price:     100.0,
			Volume:    1000.0,
			Timestamp: time.Now(),
		},
	}, nil
}

// Start starts the WebSocket data source
func (ds *WebSocketDataSource) Start(ctx context.Context) error {
	// In a real implementation, this would establish the WebSocket connection
	ds.logger.Info("Starting WebSocket data source")
	return nil
}

// Stop stops the WebSocket data source
func (ds *WebSocketDataSource) Stop(ctx context.Context) error {
	// In a real implementation, this would close the WebSocket connection
	ds.logger.Info("Stopping WebSocket data source")
	return nil
}

// RESTDataSource implements DataSource using REST APIs
type RESTDataSource struct {
	config map[string]interface{}
	logger *zap.Logger
	// Add HTTP client and other fields
}

// NewRESTDataSource creates a new REST data source
func NewRESTDataSource(config map[string]interface{}, logger *zap.Logger) *RESTDataSource {
	return &RESTDataSource{
		config: config,
		logger: logger,
	}
}

// GetData gets market data from the REST data source
func (ds *RESTDataSource) GetData(ctx context.Context, symbol string, timeRange string) ([]MarketData, error) {
	// In a real implementation, this would make HTTP requests to get data
	// For now, return some dummy data
	return []MarketData{
		{
			Symbol:    symbol,
			Price:     100.0,
			Volume:    1000.0,
			Timestamp: time.Now(),
		},
	}, nil
}

// Start starts the REST data source
func (ds *RESTDataSource) Start(ctx context.Context) error {
	// In a real implementation, this would initialize the HTTP client
	ds.logger.Info("Starting REST data source")
	return nil
}

// Stop stops the REST data source
func (ds *RESTDataSource) Stop(ctx context.Context) error {
	// In a real implementation, this would clean up the HTTP client
	ds.logger.Info("Stopping REST data source")
	return nil
}

