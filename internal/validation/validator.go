package validation

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"

	validator "github.com/go-playground/validator/v10"
)

// Validator represents a validator
type Validator struct {
	validator *validator.Validate
}

// NewValidator creates a new validator
func NewValidator() *Validator {
	v := validator.New()

	// Register custom validation functions
	v.RegisterValidation("password", validatePassword)
	v.RegisterValidation("symbol", validateSymbol)
	v.RegisterValidation("amount", validateAmount)
	v.RegisterValidation("price", validatePrice)

	// Register tag name function
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})

	return &Validator{
		validator: v,
	}
}

// Validate validates a struct
func (v *Validator) Validate(i interface{}) error {
	if err := v.validator.Struct(i); err != nil {
		// Convert validation errors to a more user-friendly format
		if validationErrors, ok := err.(validator.ValidationErrors); ok {
			var errMessages []string
			for _, e := range validationErrors {
				errMessages = append(errMessages, formatValidationError(e))
			}
			return errors.New(strings.Join(errMessages, "; "))
		}
		return err
	}
	return nil
}

// ValidateVar validates a variable
func (v *Validator) ValidateVar(field interface{}, tag string) error {
	if err := v.validator.Var(field, tag); err != nil {
		// Convert validation errors to a more user-friendly format
		if validationErrors, ok := err.(validator.ValidationErrors); ok {
			var errMessages []string
			for _, e := range validationErrors {
				errMessages = append(errMessages, formatValidationError(e))
			}
			return errors.New(strings.Join(errMessages, "; "))
		}
		return err
	}
	return nil
}

// formatValidationError formats a validation error
func formatValidationError(e validator.FieldError) string {
	field := e.Field()
	tag := e.Tag()
	param := e.Param()

	switch tag {
	case "required":
		return fmt.Sprintf("%s is required", field)
	case "email":
		return fmt.Sprintf("%s must be a valid email address", field)
	case "min":
		return fmt.Sprintf("%s must be at least %s characters long", field, param)
	case "max":
		return fmt.Sprintf("%s must be at most %s characters long", field, param)
	case "oneof":
		return fmt.Sprintf("%s must be one of: %s", field, param)
	case "password":
		return fmt.Sprintf("%s must contain at least one uppercase letter, one lowercase letter, one number, and one special character", field)
	case "symbol":
		return fmt.Sprintf("%s must be a valid trading symbol", field)
	case "amount":
		return fmt.Sprintf("%s must be a valid amount", field)
	case "price":
		return fmt.Sprintf("%s must be a valid price", field)
	default:
		return fmt.Sprintf("%s failed validation: %s", field, tag)
	}
}

// validatePassword validates a password
func validatePassword(fl validator.FieldLevel) bool {
	password := fl.Field().String()

	// Check for at least one uppercase letter
	if !regexp.MustCompile(`[A-Z]`).MatchString(password) {
		return false
	}

	// Check for at least one lowercase letter
	if !regexp.MustCompile(`[a-z]`).MatchString(password) {
		return false
	}

	// Check for at least one number
	if !regexp.MustCompile(`[0-9]`).MatchString(password) {
		return false
	}

	// Check for at least one special character
	if !regexp.MustCompile(`[^a-zA-Z0-9]`).MatchString(password) {
		return false
	}

	return true
}

// validateSymbol validates a trading symbol
func validateSymbol(fl validator.FieldLevel) bool {
	symbol := fl.Field().String()

	// Check if symbol is in the format BASE/QUOTE (e.g., BTC/USD)
	parts := strings.Split(symbol, "/")
	if len(parts) != 2 {
		return false
	}

	// Check if base and quote are valid
	base := parts[0]
	quote := parts[1]

	// Base and quote should be 2-5 characters long
	if len(base) < 2 || len(base) > 5 || len(quote) < 2 || len(quote) > 5 {
		return false
	}

	// Base and quote should only contain uppercase letters
	if !regexp.MustCompile(`^[A-Z]+$`).MatchString(base) || !regexp.MustCompile(`^[A-Z]+$`).MatchString(quote) {
		return false
	}

	return true
}

// validateAmount validates an amount
func validateAmount(fl validator.FieldLevel) bool {
	amount := fl.Field().Float()

	// Amount should be positive
	if amount <= 0 {
		return false
	}

	return true
}

// validatePrice validates a price
func validatePrice(fl validator.FieldLevel) bool {
	price := fl.Field().Float()

	// Price should be positive
	if price <= 0 {
		return false
	}

	return true
}
