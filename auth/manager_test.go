package auth_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	"github.com/qpubio/qpub-go/auth"
	"github.com/qpubio/qpub-go/internal/apikey"
	"github.com/qpubio/qpub-go/internal/crypto"
	"github.com/qpubio/qpub-go/internal/logger"
	"github.com/qpubio/qpub-go/option"
)

type mockHTTP struct {
	post func(ctx context.Context, url string, body interface{}, headers map[string]string) ([]byte, int, error)
}

func (m *mockHTTP) Post(ctx context.Context, url string, body interface{}, headers map[string]string) ([]byte, int, error) {
	return m.post(ctx, url, body, headers)
}

func createValidJWT(t *testing.T, alias string, perm map[string][]string) string {
	t.Helper()
	header := map[string]string{"alg": "HS256", "typ": "JWT", "aki": "key-1"}
	payload := map[string]interface{}{"exp": time.Now().Unix() + 3600}
	if alias != "" {
		payload["alias"] = alias
	}
	if perm != nil {
		payload["permission"] = perm
	}
	hb, _ := json.Marshal(header)
	pb, _ := json.Marshal(payload)
	return base64.StdEncoding.EncodeToString(hb) + "." + base64.StdEncoding.EncodeToString(pb) + ".test-signature"
}

func testOpts(overrides option.Option) *option.Manager {
	base := option.DefaultOption()
	base.AuthenticateRetries = 0
	merge := option.NewManagerFrom(overrides)
	o := merge.Get()
	// re-apply zero retries unless overridden
	if overrides.AuthenticateRetries == 0 && overrides.AuthURL == "" {
		o.AuthenticateRetries = 0
	}
	return option.NewManagerFrom(o)
}

func TestAPIKeyParse(t *testing.T) {
	p, err := apikey.Parse("key-1:secret-1")
	if err != nil || p.PublicID != "key-1" || p.Secret != "secret-1" {
		t.Fatalf("parse: %+v %v", p, err)
	}
	for _, cred := range []string{"key-1", "key-1:secret-1:extra", ":secret-1"} {
		if _, err := apikey.Parse(cred); err == nil {
			t.Fatalf("expected error for %q", cred)
		}
	}
}

func TestAPIKeyAuthURL(t *testing.T) {
	opts := testOpts(option.Option{APIKey: "test-api-key"})
	log := logger.NewFactory("t", opts.Get()).Create("Auth")
	m := auth.NewManager(opts, &mockHTTP{}, log)
	u, err := m.GetAuthenticateURL("ws://localhost:8080/v1")
	if err != nil || u != "ws://localhost:8080/v1?api_key=test-api-key" {
		t.Fatalf("url: %s err=%v", u, err)
	}
}

func TestCreateTokenRequestCanonical(t *testing.T) {
	opts := testOpts(option.Option{APIKey: "key-1:secret-1"})
	log := logger.NewFactory("t", opts.Get()).Create("Auth")
	var captured string
	httpMock := &mockHTTP{post: func(ctx context.Context, url string, body interface{}, headers map[string]string) ([]byte, int, error) {
		return nil, 200, nil
	}}
	_ = httpMock
	m := auth.NewManager(opts, &mockHTTP{}, log)
	_, _ = m.CreateTokenRequest(context.Background(), option.TokenOptions{
		Alias:      "client-1",
		Permission: option.Permission{"room.*": {"subscribe"}},
	})
	captured = auth.BuildCanonicalString("key-1", 123, "client-1", option.Permission{"room.*": {"subscribe"}})
	if captured != "aki=key-1\ntimestamp=123\nalias=client-1\npermission={\"room.*\":[\"subscribe\"]}" {
		t.Fatalf("canonical mismatch: %q", captured)
	}
	sig, _ := crypto.HMACSign(captured, "secret-1")
	if sig == "" {
		t.Fatal("expected signature")
	}
}

func TestRequestTokenMissingAKI(t *testing.T) {
	opts := testOpts(option.Option{})
	log := logger.NewFactory("t", opts.Get()).Create("Auth")
	m := auth.NewManager(opts, &mockHTTP{}, log)
	_, err := m.RequestToken(context.Background(), option.TokenRequest{AKI: ""})
	if err == nil || err.Error() != "Invalid token request: aki is required" {
		t.Fatalf("err=%v", err)
	}
}

func TestIssueTokenURL(t *testing.T) {
	port := 443
	opts := testOpts(option.Option{
		APIKey:   "key-1:secret-1",
		HTTPHost: "api.example.com",
		HTTPPort: &port,
		IsSecure: true,
	})
	log := logger.NewFactory("t", opts.Get()).Create("Auth")
	var gotURL string
	httpMock := &mockHTTP{post: func(ctx context.Context, url string, body interface{}, headers map[string]string) ([]byte, int, error) {
		gotURL = url
	 tok := createValidJWT(t, "", nil)
	 b, _ := json.Marshal(map[string]string{"token": tok})
	 return b, 200, nil
	}}
	m := auth.NewManager(opts, httpMock, log)
	_, err := m.IssueToken(context.Background(), option.TokenOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if gotURL != "https://api.example.com:443/v1/key/key-1/token/issue" {
		t.Fatalf("url=%s", gotURL)
	}
}
