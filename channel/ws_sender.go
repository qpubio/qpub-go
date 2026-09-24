package channel

// MessageSender sends WebSocket frames (implemented by transport/ws.Client).
type MessageSender interface {
	Send(data []byte) error
}
