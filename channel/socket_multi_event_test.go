package channel_test

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/qpubio/qpub-go/channel"
	"github.com/qpubio/qpub-go/events"
	"github.com/qpubio/qpub-go/internal/logger"
	"github.com/qpubio/qpub-go/option"
	"github.com/qpubio/qpub-go/protocol"
)

func TestSocketMultipleEventSubscriptions(t *testing.T) {
	sender := &mockSender{}
	log := logger.NewFactory("t", option.DefaultOption()).Create("Ch")
	ch := channel.NewSocketChannel("room", sender, log)

	var mu sync.Mutex
	var n1, n2 int
	h1 := func(m protocol.Message) {
		mu.Lock()
		n1++
		mu.Unlock()
	}
	h2 := func(m protocol.Message) {
		mu.Lock()
		n2++
		mu.Unlock()
	}

	go func() {
		time.Sleep(5 * time.Millisecond)
		ch.HandleIncoming(subscribedAck("room"))
	}()

	if err := ch.Subscribe(context.Background(), h1, channel.SubscribeOptions{Event: "event-1", Timeout: time.Second}); err != nil {
		t.Fatal(err)
	}
	if err := ch.Subscribe(context.Background(), h2, channel.SubscribeOptions{Event: "event-2"}); err != nil {
		t.Fatal(err)
	}
	if sender.countAction(protocol.ActionSubscribe) != 1 {
		t.Fatalf("expected single subscribe wire")
	}

	msg, _ := json.Marshal(map[string]interface{}{
		"action": protocol.ActionMessage, "channel": "room", "id": "1",
		"messages": []map[string]interface{}{
			{"event": "event-1", "data": `"a"`},
			{"event": "event-2", "data": `"b"`},
			{"event": "event-3", "data": `"c"`},
		},
	})
	ch.HandleIncoming(msg)

	mu.Lock()
	defer mu.Unlock()
	if n1 != 1 || n2 != 1 {
		t.Fatalf("n1=%d n2=%d", n1, n2)
	}
}

func TestSocketSubscribeWhenDisconnected(t *testing.T) {
	sender := &disconnectedSender{}
	ch := channel.NewSocketChannel("room", sender, logger.NewFactory("t", option.DefaultOption()).Create("Ch"))
	err := ch.Subscribe(context.Background(), func(m protocol.Message) {}, channel.SubscribeOptions{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSocketChannelOnSubscribedEvent(t *testing.T) {
	sender := &mockSender{}
	ch := channel.NewSocketChannel("room", sender, logger.NewFactory("t", option.DefaultOption()).Create("Ch"))
	done := make(chan struct{}, 1)
	ch.On(events.ChannelSubscribed, func(any) { done <- struct{}{} })

	go func() {
		time.Sleep(5 * time.Millisecond)
		ch.HandleIncoming(subscribedAck("room"))
	}()

	_ = ch.Subscribe(context.Background(), func(m protocol.Message) {}, channel.SubscribeOptions{Timeout: time.Second})
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
}

type disconnectedSender struct{}

func (disconnectedSender) Send([]byte) error   { return nil }
func (disconnectedSender) IsConnected() bool { return false }

func (m *mockSender) countAction(action protocol.ActionType) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := 0
	for _, b := range m.sent {
		var v map[string]interface{}
		_ = json.Unmarshal(b, &v)
		if int(v["action"].(float64)) == int(action) {
			n++
		}
	}
	return n
}
