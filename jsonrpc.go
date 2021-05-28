// TODO: add error struct and error codes
package jsonrpc

import "encoding/json"

var (
	errParseError     = &bodyError{-32700, "Parse error", nil}
	errInvalidRequest = &bodyError{-32600, "Invalid Request", nil}
	errMethodNotFound = &bodyError{-32601, "Method not found", nil}
	errInvalidParams  = &bodyError{-32602, "Invalid params", nil}
	errInternalError  = &bodyError{-32603, "Server error", nil}
	//errServerError    = bodyError{-32700, "Parse error", nil}
)

// TODO: add description
type body struct {
	Version string           `json:"jsonrpc"`
	ID      interface{}      `json:"id,omitempty"`
	Method  string           `json:"method,omitempty"`
	Params  *json.RawMessage `json:"params,omitempty"`
	Result  *json.RawMessage `json:"result,omitempty"`
	Error   *bodyError       `json:"error,omitempty"`
}

type bodyError struct {
	Code    int
	Message string
	Data    interface{} // defined by the server
}
