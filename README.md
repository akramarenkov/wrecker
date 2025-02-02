# Wrecker

[![Go Reference](https://pkg.go.dev/badge/github.com/akramarenkov/wrecker.svg)](https://pkg.go.dev/github.com/akramarenkov/wrecker)
[![Go Report Card](https://goreportcard.com/badge/github.com/akramarenkov/wrecker)](https://goreportcard.com/report/github.com/akramarenkov/wrecker)
[![Coverage Status](https://coveralls.io/repos/github/akramarenkov/wrecker/badge.svg)](https://coveralls.io/github/akramarenkov/wrecker)

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
    "slices"

    "github.com/akramarenkov/wrecker"
)

var ErrLimitReached = errors.New("limit is reached")

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

    wrkr := wrecker.New(opts)

    _, err := wrkr.Write(data)
    fmt.Println(err)
    fmt.Println(slices.Equal(buffer.Bytes(), data))
    fmt.Println()

    _, err = wrkr.Write(data)
    fmt.Println(err)
    fmt.Println(slices.Equal(buffer.Bytes(), slices.Concat(data, data)))
    fmt.Println()

    _, err = wrkr.Write(data)
    fmt.Println(err)
    fmt.Println(slices.Equal(buffer.Bytes(), slices.Concat(data, data)))
    fmt.Println()

    received := make([]byte, len(data))

    _, err = wrkr.Read(received)
    fmt.Println(err)
    fmt.Println(slices.Equal(received, data))
    fmt.Println()

    received2 := make([]byte, len(data))

    _, err = wrkr.Read(received2)
    fmt.Println(err)
    fmt.Println(slices.Equal(received2, data))
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
