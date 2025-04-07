package httpw

import (
	"testing"

	"github.com/akramarenkov/utr"
	"github.com/stretchr/testify/require"
)

func TestNewBadUpstreamURL(t *testing.T) {
	wrecker, err := New("http://host%2F/", nil)
	require.Error(t, err)
	require.Nil(t, wrecker)
}

func TestNewBadUnixProxyTransport(t *testing.T) {
	wrecker, err := New("http+unix:///tmp/upstream.sock", &utr.Transport{})
	require.Error(t, err)
	require.Nil(t, wrecker)
}
