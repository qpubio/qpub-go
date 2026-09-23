package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client is the HTTP transport used by REST and auth.
type Client struct {
	http *http.Client
}

func New() *Client {
	return &Client{http: &http.Client{Timeout: 60 * time.Second}}
}

func NewWithHTTP(h *http.Client) *Client {
	return &Client{http: h}
}

func (c *Client) Get(ctx context.Context, url string, headers map[string]string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	if resp.StatusCode >= 400 {
		return body, resp.StatusCode, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}
	return body, resp.StatusCode, nil
}

func (c *Client) Post(ctx context.Context, url string, body interface{}, headers map[string]string) ([]byte, int, error) {
	return c.doJSON(ctx, http.MethodPost, url, body, headers)
}

func (c *Client) Put(ctx context.Context, url string, body interface{}, headers map[string]string) ([]byte, int, error) {
	return c.doJSON(ctx, http.MethodPut, url, body, headers)
}

func (c *Client) Delete(ctx context.Context, url string, headers map[string]string) ([]byte, int, error) {
	return c.doJSON(ctx, http.MethodDelete, url, nil, headers)
}

func (c *Client) doJSON(ctx context.Context, method, url string, body interface{}, headers map[string]string) ([]byte, int, error) {
	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, 0, err
		}
		r = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, r)
	if err != nil {
		return nil, 0, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	if resp.StatusCode >= 400 {
		return respBody, resp.StatusCode, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}
	return respBody, resp.StatusCode, nil
}

// DecodeJSON unmarshals response bytes.
func DecodeJSON[T any](body []byte, out *T) error {
	if len(body) == 0 {
		return nil
	}
	return json.Unmarshal(body, out)
}
