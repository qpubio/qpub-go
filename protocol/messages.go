package protocol

import "encoding/json"

// ErrorInfo is server error payload.
type ErrorInfo struct {
	Code       int    `json:"code"`
	Href       string `json:"href"`
	Message    string `json:"message"`
	StatusCode int `json:"status_code"`
}

// ConnectionDetails from CONNECTED message.
type ConnectionDetails struct {
	Alias    string `json:"alias"`
	ClientID string `json:"client_id"`
	ServerID string `json:"server_id"`
}

// DataMessagePayload is publish/subscribe payload.
type DataMessagePayload struct {
	Alias string          `json:"alias,omitempty"`
	Event string          `json:"event,omitempty"`
	Data  json.RawMessage `json:"data,omitempty"`
}

// Message is the consumer-facing channel message.
type Message struct {
	Action    ActionType      `json:"action,omitempty"`
	Error     *ErrorInfo      `json:"error,omitempty"`
	ID        string          `json:"id,omitempty"`
	Timestamp string          `json:"timestamp,omitempty"`
	Channel   string          `json:"channel"`
	Alias     string          `json:"alias,omitempty"`
	Event     string          `json:"event,omitempty"`
	Data      json.RawMessage `json:"data,omitempty"`
}

// WireMessage is a generic JSON frame on the WebSocket.
type WireMessage struct {
	Action         ActionType           `json:"action"`
	Error          *ErrorInfo           `json:"error,omitempty"`
	ConnectionID   string               `json:"connection_id,omitempty"`
	ConnectionDetails *ConnectionDetails `json:"connection_details,omitempty"`
	Channel        string               `json:"channel,omitempty"`
	SubscriptionID string               `json:"subscription_id,omitempty"`
	ID             string               `json:"id,omitempty"`
	Timestamp      string               `json:"timestamp,omitempty"`
	Messages       []DataMessagePayload `json:"messages,omitempty"`
}

// RestPublishRequest is REST batch/single publish body.
type RestPublishRequest struct {
	Channels []string             `json:"channels,omitempty"`
	Messages []DataMessagePayload `json:"messages"`
}
