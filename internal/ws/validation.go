package ws

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/abdoElHodaky/tradSys/internal/validation"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

// MessageValidator validates WebSocket messages
type MessageValidator struct {
	logger    *zap.Logger
	validator *validation.Validator
	schemas   map[string]interface{}
}

// NewMessageValidator creates a new message validator
func NewMessageValidator(logger *zap.Logger) *MessageValidator {
	return &MessageValidator{
		logger:    logger,
		validator: validation.NewValidator(),
		schemas:   make(map[string]interface{}),
	}
}

// RegisterSchema registers a schema for a message type
func (v *MessageValidator) RegisterSchema(messageType string, schema interface{}) {
	v.schemas[messageType] = schema
}

// ValidateMessage validates a message against its schema
func (v *MessageValidator) ValidateMessage(message Message) error {
	// Check if schema exists for message type
	schema, ok := v.schemas[message.Type]
	if !ok {
		return fmt.Errorf("no schema registered for message type: %s", message.Type)
	}

	// Create a new instance of the schema type
	schemaType := reflect.TypeOf(schema)
	if schemaType.Kind() == reflect.Ptr {
		schemaType = schemaType.Elem()
	}
	instance := reflect.New(schemaType).Interface()

	// Convert message data to JSON
	var jsonData []byte
	var err error
	switch data := message.Data.(type) {
	case string:
		jsonData = []byte(data)
	case []byte:
		jsonData = data
	case map[string]interface{}:
		jsonData, err = json.Marshal(data)
		if err != nil {
			return fmt.Errorf("failed to marshal message data: %w", err)
		}
	case proto.Message:
		// For Protocol Buffers messages, we need to convert to JSON
		jsonData, err = json.Marshal(data)
		if err != nil {
			return fmt.Errorf("failed to marshal protobuf message: %w", err)
		}
	default:
		jsonData, err = json.Marshal(data)
		if err != nil {
			return fmt.Errorf("failed to marshal message data: %w", err)
		}
	}

	// Unmarshal JSON data into schema instance
	if err := json.Unmarshal(jsonData, instance); err != nil {
		return fmt.Errorf("failed to unmarshal message data: %w", err)
	}

	// Validate instance against schema
	if err := v.validator.Validate(instance); err != nil {
		return fmt.Errorf("message validation failed: %w", err)
	}

	return nil
}

// ValidateBinaryMessage validates a binary message against its schema
func (v *MessageValidator) ValidateBinaryMessage(message *WebSocketMessage) error {
	// Check if schema exists for message type
	_, ok := v.schemas[message.Type]
	if !ok {
		return fmt.Errorf("no schema registered for message type: %s", message.Type)
	}

	// For binary messages, we need to convert to the appropriate schema type
	// This depends on the specific message type and schema
	// For now, we'll just validate the basic fields

	// Validate required fields
	if message.Type == "" {
		return errors.New("message type is required")
	}

	// For specific message types, we can add more validation
	switch message.Type {
	case "marketData":
		if message.Symbol == "" {
			return errors.New("symbol is required for marketData messages")
		}
	case "order":
		if message.Symbol == "" {
			return errors.New("symbol is required for order messages")
		}
		// Additional validation for order messages
		if message.Data == nil {
			return errors.New("data is required for order messages")
		}
	case "subscription":
		if message.Channel == "" {
			return errors.New("channel is required for subscription messages")
		}
	}

	// For more complex validation, we would need to unmarshal the message data
	// into the appropriate schema type and validate it
	// This would depend on the specific message type and schema

	return nil
}

// Common message schemas

// SubscriptionMessage represents a subscription message
type SubscriptionMessage struct {
	Channel string `json:"channel" validate:"required"`
	Symbol  string `json:"symbol" validate:"required,symbol"`
}

// OrderMessage represents an order message
type OrderMessage struct {
	Symbol    string  `json:"symbol" validate:"required,symbol"`
	Side      string  `json:"side" validate:"required,oneof=buy sell"`
	OrderType string  `json:"order_type" validate:"required,oneof=market limit stop stop_limit"`
	Quantity  float64 `json:"quantity" validate:"required,amount"`
	Price     float64 `json:"price" validate:"omitempty,price"`
	StopPrice float64 `json:"stop_price" validate:"omitempty,price"`
	TimeInForce string `json:"time_in_force" validate:"required,oneof=GTC IOC FOK DAY"`
}

// CancelOrderMessage represents a cancel order message
type CancelOrderMessage struct {
	OrderID string `json:"order_id" validate:"required"`
}

// MarketDataMessage represents a market data message
type MarketDataMessage struct {
	Symbol string `json:"symbol" validate:"required,symbol"`
	Type   string `json:"type" validate:"required,oneof=trade quote orderbook"`
}

// AuthMessage represents an authentication message
type AuthMessage struct {
	Token string `json:"token" validate:"required"`
}

// ErrorMessage represents an error message
type ErrorMessage struct {
	Code    int    `json:"code" validate:"required"`
	Message string `json:"message" validate:"required"`
}

// RegisterDefaultSchemas registers default schemas for common message types
func (v *MessageValidator) RegisterDefaultSchemas() {
	v.RegisterSchema("subscription", SubscriptionMessage{})
	v.RegisterSchema("order", OrderMessage{})
	v.RegisterSchema("cancelOrder", CancelOrderMessage{})
	v.RegisterSchema("marketData", MarketDataMessage{})
	v.RegisterSchema("auth", AuthMessage{})
	v.RegisterSchema("error", ErrorMessage{})
}

// ValidateMessageMiddleware returns a middleware function for validating messages
func (v *MessageValidator) ValidateMessageMiddleware() MessageHandlerMiddleware {
	return func(next MessageHandler) MessageHandler {
		return func(ctx Context, conn *AuthenticatedConnection, msg Message) error {
			// Validate message
			if err := v.ValidateMessage(msg); err != nil {
				v.logger.Error("Message validation failed",
					zap.String("type", msg.Type),
					zap.Error(err),
					zap.String("user_id", conn.UserID),
				)

				// Send error message to client
				errorMsg := Message{
					Type: "error",
					Data: ErrorMessage{
						Code:    400,
						Message: fmt.Sprintf("Validation error: %s", err.Error()),
					},
				}
				if err := conn.SendJSON(errorMsg); err != nil {
					v.logger.Error("Failed to send error message", zap.Error(err))
				}

				return err
			}

			// Call next handler
			return next(ctx, conn, msg)
		}
	}
}

// MessageHandlerMiddleware is a middleware function for message handlers
type MessageHandlerMiddleware func(MessageHandler) MessageHandler

// Context represents a context for message handling
type Context interface {
	// Add context methods as needed
}

// SendJSON sends a JSON message to the connection
func (c *AuthenticatedConnection) SendJSON(msg interface{}) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return c.WriteMessage(1, data) // 1 = TextMessage
}
