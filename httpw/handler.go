package httpw

import (
	"bytes"
	"io"
	"maps"
	"net/http"
	"net/http/httputil"
	"net/url"
	"slices"

	"github.com/akramarenkov/wrecker/httpw/internal/gatherer"

	"github.com/akramarenkov/utr"
)

const (
	UnixSchemeHTTP  = "http+unix"
	UnixSchemeHTTPS = "https+unix"
)

const unixHostname = "unix"

type HandlerOpts struct {
	// URL of upstream server. Required parameter
	Upstream string

	// List of the blockers
	Blockers []Blocker

	// List of the spoilers
	Spoilers []Spoiler

	// Transport for proxied requests
	ProxyTransport http.RoundTripper
}

// HTTP wrecker in the form of [http.Handler].
type Handler struct {
	opts HandlerOpts

	proxy *httputil.ReverseProxy
}

// Creates a new HTTP wrecker in the form of [http.Handler].
func NewHandler(opts HandlerOpts) (*Handler, error) { //nolint:gocritic // Copy frequency is low.
	upstream, err := url.Parse(opts.Upstream)
	if err != nil {
		return nil, err
	}

	proxy, err := prepareProxy(upstream, opts.ProxyTransport)
	if err != nil {
		return nil, err
	}

	hdl := &Handler{
		opts: opts,

		proxy: proxy,
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

	for _, blocker := range hdl.opts.Blockers {
		blockerRequest := req.Clone(req.Context())
		blockerRequest.Body = io.NopCloser(bytes.NewBuffer(body))

		if block := blocker(blockerRequest); block {
			wrt.WriteHeader(http.StatusForbidden)
			return
		}
	}

	proxyRequest := req.Clone(req.Context())
	proxyRequest.Body = io.NopCloser(bytes.NewBuffer(body))

	ghr := gatherer.New(wrt)

	hdl.proxy.ServeHTTP(ghr, proxyRequest)

	for _, spoiler := range hdl.opts.Spoilers {
		headers := maps.Clone(ghr.Header())
		body := slices.Clone(ghr.Body())

		if spoil := spoiler(headers, ghr.StatusCode(), body); spoil {
			wrt.WriteHeader(http.StatusForbidden)
			return
		}
	}

	_, _ = ghr.Pass()
}
