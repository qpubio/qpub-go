// Package testing provides mocks and helpers for unit tests (qpub-js testing exports).
package testing

import (
	"context"
	"sync"

	"github.com/qpubio/qpub-go/auth"
	"github.com/qpubio/qpub-go/internal/logger"
	"github.com/qpubio/qpub-go/option"
	"github.com/qpubio/qpub-go/qpub"
	"github.com/qpubio/qpub-go/transport/httpclient"
)

// MockHTTP records POST calls.
type MockHTTP struct {
	mu    sync.Mutex
	Posts []MockPost
	PostFn func(ctx context.Context, url string, body interface{}, headers map[string]string) ([]byte, int, error)
}

type MockPost struct {
	URL     string
	Body    interface{}
	Headers map[string]string
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

// NewTestRest builds a Rest client whose auth layer uses MockHTTP.
func NewTestRest(funcs ...option.OptionFunc) (*qpub.Rest, *MockHTTP) {
	mock := &MockHTTP{}
	rest := qpub.NewRest(funcs...)
	log := logger.NewFactory("test_rest", rest.OptionManager.Get()).Create("AuthManager")
	rest.Auth = auth.NewManager(rest.OptionManager, mock, log)
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
