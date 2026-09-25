package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/qpubio/qpub-go/auth"
	"github.com/qpubio/qpub-go/internal/logger"
	"github.com/qpubio/qpub-go/internal/port"
	"github.com/qpubio/qpub-go/option"
	"github.com/qpubio/qpub-go/protocol"
)

// JobHandler processes a pulled job.
type JobHandler func(ctx context.Context, job protocol.QueueJob) (interface{}, error)

// Manager is RestQueueManager equivalent.
type Manager struct {
	http       port.HTTPClient
	auth       *auth.Manager
	opts       *option.Manager
	log        *logger.Logger
	workerID   string
	mu         sync.Mutex
	running    bool
	stopCh     chan struct{}
}

func NewManager(http port.HTTPClient, auth *auth.Manager, opts *option.Manager, log *logger.Logger, instanceID string) *Manager {
	return &Manager{
		http:     http,
		auth:     auth,
		opts:     opts,
		log:      log,
		workerID: "rest_worker_" + instanceID,
	}
}

func (m *Manager) baseURL() string {
	return option.BuildRestBaseURL(m.opts.Get())
}

func (m *Manager) authHeaders(ctx context.Context) (map[string]string, error) {
	_ = ctx
	return m.auth.GetAuthHeaders()
}

func (m *Manager) Enqueue(ctx context.Context, queueName string, payload interface{}, opts protocol.EnqueueOptions) (protocol.EnqueueResult, error) {
	headers, err := m.authHeaders(ctx)
	if err != nil {
		return protocol.EnqueueResult{}, err
	}
	var raw json.RawMessage
	if payload != nil {
		raw, err = json.Marshal(payload)
		if err != nil {
			return protocol.EnqueueResult{}, err
		}
	}
	body := protocol.EnqueueJobRequestWire{
		Payload:        raw,
		Delay:          opts.Delay,
		IdempotencyKey: opts.IdempotencyKey,
		ScheduleAt:     opts.ScheduleAt,
		Metadata:       opts.Metadata,
	}
	b, _, err := m.http.Post(ctx, m.baseURL()+"/queue/"+queueName+"/jobs", body, headers)
	if err != nil {
		return protocol.EnqueueResult{}, err
	}
	var resp protocol.EnqueueJobResponseWire
	if err := json.Unmarshal(b, &resp); err != nil {
		return protocol.EnqueueResult{}, err
	}
	return protocol.ToEnqueueResult(resp), nil
}

func (m *Manager) GetJob(ctx context.Context, queueName, jobID string) (protocol.QueueJob, error) {
	headers, err := m.authHeaders(ctx)
	if err != nil {
		return protocol.QueueJob{}, err
	}
	b, _, err := m.http.Get(ctx, m.baseURL()+"/queue/"+queueName+"/jobs/"+jobID, headers)
	if err != nil {
		return protocol.QueueJob{}, err
	}
	var w protocol.QueueJobWire
	if err := json.Unmarshal(b, &w); err != nil {
		return protocol.QueueJob{}, err
	}
	return protocol.ToQueueJob(w), nil
}

func (m *Manager) ListJobs(ctx context.Context, queueName string, opts protocol.ListJobsOptions) ([]protocol.QueueJob, error) {
	headers, err := m.authHeaders(ctx)
	if err != nil {
		return nil, err
	}
	u := m.baseURL() + "/queue/" + queueName + "/jobs"
	if opts.Status != "" {
		u += "?status=" + url.QueryEscape(opts.Status)
	}
	b, _, err := m.http.Get(ctx, u, headers)
	if err != nil {
		return nil, err
	}
	var resp protocol.JobsResponseWire
	if err := json.Unmarshal(b, &resp); err != nil {
		return nil, err
	}
	out := make([]protocol.QueueJob, 0, len(resp.Jobs))
	for _, j := range resp.Jobs {
		out = append(out, protocol.ToQueueJob(j))
	}
	return out, nil
}

func (m *Manager) CancelJob(ctx context.Context, queueName, jobID string) error {
	headers, err := m.authHeaders(ctx)
	if err != nil {
		return err
	}
	_, _, err = m.http.Delete(ctx, m.baseURL()+"/queue/"+queueName+"/jobs/"+jobID, headers)
	return err
}

func (m *Manager) RetryJob(ctx context.Context, queueName, jobID string) error {
	headers, err := m.authHeaders(ctx)
	if err != nil {
		return err
	}
	_, _, err = m.http.Post(ctx, m.baseURL()+"/queue/"+queueName+"/jobs/"+jobID+"/retry", map[string]interface{}{}, headers)
	return err
}

func (m *Manager) GetConfig(ctx context.Context, queueName string) (protocol.QueueConfig, error) {
	headers, err := m.authHeaders(ctx)
	if err != nil {
		return protocol.QueueConfig{}, err
	}
	b, _, err := m.http.Get(ctx, m.baseURL()+"/queue/"+queueName, headers)
	if err != nil {
		return protocol.QueueConfig{}, err
	}
	var w protocol.QueueConfigWire
	if err := json.Unmarshal(b, &w); err != nil {
		return protocol.QueueConfig{}, err
	}
	return protocol.ToQueueConfig(w), nil
}

func (m *Manager) UpdateConfig(ctx context.Context, queueName string, opts protocol.UpdateQueueConfigOptions) (protocol.QueueConfig, error) {
	headers, err := m.authHeaders(ctx)
	if err != nil {
		return protocol.QueueConfig{}, err
	}
	body := protocol.UpdateQueueConfigRequestWire{
		ExecutionProfile:  opts.ExecutionProfile,
		VisibilityTimeout: opts.VisibilityTimeout,
		MaxAttempts:       opts.MaxAttempts,
		WebhookURL:        opts.WebhookURL,
		WebhookSecret:     opts.WebhookSecret,
	}
	b, _, err := m.http.Put(ctx, m.baseURL()+"/queue/"+queueName, body, headers)
	if err != nil {
		return protocol.QueueConfig{}, err
	}
	var w protocol.QueueConfigWire
	if err := json.Unmarshal(b, &w); err != nil {
		return protocol.QueueConfig{}, err
	}
	return protocol.ToQueueConfig(w), nil
}

func (m *Manager) RegisterWorker(ctx context.Context, opts protocol.RegisterWorkerOptions) (protocol.WorkerRegistration, error) {
	headers, err := m.authHeaders(ctx)
	if err != nil {
		return protocol.WorkerRegistration{}, err
	}
	body := protocol.RegisterWorkerRequestWire{Name: opts.Name, Queues: opts.Queues}
	b, _, err := m.http.Post(ctx, m.baseURL()+"/workers/register", body, headers)
	if err != nil {
		return protocol.WorkerRegistration{}, err
	}
	var w protocol.WorkerResponseWire
	if err := json.Unmarshal(b, &w); err != nil {
		return protocol.WorkerRegistration{}, err
	}
	return protocol.ToWorkerRegistration(w), nil
}

func (m *Manager) Heartbeat(ctx context.Context, workerID string) (protocol.WorkerRegistration, error) {
	headers, err := m.authHeaders(ctx)
	if err != nil {
		return protocol.WorkerRegistration{}, err
	}
	body := protocol.HeartbeatRequestWire{WorkerID: workerID}
	b, _, err := m.http.Post(ctx, m.baseURL()+"/workers/heartbeat", body, headers)
	if err != nil {
		return protocol.WorkerRegistration{}, err
	}
	var w protocol.WorkerResponseWire
	if err := json.Unmarshal(b, &w); err != nil {
		return protocol.WorkerRegistration{}, err
	}
	return protocol.ToWorkerRegistration(w), nil
}

func (m *Manager) Pull(ctx context.Context, queueName string, opts protocol.PullJobsOptions) ([]protocol.QueueJob, error) {
	headers, err := m.authHeaders(ctx)
	if err != nil {
		return nil, err
	}
	workerID := opts.WorkerID
	if workerID == "" {
		workerID = m.workerID
	}
	batch := opts.BatchSize
	if batch == 0 {
		batch = 1
	}
	wait := opts.Wait
	if wait == "" {
		wait = "20s"
	}
	body := protocol.PullJobsRequestWire{WorkerID: workerID, BatchSize: batch, Wait: wait}
	b, _, err := m.http.Post(ctx, m.baseURL()+"/queue/"+queueName+"/pull", body, headers)
	if err != nil {
		return nil, err
	}
	var resp protocol.JobsResponseWire
	if err := json.Unmarshal(b, &resp); err != nil {
		return nil, err
	}
	out := make([]protocol.QueueJob, 0, len(resp.Jobs))
	for _, j := range resp.Jobs {
		out = append(out, protocol.ToQueueJob(j))
	}
	return out, nil
}

func (m *Manager) Ack(ctx context.Context, queueName, jobID string, opts protocol.AckJobOptions) error {
	headers, err := m.authHeaders(ctx)
	if err != nil {
		return err
	}
	var result json.RawMessage
	if opts.Result != nil {
		result, err = json.Marshal(opts.Result)
		if err != nil {
			return err
		}
	}
	body := protocol.AckJobRequestWire{WorkerID: opts.WorkerID, Result: result}
	_, _, err = m.http.Post(ctx, m.baseURL()+"/queue/"+queueName+"/jobs/"+jobID+"/ack", body, headers)
	return err
}

func (m *Manager) Nack(ctx context.Context, queueName, jobID string, opts protocol.NackJobOptions) error {
	headers, err := m.authHeaders(ctx)
	if err != nil {
		return err
	}
	reason := opts.Reason
	if reason == "" {
		reason = "nacked"
	}
	body := protocol.NackJobRequestWire{WorkerID: opts.WorkerID, Reason: reason, RetryDelay: opts.RetryDelay}
	_, _, err = m.http.Post(ctx, m.baseURL()+"/queue/"+queueName+"/jobs/"+jobID+"/nack", body, headers)
	return err
}

func (m *Manager) RunWorker(ctx context.Context, queueName string, handler JobHandler, opts protocol.RunWorkerOptions) error {
	workerID := opts.WorkerID
	if workerID == "" {
		workerID = m.workerID
	}
	batchSize := opts.BatchSize
	if batchSize == 0 {
		batchSize = 1
	}
	pollInterval := time.Duration(opts.PollIntervalMs) * time.Millisecond
	if pollInterval == 0 {
		pollInterval = time.Second
	}
	wait := opts.Wait
	if wait == "" {
		wait = "20s"
	}
	heartbeatInterval := time.Duration(opts.HeartbeatIntervalMs) * time.Millisecond
	if heartbeatInterval == 0 {
		heartbeatInterval = 20 * time.Second
	}

	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return fmt.Errorf("worker already running")
	}
	m.running = true
	m.stopCh = make(chan struct{})
	m.mu.Unlock()

	defer func() {
		m.mu.Lock()
		m.running = false
		m.mu.Unlock()
	}()

	m.log.Info("Starting queue worker on %s", queueName)
	lastHeartbeat := time.Time{}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-m.stopCh:
			return nil
		default:
		}

		if time.Since(lastHeartbeat) >= heartbeatInterval {
			_, _ = m.Heartbeat(ctx, workerID)
			lastHeartbeat = time.Now()
		}

		jobs, err := m.Pull(ctx, queueName, protocol.PullJobsOptions{WorkerID: workerID, BatchSize: batchSize, Wait: wait})
		if err != nil {
			m.log.Error("Queue pull failed: %v", err)
		} else {
			for _, job := range jobs {
				result, err := handler(ctx, job)
				if err != nil {
					_ = m.Nack(ctx, queueName, job.ID, protocol.NackJobOptions{WorkerID: workerID, Reason: err.Error()})
				} else {
					_ = m.Ack(ctx, queueName, job.ID, protocol.AckJobOptions{WorkerID: workerID, Result: result})
				}
			}
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-m.stopCh:
			return nil
		case <-time.After(pollInterval):
		}
	}
}

func (m *Manager) StopWorker() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.running && m.stopCh != nil {
		close(m.stopCh)
	}
}

func (m *Manager) Reset() {
	m.StopWorker()
}
