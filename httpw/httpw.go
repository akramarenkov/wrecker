package httpw

import (
	"errors"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/akramarenkov/utr"
)

var ErrOnlyStdTransport = errors.New("only transport from net/http package can be used with unix socket")

const (
	UnixSchemeHTTP  = "http+unix"
	UnixSchemeHTTPS = "https+unix"
)

const unixHostname = "unix"

// Decides whether to return an error or redirect a request to an upstream server.
//
// If it returns false, an error will be sent to a sender and a request will not be
// forwarded to an upstream server.
type Decider func(*http.Request) bool

// HTTP wrecker which provides an ability to interrupt execution of requests to an
// upstream server.
type Wrecker struct {
	deciders []Decider
	proxy    *httputil.ReverseProxy
}

// Creates a new HTTP wrecker in the form of [http.Handler].
func New(upstreamURL string, proxyTransport http.RoundTripper, deciders ...Decider) (*Wrecker, error) {
	up, err := url.Parse(upstreamURL)
	if err != nil {
		return nil, err
	}

	proxy, err := prepareProxy(up, proxyTransport)
	if err != nil {
		return nil, err
	}

	wrc := &Wrecker{
		deciders: deciders,
		proxy:    proxy,
	}

	return wrc, nil
}

func prepareProxy(
	upstreamURL *url.URL,
	proxyTransport http.RoundTripper,
) (*httputil.ReverseProxy, error) {
	if upstreamURL.Scheme == UnixSchemeHTTP || upstreamURL.Scheme == UnixSchemeHTTPS {
		return prepareUnixProxy(upstreamURL, proxyTransport)
	}

	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = upstreamURL.Scheme
			req.URL.Host = upstreamURL.Host
		},
		Transport: proxyTransport,
	}

	return proxy, nil
}

func prepareUnixProxy(
	upstreamURL *url.URL,
	proxyTransport http.RoundTripper,
) (*httputil.ReverseProxy, error) {
	var keeper utr.Keeper

	// Returning of error  cannot be tested because a known correct hostname is used
	// and each wrecker instance creates its own keeper, which eliminates duplication
	// of hostname
	_ = keeper.AddPath(unixHostname, upstreamURL.Path)

	adjusters, err := prepareUnixAdjusters(proxyTransport)
	if err != nil {
		return nil, err
	}

	if err := utr.Register(&keeper, adjusters...); err != nil {
		return nil, err
	}

	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = upstreamURL.Scheme
			req.URL.Host = unixHostname
		},
		Transport: proxyTransport,
	}

	return proxy, nil
}

func prepareUnixAdjusters(proxyTransport http.RoundTripper) ([]utr.Adjuster, error) {
	adjusters := []utr.Adjuster{
		utr.WithDefaultTransport(),
		utr.WithSchemeHTTP(UnixSchemeHTTP),
		utr.WithSchemeHTTPS(UnixSchemeHTTPS),
	}

	if proxyTransport == nil {
		return adjusters, nil
	}

	httpTransport, casted := proxyTransport.(*http.Transport)
	if !casted {
		return nil, ErrOnlyStdTransport
	}

	adjusters[0] = utr.WithTransport(httpTransport)

	return adjusters, nil
}

// Implements the [http.Handler] interface.
func (wrc *Wrecker) ServeHTTP(wrt http.ResponseWriter, req *http.Request) {
	for _, decider := range wrc.deciders {
		if pass := decider(req); !pass {
			wrt.WriteHeader(http.StatusForbidden)
			return
		}
	}

	wrc.proxy.ServeHTTP(wrt, req)
}
