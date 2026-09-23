package protocol

// ActionType matches the backend WebSocket protocol (qpub-js ActionType).
type ActionType int

const (
	ActionConnect ActionType = iota
	ActionConnected
	ActionDisconnect
	ActionDisconnected
	ActionSubscribe
	ActionSubscribed
	ActionUnsubscribe
	ActionUnsubscribed
	ActionPublish
	ActionPublished
	ActionMessage
	ActionError
	ActionPing
	ActionPong
)

// ActionStrings maps action types to wire string names.
var ActionStrings = map[ActionType]string{
	ActionConnect:      "connect",
	ActionConnected:    "connected",
	ActionDisconnect:   "disconnect",
	ActionDisconnected: "disconnected",
	ActionSubscribe:    "subscribe",
	ActionSubscribed:   "subscribed",
	ActionUnsubscribe:  "unsubscribe",
	ActionUnsubscribed: "unsubscribed",
	ActionPublish:      "publish",
	ActionPublished:    "published",
	ActionMessage:      "message",
	ActionError:        "error",
	ActionPing:         "ping",
	ActionPong:         "pong",
}
