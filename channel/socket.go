package channel

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/qpubio/qpub-go/events"
	"github.com/qpubio/qpub-go/internal/emitter"
	"github.com/qpubio/qpub-go/internal/logger"
	"github.com/qpubio/qpub-go/protocol"
	"github.com/qpubio/qpub-go/transport/ws"
)

// SubscribeOptions for socket subscribe.
type SubscribeOptions struct {
	Event   string
	Timeout time.Duration
}

// MessageHandler receives channel messages.
type MessageHandler func(protocol.Message)

// SocketChannel is a real-time pub/sub channel.
type SocketChannel struct {
	name   string
	ws     *ws.Client
	log    *logger.Logger
	events *emitter.Emitter[any]

	mu              sync.Mutex
	subscribed      bool
	pendingSubscribe bool
	paused          bool
	bufferWhilePaused bool
	pausedMessages  []protocol.Message
	handler MessageHandler
}

func NewSocketChannel(name string, wsClient *ws.Client, log *logger.Logger) *SocketChannel {
	return &SocketChannel{
		name:              name,
		ws:                wsClient,
		log:               log,
		events:            emitter.New[any](),
		bufferWhilePaused: true,
	}
}

func (c *SocketChannel) Name() string { return c.name }

func (c *SocketChannel) HasCallback() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.handler != nil
}

func (c *SocketChannel) SetPendingSubscribe(v bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.pendingSubscribe = v
}

func (c *SocketChannel) Subscribe(ctx context.Context, handler MessageHandler, opts SubscribeOptions) error {
	c.mu.Lock()
	c.handler = handler
	c.mu.Unlock()
	c.events.Emit(events.ChannelSubscribing, nil)
	msg := map[string]interface{}{
		"action":  protocol.ActionSubscribe,
		"channel": c.name,
	}
	if opts.Event != "" {
		msg["event"] = opts.Event
	}
	b, _ := json.Marshal(msg)
	if err := c.ws.Send(b); err != nil {
		return err
	}
	c.mu.Lock()
	c.subscribed = true
	c.pendingSubscribe = false
	c.mu.Unlock()
	c.events.Emit(events.ChannelSubscribed, nil)
	return nil
}

func (c *SocketChannel) Unsubscribe(ctx context.Context) error {
	_ = ctx
	c.mu.Lock()
	c.handler = nil
	c.mu.Unlock()
	msg := map[string]interface{}{
		"action":  protocol.ActionUnsubscribe,
		"channel": c.name,
	}
	b, _ := json.Marshal(msg)
	if err := c.ws.Send(b); err != nil {
		return err
	}
	c.mu.Lock()
	c.subscribed = false
	c.mu.Unlock()
	c.events.Emit(events.ChannelUnsubscribed, nil)
	return nil
}

func (c *SocketChannel) Resubscribe(ctx context.Context) error {
	c.mu.Lock()
	h := c.handler
	c.mu.Unlock()
	if h == nil {
		return nil
	}
	return c.Subscribe(ctx, h, SubscribeOptions{})
}

func (c *SocketChannel) Publish(ctx context.Context, data interface{}, opts PublishOptions) error {
	_ = ctx
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}
	msg := map[string]interface{}{
		"action":  protocol.ActionPublish,
		"channel": c.name,
		"messages": []protocol.DataMessagePayload{{
			Alias: opts.Alias,
			Event: opts.Event,
			Data:  payload,
		}},
	}
	b, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return c.ws.Send(b)
}

func (c *SocketChannel) Pause(bufferMessages bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.paused = true
	c.bufferWhilePaused = bufferMessages
	c.events.Emit(events.ChannelPaused, nil)
}

func (c *SocketChannel) Resume() {
	c.mu.Lock()
	msgs := append([]protocol.Message(nil), c.pausedMessages...)
	c.pausedMessages = nil
	c.paused = false
	h := c.handler
	c.mu.Unlock()
	c.events.Emit(events.ChannelResumed, nil)
	for _, m := range msgs {
		if h != nil {
			h(m)
		}
	}
}

func (c *SocketChannel) IsPaused() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.paused
}

func (c *SocketChannel) ClearBufferedMessages() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.pausedMessages = nil
}

func (c *SocketChannel) HandleWireMessage(raw []byte) {
	var wire struct {
		Action   protocol.ActionType          `json:"action"`
		Channel  string                       `json:"channel"`
		Messages []protocol.DataMessagePayload `json:"messages"`
		ID       string                       `json:"id"`
		Timestamp string                      `json:"timestamp"`
	}
	if err := json.Unmarshal(raw, &wire); err != nil {
		return
	}
	if wire.Channel != c.name {
		return
	}
	if wire.Action != protocol.ActionMessage {
		return
	}
	for _, p := range wire.Messages {
		m := protocol.Message{
			Action:    protocol.ActionMessage,
			Channel:   c.name,
			ID:        wire.ID,
			Timestamp: wire.Timestamp,
			Alias:     p.Alias,
			Event:     p.Event,
			Data:      p.Data,
		}
		c.dispatch(m)
	}
}

func (c *SocketChannel) dispatch(m protocol.Message) {
	c.mu.Lock()
	if c.paused && c.bufferWhilePaused {
		c.pausedMessages = append(c.pausedMessages, m)
		c.mu.Unlock()
		return
	}
	if c.paused {
		c.mu.Unlock()
		return
	}
	h := c.handler
	c.mu.Unlock()
	if h != nil {
		h(m)
	}
}

func (c *SocketChannel) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.handler = nil
	c.subscribed = false
	c.paused = false
	c.pausedMessages = nil
	c.events.RemoveAll()
}

// SocketManager manages socket channels with ref counting.
type SocketManager struct {
	channels  map[string]*SocketChannel
	refCounts map[string]int
	ws        *ws.Client
	log       *logger.Logger
}

func NewSocketManager(wsClient *ws.Client, log *logger.Logger) *SocketManager {
	return &SocketManager{
		channels:  make(map[string]*SocketChannel),
		refCounts: make(map[string]int),
		ws:        wsClient,
		log:       log,
	}
}

func (m *SocketManager) Get(name string) *SocketChannel {
	if _, ok := m.channels[name]; !ok {
		m.channels[name] = NewSocketChannel(name, m.ws, m.log)
	}
	m.refCounts[name]++
	return m.channels[name]
}

func (m *SocketManager) Release(name string) {
	count, ok := m.refCounts[name]
	if !ok || count <= 0 {
		return
	}
	if count == 1 {
		ch := m.channels[name]
		if ch != nil && ch.HasCallback() {
			m.refCounts[name] = 0
			return
		}
		if ch != nil {
			ch.Reset()
		}
		delete(m.channels, name)
		delete(m.refCounts, name)
		return
	}
	m.refCounts[name] = count - 1
}

func (m *SocketManager) Has(name string) bool {
	_, ok := m.channels[name]
	return ok
}

func (m *SocketManager) Remove(name string) {
	delete(m.channels, name)
	delete(m.refCounts, name)
}

func (m *SocketManager) All() []*SocketChannel {
	out := make([]*SocketChannel, 0, len(m.channels))
	for _, ch := range m.channels {
		out = append(out, ch)
	}
	return out
}

func (m *SocketManager) PendingSubscribeAllChannels() {
	for _, ch := range m.channels {
		ch.SetPendingSubscribe(true)
	}
}

func (m *SocketManager) ResubscribeAllChannels(ctx context.Context) {
	for _, ch := range m.channels {
		if ch.HasCallback() {
			_ = ch.Resubscribe(ctx)
		}
	}
}

func (m *SocketManager) Dispatch(raw []byte) {
	var peek struct {
		Channel string `json:"channel"`
	}
	_ = json.Unmarshal(raw, &peek)
	if ch, ok := m.channels[peek.Channel]; ok {
		ch.HandleWireMessage(raw)
	}
}

func (m *SocketManager) Reset() {
	for _, ch := range m.channels {
		ch.Reset()
	}
	m.channels = make(map[string]*SocketChannel)
	m.refCounts = make(map[string]int)
}
