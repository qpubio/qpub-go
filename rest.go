package qpub

import (
	"github.com/google/uuid"
	"github.com/qpubio/qpub-go/auth"
	"github.com/qpubio/qpub-go/channel"
	"github.com/qpubio/qpub-go/internal/logger"
	"github.com/qpubio/qpub-go/option"
	"github.com/qpubio/qpub-go/queue"
	"github.com/qpubio/qpub-go/transport/httpclient"
)

// Rest is the HTTP client for channels and queues.
type Rest struct {
	instanceID string

	OptionManager *option.Manager
	Auth          *auth.Manager
	Channels      *channel.RestManager
	Queues        *queue.Manager

	log *logger.Logger
}

// NewRest creates a REST client.
func NewRest(funcs ...option.OptionFunc) *Rest {
	om := option.NewManager(funcs...)
	instanceID := "rest_" + uuid.NewString()
	http := httpclient.New()
	logFactory := logger.NewFactory(instanceID, om.Get())
	log := logFactory.Create("REST")
	authMgr := auth.NewManager(om, http, logFactory.Create("AuthManager"))
	chMgr := channel.NewRestManager(http, authMgr, om, logFactory.Create("RestChannelManager"))
	qMgr := queue.NewManager(http, authMgr, om, logFactory.Create("RestQueueManager"), instanceID)
	return &Rest{
		instanceID:    instanceID,
		OptionManager: om,
		Auth:          authMgr,
		Channels:      chMgr,
		Queues:        qMgr,
		log:           log,
	}
}

// GetInstanceID returns instance identifier.
func (r *Rest) GetInstanceID() string { return r.instanceID }

// Reset clears client state.
func (r *Rest) Reset() {
	r.Channels.Reset()
	r.Queues.Reset()
	r.Auth.Reset()
	r.OptionManager.Reset()
}
