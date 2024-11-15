package turdserve

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"time"
)

// Client can communicate with a turdserve server.
type Client struct {
	// ConnectTimeout is the time spent attempting to connect to the server.
	ConnectTimeout time.Duration

	conn     net.Conn
	callback func([]byte) // executed on receipt of data
}

// NewClient constructs a new client. Call Connect to connect to a server.
func NewClient() *Client {
	return &Client{
		ConnectTimeout: 5 * time.Second,
		conn:           nil,
		callback: func([]byte) {
		},
	}
}

// Destroy closes an existing connection to the server.
func (c *Client) Destroy() {
	if c.conn != nil {
		c.conn.Close()
	}
}

// SetCallback sets a callback which is executed when the client receives
// data from the server.
func (c *Client) SetCallback(cb func([]byte)) *Client {
	c.callback = cb
	return c
}

// Connect connects to the server. An error is returned if the initial connection
// fails. Subsequent errors are sent down the error channel.
//
// The error channel should be opened before Connect is called.
func (c *Client) Connect(ctx context.Context, addr string, port uint16, errCh chan error) error {
	var err error
	d := net.Dialer{
		Timeout: c.ConnectTimeout,
	}
	c.conn, err = d.DialContext(ctx, "tcp", fmt.Sprintf("%s:%d", addr, port))
	if err != nil {
		return err
	}

	// Execute callback on receive
	go func(ctx context.Context) {
		reader := bufio.NewReader(c.conn)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				message, err := reader.ReadBytes(delimChar)
				if err != nil {
					if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
						return
					}
					errCh <- fmt.Errorf("failed to read message from server: %w", err)
					return
				}
				c.callback(message)
			}
		}
	}(ctx)

	// Send heartbeat to server
	go func(ctx context.Context) {
		tick := time.Tick(heartbeatInterval)
		for {
			select {
			case <-ctx.Done():
				return
			case <-tick:
				if _, err := c.conn.Write([]byte(heartbeatMsg)); err != nil {
					errCh <- fmt.Errorf("failed to send heartbeat to server: %w", err)
					c.conn.Close()
					return
				}
			}
		}
	}(ctx)

	return nil
}

// Write sends data to the server.
func (c *Client) Write(b []byte) error {
	if c.conn == nil {
		return errors.New("no connection exists")
	}

	_, err := c.conn.Write(append(b, delimChar))
	return err
}
