package httpw

import "net/http"

// Decides whether to return an error or redirect a request to an upstream server.
//
// If it returns true, an error will be sent to a sender and a request will not be
// forwarded to an upstream server.
//
// Each blocker receives a copy of the request body, which it can read independently
// from other blockers and from the upstream server.
type Blocker func(*http.Request) bool
