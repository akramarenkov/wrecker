package unhijacked

import (
	"crypto/rand"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnhijacked(t *testing.T) {
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

	var router http.ServeMux

	router.HandleFunc(
		"/",
		func(w http.ResponseWriter, r *http.Request) {
			handler(New(w), r)
		},
	)

	listener, err := net.Listen("tcp", "127.0.0.1:")
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

	expectedHeaders := http.Header{
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

	request, err := http.NewRequestWithContext(
		t.Context(),
		http.MethodGet,
		requestURL.String(),
		http.NoBody,
	)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(request)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	require.Subset(t, resp.Header, expectedHeaders)

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
