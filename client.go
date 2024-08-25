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

// Destroy closes the connection to the server.
func (c *Client) Destroy() {
	c.conn.Close()
}

// SetCallback sets a callback which is executed when the client receives
// data from the server.
func (c *Client) SetCallback(cb func([]byte)) *Client {
	c.callback = cb
	return c
}

// Connect connects to the server.
func (c *Client) Connect(addr string, port uint16) error {
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
				c.Destroy()
				panic(err)
			}
			c.callback(message)
		}
	}()

	// Send heartbeat to server
	go func() {
		for {
			if _, err := c.conn.Write([]byte(heartbeatMsg)); err != nil {
				c.Destroy()
				panic(err)
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
