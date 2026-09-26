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

func TestSocketMessageTransformAndMultiID(t *testing.T) {
	sender := &mockSender{}
	log := logger.NewFactory("t", option.DefaultOption()).Create("Ch")
	ch := channel.NewSocketChannel("room", sender, log)

	var got []string
	subscribeRoom(t, ch, sender, func(m protocol.Message) {
		got = append(got, m.ID)
	})

	msg, _ := json.Marshal(map[string]interface{}{
		"action": protocol.ActionMessage, "channel": "room", "id": "base",
		"timestamp": "2024-01-01T00:00:00Z",
		"messages": []map[string]interface{}{
			{"data": `"a"`},
			{"data": `"b"`},
		},
	})
	ch.HandleIncoming(msg)

	if len(got) != 2 || got[0] != "base-0" || got[1] != "base-1" {
		t.Fatalf("ids=%v", got)
	}
}

func TestSocketSingleMessageKeepsID(t *testing.T) {
	sender := &mockSender{}
	ch := channel.NewSocketChannel("room", sender, logger.NewFactory("t", option.DefaultOption()).Create("Ch"))
	var id string
	subscribeRoom(t, ch, sender, func(m protocol.Message) { id = m.ID })

	payload, _ := json.Marshal(map[string]interface{}{
		"action": protocol.ActionMessage, "channel": "room", "id": "only",
		"messages": []map[string]interface{}{{"data": `"x"`}},
	})
	ch.HandleIncoming(payload)
	if id != "only" {
		t.Fatalf("id=%q", id)
	}
}

func TestSocketIgnoresOtherChannelMessages(t *testing.T) {
	sender := &mockSender{}
	ch := channel.NewSocketChannel("room", sender, logger.NewFactory("t", option.DefaultOption()).Create("Ch"))
	var n int
	subscribeRoom(t, ch, sender, func(m protocol.Message) { n++ })

	other, _ := json.Marshal(map[string]interface{}{
		"action": protocol.ActionMessage, "channel": "other", "id": "1",
		"messages": []map[string]interface{}{{"data": `"x"`}},
	})
	ch.HandleIncoming(other)
	if n != 0 {
		t.Fatal("expected ignore")
	}
}

func TestSocketMalformedJSONIgnored(t *testing.T) {
	sender := &mockSender{}
	ch := channel.NewSocketChannel("room", sender, logger.NewFactory("t", option.DefaultOption()).Create("Ch"))
	ch.HandleIncoming([]byte(`{not json`))
}

func TestSocketErrorActionEmitsFailed(t *testing.T) {
	sender := &mockSender{}
	ch := channel.NewSocketChannel("room", sender, logger.NewFactory("t", option.DefaultOption()).Create("Ch"))
	done := make(chan struct{}, 1)
	ch.On(events.ChannelFailed, func(any) { done <- struct{}{} })
	raw, _ := json.Marshal(map[string]interface{}{
		"action": protocol.ActionError, "channel": "room",
		"error": map[string]interface{}{"message": "boom"},
	})
	ch.HandleIncoming(raw)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
}
