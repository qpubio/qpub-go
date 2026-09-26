package testing_test

import (
	"context"
	"errors"
	"testing"

	qtesting "github.com/qpubio/qpub-go/testing"
	"github.com/qpubio/qpub-go/option"
)

func TestMockHTTPRecordsAndOverrides(t *testing.T) {
	m := &qtesting.MockHTTP{}
	m.PostFn = func(ctx context.Context, url string, body interface{}, headers map[string]string) ([]byte, int, error) {
		return []byte(`ok`), 200, nil
	}
	b, code, err := m.Post(context.Background(), "http://example.com", nil, nil)
	if err != nil || code != 200 || string(b) != "ok" {
		t.Fatalf("post: %s %d %v", b, code, err)
	}
	if len(m.Posts) != 1 || m.Posts[0].URL != "http://example.com" {
		t.Fatalf("posts=%+v", m.Posts)
	}
}

func TestMockWSSendAndConnected(t *testing.T) {
	ws := qtesting.NewMockWS()
	if !ws.IsConnected() {
		t.Fatal("expected connected")
	}
	if err := ws.Send([]byte(`{"a":1}`)); err != nil {
		t.Fatal(err)
	}
	if len(ws.Sent) != 1 {
		t.Fatal("expected sent frame")
	}
	ws.Connected = false
	if ws.IsConnected() {
		t.Fatal("expected disconnected flag")
	}
}

func TestNewTestRestAndSocketDistinct(t *testing.T) {
	r1, _ := qtesting.NewTestRest()
	r2, _ := qtesting.NewTestRest()
	defer r1.Reset()
	defer r2.Reset()
	if r1.GetInstanceID() == r2.GetInstanceID() {
		t.Fatal("expected distinct rest ids")
	}

	s1 := qtesting.NewTestSocket(option.WithAutoConnect(false))
	s2 := qtesting.NewTestSocket(option.WithAutoConnect(false))
	defer s1.Reset()
	defer s2.Reset()
	if s1.GetInstanceID() == s2.GetInstanceID() {
		t.Fatal("expected distinct socket ids")
	}
}

func TestMockIsolation(t *testing.T) {
	a := &qtesting.MockHTTP{}
	b := &qtesting.MockHTTP{}
	a.PostFn = func(ctx context.Context, url string, body interface{}, headers map[string]string) ([]byte, int, error) {
		return nil, 0, errors.New("a")
	}
	_, _, errA := a.Post(context.Background(), "u", nil, nil)
	_, _, errB := b.Post(context.Background(), "u", nil, nil)
	if errA == nil || errB != nil {
		t.Fatalf("errA=%v errB=%v", errA, errB)
	}
}
