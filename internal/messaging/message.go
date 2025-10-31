// Package messaging provides concrete message implementations
package messaging

import (
	"time"

	"github.com/abdoElHodaky/tradSys/pkg/interfaces"
)

// StandardMessage implements the Message interface
type StandardMessage struct {
	messageType string
	payload     interface{}
	timestamp   time.Time
	metadata    map[string]interface{}
}

// NewMessage creates a new standard message
func NewMessage(messageType string, payload interface{}) interfaces.Message {
	return &StandardMessage{
		messageType: messageType,
		payload:     payload,
		timestamp:   time.Now(),
		metadata:    make(map[string]interface{}),
	}
}

// NewMessageWithMetadata creates a new message with metadata
func NewMessageWithMetadata(messageType string, payload interface{}, metadata map[string]interface{}) interfaces.Message {
	return &StandardMessage{
		messageType: messageType,
		payload:     payload,
		timestamp:   time.Now(),
		metadata:    metadata,
	}
}

// GetType returns the message type identifier
func (m *StandardMessage) GetType() string {
	return m.messageType
}

// GetPayload returns the message payload
func (m *StandardMessage) GetPayload() interface{} {
	return m.payload
}

// GetTimestamp returns when the message was created
func (m *StandardMessage) GetTimestamp() time.Time {
	return m.timestamp
}

// GetMetadata returns message metadata
func (m *StandardMessage) GetMetadata() map[string]interface{} {
	return m.metadata
}

// SetMetadata sets a metadata value
func (m *StandardMessage) SetMetadata(key string, value interface{}) {
	m.metadata[key] = value
}

// GetMetadataValue gets a specific metadata value
func (m *StandardMessage) GetMetadataValue(key string) (interface{}, bool) {
	value, exists := m.metadata[key]
	return value, exists
}

// MarketDataMessage represents market data messages
type MarketDataMessage struct {
	*StandardMessage
	Symbol   string
	Exchange string
	Price    float64
	Volume   float64
}

// NewMarketDataMessage creates a new market data message
func NewMarketDataMessage(symbol, exchange string, price, volume float64) interfaces.Message {
	payload := map[string]interface{}{
		"symbol":   symbol,
		"exchange": exchange,
		"price":    price,
		"volume":   volume,
	}
	
	msg := &MarketDataMessage{
		StandardMessage: &StandardMessage{
			messageType: "market_data",
			payload:     payload,
			timestamp:   time.Now(),
			metadata:    make(map[string]interface{}),
		},
		Symbol:   symbol,
		Exchange: exchange,
		Price:    price,
		Volume:   volume,
	}
	
	// Add typed metadata
	msg.SetMetadata("symbol", symbol)
	msg.SetMetadata("exchange", exchange)
	
	return msg
}

// OrderMessage represents order-related messages
type OrderMessage struct {
	*StandardMessage
	OrderID   string
	UserID    string
	Symbol    string
	OrderType string
	Status    string
}

// NewOrderMessage creates a new order message
func NewOrderMessage(messageType, orderID, userID, symbol, orderType, status string) interfaces.Message {
	payload := map[string]interface{}{
		"order_id":   orderID,
		"user_id":    userID,
		"symbol":     symbol,
		"order_type": orderType,
		"status":     status,
	}
	
	msg := &OrderMessage{
		StandardMessage: &StandardMessage{
			messageType: messageType,
			payload:     payload,
			timestamp:   time.Now(),
			metadata:    make(map[string]interface{}),
		},
		OrderID:   orderID,
		UserID:    userID,
		Symbol:    symbol,
		OrderType: orderType,
		Status:    status,
	}
	
	// Add typed metadata
	msg.SetMetadata("order_id", orderID)
	msg.SetMetadata("user_id", userID)
	msg.SetMetadata("symbol", symbol)
	
	return msg
}

// SystemMessage represents system-level messages
type SystemMessage struct {
	*StandardMessage
	Component string
	Level     string
	Message   string
}

// NewSystemMessage creates a new system message
func NewSystemMessage(component, level, message string) interfaces.Message {
	payload := map[string]interface{}{
		"component": component,
		"level":     level,
		"message":   message,
	}
	
	msg := &SystemMessage{
		StandardMessage: &StandardMessage{
			messageType: "system",
			payload:     payload,
			timestamp:   time.Now(),
			metadata:    make(map[string]interface{}),
		},
		Component: component,
		Level:     level,
		Message:   message,
	}
	
	// Add typed metadata
	msg.SetMetadata("component", component)
	msg.SetMetadata("level", level)
	
	return msg
}

// ErrorMessage represents error messages
type ErrorMessage struct {
	*StandardMessage
	ErrorCode string
	ErrorMsg  string
	Source    string
}

// NewErrorMessage creates a new error message
func NewErrorMessage(errorCode, errorMsg, source string) interfaces.Message {
	payload := map[string]interface{}{
		"error_code": errorCode,
		"error_msg":  errorMsg,
		"source":     source,
	}
	
	msg := &ErrorMessage{
		StandardMessage: &StandardMessage{
			messageType: "error",
			payload:     payload,
			timestamp:   time.Now(),
			metadata:    make(map[string]interface{}),
		},
		ErrorCode: errorCode,
		ErrorMsg:  errorMsg,
		Source:    source,
	}
	
	// Add typed metadata
	msg.SetMetadata("error_code", errorCode)
	msg.SetMetadata("source", source)
	
	return msg
}
