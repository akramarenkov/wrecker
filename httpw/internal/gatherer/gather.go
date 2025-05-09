// Internal package with Gatherer that gathers response body, stores headers and
// response status code without passing them to a underlying [http.ResponseWriter].
package gatherer

import (
	"bytes"
	"net/http"
)

// Gathers response body, stores headers and response status code without passing
// them to a underlying [http.ResponseWriter].
type Gatherer struct {
	buffer     bytes.Buffer
	headers    http.Header
	statusCode int
}

// Creates a new gatherer.
func New() *Gatherer {
	ghr := &Gatherer{
		headers:    make(http.Header),
		statusCode: -1,
	}

	return ghr
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
func (ghr *Gatherer) Pass(underlying http.ResponseWriter) (int, error) {
	for key, values := range ghr.Header() {
		for _, value := range values {
			underlying.Header().Add(key, value)
		}
	}

	if ghr.statusCode != -1 {
		underlying.WriteHeader(ghr.StatusCode())
	}

	return underlying.Write(ghr.Body())
}
