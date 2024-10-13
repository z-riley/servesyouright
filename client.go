package turdserve

import (
	"bufio"
	"fmt"
	"net"
	"time"
)

// Client can communicate with a turdserve server.
type Client struct {
	conn     net.Conn
	callback func([]byte)
}

// NewClient constructs a new client. Call Connect to connect to a server.
func NewClient() *Client {
	return &Client{
		conn:     nil,
		callback: func([]byte) {},
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
func (c *Client) Connect(addr string, port uint16, errCh chan error) error {
	var err error
	c.conn, err = net.Dial("tcp", fmt.Sprintf("%s:%d", addr, port))
	if err != nil {
		return err
	}

	// Execute callback on receive
	go func() {
		reader := bufio.NewReader(c.conn)
		for {
			message, err := reader.ReadBytes(delimChar)
			if err != nil {
				errCh <- fmt.Errorf("failed to read message from server: %w", err)
				return
			}
			c.callback(message)
		}
	}()

	// Send heartbeat to server
	go func() {
		for {
			if _, err := c.conn.Write([]byte(heartbeatMsg)); err != nil {
				errCh <- fmt.Errorf("failed to send heartbeat to server: %w", err)
				c.conn.Close()
				return
			}
			time.Sleep(heartbeatInterval)
		}
	}()

	return nil
}

// Write sends data to the server.
func (c *Client) Write(b []byte) error {
	_, err := c.conn.Write(append(b, delimChar))
	return err
}
