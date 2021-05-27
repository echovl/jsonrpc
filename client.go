package jsonrpc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync/atomic"
)

// Client represents a JSON-RPC Client.
type Client struct {
	next       int64
	url        string
	httpClient *http.Client
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
	}
	req := &Request{
		ID:     c.nextID(),
		Method: method,
		Params: p,
	}

	raw, err := encodeMessage(req)
	if err != nil {
		done <- fmt.Errorf("encoding jsonrpc request: %w", err)
	}

	data, err := c.send(ctx, raw)
	if err != nil {
		done <- fmt.Errorf("sending jsonrpc request: %w", err)
	}

	res, err := decodeResponse(data)
	if err != nil {
		done <- fmt.Errorf("decoding jsonrpc response: %w", err)
	}

	if err := json.Unmarshal(res.Result, reply); err != nil {
		done <- err
	}
	done <- nil
}

// send sends raw data to the http server and returns the response
func (c *Client) send(ctx context.Context, data []byte) ([]byte, error) {
	hreq, err := http.NewRequestWithContext(ctx, "POST", c.url, bytes.NewBuffer(data))
	hreq.Header.Set("Content-Type", "application/json")
	hreq.Header.Set("Accept", "application/json")

	hres, err := c.httpClient.Do(hreq)
	if err != nil {
		return nil, fmt.Errorf("failed sending request: %w", err)
	}
	defer hres.Body.Close()
	res, err := ioutil.ReadAll(hres.Body)
	if err != nil {
		return nil, fmt.Errorf("failed reading response body: %w", err)
	}
	return res, nil
}

// nextID returns the next id using atomic operations
func (c *Client) nextID() int64 {
	return atomic.AddInt64(&c.next, 1)
}
