package jsonrpc

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

var (
	errInvalidEncodedJSON    = errors.New("invalid encoded json")
	errInvalidDecodedMessage = errors.New("invalid decoded message")
	null                     = json.RawMessage([]byte("null"))
)

type rawMessage struct {
	Version string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *Error          `json:"error,omitempty"`
}

// Request represents a JSON-RPC request received by a server or to be send by a client.
type Request struct {
	ID             json.RawMessage
	Method         string
	Params         json.RawMessage
	isNotification bool
}

// Request represents the response from a JSON-RPC request.
type Response struct {
	ID     json.RawMessage
	Result json.RawMessage
	Error  *Error
}

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

func (req *Request) marshal() rawMessage {
	return rawMessage{ID: req.ID, Method: req.Method, Params: req.Params}
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

func errResponse(id json.RawMessage, err *Error) *Response {
	resp := &Response{ID: id, Result: nil, Error: err}
	// If there was an error in detecting the id in the Request object, ID should be Null
	if id == nil {
		resp.ID = null
	}
	return resp
}

// readResponse decodes a JSON-encoded body and returns a response message.
func readResponse(r io.Reader) (*Response, error) {
	msg := &rawMessage{}
	if err := json.NewDecoder(r).Decode(msg); err != nil {
		return nil, errInvalidEncodedJSON
	}
	result, err := json.Marshal(msg.Result)
	if err != nil || msg.Method != "" {
		return &Response{ID: msg.ID}, errInvalidDecodedMessage
	}
	return &Response{ID: msg.ID, Result: (json.RawMessage)(result), Error: msg.Error}, nil
}

// readRequest decodes a JSON-encoded body and returns a request message.
func readRequest(r io.Reader) (*Request, error) {
	msg := &rawMessage{}
	if err := json.NewDecoder(r).Decode(msg); err != nil {
		return nil, errInvalidEncodedJSON
	}

	req := &Request{ID: msg.ID, Method: msg.Method, Params: msg.Params}
	if msg.ID == nil {
		req.isNotification = true
	}
	//id, ok := parseID(msg.ID)
	if msg.Method == "" {
		return req, errInvalidDecodedMessage
	}
	return req, nil
}

func parseID(id interface{}) (interface{}, bool) {
	if id == nil {
		return nil, true
	}

	switch v := id.(type) {
	case float32:
		return int(v), true
	case float64:
		return int(v), true
	case string:
		return v, true
	default:
		return nil, false
	}
}
