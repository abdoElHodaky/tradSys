package commands

import (
	"context"
	"fmt"
	"path/filepath"
	"reflect"
	"time"

	"go.uber.org/zap"

	"github.com/abdoElHodaky/tradSys/internal/plugin/adaptive_loader"
	"github.com/abdoElHodaky/tradSys/internal/plugin/cqrs"
)

// LoadPluginCommand represents a request to load a plugin
type LoadPluginCommand struct {
	FilePath     string
	Timeout      time.Duration
	Priority     int
	ValidateOnly bool
}

// Validate validates the command
func (c *LoadPluginCommand) Validate() error {
	if c.FilePath == "" {
		return fmt.Errorf("file path is required")
	}
	
	if c.Timeout <= 0 {
		c.Timeout = 30 * time.Second // Default timeout
	}
	
	return nil
}

// LoadPluginCommandHandler handles the LoadPluginCommand
type LoadPluginCommandHandler struct {
	loader *adaptive_loader.AdaptivePluginLoader
	logger *zap.Logger
}

// NewLoadPluginCommandHandler creates a new LoadPluginCommandHandler
func NewLoadPluginCommandHandler(loader *adaptive_loader.AdaptivePluginLoader, logger *zap.Logger) *LoadPluginCommandHandler {
	return &LoadPluginCommandHandler{
		loader: loader,
		logger: logger,
	}
}

// Type returns the type of command this handler can process
func (h *LoadPluginCommandHandler) Type() reflect.Type {
	return reflect.TypeOf(&LoadPluginCommand{})
}

// Handle processes the LoadPluginCommand
func (h *LoadPluginCommandHandler) Handle(ctx context.Context, command interface{}) error {
	cmd, ok := command.(*LoadPluginCommand)
	if !ok {
		return fmt.Errorf("invalid command type: expected *LoadPluginCommand, got %T", command)
	}
	
	// Create a task for loading the plugin
	taskName := fmt.Sprintf("load_plugin_%s", filepath.Base(cmd.FilePath))
	
	var task *adaptive_loader.Task
	
	if cmd.ValidateOnly {
		// Create a validation-only task
		task = adaptive_loader.NewTask(taskName, func() error {
			return h.loader.ValidatePlugin(ctx, cmd.FilePath)
		})
	} else {
		// Create a full load task
		task = adaptive_loader.NewTask(taskName, func() error {
			return h.loader.LoadPlugin(ctx, cmd.FilePath)
		})
	}
	
	// Set task properties
	task.WithPriority(cmd.Priority)
	
	// Create a context with timeout
	taskCtx, cancel := context.WithTimeout(ctx, cmd.Timeout)
	defer cancel()
	
	task.WithContext(taskCtx)
	
	// Submit the task to the worker pool
	h.logger.Debug("Submitting plugin load task",
		zap.String("file_path", cmd.FilePath),
		zap.Duration("timeout", cmd.Timeout),
		zap.Int("priority", cmd.Priority),
		zap.Bool("validate_only", cmd.ValidateOnly))
	
	return h.loader.SubmitTask(task)
}

// RegisterLoadPluginCommand registers the LoadPluginCommand handler with the command bus
func RegisterLoadPluginCommand(commandBus *cqrs.CommandBus, loader *adaptive_loader.AdaptivePluginLoader, logger *zap.Logger) error {
	handler := NewLoadPluginCommandHandler(loader, logger)
	return commandBus.RegisterHandler(handler)
}
