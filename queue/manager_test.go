package queue_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/qpubio/qpub-go/auth"
	"github.com/qpubio/qpub-go/internal/logger"
	"github.com/qpubio/qpub-go/option"
	"github.com/qpubio/qpub-go/protocol"
	"github.com/qpubio/qpub-go/queue"
	"github.com/qpubio/qpub-go/transport/httpclient"
)

func TestEnqueue(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/queue/jobs/jobs" {
			t.Fatalf("path=%s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(protocol.EnqueueJobResponseWire{JobID: "j1", Status: "queued"})
	}))
	defer srv.Close()
	host := srv.URL[len("http://"):]
	om := option.NewManager(option.WithAPIKey("k:s"), func(o *option.Option) { o.HTTPHost = host }, option.WithIsSecure(false))
	httpC := httpclient.New()
	authMgr := auth.NewManager(om, httpC, logger.NewFactory("t", om.Get()).Create("Auth"))
	mgr := queue.NewManager(httpC, authMgr, om, logger.NewFactory("t", om.Get()).Create("Q"), "inst")
	res, err := mgr.Enqueue(context.Background(), "jobs", map[string]string{"x": "1"}, protocol.EnqueueOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if res.JobID != "j1" {
		t.Fatalf("res=%+v", res)
	}
}

func TestPullAndAck(t *testing.T) {
	var acked bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/queue/work/pull":
			_ = json.NewEncoder(w).Encode(protocol.JobsResponseWire{
				Jobs: []protocol.QueueJobWire{{ID: "job-1", QueueName: "work", Status: "running"}},
			})
		case "/v1/queue/work/jobs/job-1/ack":
			acked = true
			w.WriteHeader(http.StatusOK)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer srv.Close()
	host := srv.URL[len("http://"):]
	om := option.NewManager(option.WithAPIKey("k:s"), func(o *option.Option) { o.HTTPHost = host }, option.WithIsSecure(false))
	httpC := httpclient.New()
	authMgr := auth.NewManager(om, httpC, logger.NewFactory("t", om.Get()).Create("Auth"))
	mgr := queue.NewManager(httpC, authMgr, om, logger.NewFactory("t", om.Get()).Create("Q"), "inst")

	jobs, err := mgr.Pull(context.Background(), "work", protocol.PullJobsOptions{WorkerID: "w1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 1 || jobs[0].ID != "job-1" {
		t.Fatalf("jobs=%+v", jobs)
	}
	if err := mgr.Ack(context.Background(), "work", "job-1", protocol.AckJobOptions{WorkerID: "w1"}); err != nil {
		t.Fatal(err)
	}
	if !acked {
		t.Fatal("expected ack request")
	}
}
