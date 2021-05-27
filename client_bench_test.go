package jsonrpc

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sync"
	"testing"
)

var (
	getDoFunc func(req *http.Request) (*http.Response, error)
	bigJson   string
)

type mockClient struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

func (m *mockClient) Do(req *http.Request) (*http.Response, error) {
	return getDoFunc(req)
}

func init() {
	r, err := os.Open("./testdata/big.json")
	if err != nil {
		log.Fatalf("opening file: %v", err)
	}
	b, err := ioutil.ReadAll(r)
	if err != nil {
		log.Fatalf("reading file: %v", err)
	}
	bigJson = string(b)
}

func BenchmarkClientCallSeq(b *testing.B) {
	mock := &mockClient{}
	getDoFunc = func(*http.Request) (*http.Response, error) {
		r := ioutil.NopCloser(bytes.NewReader([]byte(bigJson)))
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
	for n := 2; n <= 8192; n *= 4 {
		b.Run(fmt.Sprintf("normal/%v", n), func(b *testing.B) {
			mock := &mockClient{}
			getDoFunc = func(*http.Request) (*http.Response, error) {
				r := ioutil.NopCloser(bytes.NewReader([]byte(bigJson)))
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
						client.Call(context.Background(), "echo", msg, &reply)
						wg.Done()
					}()
				}
				wg.Wait()
			}
		})
	}
}
