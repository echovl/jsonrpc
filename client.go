package jsonrpc

import (
	"bytes"
	"context"
	"encoding/json"
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
		return fmt.Errorf("waiting response: %w", ctx.Err())
	case err := <-done:
		return err
	}
}

// TODO: we should parse and send response errors to the done channel
func (c *Client) call(ctx context.Context, method string, params, reply interface{}, done chan error) {
	p, err := json.Marshal(params)
	if err != nil {
		done <- fmt.Errorf("marshaling jsonrpc params: %w", err)
		return
	}
	req := &Request{
		ID:     c.nextID(),
		Method: method,
		Params: (*json.RawMessage)(&p),
	}

	buf := &bytes.Buffer{}
	if err := encodeMessage(buf, req); err != nil {
		done <- fmt.Errorf("encoding jsonrpc request: %w", err)
		return
	}

	rc, err := c.send(ctx, buf)
	if err != nil {
		done <- fmt.Errorf("sending jsonrpc request: %w", err)
		return
	}
	defer rc.Close()

	res, err := decodeResponse(rc)
	if err != nil {
		done <- fmt.Errorf("decoding jsonrpc response: %w", err)
		return
	}

	if err := json.Unmarshal(*res.Result, reply); err != nil {
		done <- err
		return
	}
	done <- nil
}

// send sends data from r to the http server and returns a reader of the response
func (c *Client) send(ctx context.Context, r io.Reader) (io.ReadCloser, error) {
	hreq, err := http.NewRequestWithContext(ctx, "POST", c.url, r)
	hreq.Header.Set("Content-Type", "application/json")
	hreq.Header.Set("Accept", "application/json")

	hres, err := c.httpClient.Do(hreq)
	if err != nil {
		return nil, fmt.Errorf("failed sending request: %w", err)
	}
	return hres.Body, nil
}

// nextID returns the next id using atomic operations
func (c *Client) nextID() int64 {
	return atomic.AddInt64(&c.next, 1)
}
