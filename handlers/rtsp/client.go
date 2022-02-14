package rtsp

import (
	"fmt"
	"net"
	"strconv"
	"sync/atomic"
)

// Client Rtsp client
type Client struct {
	conn net.Conn
	seq  int64
}

// NewClient instantiates a new client connecting to the address specified
func NewClient(address string, port int) (*Client, error) {
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", address, port))
	if err != nil {
		return nil, err
	}
	return &Client{conn: conn, seq: 1}, nil
}

// Send will send a request to the server
func (c *Client) Send(request *Request) (*Response, error) {
	request.Headers["CSeq"] = strconv.FormatInt(c.seq, 10)
	request.Headers["User-Agent"] = "Bobcaygeon/1.0"
	atomic.AddInt64(&c.seq, 1)
	_, err := writeRequest(c.conn, request)
	if err != nil {
		return nil, err
	}
	resp, err := readResponse(c.conn)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// LocalAddress returns the local (our) address
func (c *Client) LocalAddress() string {
	return c.conn.LocalAddr().(*net.TCPAddr).IP.String()
}

// RemoteAddress returns the remote address
func (c *Client) RemoteAddress() string {
	return c.conn.RemoteAddr().(*net.TCPAddr).IP.String()
}

func (c *Client) Close() error {
	return c.conn.Close()
}
