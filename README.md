# JSON-RPC 2.0 Module for golang

A go implementation of JSON RPC 2.0 over http. Work in progress.

## Server

```go
package main

import (
        "context"
        "net/http"

        "github.com/echovl/jsonrpc"
)

type User struct {
        ID   int    `json:"id"`
        Name string `json:"name"`
}

func getUser(ctx context.Context, id string) (User, error) {
        return User{ID: 1, Name: "Jhon Doe"}, nil
}

func main() {
        server := jsonrpc.NewServer()
        server.HandleFunc("getUserById", getUser)

        http.Handle("/api", server)
        http.ListenAndServe(":4545", nil)
}

```
