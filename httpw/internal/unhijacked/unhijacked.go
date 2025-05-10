// Internal package with wrapper over [http.ResponseWriter] that does not implement
// the [http.Hijacker] interface.
package unhijacked

import "net/http"

// Wrapper over [http.ResponseWriter] that does not implement
// the [http.Hijacker] interface.
type Unhijacked struct {
	underlying http.ResponseWriter
}

// Creates a new wrapper over [http.ResponseWriter] that does not implement
// the [http.Hijacker] interface.
func New(underlying http.ResponseWriter) *Unhijacked {
	uhj := &Unhijacked{
		underlying: underlying,
	}

	return uhj
}

// Implements the [http.ResponseWriter] interface.
func (uhj *Unhijacked) Header() http.Header {
	return uhj.underlying.Header()
}

// Implements the [http.ResponseWriter] interface.
func (uhj *Unhijacked) Write(data []byte) (int, error) {
	return uhj.underlying.Write(data)
}

// Implements the [http.ResponseWriter] interface.
func (uhj *Unhijacked) WriteHeader(statusCode int) {
	uhj.underlying.WriteHeader(statusCode)
}
