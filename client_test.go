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

var port = ":4545"

func TestCallSync(t *testing.T) {
	go startServer(t)

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
}

func startServer(t *testing.T) {
	s := NewServer()
	s.HandleFunc("sum", sum)
	s.HandleFunc("random", random)

	if err := http.ListenAndServe(port, s); err != nil {
		t.Errorf("starting server: %v", err)
	}
}
