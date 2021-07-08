package jsonrpc

import (
	"encoding/json"
	"errors"
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

func (r *request) bytes() ([]byte, error) {
	msg := rawMessage{
		Version: "2.0",
		ID:      r.ID,
		Method:  r.Method,
		Params:  r.Params,
	}
	return json.Marshal(msg)
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

// bytes returns the JSON encoded representation of the Response.
func (r *Response) bytes() ([]byte, error) {
	msg := rawMessage{
		Version: "2.0",
		ID:      r.id,
		Result:  r.result,
		Error:   r.error,
	}
	return json.Marshal(msg)
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

// decodeRequestFromReader decodes a JSON-encoded body and returns a request message.
func decodeRequestFromReader(r io.Reader) (*request, error) {
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
