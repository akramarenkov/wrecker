# Wrecker

[![Go Reference](https://pkg.go.dev/badge/github.com/akramarenkov/wrecker.svg)](https://pkg.go.dev/github.com/akramarenkov/wrecker)
[![Go Report Card](https://goreportcard.com/badge/github.com/akramarenkov/wrecker)](https://goreportcard.com/report/github.com/akramarenkov/wrecker)
[![codecov](https://codecov.io/gh/akramarenkov/wrecker/branch/master/graph/badge.svg?token=Ze9aBpHbGE)](https://codecov.io/gh/akramarenkov/wrecker)

## Purpose

Library with a Wrecker which corresponds to the io.ReadWriter interface and provides completes read and/or write operations with an error after reaching the limits on completed calls and/or the size of processed data

## Usage

Example:

```go
package main

import (
    "bytes"
    "errors"
    "fmt"

    "github.com/akramarenkov/wrecker"
)

var (
    ErrLimitReached = errors.New("limit is reached")
)

func main() {
    data := []byte("some data")

    buffer := bytes.NewBuffer(nil)

    opts := wrecker.Opts{
        Error:           ErrLimitReached,
        ReadCallsLimit:  1,
        ReadSizeLimit:   2 * len(data),
        ReadWriter:      buffer,
        WriteCallsLimit: 3,
        WriteSizeLimit:  2 * len(data),
    }

    wrecker := wrecker.New(opts)

    _, err := wrecker.Write(data)
    fmt.Println(err)
    fmt.Println(buffer.String() == string(data))
    fmt.Println()

    _, err = wrecker.Write(data)
    fmt.Println(err)
    fmt.Println(buffer.String() == string(data)+string(data))
    fmt.Println()

    _, err = wrecker.Write(data)
    fmt.Println(err)
    fmt.Println(buffer.String() == string(data)+string(data))
    fmt.Println()

    payload := make([]byte, len(data))

    _, err = wrecker.Read(payload)
    fmt.Println(err)
    fmt.Println(string(payload) == string(data))
    fmt.Println()

    payload = make([]byte, len(data))

    _, err = wrecker.Read(payload)
    fmt.Println(err)
    fmt.Println(string(payload) == string(data))

    // Output:
    // <nil>
    // true
    //
    // <nil>
    // true
    //
    // limit is reached
    // true
    //
    // <nil>
    // true
    //
    // limit is reached
    // false
}
```
