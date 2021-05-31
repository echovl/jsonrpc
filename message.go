package jsonrpc

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
)

var (
	errInvalidEncodedJSON    = errors.New("invalid encoded json")
	errInvalidDecodedMessage = errors.New("invalid decoded message")
)

// message represents jsonrpc messages that can be marshal to a raw jsonrpc object
type message interface {
	marshal() rawMessage
}

func writeMessage(w io.Writer, msg message) error {
	b := msg.marshal()
	b.Version = "2.0"
	if err := json.NewEncoder(w).Encode(b); err != nil {
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

func (req *Request) marshal() rawMessage {
	return rawMessage{ID: req.ID, Method: req.Method, Params: req.Params}
}

// Request represents the response from a JSON-RPC request.
type Response struct {
	ID     interface{}
	Result *json.RawMessage
	Error  *Error
}

func (res *Response) Err() error {
	if res.Error == nil {
		return nil
	}
	return *res.Error
}

func (res *Response) marshal() rawMessage {
	return rawMessage{ID: res.ID, Result: res.Result, Error: res.Error}
}

func errResponse(id interface{}, err *Error) *Response {
	return &Response{ID: id, Result: nil, Error: err}
}

// DecodeRequest decodes a JSON-encoded body and returns a response message.
func readResponse(r io.Reader) (*Response, error) {
	msg := &rawMessage{}
	if err := json.NewDecoder(r).Decode(msg); err != nil {
		return nil, errInvalidEncodedJSON
	}
	// TODO: validate id following jsonrpc spec
	// TODO: parse error
	result, err := json.Marshal(msg.Result)
	if err != nil || msg.Method != "" {
		return &Response{ID: msg.ID}, errInvalidDecodedMessage
	}
	return &Response{ID: msg.ID, Result: (*json.RawMessage)(&result), Error: msg.Error}, nil
}

// readRequest decodes a JSON-encoded body and returns a request message.
func readRequest(r io.Reader) (*Request, error) {
	msg := &rawMessage{}
	if err := json.NewDecoder(r).Decode(msg); err != nil {
		return nil, errInvalidEncodedJSON
	}
	// TODO: validate id following jsonrpc spec
	if msg.Method == "" || isValidID(msg.ID) {
		return &Request{ID: msg.ID}, errInvalidDecodedMessage
	}
	return &Request{ID: msg.ID, Method: msg.Method, Params: msg.Params}, nil
}

func isValidID(id interface{}) bool {
	if id == nil {
		return true
	}

	switch reflect.ValueOf(id).Kind() {
	case reflect.String, reflect.Int, reflect.Float32:
		return true
	default:
		return false
	}
}
