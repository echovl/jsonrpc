package jsonrpc

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

var echoMessages = []echoMessage{
	{"darwin", 10, 23.4, true},
	{"galileo", 99, 74.45, false},
	{"tesla", 33, 0.23, true},
	{"darwin", 10, 23.4, true},
	{"galileo", 99, 74.45, false},
	{"tesla", 33, 0.23, true},
	{"darwin", 10, 23.4, true},
	{"galileo", 99, 74.45, false},
	{"tesla", 33, 0.23, true},
	{"darwin", 10, 23.4, true},
	{"galileo", 99, 74.45, false},
	{"tesla", 33, 0.23, true},
	{"darwin", 10, 23.4, true},
	{"galileo", 99, 74.45, false},
	{"tesla", 33, 0.23, true},
	{"darwin", 10, 23.4, true},
	{"galileo", 99, 74.45, false},
	{"tesla", 33, 0.23, true},
}

type echoMessage struct {
	String string  `json:"message"`
	Int    int     `json:"int"`
	Float  float32 `json:"float"`
	Bool   bool    `json:"bool"`
}

func (e *echoMessage) equal(to echoMessage) bool {
	return e.String == to.String && e.Int == to.Int && e.Float == to.Float && e.Bool == to.Bool
}

func TestClientCallAsync(t *testing.T) {
	// given
	ts := newEchoServer(t, false)
	defer ts.Close()
	client := NewClient(ts.URL)

	// when
	w := sync.WaitGroup{}
	for _, msg := range echoMessages {
		w.Add(1)
		go func(t *testing.T, msg echoMessage) {
			reply := &echoMessage{}
			err := client.Call(context.Background(), "echo", msg, &reply)
			if err != nil {
				t.Error(err)
			}

			// then
			if reply.equal(msg) {
				t.Errorf("invalid echo message\ngot: %v\nwant: %v", reply, msg)
			}
			w.Done()
		}(t, msg)
	}
	w.Wait()
}

func TestClientCallTimeout(t *testing.T) {
	// given
	ts := newEchoServer(t, true)
	ctx, cancel := context.WithTimeout(context.Background(), time.Microsecond)
	client := NewClient(ts.URL)
	defer ts.Close()
	defer cancel()

	// when
	msg := echoMessages[0]
	reply := &echoMessage{}
	err := client.Call(ctx, "echo", msg, reply)

	// then
	if err == nil || err.Error() != "jsonrpc: context deadline exceeded" {
		t.Errorf("invalid err message\ngot: %v\nwant: %v", err, "jsonrpc: context deadline exceeded")
	}
}

func newEchoServer(t testing.TB, sleep bool) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		done := make(chan string)

		go func() {
			if sleep {
				time.Sleep(time.Millisecond)
			}

			data, err := ioutil.ReadAll(r.Body)
			if err != nil {
				t.Errorf("reading server request: %w", err)
				return
			}
			defer r.Body.Close()

			req := &body{}
			if err := json.Unmarshal(data, req); err != nil {
				t.Errorf("unmarshaling server request: %w", err)
				return
			}

			msg := &echoMessage{}
			if err := json.Unmarshal(*req.Params, msg); err != nil {
				t.Errorf("unmarshaling jsonrpc params: %w", err)
			}
			done <- fmt.Sprintf(`{"jsonrpc": "2.0", "result": {"message":"%v"}, "id": %v}`, msg.String, req.ID)
		}()

		select {
		case <-r.Context().Done():
			fmt.Fprintf(rw, `{"jsonrpc": "2.0", "result": {"message":"timeout"}, "id": 1}`)
		case msg := <-done:
			fmt.Fprintf(rw, msg)

		}
	}))
}
