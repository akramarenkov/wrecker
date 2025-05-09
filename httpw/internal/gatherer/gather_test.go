package gatherer

import (
	"crypto/rand"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGatherer(t *testing.T) {
	const messageSize = 1 << 10

	message := prepareMessage(t, messageSize)

	handler := func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Key", "Value")
		w.Header().Add("Key-Multi", "ValueMulti1")
		w.Header().Add("Key-Multi", "ValueMulti2")

		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write(message)
	}

	expecter := func(t *testing.T, resp *http.Response) {
		expected := http.Header{
			"Key": []string{
				"Value",
			},
			"Key-Multi": []string{
				"ValueMulti1",
				"ValueMulti2",
			},
			"Content-Length": []string{
				strconv.FormatInt(messageSize, 10),
			},
		}

		require.Equal(t, http.StatusCreated, resp.StatusCode)
		require.Subset(t, resp.Header, expected)
	}

	testGathererBase(t, message, handler, expecter)
}

func TestGathererLargeBody(t *testing.T) {
	const messageSize = 1 << 20

	message := prepareMessage(t, messageSize)

	handler := func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Key", "Value")
		w.Header().Add("Key-Multi", "ValueMulti1")
		w.Header().Add("Key-Multi", "ValueMulti2")

		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write(message)
	}

	expecter := func(t *testing.T, resp *http.Response) {
		expected := http.Header{
			"Key": []string{
				"Value",
			},
			"Key-Multi": []string{
				"ValueMulti1",
				"ValueMulti2",
			},
		}

		unexpected := http.Header{
			"Content-Length": []string{
				strconv.FormatInt(messageSize, 10),
			},
		}

		require.Equal(t, http.StatusCreated, resp.StatusCode)
		require.Subset(t, resp.Header, expected)
		require.NotSubset(t, resp.Header, unexpected)
	}

	testGathererBase(t, message, handler, expecter)
}

func TestGathererLargeBodyManuallyContentLength(t *testing.T) {
	const messageSize = 1 << 20

	message := prepareMessage(t, messageSize)

	handler := func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Key", "Value")
		w.Header().Add("Key-Multi", "ValueMulti1")
		w.Header().Add("Key-Multi", "ValueMulti2")
		w.Header().Set("Content-Length", strconv.FormatInt(messageSize, 10))

		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write(message)
	}

	expecter := func(t *testing.T, resp *http.Response) {
		expected := http.Header{
			"Key": []string{
				"Value",
			},
			"Key-Multi": []string{
				"ValueMulti1",
				"ValueMulti2",
			},
			"Content-Length": []string{
				strconv.FormatInt(messageSize, 10),
			},
		}

		require.Equal(t, http.StatusCreated, resp.StatusCode)
		require.Subset(t, resp.Header, expected)
	}

	testGathererBase(t, message, handler, expecter)
}

func TestGathererAutomaticStatusCode(t *testing.T) {
	const messageSize = 1 << 20

	message := prepareMessage(t, messageSize)

	handler := func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Key", "Value")
		w.Header().Add("Key-Multi", "ValueMulti1")
		w.Header().Add("Key-Multi", "ValueMulti2")
		w.Header().Set("Content-Length", strconv.FormatInt(messageSize, 10))

		_, _ = w.Write(message)
	}

	expecter := func(t *testing.T, resp *http.Response) {
		expected := http.Header{
			"Key": []string{
				"Value",
			},
			"Key-Multi": []string{
				"ValueMulti1",
				"ValueMulti2",
			},
			"Content-Length": []string{
				strconv.FormatInt(messageSize, 10),
			},
		}

		require.Equal(t, http.StatusOK, resp.StatusCode)
		require.Subset(t, resp.Header, expected)
	}

	testGathererBase(t, message, handler, expecter)
}

func testGathererBase(
	t *testing.T,
	message []byte,
	handler func(w http.ResponseWriter, r *http.Request),
	expecter func(t *testing.T, resp *http.Response),
) {
	listener, err := net.Listen("tcp", "127.0.0.1:")
	require.NoError(t, err)

	defer listener.Close()

	var router http.ServeMux

	router.HandleFunc(
		"/",
		func(w http.ResponseWriter, r *http.Request) {
			ghr := New()

			handler(ghr, r)

			_, _ = ghr.Pass(w)
		},
	)

	server := &http.Server{
		Handler:     &router,
		ReadTimeout: 5 * time.Second,
	}

	serverErr := make(chan error)

	go func() {
		serverErr <- server.Serve(listener)
		close(serverErr)
	}()

	defer func() {
		require.NoError(t, server.Shutdown(t.Context()))
		require.Equal(t, http.ErrServerClosed, <-serverErr)
	}()

	requestURL := url.URL{
		Scheme: "http",
		Host:   listener.Addr().String(),
		Path:   "/",
	}

	request, err := http.NewRequestWithContext(
		t.Context(),
		http.MethodGet,
		requestURL.String(),
		http.NoBody,
	)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(request)
	require.NoError(t, err)

	expecter(t, resp)

	output, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, message, output)
	require.NoError(t, resp.Body.Close())
}

func prepareMessage(t *testing.T, size int) []byte {
	message := make([]byte, size)

	readded, err := rand.Read(message)
	require.NoError(t, err)
	require.Equal(t, size, readded)

	return message
}
