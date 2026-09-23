package qpub

import (
	"context"
	"time"

	"github.com/qpubio/qpub-go/channel"
	"github.com/qpubio/qpub-go/option"
	"github.com/qpubio/qpub-go/protocol"
)

// OptionManager manages SDK configuration.
type OptionManager interface {
	Get() option.Option
	Set(partial option.Option)
	Reset()
}

// AuthManager handles authentication.
type AuthManager interface {
	Authenticate(ctx context.Context) (*option.AuthResponse, error)
	IsAuthenticated() bool
	ShouldAutoAuthenticate() bool
	GetAuthenticateURL(baseURL string) (string, error)
	RequestToken(ctx context.Context, request option.TokenRequest) (*option.AuthResponse, error)
	GetCurrentToken() string
	GetAuthHeaders() (map[string]string, error)
	GetToken() string
	ClearToken()
	GetAuthQueryParams() (string, error)
	Reset()
	GenerateToken(ctx context.Context, opts option.TokenOptions) (string, error)
	IssueToken(ctx context.Context, opts option.TokenOptions) (string, error)
	CreateTokenRequest(ctx context.Context, opts option.TokenOptions) (option.TokenRequest, error)
	On(event string, fn func(any))
}

// Connection manages WebSocket connectivity.
type Connection interface {
	Connect(ctx context.Context) error
	Disconnect()
	IsConnected() bool
	Ping(ctx context.Context) (time.Duration, error)
	Reset()
	IsResetting() bool
	On(event string, fn func(any))
}

// RestChannel publishes over HTTP.
type RestChannel interface {
	Name() string
	Publish(ctx context.Context, data interface{}, opts channel.PublishOptions) ([]byte, error)
	Reset()
}

// SocketChannel is real-time pub/sub.
type SocketChannel interface {
	Name() string
	Publish(ctx context.Context, data interface{}, opts channel.PublishOptions) error
	Subscribe(ctx context.Context, handler channel.MessageHandler, opts channel.SubscribeOptions) error
	Unsubscribe(ctx context.Context) error
	Pause(bufferMessages bool)
	Resume()
	IsPaused() bool
	ClearBufferedMessages()
	Reset()
}

// RestQueueManager manages queue jobs over REST.
type RestQueueManager interface {
	Enqueue(ctx context.Context, queueName string, payload interface{}, opts protocol.EnqueueOptions) (protocol.EnqueueResult, error)
	GetJob(ctx context.Context, queueName, jobID string) (protocol.QueueJob, error)
	ListJobs(ctx context.Context, queueName string, opts protocol.ListJobsOptions) ([]protocol.QueueJob, error)
	CancelJob(ctx context.Context, queueName, jobID string) error
	RetryJob(ctx context.Context, queueName, jobID string) error
	GetConfig(ctx context.Context, queueName string) (protocol.QueueConfig, error)
	UpdateConfig(ctx context.Context, queueName string, opts protocol.UpdateQueueConfigOptions) (protocol.QueueConfig, error)
	RegisterWorker(ctx context.Context, opts protocol.RegisterWorkerOptions) (protocol.WorkerRegistration, error)
	Heartbeat(ctx context.Context, workerID string) (protocol.WorkerRegistration, error)
	Pull(ctx context.Context, queueName string, opts protocol.PullJobsOptions) ([]protocol.QueueJob, error)
	Ack(ctx context.Context, queueName, jobID string, opts protocol.AckJobOptions) error
	Nack(ctx context.Context, queueName, jobID string, opts protocol.NackJobOptions) error
	RunWorker(ctx context.Context, queueName string, handler func(context.Context, protocol.QueueJob) (interface{}, error), opts protocol.RunWorkerOptions) error
	StopWorker()
	Reset()
}
