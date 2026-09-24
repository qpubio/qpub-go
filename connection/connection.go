package connection

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/qpubio/qpub-go/auth"
	"github.com/qpubio/qpub-go/channel"
	"github.com/qpubio/qpub-go/events"
	"github.com/qpubio/qpub-go/internal/emitter"
	"github.com/qpubio/qpub-go/internal/logger"
	"github.com/qpubio/qpub-go/option"
	"github.com/qpubio/qpub-go/protocol"
	"github.com/qpubio/qpub-go/transport/ws"
)

// Conn manages WebSocket lifecycle.
type Conn struct {
	opts    *option.Manager
	auth    *auth.Manager
	ws      *ws.Client
	chMgr   *channel.SocketManager
	log     *logger.Logger
	events  *emitter.Emitter[any]

	mu                    sync.Mutex
	reconnectAttempts     int
	isReconnecting        bool
	intentionalDisconnect bool
	resetting             bool
	pingCounter           int
	pendingPings          map[string]pendingPing
	readCancel            context.CancelFunc
}

type pendingPing struct {
	start time.Time
	ch    chan time.Duration
}

func New(opts *option.Manager, authMgr *auth.Manager, wsClient *ws.Client, chMgr *channel.SocketManager, log *logger.Logger) *Conn {
	c := &Conn{
		opts:         opts,
		auth:         authMgr,
		ws:           wsClient,
		chMgr:        chMgr,
		log:          log,
		events:       emitter.New[any](),
		pendingPings: make(map[string]pendingPing),
	}
	c.events.Emit(events.ConnectionInitialized, nil)
	authMgr.On(events.AuthTokenExpired, func(any) { _ = c.Connect(context.Background()) })
	authMgr.On(events.AuthTokenError, func(p any) { c.handleAuthFail(p) })
	authMgr.On(events.AuthError, func(p any) { c.handleAuthFail(p) })
	return c
}

func (c *Conn) On(event string, fn func(any)) {
	c.events.On(event, fn)
}

func (c *Conn) Connect(ctx context.Context) error {
	c.mu.Lock()
	if c.resetting {
		c.mu.Unlock()
		return nil
	}
	c.mu.Unlock()

	c.events.Emit(events.ConnectionConnecting, events.ConnectionConnectingPayload{Attempt: 1})

	if c.auth.ShouldAutoAuthenticate() {
		if _, err := c.auth.Authenticate(ctx); err != nil {
			c.events.Emit(events.ConnectionFailed, events.ConnectionFailedPayload{Error: err, Attempt: 1})
			return err
		}
	}

	o := c.opts.Get()
	protocolScheme := "ws"
	if o.IsSecure {
		protocolScheme = "wss"
	}
	host := o.WSHost
	base := fmt.Sprintf("%s://%s/v1", protocolScheme, host)
	if o.WSPort != nil {
		base = fmt.Sprintf("%s://%s:%d/v1", protocolScheme, host, *o.WSPort)
	}
	url, err := c.auth.GetAuthenticateURL(base)
	if err != nil {
		return err
	}

	if err := c.ws.Connect(url); err != nil {
		c.events.Emit(events.ConnectionFailed, events.ConnectionFailedPayload{Error: err, Attempt: 1})
		return err
	}

	c.events.Emit(events.ConnectionOpened, nil)
	c.startReadLoop(ctx)

	if o.AutoResubscribe {
		c.chMgr.ResubscribeAfterReconnect(ctx)
	}
	return nil
}

func (c *Conn) startReadLoop(ctx context.Context) {
	if c.readCancel != nil {
		c.readCancel()
	}
	readCtx, cancel := context.WithCancel(ctx)
	c.readCancel = cancel
	conn := c.ws.Conn()
	if conn == nil {
		return
	}
	go func() {
		defer cancel()
		for {
			select {
			case <-readCtx.Done():
				return
			default:
			}
			_, data, err := conn.ReadMessage()
			if err != nil {
				c.onClose(err)
				return
			}
			c.handleMessage(data)
		}
	}()
}

func (c *Conn) handleMessage(data []byte) {
	var peek struct {
		Action protocol.ActionType `json:"action"`
	}
	if err := json.Unmarshal(data, &peek); err != nil {
		c.events.Emit(events.ConnectionFailed, events.ConnectionFailedPayload{
			Error:   err,
			Context: "message_processing",
		})
		return
	}
	switch peek.Action {
	case protocol.ActionConnected:
		var wire struct {
			ConnectionID      string                      `json:"connection_id"`
			ConnectionDetails *protocol.ConnectionDetails `json:"connection_details"`
		}
		_ = json.Unmarshal(data, &wire)
		c.events.Emit(events.ConnectionConnected, map[string]interface{}{
			"connectionId":      wire.ConnectionID,
			"connectionDetails": wire.ConnectionDetails,
		})
	case protocol.ActionDisconnected:
		c.events.Emit(events.ConnectionDisconnected, nil)
	case protocol.ActionPong:
		var wire struct {
			ID int `json:"id"`
		}
		_ = json.Unmarshal(data, &wire)
		c.completePing(wire.ID)
	default:
		c.chMgr.Dispatch(data)
	}
}

func (c *Conn) onClose(err error) {
	c.events.Emit(events.ConnectionClosed, map[string]interface{}{"error": err})
	c.chMgr.PendingSubscribeAllChannels()
	c.mu.Lock()
	intentional := c.intentionalDisconnect
	c.mu.Unlock()
	if !intentional && c.opts.Get().AutoReconnect {
		go c.tryReconnect(context.Background())
	}
}

func (c *Conn) tryReconnect(ctx context.Context) {
	c.mu.Lock()
	if c.isReconnecting {
		c.mu.Unlock()
		return
	}
	o := c.opts.Get()
	if c.reconnectAttempts >= o.MaxReconnectAttempts {
		c.mu.Unlock()
		return
	}
	c.isReconnecting = true
	attempt := c.reconnectAttempts + 1
	c.reconnectAttempts = attempt
	c.mu.Unlock()

	delay := time.Duration(float64(o.InitialReconnectDelayMs)*pow(o.ReconnectBackoffMultiplier, float64(attempt-1))) * time.Millisecond
	if delay > time.Duration(o.MaxReconnectDelayMs)*time.Millisecond {
		delay = time.Duration(o.MaxReconnectDelayMs) * time.Millisecond
	}
	time.Sleep(delay)
	_ = c.Connect(ctx)
	c.mu.Lock()
	c.isReconnecting = false
	c.mu.Unlock()
}

func pow(base, exp float64) float64 {
	r := 1.0
	for i := 0; i < int(exp); i++ {
		r *= base
	}
	return r
}

func (c *Conn) Disconnect() {
	c.mu.Lock()
	c.intentionalDisconnect = true
	c.mu.Unlock()
	c.events.Emit(events.ConnectionClosing, nil)
	c.ws.Disconnect()
	c.events.Emit(events.ConnectionClosed, nil)
}

func (c *Conn) IsConnected() bool {
	return c.ws.IsConnected()
}

// WaitUntilConnected blocks until the WebSocket dial completes or ctx is cancelled.
func (c *Conn) WaitUntilConnected(ctx context.Context) error {
	for {
		if c.ws.IsConnected() {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(50 * time.Millisecond):
		}
	}
}

func (c *Conn) Ping(ctx context.Context) (time.Duration, error) {
	if !c.ws.IsConnected() {
		return 0, fmt.Errorf("WebSocket is not connected")
	}
	c.mu.Lock()
	c.pingCounter++
	id := c.pingCounter
	key := fmt.Sprintf("%d", id)
	start := time.Now()
	ch := make(chan time.Duration, 1)
	c.pendingPings[key] = pendingPing{start: start, ch: ch}
	c.mu.Unlock()

	msg, _ := json.Marshal(map[string]interface{}{"action": protocol.ActionPing, "id": id})
	if err := c.ws.Send(msg); err != nil {
		return 0, err
	}
	timeout := time.Duration(c.opts.Get().PingTimeoutMs) * time.Millisecond
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	case rtt := <-ch:
		return rtt, nil
	case <-time.After(timeout):
		c.mu.Lock()
		delete(c.pendingPings, key)
		c.mu.Unlock()
		return 0, fmt.Errorf("Ping timeout")
	}
}

func (c *Conn) completePing(id int) {
	key := fmt.Sprintf("%d", id)
	c.mu.Lock()
	p, ok := c.pendingPings[key]
	delete(c.pendingPings, key)
	c.mu.Unlock()
	if ok {
		p.ch <- time.Since(p.start)
	}
}

func (c *Conn) IsResetting() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.resetting
}

func (c *Conn) Reset() {
	c.mu.Lock()
	c.resetting = true
	c.mu.Unlock()
	if c.readCancel != nil {
		c.readCancel()
	}
	c.ws.Reset()
	c.mu.Lock()
	c.reconnectAttempts = 0
	c.isReconnecting = false
	c.intentionalDisconnect = false
	c.resetting = false
	c.mu.Unlock()
	c.events.RemoveAll()
}

func (c *Conn) handleAuthFail(p any) {
	var err error
	switch v := p.(type) {
	case events.AuthErrorPayload:
		err = v.Error
	case events.AuthTokenErrorPayload:
		err = v.Error
	default:
		err = fmt.Errorf("auth error")
	}
	c.events.Emit(events.ConnectionFailed, events.ConnectionFailedPayload{Error: err})
	c.Disconnect()
}
