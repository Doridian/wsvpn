package adapters

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Doridian/wsvpn/shared"
	"github.com/Doridian/wsvpn/shared/commands"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/webtransport-go"
)

type StreamMessageType = byte

const ErrorCodeClosed = 1

const (
	// quic-go never sends UDP payloads larger than this, so no datagram can
	// be larger either. We start here and learn the actual (smaller) limit
	// from DatagramTooLargeError, which tracks Path MTU Discovery.
	optimisticMaxPayloadLen = 1452
	// WebTransport prefixes every datagram with the session's quarter stream
	// ID as a varint after quic-go's size check, so its limit includes it.
	// Our single session uses stream 0 (1 byte), 2 bytes covers up to 16383.
	datagramPrefixReserve = 2
	// PMTUD only ever raises quic-go's limit after the handshake, so retry
	// the optimistic size every so often to pick that up
	maxPayloadLenProbeInterval = time.Second
	// Never go below this, fragmentation needs room for its own header
	minMaxPayloadLen = 64
	// http3 queues only 32 datagrams per session and silently drops the
	// rest, after QUIC already acknowledged them (so congestion control never
	// backs off). We move them into our own, much deeper queue right away.
	datagramReceiveQueueLen = 1024
)

const (
	messageTypeControl StreamMessageType = iota
	messageTypePing
	messageTypePong
)

type WebTransportAdapter struct {
	adapterBase
	conn               *webtransport.Session
	stream             *webtransport.Stream
	isServer           bool
	wg                 *sync.WaitGroup
	readyWait          *sync.Cond
	isReady            atomic.Bool
	isFullyInitialized atomic.Bool
	serializationType  commands.SerializationType

	// The control stream is written from several goroutines (commands,
	// replies, pings and pongs), quic-go streams are not safe for that
	writeLock *sync.Mutex

	serveErrorLock           *sync.Mutex
	lastServeError           error
	lastServeErrorUnexpected bool

	maxPayloadLen       atomic.Uint32
	maxPayloadLenProbed atomic.Int64
}

var _ SocketAdapter = &WebTransportAdapter{}

func NewWebTransportAdapter(conn *webtransport.Session, serializationType commands.SerializationType, isServer bool) *WebTransportAdapter {
	adapter := &WebTransportAdapter{
		conn:              conn,
		isServer:          isServer,
		readyWait:         shared.MakeSimpleCond(),
		wg:                &sync.WaitGroup{},
		serializationType: serializationType,
		writeLock:         &sync.Mutex{},
		serveErrorLock:    &sync.Mutex{},
	}
	return adapter
}

func (s *WebTransportAdapter) IsServer() bool {
	return s.isServer
}

func (s *WebTransportAdapter) IsClient() bool {
	return !s.isServer
}

func (s *WebTransportAdapter) LocalAddr() net.Addr {
	return s.conn.LocalAddr()
}

func (s *WebTransportAdapter) RemoteAddr() net.Addr {
	return s.conn.RemoteAddr()
}

func (s *WebTransportAdapter) Close() error {
	if s.stream != nil {
		s.stream.CancelRead(ErrorCodeClosed)
		s.stream.CancelWrite(ErrorCodeClosed)
		_ = s.stream.Close()
	}
	return s.conn.CloseWithError(ErrorCodeClosed, "Close called")
}

func (s *WebTransportAdapter) GetTLSConnectionState() (tls.ConnectionState, bool) {
	return s.conn.SessionState().ConnectionState.TLS, true
}

func (s *WebTransportAdapter) setReady() {
	s.readyWait.L.Lock()
	s.isReady.Store(true)
	s.readyWait.Broadcast()
	s.readyWait.L.Unlock()
}

func (s *WebTransportAdapter) RefreshFeatures() {
	s.maxPayloadLen.Store(optimisticMaxPayloadLen)
}

func (s *WebTransportAdapter) Serve() (bool, error) {
	var err error

	if s.isServer {
		s.stream, err = s.conn.AcceptStream(context.Background())
	} else {
		s.stream, err = s.conn.OpenStreamSync(context.Background())
	}

	if err != nil {
		s.setReady()
		return true, err
	}

	s.RefreshFeatures()

	s.wg.Add(1)
	go s.serveStream()

	s.wg.Add(1)
	go s.serveData()

	s.isFullyInitialized.Store(true)
	s.setReady()

	s.wg.Wait()

	s.serveErrorLock.Lock()
	defer s.serveErrorLock.Unlock()
	return s.lastServeErrorUnexpected, s.lastServeError
}

func (s *WebTransportAdapter) handleServeError(err error, unexpected bool) {
	s.serveErrorLock.Lock()
	defer s.serveErrorLock.Unlock()
	if s.lastServeError == nil {
		s.lastServeError = err
		s.lastServeErrorUnexpected = unexpected
	}
}

func (s *WebTransportAdapter) serveStream() {
	defer s.wg.Done()
	defer func() {
		_ = s.Close()
	}()

	var msgLen uint16
	var msgLenLower uint8
	var msgLenUpper uint8

	var msgType StreamMessageType
	var err error
	reader := bufio.NewReader(s.stream)

serveLoop:
	for {
		msgType, err = reader.ReadByte()
		if err != nil {
			break
		}

		switch msgType {
		case messageTypeControl:
			msgLenUpper, err = reader.ReadByte()
			if err != nil {
				break
			}

			msgLenLower, err = reader.ReadByte()
			if err != nil {
				break
			}

			msgLen = uint16(msgLenLower) | (uint16(msgLenUpper) << 8)

			data := make([]byte, msgLen)
			_, err = io.ReadFull(reader, data)
			if err != nil {
				break
			}

			s.controlMessageHandler(data)

		case messageTypePing:
			err = s.writeStream([]byte{messageTypePong})
			if err != nil {
				break serveLoop
			}

		case messageTypePong:
			if s.pongHandler != nil {
				s.pongHandler()
			}
		}
	}

	if err != nil {
		s.handleServeError(err, err != io.EOF)
	}
}

func (s *WebTransportAdapter) serveData() {
	defer s.wg.Done()
	defer func() {
		_ = s.Close()
	}()

	queue := make(chan []byte, datagramReceiveQueueLen)
	defer close(queue)

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		for data := range queue {
			s.dataMessageHandler(data)
		}
	}()

	for {
		data, err := s.conn.ReceiveDatagram(context.Background())
		if err != nil {
			s.handleServeError(err, true)
			break
		}
		queue <- data
	}
}

func (s *WebTransportAdapter) WaitReady() {
	s.readyWait.L.Lock()
	defer s.readyWait.L.Unlock()
	for !s.isReady.Load() {
		s.readyWait.Wait()
	}
}

func (s *WebTransportAdapter) writeStream(msg []byte) error {
	s.writeLock.Lock()
	defer s.writeLock.Unlock()

	n, err := s.stream.Write(msg)
	if err == nil && n != len(msg) {
		return errors.New("could not write full message")
	}
	return err
}

func (s *WebTransportAdapter) WriteControlMessage(message []byte) error {
	if !s.isFullyInitialized.Load() {
		return errors.New("not able to send")
	}

	msgLen := len(message)
	if msgLen > 0xFFFF {
		return errors.New("control message too large")
	}

	buf := &bytes.Buffer{}
	buf.WriteByte(messageTypeControl)
	buf.WriteByte(uint8((msgLen >> 8) & 0xFF))
	buf.WriteByte(uint8(msgLen & 0xFF))
	buf.Write(message)

	return s.writeStream(buf.Bytes())
}

func (s *WebTransportAdapter) MaxDataPayloadLen() uint16 {
	maxPayloadLen := s.maxPayloadLen.Load()
	if maxPayloadLen < optimisticMaxPayloadLen {
		probed := s.maxPayloadLenProbed.Load()
		if time.Since(time.Unix(0, probed)) > maxPayloadLenProbeInterval &&
			s.maxPayloadLenProbed.CompareAndSwap(probed, time.Now().UnixNano()) {
			s.maxPayloadLen.CompareAndSwap(maxPayloadLen, optimisticMaxPayloadLen)
			return optimisticMaxPayloadLen
		}
	}
	return uint16(maxPayloadLen)
}

func (s *WebTransportAdapter) WriteDataMessage(message []byte) error {
	if !s.isFullyInitialized.Load() {
		return errors.New("not able to send")
	}

	maxPayloadLen := s.maxPayloadLen.Load()
	if len(message) > int(maxPayloadLen) {
		return ErrDataPayloadLimitReduced
	}

	err := s.conn.SendDatagram(message)
	if err != nil {
		tooLargeErr := &quic.DatagramTooLargeError{}
		if errors.As(err, &tooLargeErr) {
			// Always make progress: this message did not fit, so the new
			// limit must be below its length even if the reserve is off
			newMaxPayloadLen := min(
				tooLargeErr.MaxDatagramPayloadSize-datagramPrefixReserve,
				int64(len(message)-1),
			)
			if newMaxPayloadLen < minMaxPayloadLen {
				return err
			}
			s.maxPayloadLen.CompareAndSwap(maxPayloadLen, uint32(newMaxPayloadLen))
			s.maxPayloadLenProbed.Store(time.Now().UnixNano())
			return ErrDataPayloadLimitReduced
		}
	}
	return err
}

func (s *WebTransportAdapter) WritePingMessage() error {
	if !s.isFullyInitialized.Load() {
		return errors.New("not ready")
	}

	return s.writeStream([]byte{messageTypePing})
}

func (s *WebTransportAdapter) Name() string {
	return "WebTransport"
}

func (s *WebTransportAdapter) GetCommandSerializationType() commands.SerializationType {
	return s.serializationType
}
