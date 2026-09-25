package connection

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
	"github.com/qpubio/qpub-go/auth"
	"github.com/qpubio/qpub-go/channel"
	"github.com/qpubio/qpub-go/internal/logger"
	"github.com/qpubio/qpub-go/option"
	"github.com/qpubio/qpub-go/protocol"
	"github.com/qpubio/qpub-go/transport/ws"
)

var wsUpgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

func TestWebSocketReadLoopDispatchesChannelMessage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := wsUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer func() { _ = c.Close() }()

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
	om := option.NewManager(option.WithAutoConnect(false))
	om.Set(option.Option{
		WSHost:           strings.Split(hostPort, ":")[0],
		AutoAuthenticate: false,
	})
	log := logger.NewFactory("it", om.Get()).Create("WS")
	wsClient := ws.New(log)
	chMgr := channel.NewSocketManager(wsClient, log)
	conn := New(om, auth.NewManager(om, nil, log), wsClient, chMgr, log)

	if err := wsClient.Connect("ws://" + hostPort + "/v1"); err != nil {
		t.Fatal(err)
	}
	conn.startReadLoop(context.Background())

	ch := chMgr.Get("news")
	got := make(chan string, 1)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := ch.Subscribe(context.Background(), func(m protocol.Message) {
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
	case <-time.After(3 * time.Second):
		t.Fatal("timeout")
	}
	wg.Wait()
}
