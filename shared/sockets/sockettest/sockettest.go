// Package sockettest provides an unprivileged, in-process harness for the
// wsvpn data path: real socket adapters talking over loopback, attached to
// fake TUN/TAP devices instead of kernel interfaces.
package sockettest

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"fmt"
	"io"
	"log"
	"math/big"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/Doridian/water"
	"github.com/Doridian/wsvpn/shared/commands"
	"github.com/Doridian/wsvpn/shared/features"
	"github.com/Doridian/wsvpn/shared/iface"
	"github.com/Doridian/wsvpn/shared/sockets"
	"github.com/Doridian/wsvpn/shared/sockets/adapters"
	"github.com/gobwas/ws"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
	"github.com/quic-go/webtransport-go"
)

// Transport names accepted by NewAdapterPair.
const (
	TransportWS           = "ws"
	TransportWSS          = "wss"
	TransportWebTransport = "webtransport"
)

var Transports = []string{TransportWS, TransportWSS, TransportWebTransport}

// FakeTun stands in for a kernel TUN/TAP device. Packets passed to Inject are
// returned from Read (as if the kernel routed them into the tunnel); packets
// the socket writes to the device are delivered to Out.
type FakeTun struct {
	in        chan []byte
	Out       chan []byte
	closed    chan struct{}
	closeOnce sync.Once
}

func NewFakeTun() *FakeTun {
	return &FakeTun{
		in:     make(chan []byte, 1024),
		Out:    make(chan []byte, 1024),
		closed: make(chan struct{}),
	}
}

func (f *FakeTun) Inject(packet []byte) {
	select {
	case f.in <- packet:
	case <-f.closed:
	}
}

func (f *FakeTun) Read(p []byte) (int, error) {
	select {
	case pkt := <-f.in:
		return copy(p, pkt), nil
	case <-f.closed:
		return 0, io.EOF
	}
}

func (f *FakeTun) Write(p []byte) (int, error) {
	buf := make([]byte, len(p))
	copy(buf, p)
	select {
	case f.Out <- buf:
		return len(p), nil
	case <-f.closed:
		return 0, io.ErrClosedPipe
	}
}

func (f *FakeTun) Close() error {
	f.closeOnce.Do(func() { close(f.closed) })
	return nil
}

// Wrapper returns an interface wrapper backed by this fake device.
func (f *FakeTun) Wrapper() *iface.WaterInterfaceWrapper {
	return iface.NewInterfaceWrapper(&water.Interface{ReadWriteCloser: f})
}

// SelfSignedTLS returns a server config with a fresh ECDSA certificate for
// 127.0.0.1/localhost and a client config that trusts it.
func SelfSignedTLS(tb testing.TB) (serverConfig *tls.Config, clientConfig *tls.Config) {
	tb.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		tb.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "localhost"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		DNSNames:              []string{"localhost"},
		IPAddresses:           []net.IP{net.IPv4(127, 0, 0, 1)},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		tb.Fatal(err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		tb.Fatal(err)
	}

	pool := x509.NewCertPool()
	pool.AddCert(cert)

	serverConfig = &tls.Config{
		Certificates: []tls.Certificate{{Certificate: [][]byte{der}, PrivateKey: key, Leaf: cert}},
		MinVersion:   tls.VersionTLS12,
	}
	clientConfig = &tls.Config{
		RootCAs:    pool,
		ServerName: "localhost",
		MinVersion: tls.VersionTLS12,
	}
	return serverConfig, clientConfig
}

// NewAdapterPair connects a server and a client adapter over loopback using
// the given transport. Everything is torn down via tb.Cleanup.
func NewAdapterPair(tb testing.TB, transport string) (server adapters.SocketAdapter, client adapters.SocketAdapter) {
	tb.Helper()
	switch transport {
	case TransportWS:
		return newWebSocketPair(tb, false)
	case TransportWSS:
		return newWebSocketPair(tb, true)
	case TransportWebTransport:
		return newWebTransportPair(tb)
	}
	tb.Fatalf("unknown transport %q", transport)
	return nil, nil
}

func newWebSocketPair(tb testing.TB, useTLS bool) (adapters.SocketAdapter, adapters.SocketAdapter) {
	var serverTLS, clientTLS *tls.Config
	if useTLS {
		serverTLS, clientTLS = SelfSignedTLS(tb)
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		tb.Fatal(err)
	}
	tb.Cleanup(func() { _ = ln.Close() })

	type acceptResult struct {
		adapter adapters.SocketAdapter
		err     error
	}
	accepted := make(chan acceptResult, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			accepted <- acceptResult{err: err}
			return
		}
		if serverTLS != nil {
			conn = tls.Server(conn, serverTLS)
		}
		_, err = ws.Upgrade(conn)
		if err != nil {
			_ = conn.Close()
			accepted <- acceptResult{err: err}
			return
		}
		accepted <- acceptResult{adapter: adapters.NewWebSocketAdapter(conn, commands.SerializationTypeJSON, true, nil)}
	}()

	scheme := "ws"
	if useTLS {
		scheme = "wss"
	}
	dialer := ws.Dialer{TLSConfig: clientTLS, Timeout: 10 * time.Second}
	conn, br, _, err := dialer.Dial(context.Background(), fmt.Sprintf("%s://%s/", scheme, ln.Addr().String()))
	if err != nil {
		tb.Fatal(err)
	}
	client := adapters.NewWebSocketAdapter(conn, commands.SerializationTypeJSON, false, br)

	res := <-accepted
	if res.err != nil {
		_ = conn.Close()
		tb.Fatal(res.err)
	}

	tb.Cleanup(func() {
		_ = client.Close()
		_ = res.adapter.Close()
	})
	return res.adapter, client
}

func newWebTransportPair(tb testing.TB) (adapters.SocketAdapter, adapters.SocketAdapter) {
	serverTLS, clientTLS := SelfSignedTLS(tb)
	serverTLS.NextProtos = []string{http3.NextProtoH3}

	udpConn, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		tb.Fatal(err)
	}

	accepted := make(chan *webtransport.Session, 1)
	wtServer := &webtransport.Server{
		ApplicationProtocols: []string{"wsvpn"},
		H3: &http3.Server{
			TLSConfig:       serverTLS,
			EnableDatagrams: true,
			QUICConfig: &quic.Config{
				EnableStreamResetPartialDelivery: true,
				EnableDatagrams:                  true,
			},
		},
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	webtransport.ConfigureHTTP3Server(wtServer.H3)
	wtServer.H3.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess, err := wtServer.Upgrade(w, r)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		accepted <- sess
	})
	go func() { _ = wtServer.Serve(udpConn) }()
	tb.Cleanup(func() {
		_ = wtServer.Close()
		_ = udpConn.Close()
	})

	dialer := &webtransport.Transport{
		TLSClientConfig:      clientTLS,
		ApplicationProtocols: []string{"wsvpn"},
		QUICConfig: &quic.Config{
			EnableStreamResetPartialDelivery: true,
			EnableDatagrams:                  true,
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, clientSess, err := dialer.Dial(ctx, fmt.Sprintf("https://%s/", udpConn.LocalAddr().String()), http.Header{})
	if err != nil {
		tb.Fatal(err)
	}

	var serverSess *webtransport.Session
	select {
	case serverSess = <-accepted:
	case <-ctx.Done():
		tb.Fatal("timed out waiting for WebTransport session")
	}

	server := adapters.NewWebTransportAdapter(serverSess, commands.SerializationTypeJSON, true)
	client := adapters.NewWebTransportAdapter(clientSess, commands.SerializationTypeJSON, false)
	tb.Cleanup(func() {
		_ = client.Close()
		_ = server.Close()
	})
	return server, client
}

// Peer is one end of a connected socket pair.
type Peer struct {
	Socket *sockets.Socket
	Tun    *FakeTun
}

// PairOptions configures NewSocketPair.
type PairOptions struct {
	Transport     string
	MTU           int
	Fragmentation bool
	// Logger defaults to discarding output; pass a test logger to debug.
	Logger *log.Logger
}

// NewSocketPair connects two sockets over the configured transport, each
// with a fake TUN device attached, and waits until both completed version
// negotiation and are ready to pass data.
func NewSocketPair(tb testing.TB, opts PairOptions) (server *Peer, client *Peer) {
	tb.Helper()

	if opts.MTU == 0 {
		opts.MTU = 1400
	}
	logger := opts.Logger
	if logger == nil {
		logger = log.New(io.Discard, "", 0)
	}

	serverAdapter, clientAdapter := NewAdapterPair(tb, opts.Transport)

	mk := func(adapter adapters.SocketAdapter) *Peer {
		tun := NewFakeTun()
		sock := sockets.MakeSocket(logger, adapter, tun.Wrapper(), true, nil)
		sock.SetLocalFeature(features.Fragmentation, opts.Fragmentation)
		sock.SetMTU(opts.MTU)
		tb.Cleanup(sock.Close)
		return &Peer{Socket: sock, Tun: tun}
	}
	server = mk(serverAdapter)
	client = mk(clientAdapter)

	// Serve blocks until the adapter is ready, which for WebTransport needs
	// the peer to open its stream concurrently.
	var wg sync.WaitGroup
	for _, p := range []*Peer{server, client} {
		wg.Add(1)
		go func(p *Peer) {
			defer wg.Done()
			p.Socket.Serve()
			p.Socket.WaitReady()
		}(p)
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		tb.Fatal("timed out waiting for socket pair to become ready")
	}

	return server, client
}

var ErrTimeout = errors.New("timed out waiting for packet")

// Receive waits for the next packet the socket wrote to this peer's device.
func (p *Peer) Receive(timeout time.Duration) ([]byte, error) {
	select {
	case pkt := <-p.Tun.Out:
		return pkt, nil
	case <-time.After(timeout):
		return nil, ErrTimeout
	}
}
