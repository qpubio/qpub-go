package connection

import (
	"context"
	"testing"

	"github.com/qpubio/qpub-go/auth"
	"github.com/qpubio/qpub-go/channel"
	"github.com/qpubio/qpub-go/events"
	"github.com/qpubio/qpub-go/internal/emitter"
	"github.com/qpubio/qpub-go/internal/logger"
	"github.com/qpubio/qpub-go/option"
	"github.com/qpubio/qpub-go/transport/ws"
)

func TestMalformedJSONEmitsFailedWithContext(t *testing.T) {
	conn := &Conn{events: emitter.New[any]()}
	var ctx string
	conn.events.On(events.ConnectionFailed, func(p any) {
		if pl, ok := p.(events.ConnectionFailedPayload); ok {
			ctx = pl.Context
		}
	})
	conn.handleMessage([]byte(`not-json`))
	if ctx != "message_processing" {
		t.Fatalf("context=%q", ctx)
	}
}

func TestTryReconnectStopsAtMaxAttempts(t *testing.T) {
	om := option.NewManager()
	om.Set(option.Option{
		AutoReconnect:        true,
		MaxReconnectAttempts: 2,
	})
	log := logger.NewFactory("t", om.Get()).Create("Conn")
	wsClient := ws.New(log)
	chMgr := channel.NewSocketManager(wsClient, log)
	conn := New(om, auth.NewManager(om, nil, log), wsClient, chMgr, log)
	conn.reconnectAttempts = 2
	before := conn.reconnectAttempts
	conn.tryReconnect(context.Background())
	if conn.reconnectAttempts != before {
		t.Fatalf("attempts changed from %d to %d", before, conn.reconnectAttempts)
	}
}
