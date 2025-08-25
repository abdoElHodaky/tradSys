package sandbox

import (
	"path/filepath"
	"strings"
	"sync"
)

// PluginPermission represents a specific capability a plugin may have
type PluginPermission string

const (
	// File system permissions
	PermReadFiles  PluginPermission = "fs:read"
	PermWriteFiles PluginPermission = "fs:write"
	
	// Network permissions
	PermNetworkOutbound PluginPermission = "net:outbound"
	PermNetworkInbound  PluginPermission = "net:inbound"
	
	// System permissions
	PermExecuteCommands PluginPermission = "sys:exec"
	PermAccessMemory    PluginPermission = "sys:memory"
	
	// Plugin permissions
	PermLoadPlugins     PluginPermission = "plugin:load"
	PermUnloadPlugins   PluginPermission = "plugin:unload"
	
	// Event permissions
	PermPublishEvents   PluginPermission = "event:publish"
	PermSubscribeEvents PluginPermission = "event:subscribe"
)

// FileAccessType represents the type of file access
type FileAccessType int

const (
	FileAccessRead FileAccessType = iota
	FileAccessWrite
	FileAccessExecute
)

// PathAccessRule defines access rules for a specific path pattern
type PathAccessRule struct {
	Pattern     string
	AllowRead   bool
	AllowWrite  bool
	AllowExecute bool
}

// SecurityPolicy defines security constraints for a plugin
type SecurityPolicy struct {
	// Permissions granted to the plugin
	Permissions map[PluginPermission]bool
	
	// Path access rules
	PathRules []PathAccessRule
	
	// Network access rules
	AllowedHosts []string
	AllowedPorts []int
	
	mu sync.RWMutex
}

// NewDefaultSecurityPolicy creates a new security policy with default settings
func NewDefaultSecurityPolicy() *SecurityPolicy {
	return &SecurityPolicy{
		Permissions: map[PluginPermission]bool{
			PermReadFiles:      true,  // Allow reading files by default
			PermWriteFiles:     false, // Disallow writing files by default
			PermNetworkOutbound: false, // Disallow outbound network by default
			PermNetworkInbound:  false, // Disallow inbound network by default
			PermExecuteCommands: false, // Disallow executing commands by default
			PermAccessMemory:    false, // Disallow direct memory access by default
			PermLoadPlugins:     false, // Disallow loading plugins by default
			PermUnloadPlugins:   false, // Disallow unloading plugins by default
			PermPublishEvents:   true,  // Allow publishing events by default
			PermSubscribeEvents: true,  // Allow subscribing to events by default
		},
		PathRules: []PathAccessRule{
			{
				Pattern:     "./plugins",
				AllowRead:   true,
				AllowWrite:  false,
				AllowExecute: false,
			},
			{
				Pattern:     "./data",
				AllowRead:   true,
				AllowWrite:  false,
				AllowExecute: false,
			},
		},
		AllowedHosts: []string{},
		AllowedPorts: []int{},
	}
}

// GrantPermission grants a specific permission to the plugin
func (p *SecurityPolicy) GrantPermission(perm PluginPermission) {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	p.Permissions[perm] = true
}

// RevokePermission revokes a specific permission from the plugin
func (p *SecurityPolicy) RevokePermission(perm PluginPermission) {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	p.Permissions[perm] = false
}

// CheckPermission checks if a specific permission is granted
func (p *SecurityPolicy) CheckPermission(perm PluginPermission) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	allowed, exists := p.Permissions[perm]
	return exists && allowed
}

// AddPathRule adds a path access rule
func (p *SecurityPolicy) AddPathRule(rule PathAccessRule) {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	p.PathRules = append(p.PathRules, rule)
}

// CheckFileAccess checks if access to a specific file path is allowed
func (p *SecurityPolicy) CheckFileAccess(path string, accessType FileAccessType) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	// Check if the path matches any rule
	for _, rule := range p.PathRules {
		if pathMatchesPattern(path, rule.Pattern) {
			switch accessType {
			case FileAccessRead:
				return rule.AllowRead
			case FileAccessWrite:
				return rule.AllowWrite
			case FileAccessExecute:
				return rule.AllowExecute
			}
		}
	}
	
	// If no rule matches, deny access
	return false
}

// AddAllowedHost adds a host to the allowed hosts list
func (p *SecurityPolicy) AddAllowedHost(host string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	p.AllowedHosts = append(p.AllowedHosts, host)
}

// AddAllowedPort adds a port to the allowed ports list
func (p *SecurityPolicy) AddAllowedPort(port int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	p.AllowedPorts = append(p.AllowedPorts, port)
}

// CheckNetworkAccess checks if access to a specific host and port is allowed
func (p *SecurityPolicy) CheckNetworkAccess(host string, port int) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	// Check if outbound network access is allowed
	if !p.Permissions[PermNetworkOutbound] {
		return false
	}
	
	// Check if the host is allowed
	hostAllowed := false
	for _, allowedHost := range p.AllowedHosts {
		if allowedHost == "*" || allowedHost == host {
			hostAllowed = true
			break
		}
	}
	
	if !hostAllowed {
		return false
	}
	
	// Check if the port is allowed
	portAllowed := false
	for _, allowedPort := range p.AllowedPorts {
		if allowedPort == 0 || allowedPort == port {
			portAllowed = true
			break
		}
	}
	
	return portAllowed
}

// pathMatchesPattern checks if a path matches a pattern
func pathMatchesPattern(path, pattern string) bool {
	// Handle glob patterns
	if strings.Contains(pattern, "*") {
		matched, _ := filepath.Match(pattern, path)
		return matched
	}
	
	// Handle directory prefix patterns
	if strings.HasSuffix(pattern, "/") {
		return strings.HasPrefix(path, pattern)
	}
	
	// Handle exact matches
	return path == pattern
}
