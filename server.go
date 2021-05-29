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
	errServerMarshalParams   = errors.New("invalid request params type format")
	errServerUnmarshalReturn = errors.New("invalid return type format")
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
		err = fmt.Errorf("invalid handler type: expected %v, got %v", "func", hkind)
		return
	}

	numArgs = ht.NumIn()
	if numArgs != 2 && numArgs != 1 {
		err = fmt.Errorf("invalid number of args: expected %v, got %v", 2, ht.NumIn())
		return
	}

	if ctxType := ht.In(0); ctxType != typeOfContext {
		err = fmt.Errorf("invalid arg[0] type: should be context.Context")
		return
	}

	if numArgs == 2 {
		ptype = ht.In(1)
		if !isExportedOrBuiltinType(ptype) {
			err = fmt.Errorf("invalid arg type: expected exported or builtin")
			return
		}
	}

	if numOut := ht.NumOut(); numOut != 2 {
		err = fmt.Errorf("invalid number of returns: expected 2, got %v", numOut)
		return
	}

	rtype = ht.Out(0)
	if !isExportedOrBuiltinType(rtype) {
		err = fmt.Errorf("invalid return type: expected exported or builtin")
		return
	}

	if errorType := ht.Out(1); errorType != typeOfError {
		err = fmt.Errorf("invalid error type, should be exported or builtin")
		return
	}
	return
}

func (s *Server) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	// Only POST methods are jsonrpc valid calls
	if r.Method != "POST" {
		rw.WriteHeader(http.StatusNotFound)
		rw.Write([]byte("not found"))
		return
	}

	ctx := r.Context()
	req, err := decodeRequest(r.Body)
	defer r.Body.Close()
	if err != nil {
		if err := encodeMessage(rw, newErrorResponse(nil, errInvalidRequest)); err != nil {
			log.Printf("decoding request message: %v", err)
		}
		return
	}

	method, ok := s.handler.Load(req.Method)
	if !ok {
		log.Printf("method %v not found", req.Method)
		if err := encodeMessage(rw, newErrorResponse(req.ID, errMethodNotFound)); err != nil {
			log.Printf("encoding err message: %v", err)
		}
		return
	}

	htype, _ := method.(handlerType)
	result, err := callMethod(ctx, req, htype)
	if err != nil && err == errServerMarshalParams {
		if err := encodeMessage(rw, newErrorResponse(req.ID, errInvalidParams)); err != nil {
			log.Printf("encoding err message: %v", err)
		}
		return
	}
	if err != nil && err == errServerUnmarshalReturn {
		if err := encodeMessage(rw, newErrorResponse(req.ID, errInternalError)); err != nil {
			log.Printf("encoding err message: %v", err)
		}
		return
	}
	resp := &Response{ID: req.ID, Error: nil, Result: (*json.RawMessage)(&result)}
	if err := encodeMessage(rw, resp); err != nil {
		log.Printf("encoding message to http.ResponseWriter: %v", err)
	}
}

func callMethod(ctx context.Context, req *Request, htype handlerType) (json.RawMessage, error) {
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
	if err := json.Unmarshal(*req.Params, pvalue.Interface()); err != nil || reflect.DeepEqual(pzero, pvalue.Elem()) {
		// invalid params?
		log.Printf("invalid params: %v, %v", string(*req.Params), err)
		return nil, errServerMarshalParams
	}

	var outv []reflect.Value
	if pIsValue {
		outv = htype.f.Call([]reflect.Value{reflect.ValueOf(ctx), pvalue.Elem()})
	} else {
		outv = htype.f.Call([]reflect.Value{reflect.ValueOf(ctx), pvalue})
	}

	result, err := json.Marshal(outv[0].Interface())
	if err != nil {
		// internal error?
		return nil, errServerUnmarshalReturn
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
