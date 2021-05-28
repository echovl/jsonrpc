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
	path    string
}

type Version struct {
	Tag string
}

type handlerType struct {
	f     reflect.Value
	ptype reflect.Type
	rtype reflect.Type
}

func NewServer(path string) *Server {
	return &Server{path: path}
}

// handler should be a func (params) (result, error)
// params and result should be an exported type (or builtin)
func (s *Server) HandleFunc(method string, handler interface{}) error {
	h := reflect.ValueOf(handler)
	ptype, rtype, err := inspectHandler(h)
	if err != nil {
		return fmt.Errorf("inspecting handler: %v", err)
	}

	s.handler.Store(method, handlerType{h, ptype, rtype})

	return nil
}

// TODO: add context support, something like this: func (ctx, params) (result, error)
func inspectHandler(h reflect.Value) (ptype, rtype reflect.Type, err error) {
	ht := h.Type()
	if hkind := h.Kind(); hkind != reflect.Func {
		err = fmt.Errorf("invalid handler type: expected %v, got %v", "func", hkind)
		return
	}

	if ht.NumIn() != 1 {
		err = fmt.Errorf("invalid number of args: expected %v, got %v", 1, ht.NumIn())
		return
	}

	ptype = ht.In(0)
	if !isExportedOrBuiltinType(ptype) {
		err = fmt.Errorf("invalid arg type: expected exported or builtin")
		return
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
	log.Printf("handler = func (%v) (%v, error)", ptype, rtype)
	return
}

func (s *Server) handleHTTP(rw http.ResponseWriter, r *http.Request) {
	// Only POST methods are jsonrpc valid calls
	if r.Method != "POST" {
		rw.WriteHeader(http.StatusNotFound)
		rw.Write([]byte("not found"))
		return
	}

	req, err := decodeRequest(r.Body)
	defer r.Body.Close()
	if err != nil {
		if err := encodeMessage(rw, newErrorResponse(nil, errInvalidRequest)); err != nil {
			log.Printf("decoding request message: %v", err)
		}
		return
	}

	log.Printf("request:%v, %v", req, err)

	method, ok := s.handler.Load(req.Method)
	if !ok {
		log.Printf("method %v not found", req.Method)
		if err := encodeMessage(rw, newErrorResponse(req.ID, errMethodNotFound)); err != nil {
			log.Printf("encoding err message: %v", err)
		}
		return
	}

	htype, _ := method.(handlerType)
	result, err := callMethod(req, htype)
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

func callMethod(req *Request, htype handlerType) (json.RawMessage, error) {
	var pvalue reflect.Value
	pIsValue := false
	if htype.ptype.Kind() == reflect.Ptr {
		pvalue = reflect.New(htype.ptype.Elem())
	} else {
		pvalue = reflect.New(htype.ptype)
		pIsValue = true
	}
	pzero := pvalue.Elem()

	// here pvalue is guaranteed to be a ptr
	// QUESTION: if pvalue doesnt change params should be invalid?
	if err := json.Unmarshal(*req.Params, pvalue.Interface()); err != nil || reflect.DeepEqual(pzero, pvalue.Elem()) {
		// invalid params?
		log.Printf("invalid params: %v", string(*req.Params))
		return nil, errServerMarshalParams
	}

	var outv []reflect.Value
	if pIsValue {
		outv = htype.f.Call([]reflect.Value{pvalue.Elem()})
	} else {
		outv = htype.f.Call([]reflect.Value{pvalue})
	}

	result, err := json.Marshal(outv[0].Interface())
	if err != nil {
		// internal error?
		return nil, errServerUnmarshalReturn
	}
	return result, nil
}

func (s *Server) ListenAndServe(addr string) {
	http.HandleFunc(s.path, s.handleHTTP)

	http.ListenAndServe(addr, nil)
}

func isExportedOrBuiltinType(t reflect.Type) bool {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	// PkgPath will be non-empty even for an exported type,
	// so we need to check the type name as well.
	return token.IsExported(t.Name()) || t.PkgPath() == ""
}
