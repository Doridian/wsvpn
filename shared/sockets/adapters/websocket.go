package adapters

import (
	"crypto/tls"
	"errors"
	"fmt"
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

	readState  ws.State
	writeState ws.State
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
		wsa.readState = ws.StateClientSide
		wsa.writeState = ws.StateServerSide
	} else {
		wsa.readState = ws.StateServerSide
		wsa.writeState = ws.StateClientSide
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

func (s *WebSocketAdapter) Serve() (bool, error) {
	defer s.Close()

	for {
		msg, msgType, err := wsutil.ReadData(s.conn, s.readState)
		if err != nil {
			return true, err
		}

		res := true
		if msgType == ws.OpText {
			res = s.controlMessageHandler(msg)
		} else if msgType == ws.OpBinary {
			res = s.dataMessageHandler(msg)
		} else if msgType == ws.OpPing {
			_ = wsutil.WriteMessage(s.conn, s.writeState, ws.OpPong, msg)
		} else if msgType == ws.OpPong {
			if s.pongHandler != nil {
				s.pongHandler()
			}
		} else if msgType == ws.OpClose {
			return true, fmt.Errorf("client closed connection: %v", msg)
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
	return wsutil.WriteMessage(s.conn, s.writeState, ws.OpText, message)
}

func (s *WebSocketAdapter) WriteDataMessage(message []byte) error {
	s.writeLock.Lock()
	defer s.writeLock.Unlock()
	return wsutil.WriteMessage(s.conn, s.writeState, ws.OpBinary, message)
}

func (s *WebSocketAdapter) WritePingMessage() error {
	s.writeLock.Lock()
	defer s.writeLock.Unlock()
	return wsutil.WriteMessage(s.conn, s.writeState, ws.OpPing, []byte{})
}

func (s *WebSocketAdapter) Name() string {
	return "WebSocket"
}

func (s *WebSocketAdapter) GetCommandSerializationType() commands.SerializationType {
	return s.serializationType
}
