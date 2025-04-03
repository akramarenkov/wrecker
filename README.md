# Wrecker

[![Go Reference](https://pkg.go.dev/badge/github.com/akramarenkov/wrecker.svg)](https://pkg.go.dev/github.com/akramarenkov/wrecker)
[![Go Report Card](https://goreportcard.com/badge/github.com/akramarenkov/wrecker)](https://goreportcard.com/report/github.com/akramarenkov/wrecker)
[![Coverage Status](https://coveralls.io/repos/github/akramarenkov/wrecker/badge.svg)](https://coveralls.io/github/akramarenkov/wrecker)

## Purpose

Library that provides completes input/output operations with an error according to
 various criteria

## Implemented wreckers

* **iow** - wrecker which corresponds to the io.ReadWriter interface and provides
 completes read and/or write operations with an error when reaching the limits on
 completed calls and/or the size of processed data. See [README](iow/README.md)

* **httpw** - HTTP wrecker which provides an ability to interrupt execution of requests
 to an upstream server. See [README](httpw/README.md)
