package httpw_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/akramarenkov/wrecker/httpw"
)

func ExampleWrecker() {
	message := []byte("example message")

	upstreamListener, err := net.Listen("tcp", "127.0.0.1:")
	if err != nil {
		panic(err)
	}

	wreckerListener, err := net.Listen("tcp", "127.0.0.1:")
	if err != nil {
		panic(err)
	}

	upstreamURL := url.URL{
		Scheme: "http",
		Host:   upstreamListener.Addr().String(),
	}

	decider := func(req *http.Request) bool {
		return req.URL.Path != "/forbidden"
	}

	wrecker, err := httpw.New(upstreamURL.String(), nil, decider)
	if err != nil {
		panic(err)
	}

	var (
		upstreamRouter http.ServeMux
		wreckerRouter  http.ServeMux
	)

	upstreamRouter.HandleFunc(
		"/api",
		func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write(message)
		},
	)

	upstreamRouter.HandleFunc(
		"/forbidden",
		func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write(message)
		},
	)

	wreckerRouter.Handle(
		"/",
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

	faults := make(chan error)
	defer close(faults)

	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := upstreamServer.Shutdown(ctx); err != nil {
			fmt.Println("Upstream server shutdown error:", err)
		}

		if err := <-faults; !errors.Is(err, http.ErrServerClosed) {
			fmt.Println("Upstream server has terminated abnormally:", err)
		}

		if err := wreckerServer.Shutdown(ctx); err != nil {
			fmt.Println("Wrecker server shutdown error:", err)
		}

		if err := <-faults; !errors.Is(err, http.ErrServerClosed) {
			fmt.Println("Wrecker server has terminated abnormally:", err)
		}
	}()

	go func() {
		faults <- upstreamServer.Serve(upstreamListener)
	}()

	go func() {
		faults <- wreckerServer.Serve(wreckerListener)
	}()

	apiURL := url.URL{
		Scheme: "http",
		Host:   wreckerListener.Addr().String(),
		Path:   "/api",
	}

	resp, err := http.Get(apiURL.String())
	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()

	received, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	fmt.Println(
		"Is message sent by server equal to message received by client:",
		bytes.Equal(received, message),
	)

	forbiddenURL := url.URL{
		Scheme: "http",
		Host:   wreckerListener.Addr().String(),
		Path:   "/forbidden",
	}

	resp, err = http.Get(forbiddenURL.String())
	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()

	fmt.Println(
		"Is the request to a forbidden path aborted:",
		resp.StatusCode == http.StatusForbidden,
	)
	// Output:
	// Is message sent by server equal to message received by client: true
	// Is the request to a forbidden path aborted: true
}
