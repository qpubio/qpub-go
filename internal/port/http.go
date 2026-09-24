package port

import "context"

// HTTPClient is the HTTP port used by application services.
type HTTPClient interface {
	Get(ctx context.Context, url string, headers map[string]string) ([]byte, int, error)
	Post(ctx context.Context, url string, body interface{}, headers map[string]string) ([]byte, int, error)
	Put(ctx context.Context, url string, body interface{}, headers map[string]string) ([]byte, int, error)
	Delete(ctx context.Context, url string, headers map[string]string) ([]byte, int, error)
}
