package httpw

import "net/http"

// Decides whether to return an error or redirect a request to an upstream server.
//
// If it returns false, an error will be sent to a sender and a request will not be
// forwarded to an upstream server.
//
// Each decider receives a copy of the request body, which it can read independently
// from other deciders and from the upstream server.
type Decider func(*http.Request) bool
