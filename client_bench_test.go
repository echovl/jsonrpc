package jsonrpc

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"testing"
)

type mockClient struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

var (
	getDoFunc func(req *http.Request) (*http.Response, error)
)

func (m *mockClient) Do(req *http.Request) (*http.Response, error) {
	return getDoFunc(req)
}

func BenchmarkClientCallSeq(b *testing.B) {
	// given
	mock := &mockClient{}
	json := `{"jsonrpc": "2.0", "result": {"message":"echo"}, "id": 1}`
	getDoFunc = func(*http.Request) (*http.Response, error) {
		r := ioutil.NopCloser(bytes.NewReader([]byte(json)))
		return &http.Response{
			StatusCode: 200,
			Body:       r,
		}, nil
	}

	client := NewClient("http://mock.io")
	client.httpClient = mock
	for i := 0; i < b.N; i++ {
		msg := echoMessage{String: "bench", Int: 23, Float: 23.4, Bool: true}
		reply := &echoMessage{}
		client.Call(context.Background(), "echo", msg, &reply)
	}
}

func BenchmarkClientCallAsync(b *testing.B) {
	// given
	for n := 2; n <= 1024; n *= 8 {
		b.Run(fmt.Sprintf("normal/%v", n), func(b *testing.B) {
			mock := &mockClient{}
			json := `{"jsonrpc": "2.0", "result": {"message":"echo"}, "id": 1}`
			getDoFunc = func(*http.Request) (*http.Response, error) {
				r := ioutil.NopCloser(bytes.NewReader([]byte(json)))
				return &http.Response{
					StatusCode: 200,
					Body:       r,
				}, nil
			}

			client := NewClient("http://mock.io")
			client.httpClient = mock
			for i := 0; i < b.N; i++ {
				wg := sync.WaitGroup{}
				wg.Add(n)
				for j := 0; j < n; j++ {
					go func() {
						msg := echoMessage{String: "bench", Int: 23, Float: 23.4, Bool: true}
						reply := &echoMessage{}
						client.Call(context.Background(), "getReply", msg, &reply)
						wg.Done()
					}()
				}
				wg.Wait()
			}
		})
	}
}
