package option

import "sync"

// AuthOptions for custom auth URL requests.
type AuthOptions struct {
	Headers map[string]string
	Body    map[string]interface{}
}

// Permission map for token scopes.
type Permission map[string][]string

// TokenRequest is signed client token exchange payload.
type TokenRequest struct {
	AKI        string     `json:"aki"`
	Permission Permission `json:"permission,omitempty"`
	Alias      string     `json:"alias,omitempty"`
	Timestamp  int64      `json:"timestamp"`
	Signature  string     `json:"signature"`
}

// TokenOptions for generate/issue/createTokenRequest.
type TokenOptions struct {
	Permission Permission
	Alias      string
	ExpiresIn  int // seconds, default 3600
}

// AuthResponse from auth/token endpoints.
type AuthResponse struct {
	Token        string        `json:"token,omitempty"`
	TokenRequest *TokenRequest `json:"tokenRequest,omitempty"`
}

// Option is SDK configuration.
type Option struct {
	APIKey     string
	AuthURL    string
	AuthOptions *AuthOptions
	TokenRequest *TokenRequest

	Alias string

	HTTPHost string
	HTTPPort *int

	WSHost string
	WSPort *int

	IsSecure bool

	AutoConnect    bool
	AutoReconnect  bool
	AutoResubscribe bool
	AutoAuthenticate bool

	ConnectTimeoutMs int

	MaxReconnectAttempts       int
	InitialReconnectDelayMs    int
	MaxReconnectDelayMs        int
	ReconnectBackoffMultiplier float64

	ResubscribeIntervalMs int
	AuthenticateRetries   int
	AuthenticateRetryIntervalMs int

	PingTimeoutMs int

	Debug    bool
	LogLevel string
}

// DefaultOption returns production-oriented defaults for hosts, reconnect, and auth.
func DefaultOption() Option {
	return Option{
		HTTPHost: "rest.qpub.io",
		HTTPPort: nil,
		WSHost:   "socket.qpub.io",
		WSPort:   nil,
		IsSecure: true,

		AutoConnect:      true,
		AutoReconnect:    true,
		AutoResubscribe:  true,
		AutoAuthenticate: true,

		ConnectTimeoutMs: 10000,

		MaxReconnectAttempts:       10,
		InitialReconnectDelayMs:    1000,
		MaxReconnectDelayMs:        30000,
		ReconnectBackoffMultiplier: 1.5,

		ResubscribeIntervalMs:       1000,
		AuthenticateRetries:         3,
		AuthenticateRetryIntervalMs: 1000,

		PingTimeoutMs: 10000,

		Debug:    false,
		LogLevel: "error",
	}
}

// OptionFunc applies partial configuration.
type OptionFunc func(*Option)

// WithAPIKey sets the API key credential.
func WithAPIKey(key string) OptionFunc {
	return func(o *Option) { o.APIKey = key }
}

// WithAutoConnect toggles auto-connect.
func WithAutoConnect(v bool) OptionFunc {
	return func(o *Option) { o.AutoConnect = v }
}

// WithIsSecure sets HTTP/WS TLS usage.
func WithIsSecure(v bool) OptionFunc {
	return func(o *Option) { o.IsSecure = v }
}

// FromOption merges a user-built Option (use DefaultOption() first) over defaults.
func FromOption(o Option) OptionFunc {
	return func(target *Option) {
		mergeOption(target, o)
	}
}

// Manager holds merged options.
type Manager struct {
	mu   sync.RWMutex
	opts Option
}

// NewManager creates an option manager from defaults + funcs.
func NewManager(funcs ...OptionFunc) *Manager {
	opts := DefaultOption()
	for _, f := range funcs {
		f(&opts)
	}
	return &Manager{opts: opts}
}

// NewManagerFrom merges defaults with a partial Option struct.
func NewManagerFrom(partial Option) *Manager {
	opts := DefaultOption()
	mergeOption(&opts, partial)
	return &Manager{opts: opts}
}

func mergeOption(dst *Option, src Option) {
	if src.APIKey != "" {
		dst.APIKey = src.APIKey
	}
	if src.AuthURL != "" {
		dst.AuthURL = src.AuthURL
	}
	if src.AuthOptions != nil {
		dst.AuthOptions = src.AuthOptions
	}
	if src.TokenRequest != nil {
		dst.TokenRequest = src.TokenRequest
	}
	if src.Alias != "" {
		dst.Alias = src.Alias
	}
	if src.HTTPHost != "" {
		dst.HTTPHost = src.HTTPHost
	}
	if src.HTTPPort != nil {
		dst.HTTPPort = src.HTTPPort
	}
	if src.WSHost != "" {
		dst.WSHost = src.WSHost
	}
	if src.WSPort != nil {
		dst.WSPort = src.WSPort
	}
	// Bool and secure flags are applied via OptionFunc to avoid partial-struct zero-value bugs.
	if src.ConnectTimeoutMs != 0 {
		dst.ConnectTimeoutMs = src.ConnectTimeoutMs
	}
	if src.MaxReconnectAttempts != 0 {
		dst.MaxReconnectAttempts = src.MaxReconnectAttempts
	}
	if src.InitialReconnectDelayMs != 0 {
		dst.InitialReconnectDelayMs = src.InitialReconnectDelayMs
	}
	if src.MaxReconnectDelayMs != 0 {
		dst.MaxReconnectDelayMs = src.MaxReconnectDelayMs
	}
	if src.ReconnectBackoffMultiplier != 0 {
		dst.ReconnectBackoffMultiplier = src.ReconnectBackoffMultiplier
	}
	if src.ResubscribeIntervalMs != 0 {
		dst.ResubscribeIntervalMs = src.ResubscribeIntervalMs
	}
	if src.AuthenticateRetries != 0 {
		dst.AuthenticateRetries = src.AuthenticateRetries
	}
	if src.AuthenticateRetryIntervalMs != 0 {
		dst.AuthenticateRetryIntervalMs = src.AuthenticateRetryIntervalMs
	}
	if src.PingTimeoutMs != 0 {
		dst.PingTimeoutMs = src.PingTimeoutMs
	}
	dst.Debug = src.Debug
	if src.LogLevel != "" {
		dst.LogLevel = src.LogLevel
	}
}

// Get returns a copy of all options.
func (m *Manager) Get() Option {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.opts
}

// Set merges partial options.
func (m *Manager) Set(partial Option) {
	m.mu.Lock()
	defer m.mu.Unlock()
	mergeOption(&m.opts, partial)
}

// Reset restores defaults (keeps nothing from prior state).
func (m *Manager) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.opts = DefaultOption()
}

// BuildRestBaseURL builds REST API base URL.
func BuildRestBaseURL(opts Option) string {
	protocol := "http"
	if opts.IsSecure {
		protocol = "https"
	}
	host := opts.HTTPHost
	if opts.HTTPPort != nil {
		return protocol + "://" + host + ":" + itoa(*opts.HTTPPort) + "/v1"
	}
	return protocol + "://" + host + "/v1"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [12]byte
	i := len(b)
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
