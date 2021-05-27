package jsonrpc

import (
	"encoding/json"
	"fmt"
)

// message represents jsonrpc messages that can be marshal to a raw jsonrpc object
type message interface {
	marshal() body
}

func encodeMessage(msg message) ([]byte, error) {
	b := msg.marshal()
	b.Version = "2.0"
	data, err := json.Marshal(b)
	if err != nil {
		return nil, fmt.Errorf("marshaling jsonrpc message: %w", err)
	}
	return data, nil
}

// Request represents a JSON-RPC request received by a server or to be send by a client.
type Request struct {
	ID     interface{}
	Method string
	Params json.RawMessage
}

func (req *Request) marshal() body {
	return body{
		ID:     req.ID,
		Method: req.Method,
		Params: req.Params,
	}
}

// Request represents the response from a JSON-RPC request.
type Response struct {
	ID     interface{}
	Error  error
	Result json.RawMessage
}

func (res *Response) marshal() body {
	return body{
		ID:     res.ID,
		Result: res.Result,
		Error:  res.Error,
	}
}

// DecodeRequest decodes a JSON-encoded body and returns a response message.
func decodeResponse(data []byte) (Response, error) {
	msg := &body{}
	if err := json.Unmarshal(data, msg); err != nil {
		return Response{}, fmt.Errorf("unmarshaling jsonrpc message: %w", err)
	}
	// TODO: validate id following jsonrpc spec
	if msg.Method != "" {
		// if method is present, this is a request
		return Response{}, fmt.Errorf("malformed response: method present")
	}
	// TODO: parse error
	result, err := json.Marshal(msg.Result)
	if err != nil {
		return Response{}, fmt.Errorf("unmarshaling jsonrpc result :%w", err)
	}
	return Response{
		ID:     msg.ID,
		Error:  nil,
		Result: result,
	}, nil
}

// decodeRequest decodes a JSON-encoded body and returns a request message.
func decodeRequest(data []byte) (Request, error) {
	b := &body{}
	if err := json.Unmarshal(data, b); err != nil {
		return Request{}, fmt.Errorf("unmarshaling jsonrpc message: %w", err)
	}
	// TODO: validate id following jsonrpc spec
	if b.Method == "" {
		return Request{}, fmt.Errorf("malformed request: missing method")
	}
	return Request{
		ID:     b.ID,
		Method: b.Method,
		Params: b.Params,
	}, nil
}
