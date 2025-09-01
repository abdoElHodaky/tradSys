package ws

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// ValidationError represents a validation error response
type ValidationError struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

// Validate validates a struct using the validator package
func Validate(data interface{}) error {
	validate := validator.New()
	return validate.Struct(data)
}

// HandleValidationError handles validation errors for websocket connections
func HandleValidationError(conn *websocket.Conn, err error, logger *zap.Logger) {
	if err == nil {
		return
	}

	validationErr := ValidationError{
		Status:  http.StatusBadRequest,
		Message: fmt.Sprintf("Validation error: %s", err.Error()),
		Code:    "VALIDATION_ERROR",
	}

	// Log the validation error
	logger.Debug("Validation error",
		zap.String("error", err.Error()),
		zap.String("code", validationErr.Code),
	)

	// Send the validation error response to the client
	errBytes, marshalErr := json.Marshal(validationErr)
	if marshalErr != nil {
		logger.Error("Failed to marshal validation error",
			zap.Error(marshalErr),
		)
		return
	}

	writeErr := conn.WriteMessage(websocket.TextMessage, errBytes)
	if writeErr != nil {
		logger.Error("Failed to write validation error message",
			zap.Error(writeErr),
		)
	}
}

// ValidateRequest validates a request and handles any validation errors
func ValidateRequest(conn *websocket.Conn, data interface{}, logger *zap.Logger) bool {
	err := Validate(data)
	if err != nil {
		HandleValidationError(conn, err, logger)
		return false
	}
	return true
}

