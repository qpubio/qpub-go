// Package qpub is the official Go client for QPub real-time messaging.
//
// Use NewSocket for WebSocket pub/sub and NewRest for HTTP channels and queues.
package qpub

import (
	"github.com/qpubio/qpub-go/events"
	"github.com/qpubio/qpub-go/option"
	"github.com/qpubio/qpub-go/protocol"
)

// Re-export core types for convenience at the package entry point.

type (
	Option         = option.Option
	OptionFunc     = option.OptionFunc
	TokenOptions   = option.TokenOptions
	TokenRequest   = option.TokenRequest
	AuthResponse   = option.AuthResponse
	Permission     = option.Permission
	Message        = protocol.Message
	QueueJob       = protocol.QueueJob
	EnqueueOptions = protocol.EnqueueOptions
	EnqueueResult  = protocol.EnqueueResult
)

// WithAPIKey and WithAutoConnect are option helpers.
var (
	WithAPIKey     = option.WithAPIKey
	WithAutoConnect = option.WithAutoConnect
	DefaultOption  = option.DefaultOption
)

// ConnectionEvents names for connection lifecycle callbacks.
var ConnectionEvents = struct {
	Initialized  string
	Connecting   string
	Opened       string
	Connected    string
	Disconnected string
	Closing      string
	Closed       string
	Failed       string
}{
	Initialized:  events.ConnectionInitialized,
	Connecting:   events.ConnectionConnecting,
	Opened:       events.ConnectionOpened,
	Connected:    events.ConnectionConnected,
	Disconnected: events.ConnectionDisconnected,
	Closing:      events.ConnectionClosing,
	Closed:       events.ConnectionClosed,
	Failed:       events.ConnectionFailed,
}

// ChannelEvents names for socket channel lifecycle callbacks.
var ChannelEvents = struct {
	Initialized   string
	Subscribing   string
	Subscribed    string
	Unsubscribing string
	Unsubscribed  string
	Paused        string
	Resumed       string
	Failed        string
}{
	Initialized:   events.ChannelInitialized,
	Subscribing:   events.ChannelSubscribing,
	Subscribed:    events.ChannelSubscribed,
	Unsubscribing: events.ChannelUnsubscribing,
	Unsubscribed:  events.ChannelUnsubscribed,
	Paused:        events.ChannelPaused,
	Resumed:       events.ChannelResumed,
	Failed:        events.ChannelFailed,
}

// AuthEvents names for authentication callbacks.
var AuthEvents = struct {
	TokenUpdated string
	TokenExpired string
	TokenError   string
	AuthError    string
}{
	TokenUpdated: events.AuthTokenUpdated,
	TokenExpired: events.AuthTokenExpired,
	TokenError:   events.AuthTokenError,
	AuthError:    events.AuthError,
}
