package events

import "time"

// Connection event names (qpub-js ConnectionEvents).
const (
	ConnectionInitialized  = "initialized"
	ConnectionConnecting   = "connecting"
	ConnectionOpened       = "opened"
	ConnectionConnected    = "connected"
	ConnectionDisconnected = "disconnected"
	ConnectionClosing      = "closing"
	ConnectionClosed       = "closed"
	ConnectionFailed       = "failed"
)

// Channel event names.
const (
	ChannelInitialized   = "initialized"
	ChannelSubscribing   = "subscribing"
	ChannelSubscribed    = "subscribed"
	ChannelUnsubscribing = "unsubscribing"
	ChannelUnsubscribed  = "unsubscribed"
	ChannelPaused        = "paused"
	ChannelResumed       = "resumed"
	ChannelFailed        = "failed"
)

// Auth event names.
const (
	AuthTokenUpdated = "token_updated"
	AuthTokenExpired = "token_expired"
	AuthTokenError   = "token_error"
	AuthError        = "auth_error"
)

// ConnectionConnectingPayload for connecting event.
type ConnectionConnectingPayload struct {
	Attempt int
}

// ConnectionFailedPayload for failed event.
type ConnectionFailedPayload struct {
	Error   error
	Attempt int
	Context string
}

// ChannelFailedPayload for channel failures.
type ChannelFailedPayload struct {
	ChannelName string
	Error       error
	Action      string
}

// AuthTokenUpdatedPayload when token is set.
type AuthTokenUpdatedPayload struct {
	Token     string
	ExpiresAt *time.Time
}

// AuthTokenExpiredPayload when token expires.
type AuthTokenExpiredPayload struct {
	ExpiredAt time.Time
	Token     string
}

// AuthErrorPayload for auth errors.
type AuthErrorPayload struct {
	Error   error
	Context string
}

// AuthTokenErrorPayload for token errors.
type AuthTokenErrorPayload struct {
	Error error
}
