package common

import (
	"fmt"
)

// ServiceError represents a service-level error with context
type ServiceError struct {
	Service   string
	Operation string
	Err       error
}

func (e *ServiceError) Error() string {
	return fmt.Sprintf("[%s:%s] %v", e.Service, e.Operation, e.Err)
}

func (e *ServiceError) Unwrap() error {
	return e.Err
}

// NewServiceError creates a new service error with context
func NewServiceError(service, operation string, err error) *ServiceError {
	return &ServiceError{
		Service:   service,
		Operation: operation,
		Err:       err,
	}
}

// WrapServiceError wraps an error with service context
func WrapServiceError(service, operation string, err error) error {
	if err == nil {
		return nil
	}
	return NewServiceError(service, operation, err)
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string
	Value   interface{}
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation failed for field '%s' with value '%v': %s", e.Field, e.Value, e.Message)
}

// NewValidationError creates a new validation error
func NewValidationError(field string, value interface{}, message string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Value:   value,
		Message: message,
	}
}

// RepositoryError represents a repository-level error
type RepositoryError struct {
	Repository string
	Operation  string
	Err        error
}

func (e *RepositoryError) Error() string {
	return fmt.Sprintf("[%s:%s] %v", e.Repository, e.Operation, e.Err)
}

func (e *RepositoryError) Unwrap() error {
	return e.Err
}

// NewRepositoryError creates a new repository error
func NewRepositoryError(repository, operation string, err error) *RepositoryError {
	return &RepositoryError{
		Repository: repository,
		Operation:  operation,
		Err:        err,
	}
}

// WrapRepositoryError wraps an error with repository context
func WrapRepositoryError(repository, operation string, err error) error {
	if err == nil {
		return nil
	}
	return NewRepositoryError(repository, operation, err)
}
