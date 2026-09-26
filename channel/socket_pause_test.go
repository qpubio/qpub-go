package channel_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/qpubio/qpub-go/channel"
	"github.com/qpubio/qpub-go/events"
	"github.com/qpubio/qpub-go/internal/logger"
	"github.com/qpubio/qpub-go/option"
	"github.com/qpubio/qpub-go/protocol"
)

func messagePayload(data string) []byte {
	b, _ := json.Marshal(map[string]interface{}{
		"action": protocol.ActionMessage, "channel": "room", "id": "1",
		"messages": []map[string]interface{}{{"data": data}},
	})
	return b
}

func TestSocketPauseResumeEvents(t *testing.T) {
	sender := &mockSender{}
	ch := channel.NewSocketChannel("room", sender, logger.NewFactory("t", option.DefaultOption()).Create("Ch"))
	paused := make(chan struct{}, 1)
	resumed := make(chan struct{}, 1)
	ch.On(events.ChannelPaused, func(any) { paused <- struct{}{} })
	ch.On(events.ChannelResumed, func(any) { resumed <- struct{}{} })
	ch.Pause(true)
	select {
	case <-paused:
	case <-time.After(time.Second):
		t.Fatal("paused event")
	}
	ch.Resume()
	select {
	case <-resumed:
	case <-time.After(time.Second):
		t.Fatal("resumed event")
	}
}

func TestSocketPauseDropWhenBufferDisabled(t *testing.T) {
	sender := &mockSender{}
	ch := channel.NewSocketChannel("room", sender, logger.NewFactory("t", option.DefaultOption()).Create("Ch"))
	var n int
	subscribeRoom(t, ch, sender, func(m protocol.Message) { n++ })
	ch.Pause(false)
	ch.HandleIncoming(messagePayload(`"dropped"`))
	ch.Resume()
	if n != 0 {
		t.Fatalf("n=%d", n)
	}
}

func TestSocketPauseBufferMultipleAndClear(t *testing.T) {
	sender := &mockSender{}
	ch := channel.NewSocketChannel("room", sender, logger.NewFactory("t", option.DefaultOption()).Create("Ch"))
	var n int
	subscribeRoom(t, ch, sender, func(m protocol.Message) { n++ })
	ch.Pause(true)
	ch.HandleIncoming(messagePayload(`"1"`))
	ch.HandleIncoming(messagePayload(`"2"`))
	ch.ClearBufferedMessages()
	ch.Resume()
	if n != 0 {
		t.Fatalf("n=%d", n)
	}
}

func TestSocketPauseNoOpWhenAlreadyPaused(t *testing.T) {
	sender := &mockSender{}
	ch := channel.NewSocketChannel("room", sender, logger.NewFactory("t", option.DefaultOption()).Create("Ch"))
	ch.Pause(true)
	ch.Pause(true)
	if !ch.IsPaused() {
		t.Fatal("expected paused")
	}
}

func TestSocketResumeNoOpWhenNotPaused(t *testing.T) {
	sender := &mockSender{}
	ch := channel.NewSocketChannel("room", sender, logger.NewFactory("t", option.DefaultOption()).Create("Ch"))
	ch.Resume()
	if ch.IsPaused() {
		t.Fatal("expected not paused")
	}
}

func TestSocketResetClearsPauseState(t *testing.T) {
	sender := &mockSender{}
	ch := channel.NewSocketChannel("room", sender, logger.NewFactory("t", option.DefaultOption()).Create("Ch"))
	subscribeRoom(t, ch, sender, func(m protocol.Message) {})
	ch.Pause(true)
	ch.Reset()
	if ch.IsPaused() {
		t.Fatal("pause should reset")
	}
}

func TestSocketDeliverWhenNotPaused(t *testing.T) {
	sender := &mockSender{}
	ch := channel.NewSocketChannel("room", sender, logger.NewFactory("t", option.DefaultOption()).Create("Ch"))
	var n int
	subscribeRoom(t, ch, sender, func(m protocol.Message) { n++ })
	ch.HandleIncoming(messagePayload(`"hi"`))
	if n != 1 {
		t.Fatalf("n=%d", n)
	}
}
