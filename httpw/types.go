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

// Decides whether to return an error or pass response to an sender.
//
// If it returns true, an error will be sent to a sender without response headers
// and body.
//
// Each spoiler receives a copy of response headers and body.
type Spoiler func(headers http.Header, statusCode int, body []byte) bool
