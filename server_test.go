package jsonrpc

import (
	"encoding/json"
	"fmt"
	"testing"
)

type Person struct {
	Name string
	Age  int
}

func (p Person) Greet(name string) {
	fmt.Println("hi", name)
}

type Api struct{ state int }

func (api Api) Version(app string) (string, error) {
	fmt.Printf("api address %p\n", &api)
	fmt.Println(api.state)
	api.state++
	return "1.0.0", nil
}

func TestServer(t *testing.T) {
	// how http works
	/*
		http.HandleFunc("method1", func(rw http.ResponseWriter, r *http.Request) {
			rw.Write([]byte("hello mate!"))
		})

		http.ListenAndServe("127.0.0.1:8080", nil)
	*/

	// how jsonrpc server could work
	s := NewServer()
	s.HandleFunc("version", func(r Request) (interface{}, error) {
		return "1.0.0", nil
	})

	s.HandleFunc("echo", func(r Request) (interface{}, error) {
		var msg string
		err := json.Unmarshal(r.Params, &msg)
		if err != nil {
			return nil, err
		}
		return msg, nil
	})

	s.HandleFunc2("version2", func(params Person) (Version, error) {
		fmt.Println("server called with", params)
		return Version{"2.0.0"}, nil
	})

	api := Api{}
	s.HandleFunc2("version3", api.Version)

	fmt.Println(s.handlers)
}
