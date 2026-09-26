package channel_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/qpubio/qpub-go/channel"
	"github.com/qpubio/qpub-go/internal/logger"
	"github.com/qpubio/qpub-go/option"
	"github.com/qpubio/qpub-go/protocol"
)

func TestSocketUnsubscribeSpecificEventHandler(t *testing.T) {
	sender := &mockSender{}
	ch := channel.NewSocketChannel("room", sender, logger.NewFactory("t", option.DefaultOption()).Create("Ch"))

	go func() {
		time.Sleep(5 * time.Millisecond)
		ch.HandleIncoming(subscribedAck("room"))
	}()
	h1 := func(m protocol.Message) {}
	h2 := func(m protocol.Message) {}
	if err := ch.Subscribe(context.Background(), h1, channel.SubscribeOptions{Event: "e1", Timeout: time.Second}); err != nil {
		t.Fatal(err)
	}
	_ = ch.Subscribe(context.Background(), h2, channel.SubscribeOptions{Event: "e2"})

	go func() {
		time.Sleep(5 * time.Millisecond)
		ch.HandleIncoming(unsubscribedAck("room"))
	}()
	if err := ch.Unsubscribe(context.Background(), channel.UnsubscribeOptions{Event: "e1", Handler: h1, Timeout: time.Second}); err != nil {
		t.Fatal(err)
	}
	if sender.countAction(protocol.ActionUnsubscribe) != 0 {
		t.Fatal("should not unsub wire while other event handlers remain")
	}
}

func TestSocketUnsubscribeAllEventCallbacksTriggersWire(t *testing.T) {
	sender := &mockSender{}
	ch := channel.NewSocketChannel("room", sender, logger.NewFactory("t", option.DefaultOption()).Create("Ch"))
	subscribeRoom(t, ch, sender, func(m protocol.Message) {}, channel.SubscribeOptions{Event: "only"})

	go func() {
		time.Sleep(5 * time.Millisecond)
		ch.HandleIncoming(unsubscribedAck("room"))
	}()
	if err := ch.Unsubscribe(context.Background(), channel.UnsubscribeOptions{Event: "only", Timeout: time.Second}); err != nil {
		t.Fatal(err)
	}
	if sender.countAction(protocol.ActionUnsubscribe) != 1 {
		t.Fatal("expected unsubscribe wire")
	}
}

func TestSocketEventHandlerIgnoresMessagesWithoutEvent(t *testing.T) {
	sender := &mockSender{}
	ch := channel.NewSocketChannel("room", sender, logger.NewFactory("t", option.DefaultOption()).Create("Ch"))
	var n int
	subscribeRoom(t, ch, sender, func(m protocol.Message) { n++ }, channel.SubscribeOptions{Event: "evt"})

	payload, _ := json.Marshal(map[string]interface{}{
		"action": protocol.ActionMessage, "channel": "room", "id": "1",
		"messages": []map[string]interface{}{{"data": `"x"`}},
	})
	ch.HandleIncoming(payload)
	if n != 0 {
		t.Fatalf("n=%d", n)
	}
}
