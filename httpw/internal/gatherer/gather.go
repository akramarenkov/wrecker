// Internal package with Gatherer that gathers response body, stores headers and
// response status code.
package gatherer

import (
	"bufio"
	"bytes"
	"net"
	"net/http"
)

// Gathers response body, stores headers and response status code.
type Gatherer struct {
	buffer     bytes.Buffer
	headers    http.Header
	hijacked   bool
	statusCode int
	underlying http.ResponseWriter
}

// Creates a new gatherer with underlying [http.ResponseWriter].
func New(underlying http.ResponseWriter) *Gatherer {
	ghr := &Gatherer{
		headers:    make(http.Header),
		statusCode: -1,
		underlying: underlying,
	}

	return ghr
}

func (ghr *Gatherer) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, casted := ghr.underlying.(http.Hijacker)
	if !casted {
		return nil, nil, http.ErrNotSupported
	}

	ghr.hijacked = true

	return hijacker.Hijack()
}

// Implements the [http.ResponseWriter] interface.
func (ghr *Gatherer) Header() http.Header {
	return ghr.headers
}

// Implements the [http.ResponseWriter] interface.
func (ghr *Gatherer) Write(data []byte) (int, error) {
	return ghr.buffer.Write(data)
}

// Implements the [http.ResponseWriter] interface.
func (ghr *Gatherer) WriteHeader(statusCode int) {
	ghr.statusCode = statusCode
}

// Returns a gathered bytes of a response body.
func (ghr *Gatherer) Body() []byte {
	return ghr.buffer.Bytes()
}

// Returns a stored http status code.
func (ghr *Gatherer) StatusCode() int {
	return ghr.statusCode
}

// Passes a gathered data to the underlying [http.ResponseWriter].
func (ghr *Gatherer) Pass() (int, error) {
	if ghr.hijacked {
		return 0, nil
	}

	for key, values := range ghr.Header() {
		for _, value := range values {
			ghr.underlying.Header().Add(key, value)
		}
	}

	if ghr.statusCode != -1 {
		ghr.underlying.WriteHeader(ghr.StatusCode())
	}

	return ghr.underlying.Write(ghr.Body())
}
