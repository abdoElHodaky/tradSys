package ws

import (
	"time"
)

// Message represents a WebSocket message
type Message struct {
	Type       string    `json:"type"`
	Payload    []byte    `json:"payload"`
	Timestamp  time.Time `json:"timestamp"`
	Sequence   int64     `json:"sequence"`
	Channel    string    `json:"channel"`
	Sender     string    `json:"sender"`
	Recipients []string  `json:"recipients,omitempty"`
}

// Request represents a WebSocket request
type Request struct {
	Type     string `json:"type"`
	Payload  []byte `json:"payload,omitempty"`
	Channel  string `json:"channel,omitempty"`
	Token    string `json:"token,omitempty"`
	ClientID string `json:"client_id,omitempty"`
}

// Response represents a WebSocket response
type Response struct {
	Type    string `json:"type"`
	Payload []byte `json:"payload,omitempty"`
	Status  int    `json:"status"`
	Error   string `json:"error,omitempty"`
	Channel string `json:"channel,omitempty"`
}

// NewMessage creates a new WebSocket message
func NewMessage(messageType string, payload []byte, channel string, sender string) *Message {
	return &Message{
		Type:      messageType,
		Payload:   payload,
		Timestamp: time.Now(),
		Sequence:  time.Now().UnixNano(),
		Channel:   channel,
		Sender:    sender,
	}
}

// NewRequest creates a new WebSocket request
func NewRequest(requestType string, payload []byte, channel string, token string, clientID string) *Request {
	return &Request{
		Type:     requestType,
		Payload:  payload,
		Channel:  channel,
		Token:    token,
		ClientID: clientID,
	}
}

// NewResponse creates a new WebSocket response
func NewResponse(responseType string, payload []byte, status int, errorMsg string, channel string) *Response {
	return &Response{
		Type:    responseType,
		Payload: payload,
		Status:  status,
		Error:   errorMsg,
		Channel: channel,
	}
}

// SuccessResponse creates a success response
func SuccessResponse(responseType string, payload []byte, channel string) *Response {
	return NewResponse(responseType, payload, 200, "", channel)
}

// ErrorResponse creates an error response
func ErrorResponse(responseType string, errorMsg string, status int, channel string) *Response {
	return NewResponse(responseType, nil, status, errorMsg, channel)
}
