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
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *Error          `json:"error,omitempty"`
}

// request represents a JSON-RPC request received by a server or to be send by a client.
type request struct {
	ID             interface{}
	Method         string
	Params         json.RawMessage
	isNotification bool
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

func (req *request) marshal() rawMessage {
	return rawMessage{ID: req.ID, Method: req.Method, Params: req.Params}
}

// Response represents the Response from a JSON-RPC request.
type Response struct {
	id     interface{}
	result json.RawMessage
	error  *Error
}

func (r *Response) ID() interface{} {
	return r.id
}

func (r *Response) Err() error {
	if r.error == nil {
		return nil
	}
	return r.error
}

// Decode will unmarshal the Response's result into v. If there was an error in the Response, that error will be returned.
func (r *Response) Decode(v interface{}) error {
	if err := r.Err(); err != nil {
		return err
	}
	if err := json.Unmarshal(r.result, v); err != nil {
		return err
	}
	return nil
}

func (r *Response) encode(w io.Writer) error {
	msg := rawMessage{
		Version: "2.0",
		ID:      r.id,
		Result:  r.result,
		Error:   r.error,
	}
	if err := json.NewEncoder(w).Encode(msg); err != nil {
		return fmt.Errorf("marshaling jsonrpc message: %w", err)
	}
	return nil
}

func (res *Response) marshal() rawMessage {
	return rawMessage{ID: res.id, Result: res.result, Error: res.error}
}

func errResponse(id interface{}, err *Error) *Response {
	resp := &Response{id: id, result: nil, error: err}
	// If there was an error in detecting the id in the Request object, ID should be Null
	if id == nil {
		resp.id = null
	}
	return resp
}

// decodeResponseFromReader decodes a JSON-encoded response from r and stores it in resp.
func decodeResponseFromReader(r io.Reader, resp *Response) error {
	msg := &rawMessage{}
	if err := json.NewDecoder(r).Decode(msg); err != nil {
		return errInvalidEncodedJSON
	}
	result, err := json.Marshal(msg.Result)
	if err != nil || msg.Method != "" {
		resp.id = msg.ID
		return errInvalidDecodedMessage
	}

	resp.id = msg.ID
	resp.result = result
	resp.error = msg.Error

	return nil
}

// readRequest decodes a JSON-encoded body and returns a request message.
func readRequest(r io.Reader) (*request, error) {
	msg := &rawMessage{}
	if err := json.NewDecoder(r).Decode(msg); err != nil {
		return nil, errInvalidEncodedJSON
	}

	req := &request{ID: msg.ID, Method: msg.Method, Params: msg.Params}
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
