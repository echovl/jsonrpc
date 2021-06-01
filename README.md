# JSON-RPC 2.0 Module for golang

A go implementation of JSON RPC 2.0 over http. Work in progress.

## Installing

To start using this library, install Go 1.12 or above. Run the following command to retrieve the library.

```sh
$ go get -u github.com/echovl/jsonrpc
```

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
## Client

```go
package main

import (
	"context"
	"fmt"

	"github.com/echovl/jsonrpc"
)

type User struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func main() {
	client := jsonrpc.NewClient("http://127.0.0.1:4545/api")
	user := &User{}
	err := client.Call(context.Background(), "getUserById", "id", &user)
	if err != nil {
		panic(err)
	}
	fmt.Println("user: ", user)
}
```
