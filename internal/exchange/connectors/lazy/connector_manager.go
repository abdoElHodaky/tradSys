package lazy

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/architecture/coordination"
	"github.com/abdoElHodaky/tradSys/internal/architecture/fx/lazy"
	"github.com/abdoElHodaky/tradSys/internal/exchange/connectors"
	"github.com/abdoElHodaky/tradSys/proto/marketdata"
	"go.uber.org/zap"
)

// LazyConnectorManager is a lazy-loaded manager for exchange connectors
type LazyConnectorManager struct {
	// Component coordinator
	coordinator *coordination.ComponentCoordinator
	
	// Component name prefix
	componentNamePrefix string
	
	// Configuration
	config connectors.ConnectorConfig
	
	// Logger
	logger *zap.Logger
	
	// Lock manager for thread safety
	lockManager *coordination.LockManager
	
	// Connector factory
	factory connectors.ConnectorFactory
	
	// Active connectors
	activeConnectors map[string]bool
	activeConnectorsMu sync.RWMutex
}

// NewLazyConnectorManager creates a new lazy-loaded connector manager
func NewLazyConnectorManager(
	coordinator *coordination.ComponentCoordinator,
	lockManager *coordination.LockManager,
	factory connectors.ConnectorFactory,
	config connectors.ConnectorConfig,
	logger *zap.Logger,
) (*LazyConnectorManager, error) {
	componentNamePrefix := "exchange-connector-"
	
	// Register the lock for connector operations
	lockManager.RegisterLock("exchange-connectors", &sync.Mutex{})
	
	return &LazyConnectorManager{
		coordinator:         coordinator,
		componentNamePrefix: componentNamePrefix,
		config:              config,
		logger:              logger,
		lockManager:         lockManager,
		factory:             factory,
		activeConnectors:    make(map[string]bool),
	}, nil
}

// GetConnector gets a connector for an exchange
func (m *LazyConnectorManager) GetConnector(
	ctx context.Context,
	exchangeName string,
) (connectors.ExchangeConnector, error) {
	componentName := m.componentNamePrefix + exchangeName
	
	// Check if the component is already registered
	_, err := m.coordinator.GetComponentInfo(componentName)
	if err != nil {
		// Component not registered, register it
		err = m.registerConnector(ctx, exchangeName, componentName)
		if err != nil {
			return nil, err
		}
	}
	
	// Get the component
	connectorInterface, err := m.coordinator.GetComponent(ctx, componentName)
	if err != nil {
		return nil, err
	}
	
	// Cast to the actual connector type
	connector, ok := connectorInterface.(connectors.ExchangeConnector)
	if !ok {
		return nil, fmt.Errorf("invalid connector type for exchange %s", exchangeName)
	}
	
	// Update active connectors
	m.activeConnectorsMu.Lock()
	m.activeConnectors[exchangeName] = true
	m.activeConnectorsMu.Unlock()
	
	return connector, nil
}

// registerConnector registers a connector with the coordinator
func (m *LazyConnectorManager) registerConnector(
	ctx context.Context,
	exchangeName string,
	componentName string,
) error {
	// Acquire the lock to prevent concurrent connector creation
	err := m.lockManager.AcquireLock("exchange-connectors", "connector-manager")
	if err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	defer m.lockManager.ReleaseLock("exchange-connectors", "connector-manager")
	
	// Create the provider function
	providerFn := func(log *zap.Logger) (interface{}, error) {
		// Create the connector
		connector, err := m.factory.CreateConnector(exchangeName, m.config, log)
		if err != nil {
			return nil, err
		}
		
		// Initialize the connector
		err = connector.Initialize(context.Background())
		if err != nil {
			return nil, err
		}
		
		return connector, nil
	}
	
	// Create the lazy provider
	provider := lazy.NewEnhancedLazyProvider(
		componentName,
		providerFn,
		m.logger,
		nil, // Metrics will be handled by the coordinator
		lazy.WithMemoryEstimate(75*1024*1024), // 75MB estimate
		lazy.WithTimeout(45*time.Second),      // Exchange connections can take time
		lazy.WithPriority(25),                 // Medium-high priority
	)
	
	// Register with the coordinator
	return m.coordinator.RegisterComponent(
		componentName,
		"exchange-connector",
		provider,
		[]string{}, // No dependencies
	)
}

// ReleaseConnector releases a connector
func (m *LazyConnectorManager) ReleaseConnector(
	ctx context.Context,
	exchangeName string,
) error {
	componentName := m.componentNamePrefix + exchangeName
	
	// Update active connectors
	m.activeConnectorsMu.Lock()
	delete(m.activeConnectors, exchangeName)
	m.activeConnectorsMu.Unlock()
	
	// Shutdown the component
	return m.coordinator.ShutdownComponent(ctx, componentName)
}

// GetMarketData gets market data from an exchange
func (m *LazyConnectorManager) GetMarketData(
	ctx context.Context,
	exchangeName string,
	symbol string,
) (*marketdata.MarketDataResponse, error) {
	// Get the connector
	connector, err := m.GetConnector(ctx, exchangeName)
	if err != nil {
		return nil, err
	}
	
	// Call the actual method
	return connector.GetMarketData(ctx, symbol)
}

// SubscribeMarketData subscribes to market data from an exchange
func (m *LazyConnectorManager) SubscribeMarketData(
	ctx context.Context,
	exchangeName string,
	symbol string,
	callback func(*marketdata.MarketDataResponse),
) error {
	// Get the connector
	connector, err := m.GetConnector(ctx, exchangeName)
	if err != nil {
		return err
	}
	
	// Call the actual method
	return connector.SubscribeMarketData(ctx, symbol, callback)
}

// UnsubscribeMarketData unsubscribes from market data from an exchange
func (m *LazyConnectorManager) UnsubscribeMarketData(
	ctx context.Context,
	exchangeName string,
	symbol string,
) error {
	// Get the connector
	connector, err := m.GetConnector(ctx, exchangeName)
	if err != nil {
		return err
	}
	
	// Call the actual method
	return connector.UnsubscribeMarketData(ctx, symbol)
}

// ListActiveConnectors lists active connectors
func (m *LazyConnectorManager) ListActiveConnectors() []string {
	m.activeConnectorsMu.RLock()
	defer m.activeConnectorsMu.RUnlock()
	
	connectors := make([]string, 0, len(m.activeConnectors))
	for connector := range m.activeConnectors {
		connectors = append(connectors, connector)
	}
	
	return connectors
}

// ShutdownAll shuts down all connectors
func (m *LazyConnectorManager) ShutdownAll(ctx context.Context) error {
	m.activeConnectorsMu.RLock()
	activeConnectors := make([]string, 0, len(m.activeConnectors))
	for connector := range m.activeConnectors {
		activeConnectors = append(activeConnectors, connector)
	}
	m.activeConnectorsMu.RUnlock()
	
	var lastErr error
	for _, connector := range activeConnectors {
		err := m.ReleaseConnector(ctx, connector)
		if err != nil {
			lastErr = err
			m.logger.Error("Failed to release connector",
				zap.String("connector", connector),
				zap.Error(err),
			)
		}
	}
	
	return lastErr
}

