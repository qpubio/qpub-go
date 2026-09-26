package channel_test

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/qpubio/qpub-go/channel"
	"github.com/qpubio/qpub-go/internal/logger"
	"github.com/qpubio/qpub-go/option"
	"github.com/qpubio/qpub-go/protocol"
)

type mockSender struct {
	mu   sync.Mutex
	sent [][]byte
}

func (m *mockSender) Send(data []byte) error {
	m.mu.Lock()
	m.sent = append(m.sent, append([]byte(nil), data...))
	m.mu.Unlock()
	return nil
}

func (m *mockSender) IsConnected() bool { return true }

func (m *mockSender) last() map[string]interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.sent) == 0 {
		return nil
	}
	var v map[string]interface{}
	_ = json.Unmarshal(m.sent[len(m.sent)-1], &v)
	return v
}

func ackSubscribed(t *testing.T, ch *channel.SocketChannel, sender *mockSender) {
	t.Helper()
	time.Sleep(5 * time.Millisecond)
	ch.HandleIncoming(subscribedAck(ch.Name()))
}

func subscribeRoom(t *testing.T, ch *channel.SocketChannel, sender *mockSender, fn channel.MessageHandler, opts ...channel.SubscribeOptions) {
	t.Helper()
	o := channel.SubscribeOptions{Timeout: time.Second}
	if len(opts) > 0 {
		o = opts[0]
		if o.Timeout == 0 {
			o.Timeout = time.Second
		}
	}
	go ackSubscribed(t, ch, sender)
	if err := ch.Subscribe(context.Background(), fn, o); err != nil {
		t.Fatal(err)
	}
}

func unsubscribedAck(name string) []byte {
	b, _ := json.Marshal(map[string]interface{}{
		"action": protocol.ActionUnsubscribed, "channel": name, "subscription_id": "sub-1",
	})
	return b
}

func subscribedAck(name string) []byte {
	b, _ := json.Marshal(map[string]interface{}{
		"action":          protocol.ActionSubscribed,
		"channel":         name,
		"subscription_id": "sub-1",
	})
	return b
}

func TestSocketSubscribeWaitsForSubscribed(t *testing.T) {
	sender := &mockSender{}
	log := logger.NewFactory("t", option.DefaultOption()).Create("Ch")
	ch := channel.NewSocketChannel("room", sender, log)

	go func() {
		time.Sleep(10 * time.Millisecond)
		ch.HandleIncoming(subscribedAck("room"))
	}()

	err := ch.Subscribe(context.Background(), func(m protocol.Message) {}, channel.SubscribeOptions{Timeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	last := sender.last()
	if int(last["action"].(float64)) != int(protocol.ActionSubscribe) {
		t.Fatalf("subscribe wire: %v", last)
	}
}

func TestSocketEventFilter(t *testing.T) {
	sender := &mockSender{}
	log := logger.NewFactory("t", option.DefaultOption()).Create("Ch")
	ch := channel.NewSocketChannel("room", sender, log)

	go func() {
		time.Sleep(5 * time.Millisecond)
		ch.HandleIncoming(subscribedAck("room"))
	}()

	var got []string
	err := ch.Subscribe(context.Background(), func(m protocol.Message) {
		got = append(got, m.Event)
	}, channel.SubscribeOptions{Event: "foo", Timeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}

	msg, _ := json.Marshal(map[string]interface{}{
		"action": protocol.ActionMessage, "channel": "room", "id": "1",
		"messages": []map[string]interface{}{
			{"event": "bar", "data": `"x"`},
			{"event": "foo", "data": `"y"`},
		},
	})
	ch.HandleIncoming(msg)

	if len(got) != 1 || got[0] != "foo" {
		t.Fatalf("got events %v", got)
	}
}

type errSender struct {
	mockSender
	err error
}

func (e *errSender) Send(data []byte) error {
	if e.err != nil {
		return e.err
	}
	return e.mockSender.Send(data)
}

func TestSocketPublishSuccess(t *testing.T) {
	sender := &mockSender{}
	ch := channel.NewSocketChannel("room", sender, logger.NewFactory("t", option.DefaultOption()).Create("Ch"))
	if err := ch.Publish(context.Background(), "hello", channel.PublishOptions{Event: "evt"}); err != nil {
		t.Fatal(err)
	}
	last := sender.last()
	if int(last["action"].(float64)) != int(protocol.ActionPublish) {
		t.Fatalf("action=%v", last["action"])
	}
}

func TestSocketPublishWhenDisconnected(t *testing.T) {
	ch := channel.NewSocketChannel("room", disconnectedSender{}, logger.NewFactory("t", option.DefaultOption()).Create("Ch"))
	err := ch.Publish(context.Background(), "x", channel.PublishOptions{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSocketUnsubscribeWire(t *testing.T) {
	sender := &mockSender{}
	ch := channel.NewSocketChannel("room", sender, logger.NewFactory("t", option.DefaultOption()).Create("Ch"))
	subscribeRoom(t, ch, sender, func(m protocol.Message) {})
	go func() {
		time.Sleep(5 * time.Millisecond)
		ch.HandleIncoming(unsubscribedAck("room"))
	}()
	if err := ch.Unsubscribe(context.Background(), channel.UnsubscribeOptions{Timeout: time.Second}); err != nil {
		t.Fatal(err)
	}
	if sender.countAction(protocol.ActionUnsubscribe) != 1 {
		t.Fatal("expected unsubscribe")
	}
}

func TestSocketNoDoubleSubscribeWire(t *testing.T) {
	sender := &mockSender{}
	ch := channel.NewSocketChannel("room", sender, logger.NewFactory("t", option.DefaultOption()).Create("Ch"))
	subscribeRoom(t, ch, sender, func(m protocol.Message) {})
	before := sender.countAction(protocol.ActionSubscribe)
	if err := ch.Subscribe(context.Background(), func(m protocol.Message) {}, channel.SubscribeOptions{Event: "evt2"}); err != nil {
		t.Fatal(err)
	}
	if sender.countAction(protocol.ActionSubscribe) != before {
		t.Fatal("duplicate subscribe wire")
	}
}

func TestSocketPauseBuffer(t *testing.T) {
	sender := &mockSender{}
	log := logger.NewFactory("t", option.DefaultOption()).Create("Ch")
	ch := channel.NewSocketChannel("room", sender, log)

	go func() {
		time.Sleep(5 * time.Millisecond)
		ch.HandleIncoming(subscribedAck("room"))
	}()

	var n int
	err := ch.Subscribe(context.Background(), func(m protocol.Message) { n++ }, channel.SubscribeOptions{Timeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}

	ch.Pause(true)
	payload, _ := json.Marshal(map[string]interface{}{
		"action": protocol.ActionMessage, "channel": "room", "id": "1",
		"messages": []map[string]interface{}{{"data": `"hi"`}},
	})
	ch.HandleIncoming(payload)
	if n != 0 {
		t.Fatal("expected buffered")
	}
	ch.Resume()
	if n != 1 {
		t.Fatalf("n=%d", n)
	}
}
