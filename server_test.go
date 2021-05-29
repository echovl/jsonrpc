package jsonrpc

import (
	"context"
	"testing"
)

type handler struct {
	numArgs int
	name    string
	f       interface{}
}

type SimpleStruct struct {
	Str     string
	Number  int
	Boolean bool
}

type ComplexStruct struct {
	Arr     []byte
	Nested  SimpleStruct
	Complex []SimpleStruct
	Any     interface{}
}

var handlers = []handler{
	// 1 arg, 2 returns
	{1, "nil_string", func(ctx context.Context) (string, error) { return "", nil }},
	{1, "nil_bool", func(ctx context.Context) (bool, error) { return true, nil }},
	{1, "nil_int", func(ctx context.Context) (int, error) { return 1, nil }},
	{1, "nil_float32", func(ctx context.Context) (float32, error) { return 1.0, nil }},
	{1, "nil_simpleStruct", func(ctx context.Context) (SimpleStruct, error) { return SimpleStruct{}, nil }},
	{1, "nil_complexStruct", func(ctx context.Context) (ComplexStruct, error) { return ComplexStruct{}, nil }},
	{1, "nil_*string", func(ctx context.Context) (*string, error) { return nil, nil }},
	{1, "nil_*simpleStruct", func(ctx context.Context) (*SimpleStruct, error) { return &SimpleStruct{}, nil }},
	{1, "nil_*complexStruct", func(ctx context.Context) (*ComplexStruct, error) { return &ComplexStruct{}, nil }},
	// 2 args, 2 returns
	{2, "string_string", func(ctx context.Context, p string) (string, error) { return "", nil }},
	{2, "string_bool", func(ctx context.Context, p string) (bool, error) { return true, nil }},
	{2, "string_int", func(ctx context.Context, p string) (int, error) { return 0, nil }},
	{2, "string_float32", func(ctx context.Context, p string) (float32, error) { return 1.0, nil }},
	{2, "bool_string", func(ctx context.Context, p bool) (string, error) { return "", nil }},
	{2, "bool_bool", func(ctx context.Context, p bool) (bool, error) { return true, nil }},
	{2, "bool_int", func(ctx context.Context, p bool) (int, error) { return 1.0, nil }},
	{2, "bool_float32", func(ctx context.Context, p bool) (float32, error) { return 1.0, nil }},
	{2, "int_string", func(ctx context.Context, p int) (string, error) { return "", nil }},
	{2, "int_bool", func(ctx context.Context, p int) (bool, error) { return true, nil }},
	{2, "int_int", func(ctx context.Context, p int) (int, error) { return 0, nil }},
	{2, "int_float32", func(ctx context.Context, p int) (float32, error) { return 1.0, nil }},
	{2, "float32_string", func(ctx context.Context, p float32) (string, error) { return "", nil }},
	{2, "float32_bool", func(ctx context.Context, p float32) (bool, error) { return true, nil }},
	{2, "float32_int", func(ctx context.Context, p float32) (int, error) { return 1, nil }},
	{2, "float32_float32", func(ctx context.Context, p float32) (float32, error) { return 1.0, nil }},
}

func TestHandleFunc(t *testing.T) {
	server := NewServer()

	for _, h := range handlers {
		err := server.HandleFunc(h.name, h.f)
		if err != nil {
			t.Errorf("method %v registration failed: %v", h.name, err)
		}
		v, ok := server.handler.Load(h.name)
		if !ok {
			t.Errorf("method %v not registered", h.name)
		}
		htype, ok := v.(handlerType)
		if !ok {
			t.Errorf("handler with wrong type")
		}
		if htype.numArgs != h.numArgs {
			t.Errorf("handlerType with incorrect numArgs: \ngot: %v\nwant: %v\n", htype.numArgs, h.numArgs)
		}
	}
}
