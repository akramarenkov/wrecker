package httpw

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/akramarenkov/utr"
)

const (
	UnixSchemeHTTP  = "http+unix"
	UnixSchemeHTTPS = "https+unix"
)

const unixHostname = "unix"

// HTTP wrecker in the form of [http.Handler].
type Handler struct {
	deciders []Decider
	proxy    *httputil.ReverseProxy
}

// Creates a new HTTP wrecker in the form of [http.Handler].
func New(upstreamURL string, proxyTransport http.RoundTripper, deciders ...Decider) (*Handler, error) {
	up, err := url.Parse(upstreamURL)
	if err != nil {
		return nil, err
	}

	proxy, err := prepareProxy(up, proxyTransport)
	if err != nil {
		return nil, err
	}

	hdl := &Handler{
		deciders: deciders,
		proxy:    proxy,
	}

	return hdl, nil
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

	// Returning of error cannot be tested because a known correct hostname is used
	// and each wrecker instance creates its own keeper, which eliminates duplication
	// of hostname
	_ = keeper.AddPath(unixHostname, upstreamURL.Path)

	if proxyTransport == nil {
		proxyTransport = http.DefaultTransport
	}

	transport, err := utr.New(
		&keeper,
		proxyTransport,
		utr.WithSchemeHTTP(UnixSchemeHTTP),
		utr.WithSchemeHTTPS(UnixSchemeHTTPS),
	)
	if err != nil {
		return nil, err
	}

	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = upstreamURL.Scheme
			req.URL.Host = unixHostname
		},
		Transport: transport,
	}

	return proxy, nil
}

// Implements the [http.Handler] interface.
func (hdl *Handler) ServeHTTP(wrt http.ResponseWriter, req *http.Request) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return
	}

	for _, decider := range hdl.deciders {
		cloned := req.Clone(req.Context())
		cloned.Body = io.NopCloser(bytes.NewBuffer(body))

		if pass := decider(cloned); !pass {
			wrt.WriteHeader(http.StatusForbidden)
			return
		}
	}

	cloned := req.Clone(req.Context())
	cloned.Body = io.NopCloser(bytes.NewBuffer(body))

	hdl.proxy.ServeHTTP(wrt, cloned)
}
