package jsonrpc

import (
	"log"
	"testing"
)

type Api struct{ state int }

func (api Api) Version(app string) (Version, error) {
	return Version{"3.0.0"}, nil
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
	s := NewServer("/api")
	s.HandleFunc("version2", func(app string) (Version, error) {
		log.Println("server called with", app)
		return Version{"2.0.0"}, nil
	})

	api := Api{}
	s.HandleFunc("version3", api.Version)
	s.ListenAndServe(":8080")
}
