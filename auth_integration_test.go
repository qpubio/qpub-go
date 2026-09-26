package qpub_test

import (
	"testing"

	"github.com/qpubio/qpub-go"
	"github.com/qpubio/qpub-go/option"
)

func TestComponentsConstructTogether(t *testing.T) {
	socket := qpub.NewSocket(qpub.WithAPIKey("k:s"), option.WithAutoConnect(false))
	defer socket.Reset()

	if socket.Auth == nil || socket.Connection == nil || socket.Channels == nil {
		t.Fatal("missing socket services")
	}
	if !socket.Auth.ShouldAutoAuthenticate() {
		t.Fatal("expected default auto authenticate")
	}
	if socket.Connection.IsConnected() {
		t.Fatal("expected disconnected initially")
	}
	if socket.Channels.Has("x") {
		t.Fatal("expected no channels")
	}
	_ = socket.Channels.Get("x")
	if !socket.Channels.Has("x") {
		t.Fatal("expected channel after get")
	}
}

func TestRestComponentsConstructTogether(t *testing.T) {
	rest := qpub.NewRest(qpub.WithAPIKey("k:s"))
	defer rest.Reset()
	if rest.Auth == nil || rest.Channels == nil || rest.Queues == nil {
		t.Fatal("missing rest services")
	}
}
