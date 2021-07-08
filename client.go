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

// NewClient returns a new Client to handle requests to a JSON-RPC server.
// TODO: support custom httpClients
func NewClient(url string) *Client {
	return &Client{url: url, httpClient: http.DefaultClient}
}

// Call executes the named method, waits for it to complete, and returns a JSONRPC response.
func (c *Client) Call(ctx context.Context, method string, params interface{}) (*Response, error) {
	done := make(chan error)
	resp := &Response{}
	go c.call(ctx, method, params, resp, done)
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("jsonrpc: %v", ctx.Err())
	case err := <-done:
		return resp, err
	}
}

// Notify executes the named method and discards the response.
func (c *Client) Notify(ctx context.Context, method string, params interface{}) error {
	done := make(chan error)
	go c.notify(ctx, method, params, done)
	select {
	case <-ctx.Done():
		return fmt.Errorf("jsonrpc: %v", ctx.Err())
	case err := <-done:
		return err
	}
}

func (c *Client) notify(ctx context.Context, method string, params interface{}, done chan error) {
	p, err := json.Marshal(params)
	if err != nil {
		done <- fmt.Errorf("jsonrpc: marshaling params: %w", err)
		return
	}
	req := &request{ID: nil, Method: method, Params: p}
	rc, err := c.send(ctx, req)
	if err != nil {
		done <- fmt.Errorf("jsonrpc: sending request: %w", err)
		return
	}
	defer rc.Close()

	done <- nil
}

func (c *Client) call(ctx context.Context, method string, params interface{}, resp *Response, done chan error) {
	p, err := json.Marshal(params)
	if err != nil {
		done <- fmt.Errorf("jsonrpc: marshaling params: %w", err)
		return
	}
	req := &request{ID: c.nextID(), Method: method, Params: p}
	rc, err := c.send(ctx, req)
	if err != nil {
		done <- fmt.Errorf("jsonrpc: sending request: %w", err)
		return
	}
	defer rc.Close()

	if err := decodeResponseFromReader(rc, resp); err != nil {
		done <- fmt.Errorf("jsonrpc: reading response: %w", err)
		return
	}

	done <- nil
}

// send sends data from r to the http server and returns a reader of the response
func (c *Client) send(ctx context.Context, req *request) (io.ReadCloser, error) {
	b, err := req.bytes()
	if err != nil {
		return nil, err
	}
	hreq, err := http.NewRequestWithContext(ctx, "POST", c.url, bytes.NewBuffer(b))
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
func (c *Client) nextID() interface{} {
	return atomic.AddInt64(&c.next, 1)
}
