# Wrecker

[![Go Reference](https://pkg.go.dev/badge/github.com/akramarenkov/wrecker.svg)](https://pkg.go.dev/github.com/akramarenkov/wrecker)
[![Go Report Card](https://goreportcard.com/badge/github.com/akramarenkov/wrecker)](https://goreportcard.com/report/github.com/akramarenkov/wrecker)
[![codecov](https://codecov.io/gh/akramarenkov/wrecker/branch/master/graph/badge.svg?token=)](https://codecov.io/gh/akramarenkov/wrecker)

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
        WriteCallsLimit: 2,
        WriteSizeLimit:  len(data),
    }

    wrecker := wrecker.New(opts)

    if _, err := wrecker.Write(data); err != nil {
        fmt.Println(err)
    }

    fmt.Println(buffer.String() == string(data))

    if _, err := wrecker.Write(data); err != nil {
        fmt.Println(err)
    }

    fmt.Println(buffer.String() == string(data))

    payload := make([]byte, len(data))

    if _, err := wrecker.Read(payload); err != nil {
        fmt.Println(err)
    }

    fmt.Println(string(payload) == string(data))

    payload = make([]byte, len(data))

    if _, err := wrecker.Read(payload); err != nil {
        fmt.Println(err)
    }

    fmt.Println(string(payload) != string(data))

    // Output:
    // true
    // limit is reached
    // true
    // true
    // limit is reached
    // true
}
```
