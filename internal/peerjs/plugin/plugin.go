package plugin

import (
	"plugin"
	"sync"

	"github.com/abdoElHodaky/tradSys/internal/peerjs"
	"go.uber.org/zap"
)

// PeerJSPlugin defines the interface for PeerJS plugins
type PeerJSPlugin interface {
	// Initialize initializes the plugin
	Initialize(server *peerjs.PeerServer, logger *zap.Logger) error
	
	// GetName returns the name of the plugin
	GetName() string
	
	// GetVersion returns the version of the plugin
	GetVersion() string
	
	// GetDescription returns the description of the plugin
	GetDescription() string
	
	// OnPeerConnected is called when a peer connects
	OnPeerConnected(peerID string)
	
	// OnPeerDisconnected is called when a peer disconnects
	OnPeerDisconnected(peerID string)
	
	// OnMessage is called when a message is received
	OnMessage(msg *peerjs.Message) bool // Return true if the message was handled
}

// PluginInfo contains information about a plugin
type PluginInfo struct {
	Name        string
	Version     string
	Description string
}

// Constants for plugin symbols
const (
	PluginInfoSymbol = "PluginInfo"
	CreatePluginSymbol = "CreatePlugin"
)

// PluginLoader loads PeerJS plugins
type PluginLoader struct {
	pluginDir string
	plugins   map[string]PeerJSPlugin
	logger    *zap.Logger
	server    *peerjs.PeerServer
	mu        sync.RWMutex
}

// NewPluginLoader creates a new plugin loader
func NewPluginLoader(pluginDir string, server *peerjs.PeerServer, logger *zap.Logger) *PluginLoader {
	return &PluginLoader{
		pluginDir: pluginDir,
		plugins:   make(map[string]PeerJSPlugin),
		logger:    logger,
		server:    server,
	}
}

// LoadPlugins loads all plugins from the plugin directory
func (l *PluginLoader) LoadPlugins() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Check if the plugin directory exists
	if _, err := os.Stat(l.pluginDir); os.IsNotExist(err) {
		l.logger.Warn("Plugin directory does not exist", zap.String("directory", l.pluginDir))
		return nil
	}

	// Find all .so files in the plugin directory
	files, err := filepath.Glob(filepath.Join(l.pluginDir, "*.so"))
	if err != nil {
		return fmt.Errorf("failed to list plugin files: %w", err)
	}

	for _, file := range files {
		if err := l.loadPlugin(file); err != nil {
			l.logger.Error("Failed to load plugin",
				zap.String("file", file),
				zap.Error(err))
			continue
		}
	}

	l.logger.Info("Loaded PeerJS plugins", zap.Int("count", len(l.plugins)))
	return nil
}

// loadPlugin loads a single plugin
func (l *PluginLoader) loadPlugin(path string) error {
	// Open the plugin
	p, err := plugin.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open plugin: %w", err)
	}

	// Look up the plugin info
	infoSymbol, err := p.Lookup(PluginInfoSymbol)
	if err != nil {
		return fmt.Errorf("plugin does not export %s: %w", PluginInfoSymbol, err)
	}

	info, ok := infoSymbol.(*PluginInfo)
	if !ok {
		return fmt.Errorf("plugin info is not of type *PluginInfo")
	}

	// Look up the create plugin function
	createSymbol, err := p.Lookup(CreatePluginSymbol)
	if err != nil {
		return fmt.Errorf("plugin does not export %s: %w", CreatePluginSymbol, err)
	}

	createFunc, ok := createSymbol.(func() PeerJSPlugin)
	if !ok {
		return fmt.Errorf("create plugin function has wrong signature")
	}

	// Create the plugin
	plugin := createFunc()

	// Initialize the plugin
	if err := plugin.Initialize(l.server, l.logger); err != nil {
		return fmt.Errorf("failed to initialize plugin: %w", err)
	}

	// Register the plugin
	l.plugins[plugin.GetName()] = plugin

	l.logger.Info("Loaded PeerJS plugin",
		zap.String("name", plugin.GetName()),
		zap.String("version", plugin.GetVersion()),
		zap.String("description", plugin.GetDescription()))

	return nil
}

// GetPlugin returns a plugin by name
func (l *PluginLoader) GetPlugin(name string) (PeerJSPlugin, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	plugin, ok := l.plugins[name]
	return plugin, ok
}

// GetPlugins returns all loaded plugins
func (l *PluginLoader) GetPlugins() []PeerJSPlugin {
	l.mu.RLock()
	defer l.mu.RUnlock()

	plugins := make([]PeerJSPlugin, 0, len(l.plugins))
	for _, plugin := range l.plugins {
		plugins = append(plugins, plugin)
	}

	return plugins
}

// NotifyPeerConnected notifies all plugins that a peer has connected
func (l *PluginLoader) NotifyPeerConnected(peerID string) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	for _, plugin := range l.plugins {
		plugin.OnPeerConnected(peerID)
	}
}

// NotifyPeerDisconnected notifies all plugins that a peer has disconnected
func (l *PluginLoader) NotifyPeerDisconnected(peerID string) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	for _, plugin := range l.plugins {
		plugin.OnPeerDisconnected(peerID)
	}
}

// HandleMessage passes a message to all plugins until one handles it
func (l *PluginLoader) HandleMessage(msg *peerjs.Message) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()

	for _, plugin := range l.plugins {
		if plugin.OnMessage(msg) {
			return true
		}
	}

	return false
}

