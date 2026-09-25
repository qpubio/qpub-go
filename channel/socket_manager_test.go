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
