package qpub_test

import (
	"testing"

	"github.com/qpubio/qpub-go/option"
	"github.com/qpubio/qpub-go"
)

func TestSocketInstanceIDStableAcrossServices(t *testing.T) {
	socket := qpub.NewSocket(qpub.WithAPIKey("k:s"), option.WithAutoConnect(false))
	defer socket.Reset()

	id := socket.GetInstanceID()
	if id == "" {
		t.Fatal("empty instance id")
	}
	if socket.GetInstanceID() != id {
		t.Fatal("id changed on second call")
	}
	_ = socket.Auth
	if socket.GetInstanceID() != id {
		t.Fatal("id changed after auth access")
	}
	_ = socket.Connection
	if socket.GetInstanceID() != id {
		t.Fatal("id changed after connection access")
	}
	_ = socket.Channels
	if socket.GetInstanceID() != id {
		t.Fatal("id changed after channels access")
	}
}

func TestSocketInstanceIDAfterOptionSetAndReset(t *testing.T) {
	socket := qpub.NewSocket(qpub.WithAPIKey("k:s"), option.WithAutoConnect(false))
	defer socket.Reset()

	id := socket.GetInstanceID()
	socket.OptionManager.Set(option.Option{APIKey: "new-key"})
	if socket.GetInstanceID() != id {
		t.Fatal("id changed after option set")
	}
	socket.Reset()
	if socket.GetInstanceID() != id {
		t.Fatal("id changed after reset")
	}
}

func TestRestInstanceIDStableAcrossServices(t *testing.T) {
	rest := qpub.NewRest(qpub.WithAPIKey("k:s"))
	defer rest.Reset()

	id := rest.GetInstanceID()
	if id == "" {
		t.Fatal("empty instance id")
	}
	_ = rest.Channels
	_ = rest.Queues
	_ = rest.Auth
	if rest.GetInstanceID() != id {
		t.Fatal("id changed after service access")
	}
}

func TestRestInstanceIDAfterOptionSetAndReset(t *testing.T) {
	rest := qpub.NewRest(qpub.WithAPIKey("k:s"))
	defer rest.Reset()

	id := rest.GetInstanceID()
	rest.OptionManager.Set(option.Option{APIKey: "new-key"})
	if rest.GetInstanceID() != id {
		t.Fatal("id changed after option set")
	}
	rest.Reset()
	if rest.GetInstanceID() != id {
		t.Fatal("id changed after reset")
	}
}

func TestMultipleInstancesUniqueIDs(t *testing.T) {
	a := qpub.NewSocket(option.WithAutoConnect(false))
	b := qpub.NewSocket(option.WithAutoConnect(false))
	defer a.Reset()
	defer b.Reset()
	if a.GetInstanceID() == b.GetInstanceID() {
		t.Fatal("expected distinct socket instance ids")
	}

	r1 := qpub.NewRest()
	r2 := qpub.NewRest()
	defer r1.Reset()
	defer r2.Reset()
	if r1.GetInstanceID() == r2.GetInstanceID() {
		t.Fatal("expected distinct rest instance ids")
	}
}
