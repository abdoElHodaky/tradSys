package sandbox

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// PluginSandbox provides an isolated environment for plugin execution
type PluginSandbox struct {
	// Plugin identity
	PluginID string

	// Resource limits
	MemoryLimit     int64
	CPULimit        float64
	FileAccessPaths []string
	ExecutionTimeout time.Duration

	// Security policies
	SecurityPolicy *SecurityPolicy

	// Monitoring
	ResourceMonitor *ResourceMonitor
	Logger          *zap.Logger

	// Execution context
	ctx    context.Context
	cancel context.CancelFunc
	mu     sync.RWMutex
}

// NewPluginSandbox creates a new plugin sandbox
func NewPluginSandbox(pluginID string, logger *zap.Logger) *PluginSandbox {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &PluginSandbox{
		PluginID:         pluginID,
		MemoryLimit:      256 * 1024 * 1024, // 256MB default
		CPULimit:         50.0,              // 50% of one CPU core
		FileAccessPaths:  []string{},
		ExecutionTimeout: 30 * time.Second,
		SecurityPolicy:   NewDefaultSecurityPolicy(),
		ResourceMonitor:  NewResourceMonitor(logger),
		Logger:           logger,
		ctx:              ctx,
		cancel:           cancel,
	}
}

// WithMemoryLimit sets the memory limit for the sandbox
func (s *PluginSandbox) WithMemoryLimit(limitBytes int64) *PluginSandbox {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.MemoryLimit = limitBytes
	return s
}

// WithCPULimit sets the CPU limit for the sandbox
func (s *PluginSandbox) WithCPULimit(limitPercent float64) *PluginSandbox {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.CPULimit = limitPercent
	return s
}

// WithFileAccess sets the allowed file access paths
func (s *PluginSandbox) WithFileAccess(paths []string) *PluginSandbox {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.FileAccessPaths = make([]string, len(paths))
	copy(s.FileAccessPaths, paths)
	return s
}

// WithExecutionTimeout sets the execution timeout
func (s *PluginSandbox) WithExecutionTimeout(timeout time.Duration) *PluginSandbox {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.ExecutionTimeout = timeout
	return s
}

// WithSecurityPolicy sets the security policy
func (s *PluginSandbox) WithSecurityPolicy(policy *SecurityPolicy) *PluginSandbox {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.SecurityPolicy = policy
	return s
}

// Start starts the sandbox monitoring
func (s *PluginSandbox) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Start resource monitoring
	s.ResourceMonitor.StartMonitoring(s.PluginID)
	
	s.Logger.Info("Started plugin sandbox",
		zap.String("plugin_id", s.PluginID),
		zap.Int64("memory_limit_bytes", s.MemoryLimit),
		zap.Float64("cpu_limit_percent", s.CPULimit),
		zap.Duration("execution_timeout", s.ExecutionTimeout))
	
	return nil
}

// Stop stops the sandbox
func (s *PluginSandbox) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Stop resource monitoring
	s.ResourceMonitor.StopMonitoring(s.PluginID)
	
	// Cancel context
	s.cancel()
	
	s.Logger.Info("Stopped plugin sandbox",
		zap.String("plugin_id", s.PluginID))
}

// Execute executes a function within the sandbox
func (s *PluginSandbox) Execute(fn func(ctx context.Context) error) error {
	s.mu.RLock()
	timeout := s.ExecutionTimeout
	s.mu.RUnlock()
	
	// Create execution context with timeout
	execCtx, cancel := context.WithTimeout(s.ctx, timeout)
	defer cancel()
	
	// Apply resource limits
	if err := s.ApplyResourceLimits(); err != nil {
		return fmt.Errorf("failed to apply resource limits: %w", err)
	}
	
	// Execute in goroutine with result channel
	resultCh := make(chan error, 1)
	go func() {
		resultCh <- fn(execCtx)
	}()
	
	// Wait for completion or timeout
	select {
	case err := <-resultCh:
		return err
	case <-execCtx.Done():
		if execCtx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("execution timed out after %v", timeout)
		}
		return execCtx.Err()
	}
}

// ApplyResourceLimits applies resource limits to the sandbox
func (s *PluginSandbox) ApplyResourceLimits() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	// Apply memory limit
	if err := applyMemoryLimit(s.MemoryLimit); err != nil {
		return fmt.Errorf("failed to apply memory limit: %w", err)
	}
	
	// Apply CPU limit
	if err := applyCPULimit(s.CPULimit); err != nil {
		return fmt.Errorf("failed to apply CPU limit: %w", err)
	}
	
	return nil
}

// CheckPermission checks if the sandbox has a specific permission
func (s *PluginSandbox) CheckPermission(perm PluginPermission) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if s.SecurityPolicy == nil {
		return false
	}
	
	return s.SecurityPolicy.CheckPermission(perm)
}

// CheckFileAccess checks if the sandbox has access to a specific file path
func (s *PluginSandbox) CheckFileAccess(path string, accessType FileAccessType) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if s.SecurityPolicy == nil {
		return false
	}
	
	return s.SecurityPolicy.CheckFileAccess(path, accessType)
}

// applyMemoryLimit applies a memory limit to the current process
// This is a placeholder for actual implementation which would depend on the OS
func applyMemoryLimit(limitBytes int64) error {
	// In a real implementation, this would use OS-specific mechanisms
	// such as cgroups on Linux or job objects on Windows
	
	// For now, just log that we would apply the limit
	return nil
}

// applyCPULimit applies a CPU limit to the current process
// This is a placeholder for actual implementation which would depend on the OS
func applyCPULimit(limitPercent float64) error {
	// In a real implementation, this would use OS-specific mechanisms
	// such as cgroups on Linux or job objects on Windows
	
	// For now, just log that we would apply the limit
	return nil
}
