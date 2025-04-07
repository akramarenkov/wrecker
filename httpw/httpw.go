package httpw

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"time"
)

const DefaultReadTimeout = time.Second

const defaultQuicklyErrorsTimeout = time.Second

// Options of the created instance of the HTTP wrecker with an HTTP server inside.
type Opts struct {
	// Network to listen, as in [net.Listen]. Required parameter
	Network string

	// Address to listen, as in [net.Listen]. Required parameter
	Address string

	// URL of upstream server. Required parameter
	Upstream string

	// List of the deciders
	Deciders []Decider

	// Transport for proxied requests
	ProxyTransport http.RoundTripper

	// Parameters of server. Addr and Handler fields are ignored
	Server *http.Server
}

// HTTP wrecker with an HTTP server inside.
type Wrecker struct {
	err      chan error
	listener net.Listener
	server   *http.Server
}

// Creates and runs an HTTP wrecker with an HTTP server inside.
func Run(opts Opts) (*Wrecker, error) { //nolint:gocritic // Copy frequency is low.
	handler, err := New(opts.Upstream, opts.ProxyTransport, opts.Deciders...)
	if err != nil {
		return nil, err
	}

	listener, err := prepareListener(opts.Server, opts.Network, opts.Address)
	if err != nil {
		return nil, err
	}

	wrc := &Wrecker{
		err:      make(chan error, 1),
		listener: listener,
		server:   prepareServer(opts.Server, handler),
	}

	go wrc.serve()

	if err := wrc.waitQuicklyErrors(defaultQuicklyErrorsTimeout); err != nil {
		return nil, err
	}

	return wrc, nil
}

func prepareListener(srv *http.Server, network, address string) (net.Listener, error) {
	if srv != nil && srv.TLSConfig != nil {
		return tls.Listen(network, address, srv.TLSConfig)
	}

	return net.Listen(network, address)
}

func prepareServer(srv *http.Server, handler *Handler) *http.Server {
	if srv == nil {
		server := &http.Server{
			Handler:     handler,
			ReadTimeout: DefaultReadTimeout,
		}

		return server
	}

	server := &http.Server{
		Handler:                      handler,
		TLSConfig:                    srv.TLSConfig,
		DisableGeneralOptionsHandler: srv.DisableGeneralOptionsHandler,
		ReadTimeout:                  srv.ReadTimeout,
		ReadHeaderTimeout:            srv.ReadHeaderTimeout,
		WriteTimeout:                 srv.WriteTimeout,
		IdleTimeout:                  srv.IdleTimeout,
		MaxHeaderBytes:               srv.MaxHeaderBytes,
		TLSNextProto:                 srv.TLSNextProto,
		ConnState:                    srv.ConnState,
		ErrorLog:                     srv.ErrorLog,
		BaseContext:                  srv.BaseContext,
		ConnContext:                  srv.ConnContext,
		HTTP2:                        srv.HTTP2,
		Protocols:                    srv.Protocols,
	}

	return server
}

func (wrc *Wrecker) serve() {
	defer close(wrc.err)

	wrc.err <- wrc.server.Serve(wrc.listener)
}

// When calling the [http.Server.Serve] method, it can very quickly return an error
// unrelated to listening for connections.
func (wrc *Wrecker) waitQuicklyErrors(timeout time.Duration) error {
	select {
	case <-time.After(timeout):
		return nil
	case err := <-wrc.err:
		return err
	}
}

func (wrc *Wrecker) Addr() net.Addr {
	return wrc.listener.Addr()
}

// Returns a channel with errors occurring in the wrecker server.
//
// When the wrecker server is terminated by the [Shutdown] or [Close] methods,
// [http.ErrServerClosed] is returned.
func (wrc *Wrecker) Err() <-chan error {
	return wrc.err
}

// Gracefully shuts down the wrecker server, simply calling [http.Server.Shutdown].
func (wrc *Wrecker) Shutdown(ctx context.Context) error {
	return wrc.server.Shutdown(ctx)
}

// Immediately closes all listeners and connections of the wrecker server,
// simply calling [http.Server.Close].
func (wrc *Wrecker) Close() error {
	return wrc.server.Close()
}
