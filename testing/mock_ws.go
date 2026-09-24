package testing

import (
	"encoding/json"
	"sync"
)

// MockWS records outbound WebSocket frames for tests.
type MockWS struct {
	mu          sync.Mutex
	Sent        [][]byte
	Connected   bool
	OnSend      func(data []byte) error
	InboundFn   func(lastSent []byte) []byte
}

func NewMockWS() *MockWS {
	return &MockWS{Connected: true}
}

func (m *MockWS) Send(data []byte) error {
	m.mu.Lock()
	m.Sent = append(m.Sent, append([]byte(nil), data...))
	fn := m.OnSend
	inbound := m.InboundFn
	last := append([]byte(nil), data...)
	m.mu.Unlock()
	if fn != nil {
		if err := fn(data); err != nil {
			return err
		}
	}
	if inbound != nil {
		if frame := inbound(last); len(frame) > 0 {
			_ = frame
		}
	}
	return nil
}

func (m *MockWS) IsConnected() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.Connected
}

// LastSubscribeChannel returns the channel from the most recent SUBSCRIBE frame, if any.
func (m *MockWS) LastSubscribeChannel() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := len(m.Sent) - 1; i >= 0; i-- {
		var v struct {
			Action  int    `json:"action"`
			Channel string `json:"channel"`
		}
		if json.Unmarshal(m.Sent[i], &v) == nil && v.Channel != "" {
			return v.Channel
		}
	}
	return ""
}
