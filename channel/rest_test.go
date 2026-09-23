package channel_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/qpubio/qpub-go/auth"
	"github.com/qpubio/qpub-go/channel"
	"github.com/qpubio/qpub-go/internal/logger"
	"github.com/qpubio/qpub-go/option"
	"github.com/qpubio/qpub-go/protocol"
	"github.com/qpubio/qpub-go/transport/httpclient"
)

func TestRestPublish(t *testing.T) {
	var gotPath string
	var gotBody protocol.RestPublishRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	host := srv.URL[len("http://"):]
	om := option.NewManager(
		option.WithAPIKey("pub:sec"),
		func(o *option.Option) { o.HTTPHost = host },
		option.WithIsSecure(false),
	)
	httpClient := httpclient.New()
	authMgr := auth.NewManager(om, httpClient, logger.NewFactory("t", om.Get()).Create("Auth"))
	mgr := channel.NewRestManager(httpClient, authMgr, om, logger.NewFactory("t", om.Get()).Create("Ch"))
	ch := mgr.Get("my-channel")
	_, err := ch.Publish(context.Background(), "Hello!", channel.PublishOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/v1/channel/my-channel/messages" {
		t.Fatalf("path=%s", gotPath)
	}
	if len(gotBody.Messages) != 1 {
		t.Fatalf("messages=%+v", gotBody.Messages)
	}
}
