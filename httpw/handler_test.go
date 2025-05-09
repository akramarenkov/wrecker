package httpw

import (
	"testing"

	"github.com/akramarenkov/utr"
	"github.com/stretchr/testify/require"
)

func TestNewBadUpstreamURL(t *testing.T) {
	opts := HandlerOpts{
		Upstream: "http://host%2F/",
	}

	handler, err := NewHandler(opts)
	require.Error(t, err)
	require.Nil(t, handler)
}

func TestNewBadUnixProxyTransport(t *testing.T) {
	opts := HandlerOpts{
		Upstream:       "http+unix:///tmp/upstream.sock",
		ProxyTransport: &utr.Transport{},
	}

	handler, err := NewHandler(opts)
	require.Error(t, err)
	require.Nil(t, handler)
}
