package qpub

import (
	"context"

	"github.com/google/uuid"
	"github.com/qpubio/qpub-go/auth"
	"github.com/qpubio/qpub-go/channel"
	"github.com/qpubio/qpub-go/connection"
	"github.com/qpubio/qpub-go/internal/logger"
	"github.com/qpubio/qpub-go/option"
	"github.com/qpubio/qpub-go/transport/httpclient"
	"github.com/qpubio/qpub-go/transport/ws"
)

// Socket is the WebSocket client (qpub-js Socket).
type Socket struct {
	instanceID string

	OptionManager *option.Manager
	Auth          *auth.Manager
	Connection    *connection.Conn
	Channels      *channel.SocketManager

	log *logger.Logger
}

// NewSocket creates a Socket client.
func NewSocket(funcs ...option.OptionFunc) *Socket {
	om := option.NewManager(funcs...)
	instanceID := "socket_" + uuid.NewString()
	logFactory := logger.NewFactory(instanceID, om.Get())
	http := httpclient.New()
	authMgr := auth.NewManager(om, http, logFactory.Create("AuthManager"))
	wsClient := ws.New(logFactory.Create("WebSocketClient"))
	chMgr := channel.NewSocketManager(wsClient, logFactory.Create("SocketChannelManager"))
	conn := connection.New(om, authMgr, wsClient, chMgr, logFactory.Create("Connection"))
	s := &Socket{
		instanceID:    instanceID,
		OptionManager: om,
		Auth:          authMgr,
		Connection:    conn,
		Channels:      chMgr,
		log:           logFactory.Create("Socket"),
	}
	if om.Get().AutoConnect {
		go func() { _ = s.Connection.Connect(context.Background()) }()
	}
	s.log.Info("Socket instance created")
	return s
}

// GetInstanceID returns instance identifier.
func (s *Socket) GetInstanceID() string { return s.instanceID }

// Reset resets the socket instance (qpub-js order).
func (s *Socket) Reset() {
	s.Connection.Reset()
	s.Channels.Reset()
	s.Auth.Reset()
	s.OptionManager.Reset()
	s.log.Info("Socket instance reset completed")
}
