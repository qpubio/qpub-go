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

func TestSocketManagerRefCountAndRelease(t *testing.T) {
	log := logger.NewFactory("t", option.DefaultOption()).Create("Mgr")
	mgr := channel.NewSocketManager(&mockSender{}, log)

	_ = mgr.Get("room")
	_ = mgr.Get("room")
	mgr.Release("room")
	if !mgr.Has("room") {
		t.Fatal("channel should still exist")
	}

	mgr.Release("room")
	if mgr.Has("room") {
		t.Fatal("channel should be removed without callback")
	}
}

func TestSocketManagerReleaseKeepsChannelWithCallback(t *testing.T) {
	log := logger.NewFactory("t", option.DefaultOption()).Create("Mgr")
	mgr := channel.NewSocketManager(&mockSender{}, log)

	ch := mgr.Get("room")
	go func() {
		time.Sleep(5 * time.Millisecond)
		ch.HandleIncoming(subscribedAck("room"))
	}()
	if err := ch.Subscribe(context.Background(), func(m protocol.Message) {}, channel.SubscribeOptions{Timeout: time.Second}); err != nil {
		t.Fatal(err)
	}
	mgr.Release("room")
	if !mgr.Has("room") {
		t.Fatal("expected channel kept for resubscribe")
	}
}

func TestSocketManagerCreateGetHasRemove(t *testing.T) {
	sender := &mockSender{}
	mgr := channel.NewSocketManager(sender, logger.NewFactory("t", option.DefaultOption()).Create("Mgr"))
	if mgr.Has("a") {
		t.Fatal("expected missing")
	}
	ch := mgr.Get("a")
	if !mgr.Has("a") || ch.Name() != "a" {
		t.Fatal("create/get")
	}
	ch2 := mgr.Get("a")
	if ch2 != ch {
		t.Fatal("expected same instance")
	}
	mgr.Remove("a")
	if mgr.Has("a") {
		t.Fatal("expected removed")
	}
	mgr.Remove("missing") // no panic
}

func TestSocketManagerAllReturnsChannels(t *testing.T) {
	mgr := channel.NewSocketManager(&mockSender{}, logger.NewFactory("t", option.DefaultOption()).Create("Mgr"))
	_ = mgr.Get("x")
	_ = mgr.Get("y")
	all := mgr.All()
	if len(all) != 2 {
		t.Fatalf("len=%d", len(all))
	}
}

func TestSocketManagerResetClearsChannels(t *testing.T) {
	mgr := channel.NewSocketManager(&mockSender{}, logger.NewFactory("t", option.DefaultOption()).Create("Mgr"))
	_ = mgr.Get("room")
	mgr.Reset()
	if mgr.Has("room") || len(mgr.All()) != 0 {
		t.Fatal("expected empty after reset")
	}
}

func TestSocketManagerRefCountIncrementOnGet(t *testing.T) {
	mgr := channel.NewSocketManager(&mockSender{}, logger.NewFactory("t", option.DefaultOption()).Create("Mgr"))
	_ = mgr.Get("r")
	_ = mgr.Get("r")
	mgr.Release("r")
	if !mgr.Has("r") {
		t.Fatal("expected channel until second release")
	}
}

func TestSocketManagerReleaseUnknownChannel(t *testing.T) {
	mgr := channel.NewSocketManager(&mockSender{}, logger.NewFactory("t", option.DefaultOption()).Create("Mgr"))
	mgr.Release("nope")
}

func TestSocketManagerResubscribeOnlyWithCallbacks(t *testing.T) {
	sender := &mockSender{}
	mgr := channel.NewSocketManager(sender, logger.NewFactory("t", option.DefaultOption()).Create("Mgr"))
	withCB := mgr.Get("with")
	without := mgr.Get("without")

	go func() {
		time.Sleep(5 * time.Millisecond)
		withCB.HandleIncoming(subscribedAck("with"))
	}()
	_ = withCB.Subscribe(context.Background(), func(m protocol.Message) {}, channel.SubscribeOptions{Timeout: time.Second})

	mgr.PendingSubscribeAllChannels()
	mgr.ResubscribeAfterReconnect(context.Background())

	if sender.countAction(protocol.ActionSubscribe) < 2 {
		t.Fatalf("subscribe wire count=%d", sender.countAction(protocol.ActionSubscribe))
	}
	_ = without
}

func TestSocketManagerResubscribeSkipsWhenNoCallbacks(t *testing.T) {
	sender := &mockSender{}
	mgr := channel.NewSocketManager(sender, logger.NewFactory("t", option.DefaultOption()).Create("Mgr"))
	_ = mgr.Get("empty")
	before := sender.countAction(protocol.ActionSubscribe)
	mgr.ResubscribeAllChannels(context.Background())
	if sender.countAction(protocol.ActionSubscribe) != before {
		t.Fatal("expected no resubscribe without callbacks")
	}
}
