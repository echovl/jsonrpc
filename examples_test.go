package jsonrpc_test

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/echovl/jsonrpc"
)

type Version struct {
	Tag string
}

func ExampleServer() {
	done := make(chan struct{})
	s := jsonrpc.NewServer()

	go startServer(s, done)

	client := jsonrpc.NewClient("http://127.0.0.1:8080/api")
	reply := &Version{}

	err := client.Call(context.Background(), "version", "client", reply)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("app version is", reply.Tag)
	time.Sleep(2 * time.Second)
	// Output
	// app version is 1.0.0.0
}

func startServer(s *jsonrpc.Server, done chan<- struct{}) {
	err := s.HandleFunc("version", func(ctx context.Context, app string) (Version, error) {
		select {
		case <-ctx.Done():
			log.Println("handler: context canceled")
			return Version{}, nil
		case <-time.After(time.Second):
			return Version{"1.0.0"}, nil
		}
	})
	if err != nil {
		log.Fatal(err)
	}

	http.Handle("/api", s)
	done <- struct{}{}
	http.ListenAndServe(":8080", nil)
}
