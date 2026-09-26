package qpub_test

import (
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/qpubio/qpub-go"
	"github.com/qpubio/qpub-go/option"
)

var uuidSegment = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func uuidAfterPrefix(id, prefix string) string {
	return strings.TrimPrefix(id, prefix)
}

func assertSocketIDFormat(t *testing.T, id string) {
	t.Helper()
	if !strings.HasPrefix(id, "socket_") {
		t.Fatalf("expected socket_ prefix, got %q", id)
	}
	u := uuidAfterPrefix(id, "socket_")
	if !uuidSegment.MatchString(u) {
		t.Fatalf("expected UUIDv7 after prefix, got %q", u)
	}
}

func assertRestIDFormat(t *testing.T, id string) {
	t.Helper()
	if !strings.HasPrefix(id, "rest_") {
		t.Fatalf("expected rest_ prefix, got %q", id)
	}
	u := uuidAfterPrefix(id, "rest_")
	if !uuidSegment.MatchString(u) {
		t.Fatalf("expected UUIDv7 after prefix, got %q", u)
	}
}

func TestSocketInstanceIDStableAcrossServices(t *testing.T) {
	socket := qpub.NewSocket(qpub.WithAPIKey("k:s"), option.WithAutoConnect(false))
	defer socket.Reset()

	id := socket.GetInstanceID()
	if id == "" {
		t.Fatal("empty instance id")
	}
	assertSocketIDFormat(t, id)
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
	assertRestIDFormat(t, id)
	_ = rest.Channels
	_ = rest.Queues
	_ = rest.Auth
	_ = rest.OptionManager
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

func TestCrossTypeInstanceIDUniqueness(t *testing.T) {
	s1 := qpub.NewSocket(option.WithAutoConnect(false))
	s2 := qpub.NewSocket(option.WithAutoConnect(false))
	r1 := qpub.NewRest()
	r2 := qpub.NewRest()
	defer s1.Reset()
	defer s2.Reset()
	defer r1.Reset()
	defer r2.Reset()

	ids := []string{s1.GetInstanceID(), s2.GetInstanceID(), r1.GetInstanceID(), r2.GetInstanceID()}
	seen := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			t.Fatalf("duplicate id %q", id)
		}
		seen[id] = struct{}{}
	}
	assertSocketIDFormat(t, ids[0])
	assertSocketIDFormat(t, ids[1])
	assertRestIDFormat(t, ids[2])
	assertRestIDFormat(t, ids[3])
}

func TestInstanceIDTimeOrderedUUIDv7(t *testing.T) {
	s1 := qpub.NewSocket(option.WithAutoConnect(false))
	time.Sleep(2 * time.Millisecond)
	s2 := qpub.NewSocket(option.WithAutoConnect(false))
	time.Sleep(2 * time.Millisecond)
	s3 := qpub.NewSocket(option.WithAutoConnect(false))
	defer s1.Reset()
	defer s2.Reset()
	defer s3.Reset()

	u1 := uuidAfterPrefix(s1.GetInstanceID(), "socket_")
	u2 := uuidAfterPrefix(s2.GetInstanceID(), "socket_")
	u3 := uuidAfterPrefix(s3.GetInstanceID(), "socket_")
	if u1 == u2 || u2 == u3 {
		t.Fatal("expected distinct UUID segments")
	}
	if strings.Compare(u1, u2) >= 0 || strings.Compare(u2, u3) >= 0 {
		t.Fatalf("expected time-ordered UUIDv7: %q %q %q", u1, u2, u3)
	}
}

func TestRapidInstanceCreationUnique(t *testing.T) {
	const n = 100
	ids := make(map[string]struct{}, n)
	var sockets []*qpub.Socket
	for i := 0; i < n; i++ {
		s := qpub.NewSocket(option.WithAutoConnect(false))
		sockets = append(sockets, s)
		id := s.GetInstanceID()
		if _, ok := ids[id]; ok {
			t.Fatalf("duplicate id at %d", i)
		}
		ids[id] = struct{}{}
	}
	for _, s := range sockets {
		s.Reset()
	}
}

func TestTrackMultipleSocketsByID(t *testing.T) {
	registry := make(map[string]*qpub.Socket)
	var sockets []*qpub.Socket
	for i := 0; i < 5; i++ {
		s := qpub.NewSocket(option.WithAutoConnect(false), qpub.WithAPIKey("k:s"))
		sockets = append(sockets, s)
		registry[s.GetInstanceID()] = s
	}
	defer func() {
		for _, s := range sockets {
			s.Reset()
		}
	}()
	if len(registry) != 5 {
		t.Fatalf("registry size=%d", len(registry))
	}
	for _, s := range sockets {
		if registry[s.GetInstanceID()] != s {
			t.Fatal("registry lookup failed")
		}
	}
}

func TestTrackMixedSocketAndRestByID(t *testing.T) {
	s1 := qpub.NewSocket(option.WithAutoConnect(false))
	r1 := qpub.NewRest()
	s2 := qpub.NewSocket(option.WithAutoConnect(false))
	r2 := qpub.NewRest()
	defer s1.Reset()
	defer s2.Reset()
	defer r1.Reset()
	defer r2.Reset()

	type entry struct {
		kind string
	}
	reg := map[string]entry{
		s1.GetInstanceID(): {"socket"},
		r1.GetInstanceID(): {"rest"},
		s2.GetInstanceID(): {"socket"},
		r2.GetInstanceID(): {"rest"},
	}
	if len(reg) != 4 {
		t.Fatal("expected 4 entries")
	}
	var socketN, restN int
	for id := range reg {
		switch {
		case strings.HasPrefix(id, "socket_"):
			socketN++
		case strings.HasPrefix(id, "rest_"):
			restN++
		}
	}
	if socketN != 2 || restN != 2 {
		t.Fatalf("socket=%d rest=%d", socketN, restN)
	}
}

func TestInstanceBasedErrorTracking(t *testing.T) {
	s := qpub.NewSocket(option.WithAutoConnect(false))
	r := qpub.NewRest()
	defer s.Reset()
	defer r.Reset()

	type errRec struct {
		instanceID string
		kind       string
	}
	errs := []errRec{
		{s.GetInstanceID(), "socket"},
		{r.GetInstanceID(), "rest"},
	}
	if !strings.HasPrefix(errs[0].instanceID, "socket_") {
		t.Fatal(errs[0].instanceID)
	}
	if !strings.HasPrefix(errs[1].instanceID, "rest_") {
		t.Fatal(errs[1].instanceID)
	}
}

func TestDifferentConfigurationsUniqueIDs(t *testing.T) {
	s1 := qpub.NewSocket(option.WithAutoConnect(false), qpub.WithAPIKey("k1"))
	s2 := qpub.NewSocket(qpub.WithAPIKey("k2"))
	defer s1.Reset()
	defer s2.Reset()
	if s1.GetInstanceID() == s2.GetInstanceID() {
		t.Fatal("expected different ids")
	}

	r1 := qpub.NewRest(qpub.WithAPIKey("a"))
	r2 := qpub.NewRest(qpub.WithAPIKey("b"))
	defer r1.Reset()
	defer r2.Reset()
	if r1.GetInstanceID() == r2.GetInstanceID() {
		t.Fatal("expected different rest ids")
	}
}
