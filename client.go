package jsonrpc

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync/atomic"
)

// Client represents a JSON-RPC Client.
type Client struct {
	next       int64
	url        string
	httpClient httpClient
}

type httpClient interface {
	Do(*http.Request) (*http.Response, error)
}

var errClientContextCanceled = errors.New("context canceled by the client")

// NewClient returns a new Client to handle requests to the JSON-RPC server at the other end of the connection.
// TODO: support custom httpClients
func NewClient(url string) *Client {
	return &Client{url: url, httpClient: http.DefaultClient}
}

func (c *Client) Call(ctx context.Context, method string, params, reply interface{}) error {
	done := make(chan error)
	go c.call(ctx, method, params, reply, done)
	select {
	case <-ctx.Done():
		return fmt.Errorf("jsonrpc: %v", ctx.Err())
	case err := <-done:
		return err
	}
}

func (c *Client) call(ctx context.Context, method string, params, reply interface{}, done chan error) {
	p, err := json.Marshal(params)
	if err != nil {
		done <- fmt.Errorf("jsonrpc: marshaling params: %w", err)
		return
	}
	req := &Request{ID: c.nextID(), Method: method, Params: (*json.RawMessage)(&p)}

	buf := &bytes.Buffer{}
	if err := writeMessage(buf, req); err != nil {
		done <- fmt.Errorf("jsonrpc: encoding request: %w", err)
		return
	}

	rc, err := c.send(ctx, buf)
	if err != nil {
		done <- fmt.Errorf("jsonrpc: sending request: %w", err)
		return
	}
	defer rc.Close()

	res, err := readResponse(rc)
	if err != nil {
		done <- fmt.Errorf("jsonrpc: reading response: %w", err)
		return
	}
	if res.Err() != nil {
		done <- res.Err()
		return
	}

	if err := json.Unmarshal(*res.Result, reply); err != nil {
		done <- fmt.Errorf("jsonrpc: unmarshaling result: %w", err)
		return
	}
	done <- nil
}

// send sends data from r to the http server and returns a reader of the response
func (c *Client) send(ctx context.Context, r io.Reader) (io.ReadCloser, error) {
	hreq, err := http.NewRequestWithContext(ctx, "POST", c.url, r)
	if err != nil {
		return nil, err
	}
	hreq.Header.Set("Content-Type", "application/json")
	hreq.Header.Set("Accept", "application/json")

	hres, err := c.httpClient.Do(hreq)
	if err != nil {
		return nil, err
	}
	return hres.Body, nil
}

// nextID returns the next id using atomic operations
func (c *Client) nextID() int64 {
	return atomic.AddInt64(&c.next, 1)
}
