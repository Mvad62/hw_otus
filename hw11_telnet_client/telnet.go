package main

import (
	"errors"
	"io"
	"net"
	"sync"
	"time"
)

var errNotConnected = errors.New("telnet client is not connected")

type TelnetClient interface {
	Connect() error
	io.Closer
	Send() error
	Receive() error
}

func NewTelnetClient(address string, timeout time.Duration, in io.ReadCloser, out io.Writer) TelnetClient {
	return &telnetClient{
		address: address,
		timeout: timeout,
		in:      in,
		out:     out,
	}
}

type telnetClient struct {
	address string
	timeout time.Duration
	in      io.ReadCloser
	out     io.Writer

	mu   sync.Mutex
	conn net.Conn
}

func (c *telnetClient) Connect() error {
	conn, err := net.DialTimeout("tcp", c.address, c.timeout)
	if err != nil {
		return err
	}

	c.mu.Lock()
	old := c.conn
	c.conn = conn
	c.mu.Unlock()

	if old != nil {
		_ = old.Close()
	}
	return nil
}

func (c *telnetClient) Close() error {
	c.mu.Lock()
	conn := c.conn
	c.conn = nil
	c.mu.Unlock()

	if conn == nil {
		return nil
	}
	return conn.Close()
}

func (c *telnetClient) Send() error {
	c.mu.Lock()
	conn := c.conn
	c.mu.Unlock()
	if conn == nil {
		return errNotConnected
	}

	_, err := io.Copy(conn, c.in)
	return err
}

func (c *telnetClient) Receive() error {
	c.mu.Lock()
	conn := c.conn
	c.mu.Unlock()
	if conn == nil {
		return errNotConnected
	}

	_, err := io.Copy(c.out, conn)
	return err
}
