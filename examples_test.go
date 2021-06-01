package jsonrpc_test

import (
	"context"
	"net/http"

	"github.com/echovl/jsonrpc"
)

type Version struct {
	Tag string
}

func ExampleServer() {
	server := jsonrpc.NewServer()
	err := server.HandleFunc("version", func(ctx context.Context) (Version, error) {
		return Version{"1.0.0"}, nil
	})
	if err != nil {
		panic(err)
	}

	http.Handle("/api", server)
	http.ListenAndServe(":4545", nil)
}
