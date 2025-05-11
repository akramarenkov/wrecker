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

// Data of response from upstream server.
type Response struct {
	// Body of a response
	Body []byte

	// Header maps of a response
	Header http.Header

	// Request that was sent to obtain this Response
	Request *http.Request

	// Status code of a response
	StatusCode int
}

// Decides whether to return an error or pass response to an sender.
//
// If it returns true, an error will be sent to a sender without response headers
// and body.
//
// Each spoiler receives a copy of response headers and body, also a copy of the
// request body.
type Spoiler func(*Response) bool
