package jsonrpc

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"
)

type Reply struct {
	C int
}

type Args struct {
	A int
	B int
}

func sum(ctx context.Context, args Args) (Reply, error) {
	return Reply{args.A + args.B}, nil
}

func random(ctx context.Context) (Reply, error) {
	return Reply{33}, nil
}

func slow(ctx context.Context) (Reply, error) {
	time.Sleep(time.Millisecond)
	return Reply{}, nil
}

type state struct {
	N int
}

func (s *state) increaseCounter(ctx context.Context, add int) (int, error) {
	s.N += add
	return s.N, nil
}

var port = ":4545"

func TestCallSync(t *testing.T) {
	counter := &state{}
	go startServer(t, counter)

	client := NewClient("http://localhost" + port)

	// Some valid calls
	sum := &Reply{}
	err := client.Call(context.Background(), "sum", Args{1, 2}, sum)
	if err != nil {
		t.Errorf("sum: error not expected: %v", err)
	}
	if sum.C != 3 {
		t.Errorf("sum: invalid sum: expected 3, got %v", sum.C)
	}

	rnd := &Reply{}
	err = client.Call(context.Background(), "random", Args{1, 2}, rnd)
	if err != nil {
		t.Errorf("random: error not expected: %v", err)
	}

	// Invalid params
	err = client.Call(context.Background(), "sum", nil, sum)
	if err != ErrInvalidParams {
		t.Errorf("sum invalid params err:\ngot: %v\nwant: ErrInvalidParams", err)
	}

	// Unknown method
	err = client.Call(context.Background(), "unknown", nil, &struct{}{})
	if err != ErrMethodNotFound {
		t.Errorf("unknown method:\ngot: %v\nwant: ErrMethodNotFound", err)
	}

	// Context canceled
	ctx, cancel := context.WithTimeout(context.Background(), time.Microsecond)
	defer cancel()
	err = client.Call(ctx, "slow", nil, &struct{}{})
	if errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Context canceled: expected context.DeadlineExceeded, got %v", err)
	}

	// Notification
	err = client.Notify(context.Background(), "counter", 3)
	if err != nil {
		t.Errorf("counter: error not expected: %v", err)
	}
	if counter.N != 3 {
		t.Errorf("counter: bad state counter:\ngot: %v\nwant: %v", counter.N, 3)
	}

}

func BenchmarkClientSync(b *testing.B) {
	b.Run("call", func(b *testing.B) {
		client := NewClient("http://localhost" + port)
		for i := 0; i < b.N; i++ {
			var reply int
			err := client.Call(context.Background(), "counter", 6, &reply)
			if err != nil {
				panic(err)
			}
		}
	})
	b.Run("notify", func(b *testing.B) {
		client := NewClient("http://localhost" + port)
		for i := 0; i < b.N; i++ {
			err := client.Notify(context.Background(), "counter", 6)
			if err != nil {
				panic(err)
			}
		}
	})
}

func startServer(t *testing.T, counter *state) {
	s := NewServer()
	s.HandleFunc("sum", sum)
	s.HandleFunc("random", random)
	s.HandleFunc("counter", counter.increaseCounter)

	if err := http.ListenAndServe(port, s); err != nil {
		t.Errorf("starting server: %v", err)
	}
}
