package qpub_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/qpubio/qpub-go/channel"
	"github.com/qpubio/qpub-go/option"
	"github.com/qpubio/qpub-go/protocol"
	"github.com/qpubio/qpub-go/qpub"
)

var composeUpgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

func TestSocketComposeSubscribeReceivesMessage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := composeUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer c.Close()

		connected, _ := json.Marshal(map[string]interface{}{
			"action": protocol.ActionConnected, "connection_id": "c1",
		})
		_ = c.WriteMessage(websocket.TextMessage, connected)

		for {
			_, data, err := c.ReadMessage()
			if err != nil {
				return
			}
			var peek struct {
				Action  protocol.ActionType `json:"action"`
				Channel string              `json:"channel"`
			}
			if json.Unmarshal(data, &peek) != nil {
				continue
			}
			if peek.Action == protocol.ActionSubscribe {
				ack, _ := json.Marshal(map[string]interface{}{
					"action": protocol.ActionSubscribed, "channel": peek.Channel, "subscription_id": "s1",
				})
				_ = c.WriteMessage(websocket.TextMessage, ack)

				payload, _ := json.Marshal(map[string]interface{}{
					"action": protocol.ActionMessage, "channel": peek.Channel, "id": "01ARZ3NDEKTSV4RRFFQ69G5FAV",
					"messages": []map[string]interface{}{{"data": "hello"}},
				})
				_ = c.WriteMessage(websocket.TextMessage, payload)
			}
		}
	}))
	defer srv.Close()

	hostPort := strings.TrimPrefix(srv.URL, "http://")
	socket := qpub.NewSocket(
		qpub.WithAPIKey("pub:sec"),
		option.WithAutoConnect(false),
		option.WithIsSecure(false),
		func(o *option.Option) {
			o.WSHost = hostPort
			o.AutoAuthenticate = false
		},
	)
	defer socket.Reset()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := socket.Connection.Connect(ctx); err != nil {
		t.Fatal(err)
	}
	if err := socket.Connection.WaitUntilConnected(ctx); err != nil {
		t.Fatal(err)
	}

	ch := socket.Channels.Get("news")
	got := make(chan string, 1)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := ch.Subscribe(ctx, func(m qpub.Message) {
			got <- string(m.Data)
		}, channel.SubscribeOptions{Timeout: 2 * time.Second}); err != nil {
			t.Error(err)
		}
	}()

	select {
	case s := <-got:
		if s != "hello" && s != `"hello"` {
			t.Fatalf("payload %q", s)
		}
	case <-ctx.Done():
		t.Fatal("timeout waiting for message")
	}
	wg.Wait()
}
