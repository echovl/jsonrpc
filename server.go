package jsonrpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go/token"
	"log"
	"net/http"
	"reflect"
	"sync"
)

var (
	typeOfError   = reflect.TypeOf((*error)(nil)).Elem()
	typeOfContext = reflect.TypeOf((*context.Context)(nil)).Elem()
)
var (
	errServerInvalidParams = errors.New("invalid request params type format")
	errServerInvalidOutput = errors.New("invalid return type format")
)

type Server struct {
	handler sync.Map
}

type handlerType struct {
	f       reflect.Value
	ptype   reflect.Type
	rtype   reflect.Type
	numArgs int
}

func NewServer() *Server {
	return &Server{}
}

// handler should be a func (params) (result, error)
// params and result should be an exported type (or builtin)
func (s *Server) HandleFunc(method string, handler interface{}) error {
	h := reflect.ValueOf(handler)
	numArgs, ptype, rtype, err := inspectHandler(h)
	if err != nil {
		return fmt.Errorf("jsonrpc: %v", err)
	}
	s.handler.Store(method, handlerType{f: h, ptype: ptype, rtype: rtype, numArgs: numArgs})
	return nil
}

func inspectHandler(h reflect.Value) (numArgs int, ptype, rtype reflect.Type, err error) {
	ht := h.Type()
	if hkind := h.Kind(); hkind != reflect.Func {
		err = fmt.Errorf("invalid handler type: expected func, got %v", hkind)
		return
	}

	numArgs = ht.NumIn()
	if numArgs != 2 && numArgs != 1 {
		err = fmt.Errorf("invalid number of args: expected %v, got %v", 2, ht.NumIn())
		return
	}

	if ctxType := ht.In(0); ctxType != typeOfContext {
		err = fmt.Errorf("invalid first arg type: expected context.Context, got %v", ctxType)
		return
	}

	if numArgs == 2 {
		ptype = ht.In(1)
		if !isExportedOrBuiltinType(ptype) {
			err = fmt.Errorf("invalid second arg type: expected exported or builtin")
			return
		}
	}

	if numOut := ht.NumOut(); numOut != 2 {
		err = fmt.Errorf("invalid number of returns: expected 2, got %v", numOut)
		return
	}

	rtype = ht.Out(0)
	if !isExportedOrBuiltinType(rtype) {
		err = fmt.Errorf("invalid first return type: expected exported or builtin")
		return
	}

	if errorType := ht.Out(1); errorType != typeOfError {
		err = fmt.Errorf("invalid second return type: expected error, got %v", errorType)
		return
	}
	return
}

func (s *Server) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	// Only POST methods are jsonrpc valid calls
	if r.Method != "POST" {
		rw.WriteHeader(http.StatusNotFound)
		rw.Write([]byte("Not found"))
		return
	}

	ctx := r.Context()
	req, err := readRequest(r.Body)
	defer r.Body.Close()
	if errors.Is(err, errInvalidEncodedJSON) {
		sendMessage(rw, errResponse(nil, &ErrorParseError))
		return
	}
	if errors.Is(err, errInvalidDecodedMessage) {
		sendMessage(rw, errResponse(req.ID, &ErrInvalidRequest))
		return
	}

	method, ok := s.handler.Load(req.Method)
	if !ok {
		sendMessage(rw, errResponse(req.ID, &ErrMethodNotFound))
		return
	}

	htype, _ := method.(handlerType)
	result, err := callMethod(ctx, req, htype)
	if errors.Is(err, errServerInvalidParams) {
		sendMessage(rw, errResponse(req.ID, &ErrInvalidParams))
		return
	}
	if errors.Is(err, errServerInvalidOutput) {
		sendMessage(rw, errResponse(req.ID, &ErrInternalError))
		return
	}
	if err, ok := err.(Error); ok {
		sendMessage(rw, errResponse(req.ID, &err))
		return
	}

	sendMessage(rw, &Response{
		ID:     req.ID,
		Error:  nil,
		Result: (*json.RawMessage)(&result),
	})
}

func sendMessage(rw http.ResponseWriter, msg message) {
	if err := writeMessage(rw, msg); err != nil {
		log.Printf("jsonrpc: sending response: %v", err)
	}
}

func callMethod(ctx context.Context, req *Request, htype handlerType) (json.RawMessage, error) {
	var outv []reflect.Value
	if htype.numArgs == 1 {
		outv = htype.f.Call([]reflect.Value{reflect.ValueOf(ctx)})
	} else {
		var pvalue, pzero reflect.Value
		pIsValue := false
		if htype.ptype.Kind() == reflect.Ptr {
			pvalue = reflect.New(htype.ptype.Elem())
			pzero = reflect.New(htype.ptype.Elem())
		} else {
			pvalue = reflect.New(htype.ptype)
			pzero = reflect.New(htype.ptype)
			pIsValue = true
		}

		// here pvalue is guaranteed to be a ptr
		// QUESTION: if pvalue doesnt change params should be invalid?
		if req.Params == nil {
			return nil, errServerInvalidParams
		}
		if err := json.Unmarshal(*req.Params, pvalue.Interface()); err != nil || reflect.DeepEqual(pzero, pvalue.Elem()) {
			return nil, errServerInvalidParams
		}

		if pIsValue {
			outv = htype.f.Call([]reflect.Value{reflect.ValueOf(ctx), pvalue.Elem()})
		} else {
			outv = htype.f.Call([]reflect.Value{reflect.ValueOf(ctx), pvalue})
		}
	}

	outErr := outv[1].Interface()
	switch err := outErr.(type) {
	case Error:
		return nil, err
	case error:
		return nil, Error{Code: -32000, Message: err.Error()}
	default:
	}

	result, err := json.Marshal(outv[0].Interface())
	if err != nil {
		// this should not happen if the output is well defined
		return nil, errServerInvalidOutput
	}
	return result, nil
}

func isExportedOrBuiltinType(t reflect.Type) bool {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	// PkgPath will be non-empty even for an exported type,
	// so we need to check the type name as well.
	return token.IsExported(t.Name()) || t.PkgPath() == ""
}
