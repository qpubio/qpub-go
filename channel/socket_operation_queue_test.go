package channel_test

import (
	"context"
	"testing"
	"time"

	"github.com/qpubio/qpub-go/channel"
	"github.com/qpubio/qpub-go/internal/logger"
	"github.com/qpubio/qpub-go/option"
	"github.com/qpubio/qpub-go/protocol"
)

func TestOperationQueueSubscribeWhileUnsubscribePending(t *testing.T) {
	sender := &mockSender{}
	ch := channel.NewSocketChannel("room", sender, logger.NewFactory("t", option.DefaultOption()).Create("Ch"))
	subscribeRoom(t, ch, sender, func(m protocol.Message) {})

	// Start unsubscribe without waiting for ack.
	go func() {
		_ = ch.Unsubscribe(context.Background(), channel.UnsubscribeOptions{Timeout: 5 * time.Second})
	}()
	time.Sleep(10 * time.Millisecond)

	h2 := func(m protocol.Message) {}
	if err := ch.Subscribe(context.Background(), h2, channel.SubscribeOptions{}); err != nil {
		t.Fatal(err)
	}

	ch.HandleIncoming(unsubscribedAck("room"))
	time.Sleep(20 * time.Millisecond)
	if sender.countAction(protocol.ActionSubscribe) < 2 {
		t.Fatalf("subscribe sends=%d", sender.countAction(protocol.ActionSubscribe))
	}
}

func TestOperationQueueUnsubscribeWhileSubscribePending(t *testing.T) {
	sender := &mockSender{}
	ch := channel.NewSocketChannel("room", sender, logger.NewFactory("t", option.DefaultOption()).Create("Ch"))

	subDone := make(chan error, 1)
	go func() {
		subDone <- ch.Subscribe(context.Background(), func(m protocol.Message) {}, channel.SubscribeOptions{Timeout: 2 * time.Second})
	}()

	deadline := time.After(2 * time.Second)
	for sender.countAction(protocol.ActionSubscribe) == 0 {
		select {
		case <-deadline:
			t.Fatal("subscribe wire not sent")
		default:
			time.Sleep(time.Millisecond)
		}
	}
	if err := ch.Unsubscribe(context.Background()); err != nil {
		t.Fatal(err)
	}
	ch.HandleIncoming(subscribedAck("room"))
	if err := <-subDone; err != nil {
		t.Fatal(err)
	}
	time.Sleep(20 * time.Millisecond)
	if sender.countAction(protocol.ActionUnsubscribe) != 1 {
		t.Fatalf("unsub=%d", sender.countAction(protocol.ActionUnsubscribe))
	}
}

func TestOperationQueueResetClearsQueue(t *testing.T) {
	sender := &mockSender{}
	ch := channel.NewSocketChannel("room", sender, logger.NewFactory("t", option.DefaultOption()).Create("Ch"))
	go func() {
		_ = ch.Unsubscribe(context.Background(), channel.UnsubscribeOptions{Timeout: 5 * time.Second})
	}()
	time.Sleep(5 * time.Millisecond)
	_ = ch.Subscribe(context.Background(), func(m protocol.Message) {}, channel.SubscribeOptions{})
	ch.Reset()
	ch.HandleIncoming(unsubscribedAck("room"))
	time.Sleep(20 * time.Millisecond)
	if sender.countAction(protocol.ActionSubscribe) > 1 {
		t.Fatal("queue should not process after reset")
	}
}

func TestOperationQueueNoQueueWhenIdle(t *testing.T) {
	sender := &mockSender{}
	ch := channel.NewSocketChannel("room", sender, logger.NewFactory("t", option.DefaultOption()).Create("Ch"))
	subscribeRoom(t, ch, sender, func(m protocol.Message) {})
	before := sender.countAction(protocol.ActionSubscribe)
	_ = ch.Subscribe(context.Background(), func(m protocol.Message) {}, channel.SubscribeOptions{Event: "e2"})
	if sender.countAction(protocol.ActionSubscribe) != before {
		t.Fatal("expected no extra subscribe on wire")
	}
}
