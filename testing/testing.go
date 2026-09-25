// Package testing provides mocks and helpers for unit tests (qpub-js testing exports).
package testing

import (
	"context"
	"sync"

	"github.com/qpubio/qpub-go/auth"
	"github.com/qpubio/qpub-go/channel"
	"github.com/qpubio/qpub-go/internal/logger"
	"github.com/qpubio/qpub-go/internal/port"
	"github.com/qpubio/qpub-go/option"
	"github.com/qpubio/qpub-go/qpub"
	"github.com/qpubio/qpub-go/queue"
	"github.com/qpubio/qpub-go/transport/httpclient"
)

var _ port.HTTPClient = (*MockHTTP)(nil)

// MockHTTP records HTTP calls for assertions.
type MockHTTP struct {
	mu sync.Mutex

	Posts []MockPost
	Gets  []MockGet
	Puts  []MockPut
	Dels  []MockDelete

	PostFn   func(ctx context.Context, url string, body interface{}, headers map[string]string) ([]byte, int, error)
	GetFn    func(ctx context.Context, url string, headers map[string]string) ([]byte, int, error)
	PutFn    func(ctx context.Context, url string, body interface{}, headers map[string]string) ([]byte, int, error)
	DeleteFn func(ctx context.Context, url string, headers map[string]string) ([]byte, int, error)
}

type MockPost struct {
	URL     string
	Body    interface{}
	Headers map[string]string
}

type MockGet struct {
	URL     string
	Headers map[string]string
}

type MockPut struct {
	URL     string
	Body    interface{}
	Headers map[string]string
}

type MockDelete struct {
	URL     string
	Headers map[string]string
}

func (m *MockHTTP) Get(ctx context.Context, url string, headers map[string]string) ([]byte, int, error) {
	m.mu.Lock()
	m.Gets = append(m.Gets, MockGet{URL: url, Headers: headers})
	fn := m.GetFn
	m.mu.Unlock()
	if fn != nil {
		return fn(ctx, url, headers)
	}
	return nil, 200, nil
}

func (m *MockHTTP) Post(ctx context.Context, url string, body interface{}, headers map[string]string) ([]byte, int, error) {
	m.mu.Lock()
	m.Posts = append(m.Posts, MockPost{URL: url, Body: body, Headers: headers})
	fn := m.PostFn
	m.mu.Unlock()
	if fn != nil {
		return fn(ctx, url, body, headers)
	}
	return nil, 200, nil
}

func (m *MockHTTP) Put(ctx context.Context, url string, body interface{}, headers map[string]string) ([]byte, int, error) {
	m.mu.Lock()
	m.Puts = append(m.Puts, MockPut{URL: url, Body: body, Headers: headers})
	fn := m.PutFn
	m.mu.Unlock()
	if fn != nil {
		return fn(ctx, url, body, headers)
	}
	return nil, 200, nil
}

func (m *MockHTTP) Delete(ctx context.Context, url string, headers map[string]string) ([]byte, int, error) {
	m.mu.Lock()
	m.Dels = append(m.Dels, MockDelete{URL: url, Headers: headers})
	fn := m.DeleteFn
	m.mu.Unlock()
	if fn != nil {
		return fn(ctx, url, headers)
	}
	return nil, 200, nil
}

// NewTestRest builds a Rest client whose auth, channels, and queues share MockHTTP.
func NewTestRest(funcs ...option.OptionFunc) (*qpub.Rest, *MockHTTP) {
	mock := &MockHTTP{}
	rest := qpub.NewRest(funcs...)
	logFactory := logger.NewFactory("test_rest", rest.OptionManager.Get())
	rest.Auth = auth.NewManager(rest.OptionManager, mock, logFactory.Create("AuthManager"))
	rest.Channels = channel.NewRestManager(mock, rest.Auth, rest.OptionManager, logFactory.Create("RestChannelManager"))
	rest.Queues = queue.NewManager(mock, rest.Auth, rest.OptionManager, logFactory.Create("RestQueueManager"), rest.GetInstanceID())
	return rest, mock
}

// NewTestSocket builds a Socket with auto-connect disabled by default.
func NewTestSocket(funcs ...option.OptionFunc) *qpub.Socket {
	all := append([]option.OptionFunc{option.WithAutoConnect(false)}, funcs...)
	return qpub.NewSocket(all...)
}

// NewSilentHTTPClient returns real HTTP client for integration tests.
func NewSilentHTTPClient() *httpclient.Client {
	return httpclient.New()
}
