package adapters

import (
	"crypto/tls"
	"errors"
	"io"
	"log"
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

	wsState ws.State
}

var _ SocketAdapter = &WebSocketAdapter{}

func NewWebSocketAdapter(conn net.Conn, serializationType commands.SerializationType, isServer bool) *WebSocketAdapter {
	wsa := &WebSocketAdapter{
		conn:              conn,
		writeLock:         &sync.Mutex{},
		serializationType: serializationType,
		isServer:          isServer,
	}

	if isServer {
		wsa.wsState = ws.StateServerSide
	} else {
		wsa.wsState = ws.StateClientSide
	}

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

func (s *WebSocketAdapter) Serve() (bool, error) {
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
			return true, ErrDataPayloadTooLarge
		}

		msg := messageBuf[:hdr.Length]
		_, err = io.ReadFull(reader, msg)
		if err != nil {
			return false, err
		}

		log.Printf("[F] %x %v %x", hdr.OpCode, string(msg), reader.State)

		res := true
		switch hdr.OpCode {
		case ws.OpText:
			res = s.controlMessageHandler(msg)
		case ws.OpBinary:
			res = s.dataMessageHandler(msg)
		case ws.OpPing:
			err = s.writePongMessage(msg)
			if err != nil {
				return true, err
			}
		case ws.OpPong:
			if s.pongHandler != nil {
				s.pongHandler()
			}
		}

		if !res {
			break
		}
	}

	return true, errors.New("Serve terminated")
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
	return wsutil.WriteMessage(s.conn, s.wsState, ws.OpBinary, message)
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
