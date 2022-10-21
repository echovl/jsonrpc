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
	resp, err := client.Call(context.Background(), "sum", Args{1, 2})
	if err != nil {
		t.Errorf("sum: error not expected: %v", err)
	}
	if err := resp.Decode(sum); err != nil {
		t.Errorf("sum: error not expected: %v", err)
	}
	if sum.C != 3 {
		t.Errorf("sum: invalid sum: expected 3, got %v", sum.C)
	}

	rnd := &Reply{}
	resp, err = client.Call(context.Background(), "random", Args{1, 2})
	if err != nil {
		t.Errorf("random: error not expected: %v", err)
	}
	if err := resp.Decode(rnd); err != nil {
		t.Errorf("random: error not expected: %v", err)
	}

	// Invalid params
	resp, _ = client.Call(context.Background(), "sum", nil)
	if resp.error == nil || *resp.error != *ErrInvalidParams {
		t.Errorf("sum invalid params err:\ngot: %v\nwant: ErrInvalidParams", resp.error)
	}

	// Unknown method
	resp, _ = client.Call(context.Background(), "unknown", nil)
	if resp.error == nil || *resp.error != *ErrMethodNotFound {
		t.Errorf("unknown method:\ngot: %v\nwant: ErrMethodNotFound", err)
	}

	// Context canceled
	ctx, cancel := context.WithTimeout(context.Background(), time.Microsecond)
	defer cancel()
	resp, _ = client.Call(ctx, "slow", nil)
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
			resp, err := client.Call(context.Background(), "counter", 6)
			if err != nil {
				panic(err)
			}
			if err := resp.Decode(&reply); err != nil {
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

func startServerAddCORS(t *testing.T, counter *state) {
	s := NewServer()
	s.Cors = map[string]string{
		"Access-Control-Allow-Origin":"*",
		"Access-Control-Allow-Methods":"POST,GET,OPTIONS",
		"Access-Control-Allow-Headers":"Content-Type",
	}
	s.HandleFunc("sum", sum)
	s.HandleFunc("random", random)
	s.HandleFunc("counter", counter.increaseCounter)

	if err := http.ListenAndServe(port, s); err != nil {
		t.Errorf("starting server: %v", err)
	}
}
