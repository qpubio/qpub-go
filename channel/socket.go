package channel

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/qpubio/qpub-go/events"
	"github.com/qpubio/qpub-go/internal/emitter"
	"github.com/qpubio/qpub-go/internal/logger"
	"github.com/qpubio/qpub-go/protocol"
)

// SubscribeOptions for socket subscribe.
type SubscribeOptions struct {
	Event   string
	Timeout time.Duration
}

// UnsubscribeOptions for socket unsubscribe (event-specific or full channel).
type UnsubscribeOptions struct {
	Event   string
	Handler MessageHandler
	Timeout time.Duration
}

// MessageHandler receives channel messages (invoked serially per channel).
type MessageHandler func(protocol.Message)

type queuedOperation struct {
	kind    string // subscribe | unsubscribe
	handler MessageHandler
	subOpts SubscribeOptions
	unsub   UnsubscribeOptions
}

// SocketChannel is a real-time pub/sub channel.
type SocketChannel struct {
	name   string
	ws     MessageSender
	log    *logger.Logger
	events *emitter.Emitter[any]

	mu                 sync.Mutex
	subscribed         bool
	pendingSubscribe   bool
	pendingUnsubscribe bool
	paused             bool
	bufferWhilePaused  bool
	pausedMessages     []protocol.Message
	catchAllHandler    MessageHandler
	eventCallbacks     map[string][]MessageHandler
	operationQueue     []queuedOperation
	handlerSerial      sync.Mutex
	subDone            chan struct{}
	unsubDone          chan struct{}
	reconnectPending   bool
}

func NewSocketChannel(name string, ws MessageSender, log *logger.Logger) *SocketChannel {
	return &SocketChannel{
		name:              name,
		ws:                ws,
		log:               log,
		events:            emitter.New[any](),
		eventCallbacks:    make(map[string][]MessageHandler),
		bufferWhilePaused: true,
	}
}

func (c *SocketChannel) Name() string { return c.name }

// On registers a channel lifecycle listener.
func (c *SocketChannel) On(event string, fn func(any)) {
	c.events.On(event, fn)
}

func (c *SocketChannel) HasCallback() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.catchAllHandler != nil || len(c.eventCallbacks) > 0
}

func (c *SocketChannel) SetPendingSubscribe(v bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.reconnectPending = v
	c.pendingSubscribe = v
	if v {
		c.subscribed = false
	}
}

func (c *SocketChannel) NeedsResubscribe() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.reconnectPending && c.hasCallbackLocked()
}

func (c *SocketChannel) hasCallbackLocked() bool {
	return c.catchAllHandler != nil || len(c.eventCallbacks) > 0
}

func (c *SocketChannel) Subscribe(ctx context.Context, handler MessageHandler, opts SubscribeOptions) error {
	if !c.ws.IsConnected() {
		return errors.New("Cannot subscribe: WebSocket is not connected")
	}

	c.mu.Lock()
	if opts.Event != "" {
		if c.pendingUnsubscribe {
			c.operationQueue = append(c.operationQueue, queuedOperation{
				kind: "subscribe", handler: handler, subOpts: opts,
			})
			c.mu.Unlock()
			return nil
		}
		if c.subscribed || c.pendingSubscribe {
			c.mu.Unlock()
			c.addEventHandler(opts.Event, handler)
			return nil
		}
		c.mu.Unlock()
		return c.subscribeNetwork(ctx, handler, opts, true)
	}

	if c.pendingSubscribe || c.pendingUnsubscribe {
		c.operationQueue = append(c.operationQueue, queuedOperation{
			kind: "subscribe", handler: handler, subOpts: opts,
		})
		c.mu.Unlock()
		return nil
	}
	if c.subscribed && !c.pendingSubscribe {
		c.mu.Lock()
		c.catchAllHandler = handler
		c.eventCallbacks = make(map[string][]MessageHandler)
		c.mu.Unlock()
		return nil
	}
	c.mu.Unlock()
	return c.subscribeNetwork(ctx, handler, opts, false)
}

func (c *SocketChannel) subscribeNetwork(ctx context.Context, handler MessageHandler, opts SubscribeOptions, eventMode bool) error {
	done := make(chan struct{}, 1)
	c.mu.Lock()
	if eventMode {
		c.addEventHandlerLocked(opts.Event, handler)
	} else {
		c.catchAllHandler = handler
		c.eventCallbacks = make(map[string][]MessageHandler)
	}
	c.pendingSubscribe = true
	c.subDone = done
	c.mu.Unlock()
	c.events.Emit(events.ChannelSubscribing, nil)

	if err := c.sendSubscribeWire(); err != nil {
		c.mu.Lock()
		c.pendingSubscribe = false
		c.mu.Unlock()
		return err
	}
	return waitAck(ctx, done, opts.Timeout, "subscribe")
}

func (c *SocketChannel) addEventHandler(event string, handler MessageHandler) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.addEventHandlerLocked(event, handler)
}

func (c *SocketChannel) addEventHandlerLocked(event string, handler MessageHandler) {
	list := c.eventCallbacks[event]
	for _, h := range list {
		if handlerIdentity(h) == handlerIdentity(handler) {
			return
		}
	}
	c.eventCallbacks[event] = append(list, handler)
}

func (c *SocketChannel) sendSubscribeWire() error {
	msg := map[string]interface{}{
		"action":  protocol.ActionSubscribe,
		"channel": c.name,
	}
	b, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return c.ws.Send(b)
}

func (c *SocketChannel) Unsubscribe(ctx context.Context, opts ...UnsubscribeOptions) error {
	var o UnsubscribeOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	if o.Timeout <= 0 {
		o.Timeout = 10 * time.Second
	}

	c.mu.Lock()
	if o.Event != "" {
		if c.subscribed && !c.pendingUnsubscribe {
			needsNetwork := c.removeEventHandlerLocked(o.Event, o.Handler)
			if !needsNetwork {
				c.mu.Unlock()
				return nil
			}
		} else if c.pendingSubscribe || c.pendingUnsubscribe {
			c.operationQueue = append(c.operationQueue, queuedOperation{
				kind: "unsubscribe", handler: o.Handler, unsub: o,
			})
			c.mu.Unlock()
			return nil
		}
		c.mu.Unlock()
		return c.unsubscribeNetwork(ctx, o)
	}

	if !c.subscribed && !c.pendingSubscribe {
		c.mu.Unlock()
		return nil
	}
	if c.pendingSubscribe || c.pendingUnsubscribe {
		c.operationQueue = append(c.operationQueue, queuedOperation{
			kind: "unsubscribe", unsub: o,
		})
		c.mu.Unlock()
		return nil
	}
	c.mu.Unlock()
	return c.unsubscribeNetwork(ctx, o)
}

func (c *SocketChannel) removeEventHandlerLocked(event string, handler MessageHandler) (needsNetwork bool) {
	list, ok := c.eventCallbacks[event]
	if !ok {
		return false
	}
	if handler != nil {
		id := handlerIdentity(handler)
		filtered := list[:0]
		for _, h := range list {
			if handlerIdentity(h) != id {
				filtered = append(filtered, h)
			}
		}
		list = filtered
		if len(list) == 0 {
			delete(c.eventCallbacks, event)
		} else {
			c.eventCallbacks[event] = list
		}
	} else {
		delete(c.eventCallbacks, event)
	}
	if len(c.eventCallbacks) == 0 && c.catchAllHandler == nil && c.subscribed {
		return true
	}
	return false
}

func (c *SocketChannel) unsubscribeNetwork(ctx context.Context, opts UnsubscribeOptions) error {
	done := make(chan struct{}, 1)
	c.mu.Lock()
	c.pendingUnsubscribe = true
	c.unsubDone = done
	c.mu.Unlock()
	c.events.Emit(events.ChannelUnsubscribing, nil)

	msg := map[string]interface{}{
		"action":  protocol.ActionUnsubscribe,
		"channel": c.name,
	}
	b, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	if err := c.ws.Send(b); err != nil {
		c.mu.Lock()
		c.pendingUnsubscribe = false
		c.mu.Unlock()
		return err
	}
	return waitAck(ctx, done, opts.Timeout, "unsubscribe")
}

func waitAck(ctx context.Context, done <-chan struct{}, timeout time.Duration, op string) error {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	tctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	select {
	case <-tctx.Done():
		return fmt.Errorf("%s: %w", op, tctx.Err())
	case <-done:
		return nil
	}
}

func (c *SocketChannel) Resubscribe(ctx context.Context) error {
	c.mu.Lock()
	catchAll := c.catchAllHandler
	eventsCopy := make(map[string][]MessageHandler, len(c.eventCallbacks))
	for ev, hs := range c.eventCallbacks {
		eventsCopy[ev] = append([]MessageHandler(nil), hs...)
	}
	c.subscribed = false
	c.pendingSubscribe = false
	c.pendingUnsubscribe = false
	c.mu.Unlock()

	if catchAll == nil && len(eventsCopy) == 0 {
		return nil
	}

	done := make(chan struct{}, 1)
	c.mu.Lock()
	c.catchAllHandler = catchAll
	c.eventCallbacks = eventsCopy
	c.pendingSubscribe = true
	c.subDone = done
	c.mu.Unlock()
	c.events.Emit(events.ChannelSubscribing, nil)

	if err := c.sendSubscribeWire(); err != nil {
		c.mu.Lock()
		c.pendingSubscribe = false
		c.mu.Unlock()
		return err
	}
	return waitAck(ctx, done, 10*time.Second, "subscribe")
}

func (c *SocketChannel) Publish(ctx context.Context, data interface{}, opts PublishOptions) error {
	_ = ctx
	if !c.ws.IsConnected() {
		return errors.New("Cannot publish: WebSocket is not connected")
	}
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
	c.mu.Unlock()
	c.events.Emit(events.ChannelResumed, nil)
	for _, m := range msgs {
		c.deliverMessage(m)
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

func (c *SocketChannel) HandleIncoming(raw []byte) {
	var wire struct {
		Action         protocol.ActionType           `json:"action"`
		Channel        string                        `json:"channel"`
		SubscriptionID string                        `json:"subscription_id"`
		Messages       []protocol.DataMessagePayload `json:"messages"`
		ID             string                        `json:"id"`
		Timestamp      string                        `json:"timestamp"`
		Error          *protocol.ErrorInfo           `json:"error"`
	}
	if err := json.Unmarshal(raw, &wire); err != nil {
		return
	}
	if wire.Channel != "" && wire.Channel != c.name {
		return
	}

	switch wire.Action {
	case protocol.ActionSubscribed:
		c.mu.Lock()
		c.subscribed = true
		c.pendingSubscribe = false
		c.reconnectPending = false
		done := c.subDone
		c.subDone = nil
		c.mu.Unlock()
		c.events.Emit(events.ChannelSubscribed, map[string]string{
			"channelName":    c.name,
			"subscriptionId": wire.SubscriptionID,
		})
		signalDone(done)
		c.processOperationQueue()

	case protocol.ActionUnsubscribed:
		c.mu.Lock()
		c.subscribed = false
		c.pendingUnsubscribe = false
		c.catchAllHandler = nil
		c.eventCallbacks = make(map[string][]MessageHandler)
		done := c.unsubDone
		c.unsubDone = nil
		c.mu.Unlock()
		c.events.Emit(events.ChannelUnsubscribed, map[string]string{
			"channelName":    c.name,
			"subscriptionId": wire.SubscriptionID,
		})
		signalDone(done)
		c.processOperationQueue()

	case protocol.ActionMessage:
		c.mu.Lock()
		subscribed := c.subscribed
		c.mu.Unlock()
		if !subscribed {
			return
		}
		for i, p := range wire.Messages {
			id := wire.ID
			if len(wire.Messages) > 1 {
				id = wire.ID + "-" + itoa(i)
			}
			m := protocol.Message{
				Action:    protocol.ActionMessage,
				Channel:   c.name,
				ID:        id,
				Timestamp: wire.Timestamp,
				Alias:     p.Alias,
				Event:     p.Event,
				Data:      p.Data,
			}
			c.deliverMessage(m)
		}

	case protocol.ActionError:
		msg := "channel error"
		if wire.Error != nil {
			msg = wire.Error.Message
		}
		c.events.Emit(events.ChannelFailed, events.ChannelFailedPayload{
			ChannelName: c.name,
			Error:       errors.New(msg),
			Action:      "channel_operation",
		})
	}
}

func signalDone(ch chan struct{}) {
	if ch == nil {
		return
	}
	select {
	case ch <- struct{}{}:
	default:
	}
}

func (c *SocketChannel) processOperationQueue() {
	c.mu.Lock()
	if c.pendingSubscribe || c.pendingUnsubscribe || len(c.operationQueue) == 0 {
		c.mu.Unlock()
		return
	}
	op := c.operationQueue[0]
	c.operationQueue = c.operationQueue[1:]
	c.mu.Unlock()

	ctx := context.Background()
	switch op.kind {
	case "subscribe":
		_ = c.Subscribe(ctx, op.handler, op.subOpts)
	case "unsubscribe":
		_ = c.Unsubscribe(ctx, op.unsub)
	}
}

func (c *SocketChannel) deliverMessage(m protocol.Message) {
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
	var eventHandlers []MessageHandler
	if m.Event != "" {
		eventHandlers = append([]MessageHandler(nil), c.eventCallbacks[m.Event]...)
	}
	catchAll := c.catchAllHandler
	c.mu.Unlock()

	if len(eventHandlers) > 0 {
		for _, h := range eventHandlers {
			c.invokeHandler(h, m)
		}
		return
	}
	if catchAll != nil {
		c.invokeHandler(catchAll, m)
	}
}

func (c *SocketChannel) invokeHandler(h MessageHandler, m protocol.Message) {
	if h == nil {
		return
	}
	c.handlerSerial.Lock()
	defer c.handlerSerial.Unlock()
	h(m)
}

func (c *SocketChannel) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.catchAllHandler = nil
	c.eventCallbacks = make(map[string][]MessageHandler)
	c.subscribed = false
	c.pendingSubscribe = false
	c.pendingUnsubscribe = false
	c.paused = false
	c.pausedMessages = nil
	c.operationQueue = nil
	c.events.RemoveAll()
}

func handlerIdentity(h MessageHandler) uintptr {
	if h == nil {
		return 0
	}
	return reflect.ValueOf(h).Pointer()
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b [8]byte
	n := len(b)
	for i > 0 {
		n--
		b[n] = byte('0' + i%10)
		i /= 10
	}
	return string(b[n:])
}

// SocketManager manages socket channels with ref counting.
type SocketManager struct {
	channels  map[string]*SocketChannel
	refCounts map[string]int
	ws        MessageSender
	log       *logger.Logger
}

func NewSocketManager(ws MessageSender, log *logger.Logger) *SocketManager {
	return &SocketManager{
		channels:  make(map[string]*SocketChannel),
		refCounts: make(map[string]int),
		ws:        ws,
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

func (m *SocketManager) ResubscribeAfterReconnect(ctx context.Context) {
	for _, ch := range m.channels {
		if ch.NeedsResubscribe() {
			_ = ch.Resubscribe(ctx)
		}
	}
}

func (m *SocketManager) Dispatch(raw []byte) {
	var peek struct {
		Channel string              `json:"channel"`
		Action  protocol.ActionType `json:"action"`
	}
	if err := json.Unmarshal(raw, &peek); err != nil {
		return
	}
	if peek.Channel == "" {
		return
	}
	if ch, ok := m.channels[peek.Channel]; ok {
		ch.HandleIncoming(raw)
	}
}

func (m *SocketManager) Reset() {
	for _, ch := range m.channels {
		ch.Reset()
	}
	m.channels = make(map[string]*SocketChannel)
	m.refCounts = make(map[string]int)
}
