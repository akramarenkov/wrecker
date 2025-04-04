package httpw

import (
	"crypto/rand"
	"io"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"testing"
	"time"

	"github.com/akramarenkov/utr"
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
		wreckerPath           = "/"
	)

	decider := func(req *http.Request) bool {
		return req.URL.Path != upstreamPathForbidden
	}

	message := prepareMessage(t)

	upstreamListener := prepareUpstreamListener(t, useUpstreamUnix)

	wreckerListener, err := net.Listen("tcp", "127.0.0.1:")
	require.NoError(t, err)

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

	wrecker, err := New(upstreamURL.String(), proxyTransport, decider)
	require.NoError(t, err)

	var (
		upstreamRouter http.ServeMux
		wreckerRouter  http.ServeMux
	)

	upstreamRouter.HandleFunc(
		upstreamPath,
		func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write(message)
		},
	)

	upstreamRouter.HandleFunc(
		upstreamPathForbidden,
		func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write(message)
		},
	)

	wreckerRouter.Handle(
		wreckerPath,
		wrecker,
	)

	upstreamServer := &http.Server{
		Handler:     &upstreamRouter,
		ReadTimeout: time.Second,
	}

	wreckerServer := &http.Server{
		Handler:     &wreckerRouter,
		ReadTimeout: time.Second,
	}

	serverErr := make(chan error)
	defer close(serverErr)

	defer func() {
		require.NoError(t, upstreamServer.Shutdown(t.Context()))
		require.Equal(t, http.ErrServerClosed, <-serverErr)

		require.NoError(t, wreckerServer.Shutdown(t.Context()))
		require.Equal(t, http.ErrServerClosed, <-serverErr)
	}()

	go func() {
		serverErr <- upstreamServer.Serve(upstreamListener)
	}()

	go func() {
		serverErr <- wreckerServer.Serve(wreckerListener)
	}()

	client := http.DefaultClient

	requestURL := url.URL{
		Scheme: "http",
		Host:   wreckerListener.Addr().String(),
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
		Host:   wreckerListener.Addr().String(),
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

func TestWreckerBadUpstreamURL(t *testing.T) {
	wrecker, err := New("http://host%2F/", nil)
	require.Error(t, err)
	require.Nil(t, wrecker)
}

func TestWreckerBadUnixProxyTransport(t *testing.T) {
	wrecker, err := New("http+unix:///tmp/upstream.sock", &utr.Transport{})
	require.Error(t, err)
	require.Nil(t, wrecker)
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

func prepareMessage(t *testing.T) []byte {
	const messageSize = 1024

	message := make([]byte, messageSize)

	readded, err := rand.Read(message)
	require.NoError(t, err)
	require.Equal(t, messageSize, readded)

	return message
}
