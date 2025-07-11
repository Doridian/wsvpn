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

	"github.com/Doridian/wsvpn/shared"
	"github.com/Doridian/wsvpn/shared/commands"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/webtransport-go"
)

type StreamMessageType = byte

const ErrorCodeClosed = 1

const (
	messageTypeControl StreamMessageType = iota
	messageTypePing
	messageTypePong
)

type WebTransportAdapter struct {
	adapterBase
	conn               *webtransport.Session
	netConn            net.Conn
	stream             *webtransport.Stream
	isServer           bool
	wg                 *sync.WaitGroup
	readyWait          *sync.Cond
	isReady            bool
	isFullyInitialized bool
	serializationType  commands.SerializationType

	lastServeError           error
	lastServeErrorUnexpected bool

	maxPayloadLen uint16
}

var _ SocketAdapter = &WebTransportAdapter{}

func NewWebTransportAdapter(conn *webtransport.Session, netConn net.Conn, serializationType commands.SerializationType, isServer bool) *WebTransportAdapter {
	adapter := &WebTransportAdapter{
		conn:               conn,
		netConn:            netConn,
		isServer:           isServer,
		readyWait:          shared.MakeSimpleCond(),
		wg:                 &sync.WaitGroup{},
		isReady:            false,
		isFullyInitialized: false,
		serializationType:  serializationType,
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
	err := s.conn.CloseWithError(ErrorCodeClosed, "Close called")
	if s.netConn != nil {
		_ = s.netConn.Close()
	}
	return err
}

func (s *WebTransportAdapter) GetTLSConnectionState() (tls.ConnectionState, bool) {
	return s.conn.ConnectionState().TLS, true
}

func (s *WebTransportAdapter) setReady() {
	s.isReady = true
	s.readyWait.Broadcast()
}

func (s *WebTransportAdapter) RefreshFeatures() {
	// The estimate from the errors below is sadly quite bad
	// By bad I mean there is often no error when the payload
	// is clearly too large and doesn't get sent
	s.maxPayloadLen = uint16(1200 - 16)
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

	s.isFullyInitialized = true
	s.setReady()

	s.wg.Wait()

	return s.lastServeErrorUnexpected, s.lastServeError
}

func (s *WebTransportAdapter) handleServeError(err error, unexpected bool) {
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
			_, err = s.stream.Write([]byte{messageTypePong})
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

	for {
		data, err := s.conn.ReceiveDatagram(context.Background())
		if err != nil {
			s.handleServeError(err, true)
			break
		}
		s.dataMessageHandler(data)
	}
}

func (s *WebTransportAdapter) WaitReady() {
	for !s.isReady {
		s.readyWait.L.Lock()
		s.readyWait.Wait()
		s.readyWait.L.Unlock()
	}
}

func (s *WebTransportAdapter) WriteControlMessage(message []byte) error {
	if !s.isFullyInitialized {
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

	msg := buf.Bytes()

	n, err := s.stream.Write(msg)
	if err == nil && n != len(msg) {
		return errors.New("could not write full message")
	}

	return err
}

func (s *WebTransportAdapter) MaxDataPayloadLen() uint16 {
	return s.maxPayloadLen
}

func (s *WebTransportAdapter) WriteDataMessage(message []byte) error {
	if !s.isFullyInitialized {
		return errors.New("not able to send")
	}

	if len(message) > int(s.maxPayloadLen) {
		return errors.New("data payload too large")
	}

	err := s.conn.SendDatagram(message)
	if err != nil {
		tooLargeErr := &quic.DatagramTooLargeError{}
		if errors.As(err, &tooLargeErr) {
			if tooLargeErr.MaxDatagramPayloadSize < int64(s.maxPayloadLen) {
				s.maxPayloadLen = uint16(tooLargeErr.MaxDatagramPayloadSize)
				return ErrDataPayloadLimitReduced
			}
		}
	}
	return err
}

func (s *WebTransportAdapter) WritePingMessage() error {
	if !s.isReady {
		return errors.New("not ready")
	}

	_, err := s.stream.Write([]byte{messageTypePing})
	return err
}

func (s *WebTransportAdapter) Name() string {
	return "WebTransport"
}

func (s *WebTransportAdapter) GetCommandSerializationType() commands.SerializationType {
	return s.serializationType
}
