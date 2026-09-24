package connection

import (
	"encoding/json"
	"sync"
	"testing"

	"github.com/qpubio/qpub-go/channel"
	"github.com/qpubio/qpub-go/internal/logger"
	"github.com/qpubio/qpub-go/option"
	"github.com/qpubio/qpub-go/protocol"
)

type dispatchWS struct{}

func (dispatchWS) Send([]byte) error { return nil }

func TestHandleMessageRoutesDataMessageWithStringID(t *testing.T) {
	chMgr := channel.NewSocketManager(dispatchWS{}, logger.NewFactory("t", option.DefaultOption()).Create("Conn"))
	conn := &Conn{chMgr: chMgr}

	ch := chMgr.Get("my-channel")
	var wg sync.WaitGroup
	wg.Add(1)
	_ = ch.Subscribe(t.Context(), func(m protocol.Message) {
		wg.Done()
	}, channel.SubscribeOptions{})

	subAck, _ := json.Marshal(map[string]interface{}{
		"action":          protocol.ActionSubscribed,
		"channel":         "my-channel",
		"subscription_id": "sub-1",
	})
	conn.handleMessage(subAck)

	msg, _ := json.Marshal(map[string]interface{}{
		"action":    protocol.ActionMessage,
		"channel":   "my-channel",
		"id":        "01ARZ3NDEKTSV4RRFFQ69G5FAV",
		"timestamp": "2024-01-01T00:00:00Z",
		"messages":  []map[string]interface{}{{"data": `"hello"`}},
	})
	conn.handleMessage(msg)

	wg.Wait()
}
