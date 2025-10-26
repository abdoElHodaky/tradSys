package websocket

// SubscribeRequest represents a subscription request
type SubscribeRequest struct {
	Topic    string `json:"topic"`
	ClientId string `json:"client_id"`
	Symbol   string `json:"symbol,omitempty"`
}

// SubscribeResponse represents a subscription response
type SubscribeResponse struct {
	Success        bool   `json:"success"`
	Message        string `json:"message,omitempty"`
	Channel        string `json:"channel,omitempty"`
	SubscriptionId string `json:"subscription_id,omitempty"`
}

// UnsubscribeRequest represents an unsubscription request
type UnsubscribeRequest struct {
	SubscriptionId string `json:"subscription_id"`
	ClientId       string `json:"client_id"`
}

// UnsubscribeResponse represents an unsubscription response
type UnsubscribeResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Channel string `json:"channel,omitempty"`
}

// PublishRequest represents a publish request
type PublishRequest struct {
	Topic string `json:"topic"`
	Data  []byte `json:"data"`
}

// PublishResponse represents a publish response
type PublishResponse struct {
	Success    bool   `json:"success"`
	Message    string `json:"message,omitempty"`
	Recipients int    `json:"recipients,omitempty"`
}

// GetConnectionsRequest represents a request to get connections
type GetConnectionsRequest struct {
	Topic string `json:"topic,omitempty"`
}

// ConnectionInfo represents a WebSocket connection info
type ConnectionInfo struct {
	ClientId      string   `json:"client_id"`
	UserId        string   `json:"user_id"`
	ConnectedAt   int64    `json:"connected_at"`
	Subscriptions []string `json:"subscriptions"`
	IpAddress     string   `json:"ip_address,omitempty"`
	UserAgent     string   `json:"user_agent,omitempty"`
}

// GetConnectionsResponse represents a response with connections
type GetConnectionsResponse struct {
	Success          bool              `json:"success"`
	Message          string            `json:"message,omitempty"`
	Connections      []*ConnectionInfo `json:"connections,omitempty"`
	Count            int               `json:"count"`
	TotalConnections int32             `json:"total_connections,omitempty"`
}
