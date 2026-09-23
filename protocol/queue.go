package protocol

import "encoding/json"

// Wire DTOs (snake_case JSON).

type EnqueueJobRequestWire struct {
	Payload        json.RawMessage        `json:"payload,omitempty"`
	IdempotencyKey string                 `json:"idempotency_key,omitempty"`
	Delay          string                 `json:"delay,omitempty"`
	ScheduleAt     *string                `json:"schedule_at,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

type EnqueueJobResponseWire struct {
	JobID  string `json:"job_id"`
	Status string `json:"status"`
}

type QueueJobWire struct {
	ID             string                 `json:"id"`
	QueueName      string                 `json:"queue_name"`
	Status         string                 `json:"status"`
	Payload        json.RawMessage        `json:"payload,omitempty"`
	Result         json.RawMessage        `json:"result,omitempty"`
	IdempotencyKey string                 `json:"idempotency_key,omitempty"`
	Attempt        *int                   `json:"attempt,omitempty"`
	MaxAttempts    *int                   `json:"max_attempts,omitempty"`
	ScheduleAt     string                 `json:"schedule_at,omitempty"`
	StartedAt      string                 `json:"started_at,omitempty"`
	CompletedAt    string                 `json:"completed_at,omitempty"`
	WorkerID       string                 `json:"worker_id,omitempty"`
	ErrorMessage   string                 `json:"error_message,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt      string                 `json:"created_at,omitempty"`
	UpdatedAt      string                 `json:"updated_at,omitempty"`
}

type JobsResponseWire struct {
	Jobs []QueueJobWire `json:"jobs"`
}

type PullJobsRequestWire struct {
	WorkerID  string `json:"worker_id"`
	BatchSize int    `json:"batch_size"`
	Wait      string `json:"wait"`
}

type AckJobRequestWire struct {
	WorkerID string          `json:"worker_id"`
	Result   json.RawMessage `json:"result,omitempty"`
}

type NackJobRequestWire struct {
	WorkerID   string `json:"worker_id"`
	Reason     string `json:"reason"`
	RetryDelay string `json:"retry_delay,omitempty"`
}

type QueueConfigWire struct {
	Name               string `json:"name,omitempty"`
	ExecutionProfile   string `json:"execution_profile,omitempty"`
	VisibilityTimeout  string `json:"visibility_timeout,omitempty"`
	MaxAttempts        *int   `json:"max_attempts,omitempty"`
	Retention          string `json:"retention,omitempty"`
	MaxPayloadBytes    *int   `json:"max_payload_bytes,omitempty"`
	WebhookURL         string `json:"webhook_url,omitempty"`
	WebhookSecret      string `json:"webhook_secret,omitempty"`
	CreatedAt          string `json:"created_at,omitempty"`
	UpdatedAt          string `json:"updated_at,omitempty"`
}

type UpdateQueueConfigRequestWire struct {
	ExecutionProfile  string `json:"execution_profile,omitempty"`
	VisibilityTimeout string `json:"visibility_timeout,omitempty"`
	MaxAttempts       *int   `json:"max_attempts,omitempty"`
	WebhookURL        string `json:"webhook_url,omitempty"`
	WebhookSecret     string `json:"webhook_secret,omitempty"`
}

type RegisterWorkerRequestWire struct {
	Name   string   `json:"name"`
	Queues []string `json:"queues"`
}

type WorkerResponseWire struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Queues     []string `json:"queues"`
	LastSeenAt string   `json:"last_seen_at,omitempty"`
	CreatedAt  string   `json:"created_at,omitempty"`
	UpdatedAt  string   `json:"updated_at,omitempty"`
}

type HeartbeatRequestWire struct {
	WorkerID string `json:"worker_id"`
}

// Public domain types (camelCase fields in Go).

type QueueJob struct {
	ID             string
	QueueName      string
	Status         string
	Payload        json.RawMessage
	Result         json.RawMessage
	IdempotencyKey string
	Attempt        *int
	MaxAttempts    *int
	ScheduleAt     string
	StartedAt      string
	CompletedAt    string
	WorkerID       string
	ErrorMessage   string
	Metadata       map[string]interface{}
	CreatedAt      string
	UpdatedAt      string
}

type EnqueueOptions struct {
	Delay          string
	IdempotencyKey string
	ScheduleAt     *string
	Metadata       map[string]interface{}
}

type EnqueueResult struct {
	JobID  string
	Status string
}

type ListJobsOptions struct {
	Status string
}

type PullJobsOptions struct {
	WorkerID  string
	BatchSize int
	Wait      string
}

type AckJobOptions struct {
	WorkerID string
	Result   interface{}
}

type NackJobOptions struct {
	WorkerID   string
	Reason     string
	RetryDelay string
}

type QueueConfig struct {
	Name              string
	ExecutionProfile  string
	VisibilityTimeout string
	MaxAttempts       *int
	Retention         string
	MaxPayloadBytes   *int
	WebhookURL        string
	WebhookSecret     string
	CreatedAt         string
	UpdatedAt         string
}

type UpdateQueueConfigOptions struct {
	ExecutionProfile  string
	VisibilityTimeout string
	MaxAttempts       *int
	WebhookURL        string
	WebhookSecret     string
}

type RegisterWorkerOptions struct {
	Name   string
	Queues []string
}

type WorkerRegistration struct {
	ID         string
	Name       string
	Queues     []string
	LastSeenAt string
	CreatedAt  string
	UpdatedAt  string
}

type RunWorkerOptions struct {
	WorkerID            string
	BatchSize           int
	PollIntervalMs      int
	Wait                string
	HeartbeatIntervalMs int
}

func ToQueueJob(w QueueJobWire) QueueJob {
	return QueueJob{
		ID:             w.ID,
		QueueName:      w.QueueName,
		Status:         w.Status,
		Payload:        w.Payload,
		Result:         w.Result,
		IdempotencyKey: w.IdempotencyKey,
		Attempt:        w.Attempt,
		MaxAttempts:    w.MaxAttempts,
		ScheduleAt:     w.ScheduleAt,
		StartedAt:      w.StartedAt,
		CompletedAt:    w.CompletedAt,
		WorkerID:       w.WorkerID,
		ErrorMessage:   w.ErrorMessage,
		Metadata:       w.Metadata,
		CreatedAt:      w.CreatedAt,
		UpdatedAt:      w.UpdatedAt,
	}
}

func ToEnqueueResult(w EnqueueJobResponseWire) EnqueueResult {
	return EnqueueResult{JobID: w.JobID, Status: w.Status}
}

func ToQueueConfig(w QueueConfigWire) QueueConfig {
	return QueueConfig{
		Name:              w.Name,
		ExecutionProfile:  w.ExecutionProfile,
		VisibilityTimeout: w.VisibilityTimeout,
		MaxAttempts:       w.MaxAttempts,
		Retention:         w.Retention,
		MaxPayloadBytes:   w.MaxPayloadBytes,
		WebhookURL:        w.WebhookURL,
		WebhookSecret:     w.WebhookSecret,
		CreatedAt:         w.CreatedAt,
		UpdatedAt:         w.UpdatedAt,
	}
}

func ToWorkerRegistration(w WorkerResponseWire) WorkerRegistration {
	return WorkerRegistration{
		ID:         w.ID,
		Name:       w.Name,
		Queues:     w.Queues,
		LastSeenAt: w.LastSeenAt,
		CreatedAt:  w.CreatedAt,
		UpdatedAt:  w.UpdatedAt,
	}
}
