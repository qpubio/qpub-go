package auth_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/qpubio/qpub-go/auth"
	"github.com/qpubio/qpub-go/events"
	"github.com/qpubio/qpub-go/internal/apikey"
	"github.com/qpubio/qpub-go/internal/crypto"
	"github.com/qpubio/qpub-go/internal/jwt"
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
	o := option.DefaultOption()
	o.AuthenticateRetries = 0
	o.AuthenticateRetryIntervalMs = 1000
	if overrides.APIKey != "" {
		o.APIKey = overrides.APIKey
	}
	if overrides.AuthURL != "" {
		o.AuthURL = overrides.AuthURL
	}
	if overrides.AuthenticateRetries > 0 {
		o.AuthenticateRetries = overrides.AuthenticateRetries
	}
	if overrides.AuthenticateRetryIntervalMs > 0 {
		o.AuthenticateRetryIntervalMs = overrides.AuthenticateRetryIntervalMs
	}
	if overrides.AutoAuthenticate {
		o.AutoAuthenticate = overrides.AutoAuthenticate
	}
	if overrides.HTTPHost != "" {
		o.HTTPHost = overrides.HTTPHost
	}
	if overrides.HTTPPort != nil {
		o.HTTPPort = overrides.HTTPPort
	}
	if overrides.IsSecure {
		o.IsSecure = overrides.IsSecure
	}
	if overrides.AuthOptions != nil {
		o.AuthOptions = overrides.AuthOptions
	}
	return option.NewManagerFrom(o)
}

func newAuth(t *testing.T, opts *option.Manager, http *mockHTTP) *auth.Manager {
	t.Helper()
	log := logger.NewFactory("t", opts.Get()).Create("Auth")
	return auth.NewManager(opts, http, log)
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

func TestAPIKeyAuthNoTokenNeeded(t *testing.T) {
	opts := testOpts(option.Option{APIKey: "test-api-key"})
	var posts int32
	httpMock := &mockHTTP{post: func(ctx context.Context, url string, body interface{}, headers map[string]string) ([]byte, int, error) {
		atomic.AddInt32(&posts, 1)
		return nil, 200, nil
	}}
	m := newAuth(t, opts, httpMock)
	resp, err := m.Authenticate(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if resp != nil {
		t.Fatalf("resp=%+v", resp)
	}
	if posts != 0 {
		t.Fatalf("posts=%d", posts)
	}
	if m.IsAuthenticated() {
		t.Fatal("expected not authenticated with api key only")
	}
}

func TestAPIKeyAuthURL(t *testing.T) {
	opts := testOpts(option.Option{APIKey: "test-api-key"})
	m := newAuth(t, opts, &mockHTTP{})
	u, err := m.GetAuthenticateURL("ws://localhost:8080/v1")
	if err != nil || u != "ws://localhost:8080/v1?api_key=test-api-key" {
		t.Fatalf("url: %s err=%v", u, err)
	}
}

func TestAuthURLSetsToken(t *testing.T) {
	tok := createValidJWT(t, "", nil)
	opts := testOpts(option.Option{
		AuthURL: "https://auth.example.com/token",
		AuthOptions: &option.AuthOptions{
			Body:    map[string]interface{}{"userId": "test-user"},
			Headers: map[string]string{"Custom-Header": "test"},
		},
	})
	var gotURL string
	var gotBody interface{}
	var gotHeaders map[string]string
	httpMock := &mockHTTP{post: func(ctx context.Context, url string, body interface{}, headers map[string]string) ([]byte, int, error) {
		gotURL, gotBody, gotHeaders = url, body, headers
		b, _ := json.Marshal(map[string]string{"token": tok})
		return b, 200, nil
	}}
	m := newAuth(t, opts, httpMock)
	var updated int
	m.On(events.AuthTokenUpdated, func(any) { updated++ })
	resp, err := m.Authenticate(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if resp == nil || resp.Token != tok {
		t.Fatalf("resp=%+v", resp)
	}
	if gotURL != "https://auth.example.com/token" {
		t.Fatalf("url=%s", gotURL)
	}
	if gotBody.(map[string]interface{})["userId"] != "test-user" {
		t.Fatalf("body=%v", gotBody)
	}
	if gotHeaders["Custom-Header"] != "test" {
		t.Fatalf("headers=%v", gotHeaders)
	}
	if !m.IsAuthenticated() || m.GetCurrentToken() != tok {
		t.Fatal("token not set")
	}
	if updated < 1 {
		t.Fatal("expected token_updated event")
	}
}

func TestAuthURLTokenRequestFlow(t *testing.T) {
	port := 443
	opts := testOpts(option.Option{
		AuthURL:  "https://auth.example.com/token",
		HTTPHost: "api.example.com",
		HTTPPort: &port,
		IsSecure: true,
	})
	tok := createValidJWT(t, "", nil)
	tr := option.TokenRequest{AKI: "key-1", Signature: "test-signature", Timestamp: 123}
	var calls int
	httpMock := &mockHTTP{post: func(ctx context.Context, url string, body interface{}, headers map[string]string) ([]byte, int, error) {
		calls++
		if calls == 1 {
			b, _ := json.Marshal(map[string]interface{}{"tokenRequest": tr})
			return b, 200, nil
		}
		if !strings.Contains(url, "/key/key-1/token/request") {
			t.Fatalf("unexpected url %s", url)
		}
		b, _ := json.Marshal(map[string]string{"token": tok})
		return b, 200, nil
	}}
	m := newAuth(t, opts, httpMock)
	resp, err := m.Authenticate(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("calls=%d", calls)
	}
	if resp == nil || resp.Token != tok {
		t.Fatalf("resp=%+v", resp)
	}
	if !m.IsAuthenticated() {
		t.Fatal("not authenticated")
	}
}

func TestAuthURLInvalidResponse(t *testing.T) {
	opts := testOpts(option.Option{AuthURL: "https://auth.example.com/token"})
	httpMock := &mockHTTP{post: func(ctx context.Context, url string, body interface{}, headers map[string]string) ([]byte, int, error) {
		return []byte(`{}`), 200, nil
	}}
	m := newAuth(t, opts, httpMock)
	_, err := m.Authenticate(context.Background())
	if err == nil || !strings.Contains(err.Error(), "Invalid response") {
		t.Fatalf("err=%v", err)
	}
}

func TestCreateTokenRequestCanonical(t *testing.T) {
	opts := testOpts(option.Option{APIKey: "key-1:secret-1"})
	m := newAuth(t, opts, &mockHTTP{})
	_, err := m.CreateTokenRequest(context.Background(), option.TokenOptions{
		Alias:      "client-1",
		Permission: option.Permission{"room.*": {"subscribe"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	captured := auth.BuildCanonicalString("key-1", 123, "client-1", option.Permission{"room.*": {"subscribe"}})
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
	m := newAuth(t, opts, &mockHTTP{})
	_, err := m.RequestToken(context.Background(), option.TokenRequest{AKI: ""})
	if err == nil || err.Error() != "Invalid token request: aki is required" {
		t.Fatalf("err=%v", err)
	}
}

func TestRequestTokenSuccess(t *testing.T) {
	port := 443
	opts := testOpts(option.Option{HTTPHost: "api.example.com", HTTPPort: &port, IsSecure: true})
	tok := createValidJWT(t, "", nil)
	tr := option.TokenRequest{AKI: "key-1", Signature: "sig", Timestamp: 1}
	var gotURL string
	httpMock := &mockHTTP{post: func(ctx context.Context, url string, body interface{}, headers map[string]string) ([]byte, int, error) {
		gotURL = url
		b, _ := json.Marshal(map[string]string{"token": tok})
		return b, 200, nil
	}}
	m := newAuth(t, opts, httpMock)
	resp, err := m.RequestToken(context.Background(), tr)
	if err != nil {
		t.Fatal(err)
	}
	if gotURL != "https://api.example.com:443/v1/key/key-1/token/request" {
		t.Fatalf("url=%s", gotURL)
	}
	if resp.Token != tok || !m.IsAuthenticated() {
		t.Fatalf("resp=%+v auth=%v", resp, m.IsAuthenticated())
	}
}

func TestRequestTokenHTTPError(t *testing.T) {
	port := 443
	opts := testOpts(option.Option{HTTPHost: "api.example.com", HTTPPort: &port, IsSecure: true})
	httpMock := &mockHTTP{post: func(ctx context.Context, url string, body interface{}, headers map[string]string) ([]byte, int, error) {
		return nil, 0, errors.New("Network error")
	}}
	m := newAuth(t, opts, httpMock)
	_, err := m.RequestToken(context.Background(), option.TokenRequest{AKI: "key-1", Signature: "s", Timestamp: 1})
	if err == nil || err.Error() != "Network error" {
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
	var gotURL string
	httpMock := &mockHTTP{post: func(ctx context.Context, url string, body interface{}, headers map[string]string) ([]byte, int, error) {
		gotURL = url
		tok := createValidJWT(t, "", nil)
		b, _ := json.Marshal(map[string]string{"token": tok})
		return b, 200, nil
	}}
	m := newAuth(t, opts, httpMock)
	_, err := m.IssueToken(context.Background(), option.TokenOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if gotURL != "https://api.example.com:443/v1/key/key-1/token/issue" {
		t.Fatalf("url=%s", gotURL)
	}
}

func TestGenerateToken(t *testing.T) {
	opts := testOpts(option.Option{APIKey: "key-1:secret-1"})
	m := newAuth(t, opts, &mockHTTP{})
	token, err := m.GenerateToken(context.Background(), option.TokenOptions{
		ExpiresIn:  7200,
		Alias:      "user-1",
		Permission: option.Permission{"room.*": {"subscribe"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	dec, err := jwt.Decode(token)
	if err != nil {
		t.Fatal(err)
	}
	if dec.Header.AKI != "key-1" || dec.Payload.Alias != "user-1" {
		t.Fatalf("dec=%+v", dec)
	}
	if dec.Payload.Permission["room.*"][0] != "subscribe" {
		t.Fatalf("perm=%v", dec.Payload.Permission)
	}
}

func TestGenerateTokenNoAPIKey(t *testing.T) {
	opts := testOpts(option.Option{})
	m := newAuth(t, opts, &mockHTTP{})
	_, err := m.GenerateToken(context.Background(), option.TokenOptions{})
	if err == nil || err.Error() != "API key is required" {
		t.Fatalf("err=%v", err)
	}
}

func TestAuthenticateRetries(t *testing.T) {
	opts := testOpts(option.Option{
		AuthURL:                     "https://auth.example.com/token",
		AuthenticateRetries:         2,
		AuthenticateRetryIntervalMs: 50,
	})
	tok := createValidJWT(t, "", nil)
	var calls int
	httpMock := &mockHTTP{post: func(ctx context.Context, url string, body interface{}, headers map[string]string) ([]byte, int, error) {
		calls++
		if calls < 3 {
			return nil, 0, fmt.Errorf("fail %d", calls)
		}
		b, _ := json.Marshal(map[string]string{"token": tok})
		return b, 200, nil
	}}
	m := newAuth(t, opts, httpMock)
	start := time.Now()
	_, err := m.Authenticate(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if calls != 3 {
		t.Fatalf("calls=%d", calls)
	}
	if time.Since(start) < 100*time.Millisecond {
		t.Fatal("expected retry delay")
	}
	if !m.IsAuthenticated() {
		t.Fatal("not authenticated")
	}
}

func TestAuthenticateMaxRetriesExceeded(t *testing.T) {
	opts := testOpts(option.Option{
		AuthURL:                     "https://auth.example.com/token",
		AuthenticateRetries:         2,
		AuthenticateRetryIntervalMs: 10,
	})
	var calls int
	httpMock := &mockHTTP{post: func(ctx context.Context, url string, body interface{}, headers map[string]string) ([]byte, int, error) {
		calls++
		return nil, 0, errors.New("Persistent error")
	}}
	m := newAuth(t, opts, httpMock)
	_, err := m.Authenticate(context.Background())
	if err == nil || err.Error() != "Persistent error" {
		t.Fatalf("err=%v", err)
	}
	if calls != 3 {
		t.Fatalf("calls=%d", calls)
	}
}

func TestAuthStateAndHeaders(t *testing.T) {
	opts := testOpts(option.Option{AuthURL: "https://auth.example.com/token"})
	tok := createValidJWT(t, "", nil)
	httpMock := &mockHTTP{post: func(ctx context.Context, url string, body interface{}, headers map[string]string) ([]byte, int, error) {
		b, _ := json.Marshal(map[string]string{"token": tok})
		return b, 200, nil
	}}
	m := newAuth(t, opts, httpMock)
	if m.IsAuthenticated() || m.GetCurrentToken() != "" {
		t.Fatal("initial state")
	}
	if _, err := m.Authenticate(context.Background()); err != nil {
		t.Fatal(err)
	}
	h, err := m.GetAuthHeaders()
	if err != nil {
		t.Fatal(err)
	}
	if h["Authorization"] != "Bearer "+tok {
		t.Fatalf("headers=%v", h)
	}
	m.ClearToken()
	if m.IsAuthenticated() || m.GetCurrentToken() != "" {
		t.Fatal("after clear")
	}
}

func TestShouldAutoAuthenticate(t *testing.T) {
	opts := testOpts(option.Option{AutoAuthenticate: true})
	m := newAuth(t, opts, &mockHTTP{})
	if !m.ShouldAutoAuthenticate() {
		t.Fatal("expected true")
	}
	m2 := newAuth(t, option.NewManager(func(o *option.Option) { o.AutoAuthenticate = false }), &mockHTTP{})
	if m2.ShouldAutoAuthenticate() {
		t.Fatal("expected false")
	}
}

func TestAuthenticateNoCredentials(t *testing.T) {
	opts := testOpts(option.Option{})
	m := newAuth(t, opts, &mockHTTP{})
	_, err := m.Authenticate(context.Background())
	if err == nil || !strings.Contains(err.Error(), "Either authUrl or apiKey") {
		t.Fatalf("err=%v", err)
	}
}

func TestAuthenticateHTTPError(t *testing.T) {
	opts := testOpts(option.Option{AuthURL: "https://auth.example.com/token"})
	httpMock := &mockHTTP{post: func(ctx context.Context, url string, body interface{}, headers map[string]string) ([]byte, int, error) {
		return nil, 0, errors.New("Connection timeout")
	}}
	m := newAuth(t, opts, httpMock)
	_, err := m.Authenticate(context.Background())
	if err == nil || err.Error() != "Connection timeout" {
		t.Fatalf("err=%v", err)
	}
}

func TestAuthenticateMalformedJWT(t *testing.T) {
	opts := testOpts(option.Option{AuthURL: "https://auth.example.com/token"})
	httpMock := &mockHTTP{post: func(ctx context.Context, url string, body interface{}, headers map[string]string) ([]byte, int, error) {
		b, _ := json.Marshal(map[string]string{"token": "invalid-jwt-token"})
		return b, 200, nil
	}}
	m := newAuth(t, opts, httpMock)
	_, err := m.Authenticate(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAuthenticateTokenWithoutAliasOrPermission(t *testing.T) {
	for _, tc := range []struct {
		name string
		tok  string
	}{
		{"no alias", createValidJWT(t, "", nil)},
		{"with alias", createValidJWT(t, "test-user", nil)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			opts := testOpts(option.Option{AuthURL: "https://auth.example.com/token"})
			httpMock := &mockHTTP{post: func(ctx context.Context, url string, body interface{}, headers map[string]string) ([]byte, int, error) {
				b, _ := json.Marshal(map[string]string{"token": tc.tok})
				return b, 200, nil
			}}
			m := newAuth(t, opts, httpMock)
			resp, err := m.Authenticate(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if resp.Token != tc.tok || !m.IsAuthenticated() {
				t.Fatalf("resp=%+v", resp)
			}
		})
	}
}
