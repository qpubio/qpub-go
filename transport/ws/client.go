package ws

import (
	"sync"

	"github.com/gorilla/websocket"
	"github.com/qpubio/qpub-go/internal/logger"
)

// Client wraps gorilla WebSocket.
type Client struct {
	log     *logger.Logger
	mu      sync.Mutex
	writeMu sync.Mutex // serializes WriteMessage (gorilla allows one writer)
	conn    *websocket.Conn
	dialer  websocket.Dialer
}

func New(log *logger.Logger) *Client {
	return &Client{log: log}
}

func (c *Client) Connect(url string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != nil {
		_ = c.conn.Close()
		c.conn = nil
	}
	conn, _, err := c.dialer.Dial(url, nil)
	if err != nil {
		return err
	}
	c.conn = conn
	return nil
}

func (c *Client) Conn() *websocket.Conn {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn
}

func (c *Client) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn != nil
}

func (c *Client) Send(data []byte) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	c.mu.Lock()
	conn := c.conn
	c.mu.Unlock()
	if conn == nil {
		return websocket.ErrCloseSent
	}
	return conn.WriteMessage(websocket.TextMessage, data)
}

func (c *Client) Disconnect() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != nil {
		_ = c.conn.Close()
		c.conn = nil
	}
}

func (c *Client) Reset() {
	c.Disconnect()
}
