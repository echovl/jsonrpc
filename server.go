package jsonrpc

import (
	"encoding/json"
	"fmt"
	"go/token"
	"io/ioutil"
	"log"
	"net/http"
	"reflect"
)

var typeOfError = reflect.TypeOf((*error)(nil)).Elem()

type Server struct {
	handlers map[string]handlerType
	path     string
}

type Version struct {
	Tag string
}

type handlerType struct {
	function   reflect.Value
	paramsType reflect.Type
	resultType reflect.Type
}

func NewServer(path string) *Server {
	return &Server{handlers: make(map[string]handlerType), path: path}
}

func (s *Server) HandleFunc(method string, handler func(Request) (interface{}, error)) {
	fmt.Printf("registering func for method %v\n", method)
}

// handler should be a func (params) (result, error)
// params and result should be an exported type (or builtin)
func (s *Server) HandleFunc2(method string, handler interface{}) {
	fmt.Printf("handlefunc2: registering func for method %v\n", method)

	// validate handler func
	h := reflect.ValueOf(handler)
	ht := h.Type()

	// handler should be a func/method
	if h.Kind() != reflect.Func {
		panic("invalid handler: should be of type func")
	}

	// handler should have one arg
	if ht.NumIn() != 1 {
		panic("invalid number of args: should be one")
	}

	// arg should be exported or builtin
	paramsType := ht.In(0)
	if !isExportedOrBuiltinType(paramsType) {
		panic("invalid argtype")
	}

	// handler should have two returns
	if ht.NumOut() != 2 {
		panic("invalid number of returns: should be two")
	}

	// first return should be exported or builtin
	resultType := ht.Out(0)
	if !isExportedOrBuiltinType(resultType) {
		panic("invalid resultType")
	}

	// second return should be an error
	if errorType := ht.Out(1); errorType != typeOfError {
		panic("invalid errorType")
	}

	fmt.Println("func def: ", paramsType, resultType)

	htype := handlerType{
		function:   h,
		paramsType: paramsType,
		resultType: resultType,
	}

	s.handlers[method] = htype
}

func (s *Server) ListenAndServe(addr string) {
	http.HandleFunc(s.path, func(rw http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			rw.WriteHeader(http.StatusNotFound)
			rw.Write([]byte("not found"))
		}

		// read method
		b, _ := ioutil.ReadAll(r.Body)
		defer r.Body.Close()
		req, err := decodeRequest(b)
		if err != nil {
			//handle decode error
			log.Fatal(err)
		}

		fmt.Println("handling method", req.Method)

		// get handler type
		htype, ok := s.handlers[req.Method]
		if !ok {
			// handle method not found
			rw.WriteHeader(http.StatusNotFound)
			rw.Write([]byte("not found"))
		}

		fmt.Println(htype.paramsType, htype.resultType)

		// build params and call the handler func
		//params := reflect.Indirect(reflect.Zero(htype.paramsType))
		json.Unmarshal(req.Params, nil)

		rw.Write([]byte(`{"jsonrpc": "2.0", "result": {"message":"hello"}, "id": 1}`))
	})

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
