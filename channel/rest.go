package channel

import (
	"context"
	"encoding/json"

	"github.com/qpubio/qpub-go/auth"
	"github.com/qpubio/qpub-go/internal/emitter"
	"github.com/qpubio/qpub-go/internal/logger"
	"github.com/qpubio/qpub-go/internal/port"
	"github.com/qpubio/qpub-go/option"
	"github.com/qpubio/qpub-go/protocol"
)

// PublishOptions for channel publish.
type PublishOptions struct {
	Event string
	Alias string
}

// RestChannel publishes via HTTP.
type RestChannel struct {
	name   string
	http   port.HTTPClient
	auth   *auth.Manager
	opts   *option.Manager
	log    *logger.Logger
	events *emitter.Emitter[any]
}

func NewRestChannel(name string, http port.HTTPClient, auth *auth.Manager, opts *option.Manager, log *logger.Logger) *RestChannel {
	return &RestChannel{name: name, http: http, auth: auth, opts: opts, log: log, events: emitter.New[any]()}
}

func (c *RestChannel) Name() string { return c.name }

func (c *RestChannel) Publish(ctx context.Context, data interface{}, opts PublishOptions) ([]byte, error) {
	headers, err := c.auth.GetAuthHeaders()
	if err != nil {
		return nil, err
	}
	payload, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	req := protocol.RestPublishRequest{
		Messages: []protocol.DataMessagePayload{{
			Alias: opts.Alias,
			Event: opts.Event,
			Data:  payload,
		}},
	}
	url := option.BuildRestBaseURL(c.opts.Get()) + "/channel/" + c.name + "/messages"
	b, _, err := c.http.Post(ctx, url, req, headers)
	return b, err
}

func (c *RestChannel) Reset() {
	c.events.RemoveAll()
}

// RestManager manages REST channels.
type RestManager struct {
	channels map[string]*RestChannel
	http     port.HTTPClient
	auth     *auth.Manager
	opts     *option.Manager
	log      *logger.Logger
}

func NewRestManager(http port.HTTPClient, auth *auth.Manager, opts *option.Manager, log *logger.Logger) *RestManager {
	return &RestManager{
		channels: make(map[string]*RestChannel),
		http:     http,
		auth:     auth,
		opts:     opts,
		log:      log,
	}
}

func (m *RestManager) Get(name string) *RestChannel {
	if ch, ok := m.channels[name]; ok {
		return ch
	}
	ch := NewRestChannel(name, m.http, m.auth, m.opts, m.log)
	m.channels[name] = ch
	return ch
}

func (m *RestManager) Has(name string) bool {
	_, ok := m.channels[name]
	return ok
}

func (m *RestManager) Remove(name string) {
	delete(m.channels, name)
}

func (m *RestManager) PublishBatch(ctx context.Context, channels []string, messages []protocol.DataMessagePayload) ([]byte, error) {
	headers, err := m.auth.GetAuthHeaders()
	if err != nil {
		return nil, err
	}
	req := protocol.RestPublishRequest{Channels: channels, Messages: messages}
	url := option.BuildRestBaseURL(m.opts.Get()) + "/channels/messages"
	b, _, err := m.http.Post(ctx, url, req, headers)
	return b, err
}

func (m *RestManager) Reset() {
	for _, ch := range m.channels {
		ch.Reset()
	}
	m.channels = make(map[string]*RestChannel)
}
