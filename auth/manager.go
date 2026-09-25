package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/qpubio/qpub-go/events"
	"github.com/qpubio/qpub-go/internal/apikey"
	"github.com/qpubio/qpub-go/internal/crypto"
	"github.com/qpubio/qpub-go/internal/emitter"
	"github.com/qpubio/qpub-go/internal/jwt"
	"github.com/qpubio/qpub-go/internal/logger"
	"github.com/qpubio/qpub-go/option"
	"github.com/qpubio/qpub-go/transport/httpclient"
)

// HTTPDoer is the HTTP dependency for auth (mockable).
type HTTPDoer interface {
	Post(ctx context.Context, url string, body interface{}, headers map[string]string) ([]byte, int, error)
}

// Manager handles API key, token, and auth URL flows.
type Manager struct {
	opts   *option.Manager
	http   HTTPDoer
	log    *logger.Logger
	events *emitter.Emitter[any]

	mu           sync.Mutex
	currentToken string
	refreshTimer *time.Timer
	resetting    bool
	cancel       context.CancelFunc
	ctx          context.Context
}

func NewManager(opts *option.Manager, http HTTPDoer, log *logger.Logger) *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	m := &Manager{
		opts:   opts,
		http:   http,
		log:    log,
		events: emitter.New[any](),
		ctx:    ctx,
		cancel: cancel,
	}
	if tr := opts.Get().TokenRequest; tr != nil {
		go func() {
			_, _ = m.RequestToken(context.Background(), *tr)
		}()
	}
	return m
}

func (m *Manager) On(event string, fn func(any)) {
	m.events.On(event, fn)
}

func (m *Manager) Authenticate(ctx context.Context) (*option.AuthResponse, error) {
	m.mu.Lock()
	resetting := m.resetting
	m.mu.Unlock()
	if resetting {
		return nil, nil
	}

	o := m.opts.Get()
	retries := o.AuthenticateRetries
	if retries == 0 && o.AuthenticateRetryIntervalMs == 1000 && !o.Debug {
		// keep zero retries when explicitly 0 in tests
	}
	retryInterval := time.Duration(o.AuthenticateRetryIntervalMs) * time.Millisecond

	for attempt := 0; attempt <= retries; attempt++ {
		if m.aborted() {
			return nil, nil
		}
		if o.TokenRequest != nil {
			resp, err := m.RequestToken(ctx, *o.TokenRequest)
			if err != nil {
				if attempt == retries {
					return nil, err
				}
				time.Sleep(retryInterval)
				continue
			}
			return resp, nil
		}
		resp, err := m.authenticateOnce(ctx)
		if err == nil {
			if resp != nil && resp.TokenRequest != nil {
				return m.RequestToken(ctx, *resp.TokenRequest)
			}
			return resp, nil
		}
		if m.aborted() {
			return nil, nil
		}
		if attempt == retries {
			return nil, err
		}
		time.Sleep(retryInterval)
	}
	return nil, nil
}

func (m *Manager) aborted() bool {
	select {
	case <-m.ctx.Done():
		return true
	default:
		return false
	}
}

func (m *Manager) authenticateOnce(ctx context.Context) (*option.AuthResponse, error) {
	o := m.opts.Get()
	if o.AuthURL == "" && o.APIKey == "" {
		return nil, errors.New("Either authUrl or apiKey must be provided")
	}
	if o.AuthURL == "" {
		return nil, nil
	}
	headers := map[string]string{}
	var body map[string]interface{}
	if o.AuthOptions != nil {
		for k, v := range o.AuthOptions.Headers {
			headers[k] = v
		}
		body = o.AuthOptions.Body
	}
	b, _, err := m.http.Post(ctx, o.AuthURL, body, headers)
	if err != nil {
		return nil, err
	}
	var resp option.AuthResponse
	if err := json.Unmarshal(b, &resp); err != nil {
		return nil, err
	}
	if resp.Token != "" {
		if _, err := jwt.Decode(resp.Token); err != nil {
			return nil, err
		}
		m.setToken(resp.Token)
		return &resp, nil
	}
	if resp.TokenRequest != nil {
		return &resp, nil
	}
	return nil, errors.New("Invalid response: expected token or tokenRequest")
}

func (m *Manager) ShouldAutoAuthenticate() bool {
	o := m.opts.Get()
	return o.AutoAuthenticate
}

func (m *Manager) IsAuthenticated() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.currentToken != ""
}

func (m *Manager) GetCurrentToken() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.currentToken
}

func (m *Manager) GetToken() string {
	m.mu.Lock()
	tok := m.currentToken
	m.mu.Unlock()
	if tok != "" && !jwt.IsExpired(tok) {
		return tok
	}
	if tok != "" {
		m.events.Emit(events.AuthTokenExpired, events.AuthTokenExpiredPayload{ExpiredAt: time.Now(), Token: tok})
		m.ClearToken()
	}
	return ""
}

func (m *Manager) setToken(token string) {
	m.mu.Lock()
	m.currentToken = token
	m.mu.Unlock()
	dec, _ := jwt.Decode(token)
	var expiresAt *time.Time
	if dec.Payload.Exp > 0 {
		t := time.Unix(dec.Payload.Exp, 0)
		expiresAt = &t
	}
	m.events.Emit(events.AuthTokenUpdated, events.AuthTokenUpdatedPayload{Token: token, ExpiresAt: expiresAt})
	m.scheduleRefresh(token)
}

func (m *Manager) scheduleRefresh(token string) {
	m.mu.Lock()
	if m.refreshTimer != nil {
		m.refreshTimer.Stop()
	}
	m.mu.Unlock()

	dec, err := jwt.Decode(token)
	if err != nil {
		return
	}
	buffer := 60 * time.Second
	delay := time.Until(time.Unix(dec.Payload.Exp, 0).Add(-buffer))
	if delay <= 0 {
		m.events.Emit(events.AuthTokenExpired, events.AuthTokenExpiredPayload{ExpiredAt: time.Now()})
		m.ClearToken()
		return
	}
	m.mu.Lock()
	m.refreshTimer = time.AfterFunc(delay, func() {
		m.events.Emit(events.AuthTokenExpired, events.AuthTokenExpiredPayload{ExpiredAt: time.Now()})
		m.ClearToken()
	})
	m.mu.Unlock()
}

func (m *Manager) ClearToken() {
	m.mu.Lock()
	m.currentToken = ""
	if m.refreshTimer != nil {
		m.refreshTimer.Stop()
		m.refreshTimer = nil
	}
	m.mu.Unlock()
}

func (m *Manager) GetAuthHeaders() (map[string]string, error) {
	o := m.opts.Get()
	if tok := m.GetToken(); tok != "" {
		return map[string]string{"Authorization": "Bearer " + tok}, nil
	}
	if o.APIKey != "" {
		h := map[string]string{"Authorization": "Basic " + base64.StdEncoding.EncodeToString([]byte(o.APIKey))}
		if o.Alias != "" {
			h["X-Alias"] = o.Alias
		}
		return h, nil
	}
	return nil, errors.New("No authentication credentials provided")
}

func (m *Manager) GetAuthQueryParams() (string, error) {
	o := m.opts.Get()
	if tok := m.GetToken(); tok != "" {
		return "access_token=" + url.QueryEscape(tok), nil
	}
	if o.APIKey != "" {
		params := "api_key=" + url.QueryEscape(o.APIKey)
		if o.Alias != "" {
			params += "&alias=" + url.QueryEscape(o.Alias)
		}
		return params, nil
	}
	return "", errors.New("No authentication credentials available")
}

func (m *Manager) GetAuthenticateURL(baseURL string) (string, error) {
	q, err := m.GetAuthQueryParams()
	if err != nil {
		return "", err
	}
	sep := "?"
	if strings.Contains(baseURL, "?") {
		sep = "&"
	}
	return baseURL + sep + q, nil
}

func (m *Manager) GenerateToken(ctx context.Context, opts option.TokenOptions) (string, error) {
	_ = ctx
	o := m.opts.Get()
	if o.APIKey == "" {
		return "", errors.New("API key is required")
	}
	parsed, err := apikey.Parse(o.APIKey)
	if err != nil {
		return "", fmt.Errorf("Token generation failed: %w", err)
	}
	expiresIn := opts.ExpiresIn
	if expiresIn == 0 {
		expiresIn = 3600
	}
	payload := jwt.Payload{Exp: time.Now().Unix() + int64(expiresIn)}
	if opts.Alias != "" {
		payload.Alias = opts.Alias
	}
	if opts.Permission != nil {
		payload.Permission = opts.Permission
	}
	return jwt.Sign(payload, parsed.PublicID, parsed.Secret)
}

func (m *Manager) IssueToken(ctx context.Context, opts option.TokenOptions) (string, error) {
	o := m.opts.Get()
	if o.APIKey == "" {
		return "", errors.New("API key is required for issuing tokens")
	}
	parsed, err := apikey.Parse(o.APIKey)
	if err != nil {
		return "", fmt.Errorf("Token issuance failed: %w", err)
	}
	base := option.BuildRestBaseURL(o)
	u := base + "/key/" + parsed.PublicID + "/token/issue"
	headers := map[string]string{
		"Authorization": "Basic " + base64.StdEncoding.EncodeToString([]byte(o.APIKey)),
	}
	b, _, err := m.http.Post(ctx, u, opts, headers)
	if err != nil {
		return "", fmt.Errorf("Token issuance failed: %w", err)
	}
	var resp struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(b, &resp); err != nil {
		return "", fmt.Errorf("Token issuance failed: %w", err)
	}
	if resp.Token == "" {
		return "", errors.New("Invalid token response from QPub server")
	}
	if _, err := jwt.Decode(resp.Token); err != nil {
		return "", fmt.Errorf("Token issuance failed: %w", err)
	}
	return resp.Token, nil
}

func (m *Manager) CreateTokenRequest(ctx context.Context, opts option.TokenOptions) (option.TokenRequest, error) {
	_ = ctx
	o := m.opts.Get()
	if o.APIKey == "" {
		return option.TokenRequest{}, errors.New("API key is required for creating token requests")
	}
	parsed, err := apikey.Parse(o.APIKey)
	if err != nil {
		return option.TokenRequest{}, fmt.Errorf("Token request creation failed: %w", err)
	}
	ts := time.Now().Unix()
	canonical := BuildCanonicalString(parsed.PublicID, ts, opts.Alias, opts.Permission)
	sig, err := crypto.HMACSign(canonical, parsed.Secret)
	if err != nil {
		return option.TokenRequest{}, fmt.Errorf("Token request creation failed: %w", err)
	}
	req := option.TokenRequest{
		AKI:       parsed.PublicID,
		Timestamp: ts,
		Signature: sig,
	}
	if opts.Alias != "" {
		req.Alias = opts.Alias
	}
	if opts.Permission != nil {
		req.Permission = opts.Permission
	}
	return req, nil
}

func (m *Manager) RequestToken(ctx context.Context, request option.TokenRequest) (*option.AuthResponse, error) {
	if request.AKI == "" {
		return nil, errors.New("Invalid token request: aki is required")
	}
	o := m.opts.Get()
	u := option.BuildRestBaseURL(o) + "/key/" + request.AKI + "/token/request"
	b, _, err := m.http.Post(ctx, u, request, map[string]string{"Content-Type": "application/json"})
	if err != nil {
		return nil, err
	}
	var resp option.AuthResponse
	if err := json.Unmarshal(b, &resp); err != nil {
		return nil, err
	}
	if resp.Token == "" {
		return nil, errors.New("Invalid response: token not found")
	}
	if _, err := jwt.Decode(resp.Token); err != nil {
		return nil, err
	}
	m.setToken(resp.Token)
	return &resp, nil
}

func (m *Manager) Reset() {
	m.mu.Lock()
	m.resetting = true
	m.mu.Unlock()
	m.cancel()
	ctx, cancel := context.WithCancel(context.Background())
	m.mu.Lock()
	m.cancel = cancel
	m.ctx = ctx
	m.resetting = false
	m.mu.Unlock()
	m.ClearToken()
	m.events.RemoveAll()
}

// BuildCanonicalString builds the newline-delimited string signed for token requests.
func BuildCanonicalString(aki string, timestamp int64, alias string, permission option.Permission) string {
	lines := []string{
		"aki=" + aki,
		fmt.Sprintf("timestamp=%d", timestamp),
	}
	if alias != "" {
		lines = append(lines, "alias="+alias)
	}
	if permission != nil {
		sorted := sortKeysRecursive(permission)
		b, _ := json.Marshal(sorted)
		lines = append(lines, "permission="+string(b))
	}
	return strings.Join(lines, "\n")
}

func sortKeysRecursive(value interface{}) interface{} {
	switch v := value.(type) {
	case map[string][]string:
		out := make(map[string]interface{})
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			out[k] = sortKeysRecursive(v[k])
		}
		return out
	case []string:
		cp := append([]string(nil), v...)
		return cp
	case option.Permission:
		return sortKeysRecursive(map[string][]string(v))
	default:
		return value
	}
}

// Ensure Manager uses httpclient when constructed from facade.
var _ HTTPDoer = (*httpclient.Client)(nil)
