package connection

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/qpubio/qpub-go/auth"
	"github.com/qpubio/qpub-go/channel"
	"github.com/qpubio/qpub-go/events"
	"github.com/qpubio/qpub-go/internal/emitter"
	"github.com/qpubio/qpub-go/internal/logger"
	"github.com/qpubio/qpub-go/option"
	"github.com/qpubio/qpub-go/protocol"
	"github.com/qpubio/qpub-go/transport/ws"
)

var connUpgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

type authHTTPMock struct {
	posts int32
}

func (a *authHTTPMock) Post(ctx context.Context, url string, body interface{}, headers map[string]string) ([]byte, int, error) {
	atomic.AddInt32(&a.posts, 1)
	return nil, 500, errors.New("auth failed")
}

func wsOptsFromServerURL(srvURL string, extra ...option.OptionFunc) *option.Manager {
	hostPort := strings.TrimPrefix(srvURL, "http://")
	parts := strings.Split(hostPort, ":")
	host := parts[0]
	port := 80
	if len(parts) == 2 {
		if p, err := strconv.Atoi(parts[1]); err == nil {
			port = p
		}
	}
	funcs := []option.OptionFunc{
		option.WithAPIKey("k"),
		option.WithAutoConnect(false),
		option.WithIsSecure(false),
		func(o *option.Option) {
			o.WSHost = host
			o.WSPort = &port
			o.AutoAuthenticate = false
		},
	}
	funcs = append(funcs, extra...)
	return option.NewManager(funcs...)
}

func newTestConn(t *testing.T, om *option.Manager, httpDoer auth.HTTPDoer) *Conn {
	t.Helper()
	log := logger.NewFactory("t", om.Get()).Create("Conn")
	wsClient := ws.New(log)
	chMgr := channel.NewSocketManager(wsClient, log)
	authMgr := auth.NewManager(om, httpDoer, log)
	return New(om, authMgr, wsClient, chMgr, log)
}

func TestNodePingWatchdogNotApplicable(t *testing.T) {
	t.Skip("N/A: qpub-js Node ws server ping watchdog has no equivalent in gorilla WebSocket client")
}

func TestHandleMessageDisconnectedAction(t *testing.T) {
	conn := &Conn{events: emitter.New[any]()}
	var disconnected bool
	conn.events.On(events.ConnectionDisconnected, func(any) { disconnected = true })
	raw, _ := json.Marshal(map[string]interface{}{"action": protocol.ActionDisconnected})
	conn.handleMessage(raw)
	if !disconnected {
		t.Fatal("expected disconnected event")
	}
}

func TestAuthFailureEmitsConnectionFailed(t *testing.T) {
	om := option.NewManager(
		option.WithAPIKey("k:s"),
		option.WithAutoConnect(false),
		func(o *option.Option) { o.AuthURL = "https://auth.example.com/token" },
	)
	conn := newTestConn(t, om, &authHTTPMock{})
	var failed bool
	conn.On(events.ConnectionFailed, func(any) { failed = true })
	err := conn.Connect(context.Background())
	if err == nil {
		t.Fatal("expected auth error")
	}
	if !failed {
		t.Fatal("expected connection failed event")
	}
}

func TestConnectSkipsAuthWhenAutoAuthenticateDisabled(t *testing.T) {
	var gotURL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotURL = r.URL.String()
		c, _ := connUpgrader.Upgrade(w, r, nil)
		defer func() { _ = c.Close() }()
		connected, _ := json.Marshal(map[string]interface{}{
			"action": protocol.ActionConnected, "connection_id": "c1",
		})
		_ = c.WriteMessage(websocket.TextMessage, connected)
	}))
	defer srv.Close()

	om := wsOptsFromServerURL(srv.URL, func(o *option.Option) { o.APIKey = "pub:sec" })
	httpMock := &authHTTPMock{}
	conn := newTestConn(t, om, httpMock)
	if err := conn.Connect(context.Background()); err != nil {
		t.Fatal(err)
	}
	if httpMock.posts != 0 {
		t.Fatalf("posts=%d", httpMock.posts)
	}
	if !strings.Contains(gotURL, "api_key=") {
		t.Fatalf("url=%s", gotURL)
	}
	conn.Disconnect()
}

func TestConnectWSURLIncludesPortWhenSet(t *testing.T) {
	var hostHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hostHeader = r.Host
		c, _ := connUpgrader.Upgrade(w, r, nil)
		defer func() { _ = c.Close() }()
	}))
	defer srv.Close()

	om := wsOptsFromServerURL(srv.URL)
	conn := newTestConn(t, om, &authHTTPMock{})
	if err := conn.Connect(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(hostHeader, ":") {
		t.Fatalf("host=%s", hostHeader)
	}
	conn.Disconnect()
}

func TestDisconnectAndIsConnected(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, _ := connUpgrader.Upgrade(w, r, nil)
		defer func() { _ = c.Close() }()
		time.Sleep(50 * time.Millisecond)
	}))
	defer srv.Close()

	om := wsOptsFromServerURL(srv.URL)
	conn := newTestConn(t, om, &authHTTPMock{})
	if err := conn.Connect(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !conn.IsConnected() {
		t.Fatal("expected connected")
	}
	conn.Disconnect()
	if conn.IsConnected() {
		t.Fatal("expected disconnected")
	}
}

func TestHandleAuthFailDisconnects(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, _ := connUpgrader.Upgrade(w, r, nil)
		defer func() { _ = c.Close() }()
	}))
	defer srv.Close()
	om := wsOptsFromServerURL(srv.URL)
	conn := newTestConn(t, om, &authHTTPMock{})
	if err := conn.Connect(context.Background()); err != nil {
		t.Fatal(err)
	}
	conn.handleAuthFail(events.AuthErrorPayload{Error: errors.New("token bad")})
	if conn.IsConnected() {
		t.Fatal("expected disconnect after auth fail")
	}
}

func TestResetClearsReconnectState(t *testing.T) {
	log := logger.NewFactory("t", option.DefaultOption()).Create("Conn")
	conn := &Conn{
		events:            emitter.New[any](),
		ws:                ws.New(log),
		pendingPings:      make(map[string]pendingPing),
		reconnectAttempts: 3,
		isReconnecting:    true,
	}
	conn.Reset()
	if conn.reconnectAttempts != 0 || conn.isReconnecting {
		t.Fatal("reset did not clear reconnect state")
	}
}
