package jsonrpc

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
)

// message represents jsonrpc messages that can be marshal to a raw jsonrpc object
type message interface {
	marshal() body
}

func encodeMessage(w io.Writer, msg message) error {
	b := msg.marshal()
	b.Version = "2.0"
	log.Printf("%v", msg)
	if err := json.NewEncoder(w).Encode(msg); err != nil {
		return fmt.Errorf("marshaling jsonrpc message: %w", err)
	}
	return nil
}

// Request represents a JSON-RPC request received by a server or to be send by a client.
type Request struct {
	ID     interface{}
	Method string
	Params *json.RawMessage
}

func (req *Request) marshal() body {
	return body{ID: req.ID, Method: req.Method, Params: req.Params}
}

// Request represents the response from a JSON-RPC request.
type Response struct {
	ID     interface{}
	Result *json.RawMessage
	Error  *bodyError
}

func (res *Response) marshal() body {
	return body{ID: res.ID, Result: res.Result, Error: res.Error}
}

func newErrorResponse(id interface{}, err *bodyError) *Response {
	return &Response{ID: id, Result: nil, Error: err}
}

// DecodeRequest decodes a JSON-encoded body and returns a response message.
func decodeResponse(r io.Reader) (*Response, error) {
	msg := &body{}
	if err := json.NewDecoder(r).Decode(msg); err != nil {
		return nil, fmt.Errorf("unmarshaling jsonrpc message: %w", err)
	}
	// TODO: validate id following jsonrpc spec
	if msg.Method != "" {
		// if method is present, this is a request
		return nil, fmt.Errorf("malformed response: method present")
	}
	// TODO: parse error
	result, err := json.Marshal(msg.Result)
	if err != nil {
		return nil, fmt.Errorf("unmarshaling jsonrpc result :%w", err)
	}
	return &Response{ID: msg.ID, Result: (*json.RawMessage)(&result), Error: nil}, nil
}

// decodeRequest decodes a JSON-encoded body and returns a request message.
func decodeRequest(r io.Reader) (*Request, error) {
	b := &body{}
	if err := json.NewDecoder(r).Decode(b); err != nil {
		return nil, fmt.Errorf("unmarshaling jsonrpc message: %w", err)
	}
	// TODO: validate id following jsonrpc spec
	if b.Method == "" {
		return nil, fmt.Errorf("malformed request: missing method")
	}
	return &Request{ID: b.ID, Method: b.Method, Params: b.Params}, nil
}
