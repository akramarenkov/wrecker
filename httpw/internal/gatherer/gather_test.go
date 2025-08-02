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

	"github.com/akramarenkov/wrecker/httpw/internal/unhijacked"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
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

		_, err := w.Write(message)
		assert.NoError(t, err)
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

		_, err := w.Write(message)
		assert.NoError(t, err)
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

		_, err := w.Write(message)
		assert.NoError(t, err)
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

		_, err := w.Write(message)
		assert.NoError(t, err)
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
	var router http.ServeMux

	router.HandleFunc(
		"/",
		func(w http.ResponseWriter, r *http.Request) {
			ghr := New(w)

			handler(ghr, r)

			_, err := ghr.Pass()
			assert.NoError(t, err)
		},
	)

	var blank net.ListenConfig

	listener, err := blank.Listen(t.Context(), "tcp", "127.0.0.1:")
	require.NoError(t, err)

	defer listener.Close()

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

func TestGathererWebSocket(t *testing.T) {
	testGathererWebSocketBase(t, false)
}

func TestGathererWebSocketUnhijackedUnderlying(t *testing.T) {
	testGathererWebSocketBase(t, true)
}

func testGathererWebSocketBase(t *testing.T, useUnhijacked bool) {
	const messageSize = 1 << 20

	message := prepareMessage(t, messageSize)

	headers := http.Header{"Key": []string{"Value"}}
	expectedHeaders := http.Header{"Key": []string{"Value"}}

	var upgrader websocket.Upgrader

	handler := func(w http.ResponseWriter, r *http.Request) {
		if !assert.Subset(t, r.Header, expectedHeaders) {
			return
		}

		conn, err := upgrader.Upgrade(w, r, headers)

		if useUnhijacked {
			assert.Error(t, err)
			return
		}

		if !assert.NoError(t, err) {
			return
		}

		kind, msg, err := conn.ReadMessage()
		if !assert.NoError(t, err) {
			return
		}

		assert.NoError(t, conn.WriteMessage(kind, msg))
	}

	pickResponseWriter := func(w http.ResponseWriter) http.ResponseWriter {
		if useUnhijacked {
			return unhijacked.New(w)
		}

		return w
	}

	var router http.ServeMux

	router.HandleFunc(
		"/",
		func(w http.ResponseWriter, r *http.Request) {
			ghr := New(pickResponseWriter(w))

			handler(ghr, r)

			_, err := ghr.Pass()
			assert.NoError(t, err)
		},
	)

	var blank net.ListenConfig

	listener, err := blank.Listen(t.Context(), "tcp", "127.0.0.1:")
	require.NoError(t, err)

	defer listener.Close()

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
		Scheme: "ws",
		Host:   listener.Addr().String(),
		Path:   "/",
	}

	conn, resp, err := websocket.DefaultDialer.DialContext(
		t.Context(),
		requestURL.String(),
		headers,
	)

	if useUnhijacked {
		require.Error(t, err)
		require.NoError(t, resp.Body.Close())
		require.Equal(t, http.StatusInternalServerError, resp.StatusCode)
		require.NotSubset(t, resp.Header, expectedHeaders)

		return
	}

	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	require.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)
	require.Subset(t, resp.Header, expectedHeaders)

	require.NoError(t, conn.WriteMessage(websocket.TextMessage, message))

	kind, readded, err := conn.ReadMessage()
	require.NoError(t, err)
	require.Equal(t, message, readded)
	require.Equal(t, websocket.TextMessage, kind)
}

func prepareMessage(t *testing.T, size int) []byte {
	message := make([]byte, size)

	readded, err := rand.Read(message)
	require.NoError(t, err)
	require.Equal(t, size, readded)

	return message
}
