package connection

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/qpubio/qpub-go/option"
	"github.com/qpubio/qpub-go/protocol"
)

func TestCompletePingRTT(t *testing.T) {
	c := &Conn{
		pendingPings: make(map[string]pendingPing),
	}
	start := time.Now()
	ch := make(chan time.Duration, 1)
	c.pendingPings["7"] = pendingPing{start: start, ch: ch}

	c.completePing(7)

	select {
	case rtt := <-ch:
		if rtt < 0 {
			t.Fatalf("rtt=%v", rtt)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
}

func TestPingWhenNotConnected(t *testing.T) {
	conn := newTestConn(t, option.NewManager(option.WithAutoConnect(false)), &authHTTPMock{})
	_, err := conn.Ping(context.Background())
	if err == nil || err.Error() != "WebSocket is not connected" {
		t.Fatalf("err=%v", err)
	}
}

func TestPingTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, _ := connUpgrader.Upgrade(w, r, nil)
		defer func() { _ = c.Close() }()
		for {
			_, data, err := c.ReadMessage()
			if err != nil {
				return
			}
			var peek struct {
				Action protocol.ActionType `json:"action"`
				ID     int                 `json:"id"`
			}
			if json.Unmarshal(data, &peek) == nil && peek.Action == protocol.ActionPing {
				// Do not respond with pong — force client timeout.
				continue
			}
		}
	}))
	defer srv.Close()

	om := wsOptsFromServerURL(srv.URL, func(o *option.Option) { o.PingTimeoutMs = 100 })
	conn := newTestConn(t, om, &authHTTPMock{})
	if err := conn.Connect(context.Background()); err != nil {
		t.Fatal(err)
	}
	_, err := conn.Ping(context.Background())
	if err == nil || err.Error() != "Ping timeout" {
		t.Fatalf("err=%v", err)
	}
	conn.Disconnect()
}

func TestConcurrentPings(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, _ := connUpgrader.Upgrade(w, r, nil)
		defer func() { _ = c.Close() }()
		for {
			_, data, err := c.ReadMessage()
			if err != nil {
				return
			}
			var peek struct {
				Action protocol.ActionType `json:"action"`
				ID     int                 `json:"id"`
			}
			if json.Unmarshal(data, &peek) == nil && peek.Action == protocol.ActionPing {
				pong, _ := json.Marshal(map[string]interface{}{"action": protocol.ActionPong, "id": peek.ID})
				_ = c.WriteMessage(websocket.TextMessage, pong)
			}
		}
	}))
	defer srv.Close()

	om := wsOptsFromServerURL(srv.URL, func(o *option.Option) { o.PingTimeoutMs = 2000 })
	conn := newTestConn(t, om, &authHTTPMock{})
	if err := conn.Connect(context.Background()); err != nil {
		t.Fatal(err)
	}
	conn.startReadLoop(context.Background())

	type result struct {
		rtt time.Duration
		err error
	}
	ch := make(chan result, 2)
	go func() {
		rtt, err := conn.Ping(context.Background())
		ch <- result{rtt, err}
	}()
	go func() {
		rtt, err := conn.Ping(context.Background())
		ch <- result{rtt, err}
	}()
	for i := 0; i < 2; i++ {
		select {
		case res := <-ch:
			if res.err != nil {
				t.Fatalf("ping err=%v", res.err)
			}
			if res.rtt < 0 {
				t.Fatal("negative rtt")
			}
		case <-time.After(3 * time.Second):
			t.Fatal("timeout")
		}
	}
	conn.Disconnect()
}

func TestCompletePingIgnoresUnknownID(t *testing.T) {
	c := &Conn{pendingPings: make(map[string]pendingPing)}
	c.completePing(999)
}
