package httpw

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestWrecker(t *testing.T) {
	t.Run(
		"default_proxy_transport",
		func(t *testing.T) {
			t.Parallel()
			testWreckerBase(t, nil, false)
		},
	)

	t.Run(
		"custom_proxy_transport",
		func(t *testing.T) {
			t.Parallel()
			testWreckerBase(t, &http.Transport{}, false)
		},
	)
}

func TestWreckerUnix(t *testing.T) {
	t.Run(
		"default_proxy_transport",
		func(t *testing.T) {
			t.Parallel()
			testWreckerBase(t, nil, true)
		},
	)

	t.Run(
		"custom_proxy_transport",
		func(t *testing.T) {
			t.Parallel()
			testWreckerBase(t, &http.Transport{}, true)
		},
	)
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

	blocker := func(req *http.Request) bool {
		return req.URL.Path == upstreamPathForbidden
	}

	message := prepareMessage(t, 1024)

	upstreamServer, upstreamListener, upstreamErr := prepareUpstreamServer(
		t,
		useUpstreamUnix,
		nil,
		requestPath{message, upstreamPath},
		requestPath{message, upstreamPathForbidden},
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
		Blockers:       []Blocker{blocker},
		ProxyTransport: proxyTransport,
	}

	wrecker, err := New(opts)
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

func TestWreckerSpolier(t *testing.T) {
	const (
		upstreamPath        = "/api"
		upstreamPathBadData = "/bad/data"

		badData = "spolier"
	)

	spolier := func(_ http.Header, _ int, body []byte) bool {
		return string(body) == badData
	}

	message := prepareMessage(t, 1<<10)

	upstreamServer, upstreamListener, upstreamErr := prepareUpstreamServer(
		t,
		false,
		nil,
		requestPath{message, upstreamPath},
		requestPath{[]byte(badData), upstreamPathBadData},
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
		Spoilers: []Spoiler{spolier},
	}

	wrecker, err := New(opts)
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

	requestURLBadData := url.URL{
		Scheme: "http",
		Host:   wrecker.Addr().String(),
		Path:   upstreamPathBadData,
	}

	request, err = http.NewRequestWithContext(
		t.Context(),
		http.MethodGet,
		requestURLBadData.String(),
		http.NoBody,
	)
	require.NoError(t, err)

	resp, err = client.Do(request)
	require.NoError(t, err)
	require.Equal(t, http.StatusForbidden, resp.StatusCode)
	require.NoError(t, resp.Body.Close())
}

func TestWreckerRequestCancel(t *testing.T) {
	t.Run(
		"with_server_close",
		func(t *testing.T) {
			t.Parallel()
			testWreckerRequestCancelBase(t, false)
		},
	)

	t.Run(
		"with_server_shutdown",
		func(t *testing.T) {
			t.Parallel()
			testWreckerRequestCancelBase(t, true)
		},
	)
}

func testWreckerRequestCancelBase(t *testing.T, useServerClose bool) {
	const upstreamPath = "/api"

	message := prepareMessage(t, 1<<28)

	upstreamServer, upstreamListener, upstreamErr := prepareUpstreamServer(
		t,
		false,
		nil,
		requestPath{message, upstreamPath},
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

	wrecker, err := New(opts)
	require.NoError(t, err)

	defer func() {
		if useServerClose {
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

	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
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

func TestWreckerTLS(t *testing.T) {
	const upstreamPath = "/api"

	message := prepareMessage(t, 1024)

	upstreamCaPool, upstreamServerCerts, upstreamClientCerts := genTempPKI(t, "127.0.0.1")

	upstreamTLSConfig := &tls.Config{
		Certificates: upstreamServerCerts,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    upstreamCaPool,
		MinVersion:   tls.VersionTLS13,
	}

	upstreamServer, upstreamListener, upstreamErr := prepareUpstreamServer(
		t,
		false,
		upstreamTLSConfig,
		requestPath{message, upstreamPath},
	)

	defer func() {
		require.NoError(t, upstreamServer.Shutdown(t.Context()))
		require.Equal(t, http.ErrServerClosed, <-upstreamErr)
	}()

	upstreamURL := url.URL{
		Scheme: "https",
		Host:   upstreamListener.Addr().String(),
	}

	wreckerCaPool, wreckerServerCerts, wreckerClientCerts := genTempPKI(t, "127.0.0.1")

	opts := Opts{
		Network:  "tcp",
		Address:  "127.0.0.1:",
		Upstream: upstreamURL.String(),
		ProxyTransport: &http.Transport{
			TLSClientConfig: &tls.Config{
				Certificates: upstreamClientCerts,
				MinVersion:   tls.VersionTLS13,
				RootCAs:      upstreamCaPool,
			},
		},
		Server: &http.Server{
			ReadTimeout: DefaultReadTimeout,
			TLSConfig: &tls.Config{
				Certificates: wreckerServerCerts,
				ClientAuth:   tls.RequireAndVerifyClientCert,
				ClientCAs:    wreckerCaPool,
				MinVersion:   tls.VersionTLS13,
			},
		},
	}

	wrecker, err := New(opts)
	require.NoError(t, err)

	defer func() {
		require.NoError(t, wrecker.Shutdown(t.Context()))
		require.Equal(t, http.ErrServerClosed, <-wrecker.Err())
	}()

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				Certificates: wreckerClientCerts,
				MinVersion:   tls.VersionTLS13,
				RootCAs:      wreckerCaPool,
			},
		},
	}

	requestURL := url.URL{
		Scheme: "https",
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
}

func TestRunBadUpstreamURL(t *testing.T) {
	opts := Opts{
		Upstream: "http://host%2F/",
	}

	wrecker, err := New(opts)
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

	wrecker, err := New(opts)
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
				//nolint:nilnil // Special for test which require an error
				GetCertificate: func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
					return nil, nil
				},
				// Involved in HTTP2 misconfiguration
				//nolint:gosec // Special for test which require an error
				CipherSuites: []uint16{tls.TLS_RSA_WITH_RC4_128_SHA},
			},
			ReadTimeout: DefaultReadTimeout,
			// Involved in HTTP2 misconfiguration
			Protocols: &protos,
		},
	}

	wrecker, err := New(opts)
	require.Error(t, err)
	require.Nil(t, wrecker)
}

type requestPath struct {
	Message []byte
	Pattern string
}

func prepareUpstreamServer(
	t *testing.T,
	useUpstreamUnix bool,
	tlsConfig *tls.Config,
	requestPaths ...requestPath,
) (*http.Server, net.Listener, chan error) {
	listener := prepareUpstreamListener(t, useUpstreamUnix, tlsConfig)

	serverErr := make(chan error)

	var router http.ServeMux

	for _, path := range requestPaths {
		router.HandleFunc(
			path.Pattern,
			func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write(path.Message)
			},
		)
	}

	server := &http.Server{
		Handler:     &router,
		ReadTimeout: 5 * time.Second,
		TLSConfig:   tlsConfig,
	}

	go func() {
		serverErr <- server.Serve(listener)
		close(serverErr)
	}()

	return server, listener, serverErr
}

func prepareUpstreamListener(t *testing.T, useUpstreamUnix bool, tlsConfig *tls.Config) net.Listener {
	if useUpstreamUnix {
		socketPath := filepath.Join(t.TempDir(), "upstream.sock")

		listener, err := selectListener("unix", socketPath, tlsConfig)
		require.NoError(t, err)

		return listener
	}

	listener, err := selectListener("tcp", "127.0.0.1:", tlsConfig)
	require.NoError(t, err)

	return listener
}

func selectListener(network, address string, tlsConfig *tls.Config) (net.Listener, error) {
	if tlsConfig == nil {
		return net.Listen(network, address)
	}

	return tls.Listen(network, address, tlsConfig)
}

func prepareMessage(t *testing.T, size int) []byte {
	message := make([]byte, size)

	readded, err := rand.Read(message)
	require.NoError(t, err)
	require.Equal(t, size, readded)

	return message
}

func genTempPKI(
	t *testing.T,
	address string,
) (*x509.CertPool, []tls.Certificate, []tls.Certificate) {
	const (
		certLifeTimeInDays = 1
		keySize            = 1024

		caSN     = 689023454
		clientSN = 689023455
		serverSN = 689023456
	)

	ip := net.ParseIP(address)
	require.NotNil(t, ip)

	notBefore := time.Now()
	notAfter := notBefore.AddDate(0, 0, certLifeTimeInDays)

	caTempl := &x509.Certificate{
		SerialNumber: big.NewInt(caSN),
		Subject: pkix.Name{
			CommonName: "Temporary CA",
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
	}

	serverTempl := &x509.Certificate{
		SerialNumber: big.NewInt(serverSN),
		Subject: pkix.Name{
			CommonName: "Temporary server",
		},
		NotBefore:   notBefore,
		NotAfter:    notAfter,
		IPAddresses: []net.IP{ip},
		KeyUsage:    x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}

	clientTempl := &x509.Certificate{
		SerialNumber: big.NewInt(clientSN),
		Subject: pkix.Name{
			CommonName: "Temporary client",
		},
		NotBefore:   notBefore,
		NotAfter:    notAfter,
		KeyUsage:    x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}

	caPool, caKey := genCA(t, caTempl, keySize)
	serverCert := genNodeTLSCert(t, serverTempl, keySize, caTempl, caKey)
	clientCert := genNodeTLSCert(t, clientTempl, keySize, caTempl, caKey)

	return caPool, []tls.Certificate{serverCert}, []tls.Certificate{clientCert}
}

func genCA(
	t *testing.T,
	templ *x509.Certificate,
	keySize int,
) (*x509.CertPool, *rsa.PrivateKey) {
	key, err := rsa.GenerateKey(rand.Reader, keySize)
	require.NoError(t, err)

	cert, err := x509.CreateCertificate(rand.Reader, templ, templ, &key.PublicKey, key)
	require.NoError(t, err)

	x509Cert, err := x509.ParseCertificate(cert)
	require.NoError(t, err)

	pool := x509.NewCertPool()
	pool.AddCert(x509Cert)

	return pool, key
}

func genNodeTLSCert(
	t *testing.T,
	nodeTempl *x509.Certificate,
	keySize int,
	caTempl *x509.Certificate,
	caKey *rsa.PrivateKey,
) tls.Certificate {
	key, err := rsa.GenerateKey(rand.Reader, keySize)
	require.NoError(t, err)

	cert, err := x509.CreateCertificate(rand.Reader, nodeTempl, caTempl, &key.PublicKey, caKey)
	require.NoError(t, err)

	tlsCert := tls.Certificate{
		Certificate: [][]byte{cert},
		PrivateKey:  key,
	}

	return tlsCert
}
