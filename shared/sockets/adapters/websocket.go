package adapters

import (
	"bufio"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"sync"

	"github.com/Doridian/wsvpn/shared/commands"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
)

type WebSocketAdapter struct {
	adapterBase
	conn              net.Conn
	writeLock         *sync.Mutex
	serializationType commands.SerializationType
	isServer          bool
	initial           *bufio.Reader

	wsState    ws.State
	dataWriter *wsutil.Writer
}

var _ SocketAdapter = &WebSocketAdapter{}

func NewWebSocketAdapter(conn net.Conn, serializationType commands.SerializationType, isServer bool, initial *bufio.Reader) *WebSocketAdapter {
	wsa := &WebSocketAdapter{
		conn:              conn,
		writeLock:         &sync.Mutex{},
		serializationType: serializationType,
		isServer:          isServer,
		initial:           initial,
	}

	if isServer {
		wsa.wsState = ws.StateServerSide
	} else {
		wsa.wsState = ws.StateClientSide
	}
	wsa.dataWriter = wsutil.NewWriter(conn, wsa.wsState, ws.OpBinary)

	return wsa
}

func (s *WebSocketAdapter) IsServer() bool {
	return s.isServer
}

func (s *WebSocketAdapter) IsClient() bool {
	return !s.isServer
}

func (s *WebSocketAdapter) LocalAddr() net.Addr {
	return s.conn.LocalAddr()
}

func (s *WebSocketAdapter) RemoteAddr() net.Addr {
	return s.conn.RemoteAddr()
}

func (s *WebSocketAdapter) RefreshFeatures() {

}

func (s *WebSocketAdapter) GetTLSConnectionState() (tls.ConnectionState, bool) {
	tlsConn, ok := s.conn.(*tls.Conn)
	if !ok {
		return tls.ConnectionState{}, false
	}
	return tlsConn.ConnectionState(), true
}

func (s *WebSocketAdapter) writePongMessage(data []byte) error {
	s.writeLock.Lock()
	defer s.writeLock.Unlock()
	return wsutil.WriteMessage(s.conn, s.wsState, ws.OpPong, data)
}

func (s *WebSocketAdapter) handleFrame(hdr ws.Header, data []byte) error {
	switch hdr.OpCode {
	case ws.OpText:
		res := s.controlMessageHandler(data)
		if !res {
			return errors.New("error in control message")
		}
	case ws.OpBinary:
		res := s.dataMessageHandler(data)
		if !res {
			return errors.New("error in data message")
		}
	case ws.OpPing:
		err := s.writePongMessage(data)
		if err != nil {
			return err
		}
	case ws.OpPong:
		if s.pongHandler != nil {
			s.pongHandler()
		}
	}
	return nil
}

func (s *WebSocketAdapter) handleInitial() error {
	if s.initial == nil {
		return nil
	}

	defer func() {
		ws.PutReader(s.initial)
		s.initial = nil
	}()

	f, err := ws.ReadFrame(s.initial)
	if err == nil {
		return s.handleFrame(f.Header, f.Payload)
	}

	if errors.Is(err, io.EOF) {
		return nil
	}
	return err
}

func (s *WebSocketAdapter) Serve() (bool, error) {
	err := s.handleInitial()
	if err != nil {
		return true, err
	}

	reader := wsutil.NewReader(s.conn, s.wsState)
	messageBuf := make([]byte, s.MaxDataPayloadLen())

	for {
		hdr, err := reader.NextFrame()
		if err != nil {
			return true, err
		}

		if hdr.OpCode == ws.OpClose {
			return false, errors.New("received close frame")
		}

		if hdr.Length > int64(len(messageBuf)) {
			return true, errors.New("data payload too large")
		}

		msg := messageBuf[:hdr.Length]
		_, err = io.ReadFull(reader, msg)
		if err != nil {
			return true, err
		}

		err = s.handleFrame(hdr, msg)
		if err != nil {
			return true, err
		}
	}
}

func (s *WebSocketAdapter) WaitReady() {
	s.writeLock.Lock()
	defer s.writeLock.Unlock()
}

func (s *WebSocketAdapter) Close() error {
	s.writeLock.Lock()
	defer s.writeLock.Unlock()
	return s.conn.Close()
}

func (s *WebSocketAdapter) MaxDataPayloadLen() uint16 {
	return 0xFFFF
}

func (s *WebSocketAdapter) WriteControlMessage(message []byte) error {
	s.writeLock.Lock()
	defer s.writeLock.Unlock()
	return wsutil.WriteMessage(s.conn, s.wsState, ws.OpText, message)
}

func (s *WebSocketAdapter) WriteDataMessage(message []byte) error {
	s.writeLock.Lock()
	defer s.writeLock.Unlock()
	_, err := s.dataWriter.Write(message)
	if err != nil {
		return err
	}
	err = s.dataWriter.Flush()
	return err
}

func (s *WebSocketAdapter) WritePingMessage() error {
	s.writeLock.Lock()
	defer s.writeLock.Unlock()
	return wsutil.WriteMessage(s.conn, s.wsState, ws.OpPing, []byte{})
}

func (s *WebSocketAdapter) Name() string {
	return "WebSocket"
}

func (s *WebSocketAdapter) GetCommandSerializationType() commands.SerializationType {
	return s.serializationType
}
