package httpw

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestWrecker(t *testing.T) {
	testWreckerBase(t, nil, false)
	testWreckerBase(t, &http.Transport{}, false)
}

func TestWreckerUnix(t *testing.T) {
	testWreckerBase(t, nil, true)
	testWreckerBase(t, &http.Transport{}, true)
}

func testWreckerBase(
	t *testing.T,
	proxyTransport http.RoundTripper,
	useUpstreamUnix bool,
) {
	const (
		upstreamPath          = "/api"
		upstreamPathForbidden = "/forbidden"
	)

	decider := func(req *http.Request) bool {
		return req.URL.Path != upstreamPathForbidden
	}

	message := prepareMessage(t, 1024)

	upstreamServer, upstreamListener, upstreamErr := prepareUpstreamServer(
		t,
		message,
		useUpstreamUnix,
		upstreamPath,
		upstreamPathForbidden,
	)

	defer func() {
		require.NoError(t, upstreamServer.Shutdown(t.Context()))
		require.Equal(t, http.ErrServerClosed, <-upstreamErr)
	}()

	upstreamURL := url.URL{
		Scheme: "http",
		Host:   upstreamListener.Addr().String(),
	}

	if useUpstreamUnix {
		upstreamURL = url.URL{
			Scheme: UnixSchemeHTTP,
			Path:   upstreamListener.Addr().String(),
		}
	}

	opts := Opts{
		Network:        "tcp",
		Address:        "127.0.0.1:",
		Upstream:       upstreamURL.String(),
		Deciders:       []Decider{decider},
		ProxyTransport: proxyTransport,
	}

	wrecker, err := Run(opts)
	require.NoError(t, err)

	defer func() {
		require.NoError(t, wrecker.Shutdown(t.Context()))
		require.Equal(t, http.ErrServerClosed, <-wrecker.Err())
	}()

	client := http.DefaultClient

	requestURL := url.URL{
		Scheme: "http",
		Host:   wrecker.Addr().String(),
		Path:   upstreamPath,
	}

	request, err := http.NewRequestWithContext(
		t.Context(),
		http.MethodGet,
		requestURL.String(),
		http.NoBody,
	)
	require.NoError(t, err)

	resp, err := client.Do(request)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	output, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, message, output)
	require.NoError(t, resp.Body.Close())

	requestURLForbidden := url.URL{
		Scheme: "http",
		Host:   wrecker.Addr().String(),
		Path:   upstreamPathForbidden,
	}

	request, err = http.NewRequestWithContext(
		t.Context(),
		http.MethodGet,
		requestURLForbidden.String(),
		http.NoBody,
	)
	require.NoError(t, err)

	resp, err = client.Do(request)
	require.NoError(t, err)
	require.Equal(t, http.StatusForbidden, resp.StatusCode)
	require.NoError(t, resp.Body.Close())
}

func TestWreckerRequestCancel(t *testing.T) {
	testWreckerRequestCancelBase(t, false)
	testWreckerRequestCancelBase(t, true)
}

func testWreckerRequestCancelBase(t *testing.T, useServerClose bool) {
	const (
		upstreamPath = "/api"
		wreckerPath  = "/"
	)

	message := prepareMessage(t, 1<<27)

	upstreamServer, upstreamListener, upstreamErr := prepareUpstreamServer(
		t,
		message,
		false,
		upstreamPath,
	)

	defer func() {
		require.NoError(t, upstreamServer.Shutdown(t.Context()))
		require.Equal(t, http.ErrServerClosed, <-upstreamErr)
	}()

	upstreamURL := url.URL{
		Scheme: "http",
		Host:   upstreamListener.Addr().String(),
	}

	opts := Opts{
		Network:  "tcp",
		Address:  "127.0.0.1:",
		Upstream: upstreamURL.String(),
	}

	wrecker, err := Run(opts)
	require.NoError(t, err)

	defer func() {
		if useServerClose {
			// Does not interrupt reading of the body with an error
			require.NoError(t, wrecker.Close())
		} else {
			require.NoError(t, wrecker.Shutdown(t.Context()))
		}

		require.Equal(t, http.ErrServerClosed, <-wrecker.Err())
	}()

	client := http.DefaultClient

	requestURL := url.URL{
		Scheme: "http",
		Host:   wrecker.Addr().String(),
		Path:   upstreamPath,
	}

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Millisecond)
	defer cancel()

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		requestURL.String(),
		io.NopCloser(bytes.NewBuffer(message)),
	)
	require.NoError(t, err)

	//nolint:bodyclose // False positive
	resp, err := client.Do(request)
	require.Error(t, err)
	require.Nil(t, resp)
}

func TestRunBadUpstreamURL(t *testing.T) {
	opts := Opts{
		Upstream: "http://host%2F/",
	}

	wrecker, err := Run(opts)
	require.Error(t, err)
	require.Nil(t, wrecker)
}

func TestRunListenFailed(t *testing.T) {
	upstreamListener, err := net.Listen("tcp", "127.0.0.1:")
	require.NoError(t, err)

	defer upstreamListener.Close()

	upstreamURL := url.URL{
		Scheme: "http",
		Host:   upstreamListener.Addr().String(),
	}

	opts := Opts{
		Network:  upstreamListener.Addr().Network(),
		Address:  upstreamListener.Addr().String(),
		Upstream: upstreamURL.String(),
	}

	wrecker, err := Run(opts)
	require.Error(t, err)
	require.Nil(t, wrecker)
}

func TestRunQuicklyErrorsViaHTTP2Misconfiguration(t *testing.T) {
	var protos http.Protocols

	// Involved in HTTP2 misconfiguration
	protos.SetUnencryptedHTTP2(true)

	opts := Opts{
		Network:  "tcp",
		Address:  "127.0.0.1:",
		Upstream: "http://127.0.0.1",
		Server: &http.Server{
			TLSConfig: &tls.Config{
				// Doesn't cause any problems with TLS and HTTP2 misconfiguration in
				// this case, used for simplicity to avoid generating certificates
				GetCertificate: func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
					return nil, nil
				},
				// Involved in HTTP2 misconfiguration
				CipherSuites: []uint16{tls.TLS_RSA_WITH_RC4_128_SHA},
			},
			// Involved in HTTP2 misconfiguration
			Protocols: &protos,
		},
	}

	wrecker, err := Run(opts)
	require.Error(t, err)
	require.Nil(t, wrecker)
}

func prepareUpstreamServer(
	t *testing.T,
	message []byte,
	useUpstreamUnix bool,
	requestPaths ...string,
) (*http.Server, net.Listener, chan error) {
	listener := prepareUpstreamListener(t, useUpstreamUnix)

	serverErr := make(chan error)

	var router http.ServeMux

	for _, path := range requestPaths {
		router.HandleFunc(
			path,
			func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write(message)
			},
		)
	}

	server := &http.Server{
		Handler:     &router,
		ReadTimeout: time.Second,
	}

	go func() {
		serverErr <- server.Serve(listener)
		close(serverErr)
	}()

	return server, listener, serverErr
}

func prepareUpstreamListener(t *testing.T, useUpstreamUnix bool) net.Listener {
	if useUpstreamUnix {
		socketPath := filepath.Join(t.TempDir(), "upstream.sock")

		listener, err := net.Listen("unix", socketPath)
		require.NoError(t, err)

		return listener
	}

	listener, err := net.Listen("tcp", "127.0.0.1:")
	require.NoError(t, err)

	return listener
}

func prepareMessage(t *testing.T, size int) []byte {
	message := make([]byte, size)

	readded, err := rand.Read(message)
	require.NoError(t, err)
	require.Equal(t, size, readded)

	return message
}
