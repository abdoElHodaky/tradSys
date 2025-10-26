package config

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
)

// ConfigInterface defines the interface for configuration management
type ConfigInterface interface {
	// Load loads configuration from environment variables
	Load() error
	// Validate validates the configuration
	Validate() error
	// Get retrieves a configuration value by key
	Get(key string) interface{}
	// Set sets a configuration value
	Set(key string, value interface{}) error
	// GetString retrieves a string configuration value
	GetString(key string) string
	// GetInt retrieves an integer configuration value
	GetInt(key string) int
	// GetBool retrieves a boolean configuration value
	GetBool(key string) bool
	// GetDuration retrieves a duration configuration value
	GetDuration(key string) time.Duration
}

// BaseConfig provides common configuration functionality
type BaseConfig struct {
	values map[string]interface{}
	logger *zap.Logger
	prefix string
}

// NewBaseConfig creates a new base configuration
func NewBaseConfig(prefix string, logger *zap.Logger) *BaseConfig {
	return &BaseConfig{
		values: make(map[string]interface{}),
		logger: logger,
		prefix: prefix,
	}
}

// LoadFromEnv loads configuration from environment variables using struct tags
func (bc *BaseConfig) LoadFromEnv(config interface{}) error {
	v := reflect.ValueOf(config)
	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("config must be a pointer to a struct")
	}

	v = v.Elem()
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)

		// Skip unexported fields
		if !field.CanSet() {
			continue
		}

		// Get environment variable name from tag or field name
		envName := fieldType.Tag.Get("env")
		if envName == "" {
			envName = bc.fieldToEnvName(fieldType.Name)
		}

		// Add prefix if configured
		if bc.prefix != "" {
			envName = bc.prefix + "_" + envName
		}

		// Get environment variable value
		envValue := os.Getenv(envName)
		if envValue == "" {
			// Check for default value
			defaultValue := fieldType.Tag.Get("default")
			if defaultValue != "" {
				envValue = defaultValue
			} else {
				// Check if field is required
				if fieldType.Tag.Get("required") == "true" {
					return fmt.Errorf("required environment variable %s is not set", envName)
				}
				continue
			}
		}

		// Set field value based on type
		if err := bc.setFieldValue(field, envValue, envName); err != nil {
			return fmt.Errorf("failed to set field %s: %w", fieldType.Name, err)
		}

		// Store in values map for Get method
		bc.values[strings.ToLower(fieldType.Name)] = field.Interface()
	}

	return nil
}

// setFieldValue sets a field value from string
func (bc *BaseConfig) setFieldValue(field reflect.Value, value, envName string) error {
	switch field.Kind() {
	case reflect.String:
		field.SetString(value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if field.Type() == reflect.TypeOf(time.Duration(0)) {
			// Handle duration
			duration, err := time.ParseDuration(value)
			if err != nil {
				return fmt.Errorf("invalid duration format for %s: %w", envName, err)
			}
			field.SetInt(int64(duration))
		} else {
			// Handle integer
			intValue, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid integer format for %s: %w", envName, err)
			}
			field.SetInt(intValue)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		uintValue, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid unsigned integer format for %s: %w", envName, err)
		}
		field.SetUint(uintValue)
	case reflect.Float32, reflect.Float64:
		floatValue, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("invalid float format for %s: %w", envName, err)
		}
		field.SetFloat(floatValue)
	case reflect.Bool:
		boolValue, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid boolean format for %s: %w", envName, err)
		}
		field.SetBool(boolValue)
	case reflect.Slice:
		// Handle string slices (comma-separated values)
		if field.Type().Elem().Kind() == reflect.String {
			values := strings.Split(value, ",")
			for i, v := range values {
				values[i] = strings.TrimSpace(v)
			}
			field.Set(reflect.ValueOf(values))
		} else {
			return fmt.Errorf("unsupported slice type for field %s", envName)
		}
	default:
		return fmt.Errorf("unsupported field type %s for %s", field.Kind(), envName)
	}

	return nil
}

// fieldToEnvName converts a field name to environment variable name
func (bc *BaseConfig) fieldToEnvName(fieldName string) string {
	// Convert CamelCase to UPPER_SNAKE_CASE
	var result strings.Builder
	for i, r := range fieldName {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune('_')
		}
		result.WriteRune(r)
	}
	return strings.ToUpper(result.String())
}

// Get retrieves a configuration value by key
func (bc *BaseConfig) Get(key string) interface{} {
	return bc.values[strings.ToLower(key)]
}

// Set sets a configuration value
func (bc *BaseConfig) Set(key string, value interface{}) error {
	bc.values[strings.ToLower(key)] = value
	return nil
}

// GetString retrieves a string configuration value
func (bc *BaseConfig) GetString(key string) string {
	if value, ok := bc.values[strings.ToLower(key)].(string); ok {
		return value
	}
	return ""
}

// GetInt retrieves an integer configuration value
func (bc *BaseConfig) GetInt(key string) int {
	if value, ok := bc.values[strings.ToLower(key)].(int); ok {
		return value
	}
	return 0
}

// GetBool retrieves a boolean configuration value
func (bc *BaseConfig) GetBool(key string) bool {
	if value, ok := bc.values[strings.ToLower(key)].(bool); ok {
		return value
	}
	return false
}

// GetDuration retrieves a duration configuration value
func (bc *BaseConfig) GetDuration(key string) time.Duration {
	if value, ok := bc.values[strings.ToLower(key)].(time.Duration); ok {
		return value
	}
	return 0
}

// Validate performs basic validation on configuration values
func (bc *BaseConfig) Validate() error {
	// Override in specific config implementations
	return nil
}

// ConfigValidator provides validation utilities
type ConfigValidator struct {
	errors []string
}

// NewConfigValidator creates a new config validator
func NewConfigValidator() *ConfigValidator {
	return &ConfigValidator{
		errors: make([]string, 0),
	}
}

// RequireString validates that a string field is not empty
func (cv *ConfigValidator) RequireString(value, fieldName string) *ConfigValidator {
	if strings.TrimSpace(value) == "" {
		cv.errors = append(cv.errors, fmt.Sprintf("%s is required", fieldName))
	}
	return cv
}

// RequirePositiveInt validates that an integer field is positive
func (cv *ConfigValidator) RequirePositiveInt(value int, fieldName string) *ConfigValidator {
	if value <= 0 {
		cv.errors = append(cv.errors, fmt.Sprintf("%s must be positive", fieldName))
	}
	return cv
}

// RequirePositiveDuration validates that a duration field is positive
func (cv *ConfigValidator) RequirePositiveDuration(value time.Duration, fieldName string) *ConfigValidator {
	if value <= 0 {
		cv.errors = append(cv.errors, fmt.Sprintf("%s must be positive", fieldName))
	}
	return cv
}

// RequireOneOf validates that a string field is one of the allowed values
func (cv *ConfigValidator) RequireOneOf(value, fieldName string, allowedValues []string) *ConfigValidator {
	for _, allowed := range allowedValues {
		if value == allowed {
			return cv
		}
	}
	cv.errors = append(cv.errors, fmt.Sprintf("%s must be one of: %v", fieldName, allowedValues))
	return cv
}

// RequireRange validates that an integer field is within a range
func (cv *ConfigValidator) RequireRange(value, min, max int, fieldName string) *ConfigValidator {
	if value < min || value > max {
		cv.errors = append(cv.errors, fmt.Sprintf("%s must be between %d and %d", fieldName, min, max))
	}
	return cv
}

// Validate returns validation errors if any
func (cv *ConfigValidator) Validate() error {
	if len(cv.errors) > 0 {
		return fmt.Errorf("configuration validation failed: %s", strings.Join(cv.errors, "; "))
	}
	return nil
}

// HotReloadConfig provides hot-reloading configuration capabilities
type HotReloadConfig struct {
	*BaseConfig
	reloadFunc func() error
	stopChan   chan struct{}
	interval   time.Duration
}

// NewHotReloadConfig creates a new hot-reload configuration
func NewHotReloadConfig(baseConfig *BaseConfig, reloadFunc func() error, interval time.Duration) *HotReloadConfig {
	return &HotReloadConfig{
		BaseConfig: baseConfig,
		reloadFunc: reloadFunc,
		stopChan:   make(chan struct{}),
		interval:   interval,
	}
}

// StartWatching starts watching for configuration changes
func (hrc *HotReloadConfig) StartWatching() {
	go func() {
		ticker := time.NewTicker(hrc.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := hrc.reloadFunc(); err != nil {
					hrc.logger.Error("Failed to reload configuration", zap.Error(err))
				} else {
					hrc.logger.Info("Configuration reloaded successfully")
				}
			case <-hrc.stopChan:
				return
			}
		}
	}()
}

// StopWatching stops watching for configuration changes
func (hrc *HotReloadConfig) StopWatching() {
	close(hrc.stopChan)
}
