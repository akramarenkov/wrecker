# HTTP wrecker

## Purpose

HTTP wrecker which provides an ability to interrupt execution of requests
 to an upstream server

## Usage

Example:

```go
package main

import (
    "bytes"
    "context"
    "errors"
    "fmt"
    "io"
    "net"
    "net/http"
    "net/url"
    "time"

    "github.com/akramarenkov/wrecker/httpw"
)

func main() {
    message := []byte("example message")

    upstreamListener, err := net.Listen("tcp", "127.0.0.1:")
    if err != nil {
        panic(err)
    }

    var upstreamRouter http.ServeMux

    upstreamRouter.HandleFunc(
        "/api",
        func(w http.ResponseWriter, _ *http.Request) {
            _, _ = w.Write(message)
        },
    )

    upstreamRouter.HandleFunc(
        "/forbidden",
        func(w http.ResponseWriter, _ *http.Request) {
            _, _ = w.Write(message)
        },
    )

    upstreamServer := &http.Server{
        Handler:     &upstreamRouter,
        ReadTimeout: time.Second,
    }

    upstreamErr := make(chan error)
    defer close(upstreamErr)

    defer func() {
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()

        if err := upstreamServer.Shutdown(ctx); err != nil {
            fmt.Println("Upstream server shutdown error:", err)
        }

        if err := <-upstreamErr; !errors.Is(err, http.ErrServerClosed) {
            fmt.Println("Upstream server has terminated abnormally:", err)
        }
    }()

    go func() {
        upstreamErr <- upstreamServer.Serve(upstreamListener)
    }()

    upstreamURL := url.URL{
        Scheme: "http",
        Host:   upstreamListener.Addr().String(),
    }

    opts := httpw.Opts{
        Network:  "tcp",
        Address:  "127.0.0.1:",
        Upstream: upstreamURL.String(),
        Deciders: []httpw.Decider{
            func(req *http.Request) bool {
                return req.URL.Path != "/forbidden"
            },
        },
    }

    wrecker, err := httpw.Run(opts)
    if err != nil {
        panic(err)
    }

    defer func() {
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()

        if err := wrecker.Shutdown(ctx); err != nil {
            fmt.Println("Wrecker shutdown error:", err)
        }

        if err := <-wrecker.Err(); !errors.Is(err, http.ErrServerClosed) {
            fmt.Println("Wrecker has terminated abnormally:", err)
        }
    }()

    apiURL := url.URL{
        Scheme: "http",
        Host:   wrecker.Addr().String(),
        Path:   "/api",
    }

    resp, err := http.Get(apiURL.String())
    if err != nil {
        panic(err)
    }

    defer resp.Body.Close()

    received, err := io.ReadAll(resp.Body)
    if err != nil {
        panic(err)
    }

    fmt.Println(
        "Is message sent by server equal to message received by client:",
        bytes.Equal(received, message),
    )

    forbiddenURL := url.URL{
        Scheme: "http",
        Host:   wrecker.Addr().String(),
        Path:   "/forbidden",
    }

    resp, err = http.Get(forbiddenURL.String())
    if err != nil {
        panic(err)
    }

    defer resp.Body.Close()

    fmt.Println(
        "Is the request to a forbidden path aborted:",
        resp.StatusCode == http.StatusForbidden,
    )
    // Output:
    // Is message sent by server equal to message received by client: true
    // Is the request to a forbidden path aborted: true
}
```
