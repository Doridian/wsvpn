package adapters

import (
	"crypto/tls"
	"errors"
	"sync"

	"github.com/Doridian/wsvpn/shared/commands"
	"github.com/gorilla/websocket"
)

type WebSocketAdapter struct {
	adapterBase
	conn              *websocket.Conn
	writeLock         *sync.Mutex
	serializationType commands.SerializationType
	isServer          bool
}

var _ SocketAdapter = &WebSocketAdapter{}

func NewWebSocketAdapter(conn *websocket.Conn, serializationType commands.SerializationType, isServer bool) *WebSocketAdapter {
	return &WebSocketAdapter{
		conn:              conn,
		writeLock:         &sync.Mutex{},
		serializationType: serializationType,
		isServer:          isServer,
	}
}

func (s *WebSocketAdapter) IsServer() bool {
	return s.isServer
}

func (s *WebSocketAdapter) IsClient() bool {
	return !s.isServer
}

func (s *WebSocketAdapter) RefreshFeatures() {

}

func (s *WebSocketAdapter) GetTLSConnectionState() (tls.ConnectionState, bool) {
	tlsConn, ok := s.conn.UnderlyingConn().(*tls.Conn)
	if !ok {
		return tls.ConnectionState{}, false
	}
	return tlsConn.ConnectionState(), true
}

func (s *WebSocketAdapter) Serve() (error, bool) {
	s.conn.SetPongHandler(func(appData string) error {
		if s.pongHandler != nil {
			s.pongHandler()
		}
		return nil
	})

	for {
		msgType, msg, err := s.conn.ReadMessage()
		if err != nil {
			return err, websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway)
		}

		var res bool
		if msgType == websocket.TextMessage {
			res = s.controlMessageHandler(msg)
		} else if msgType == websocket.BinaryMessage {
			res = s.dataMessageHandler(msg)
		} else {
			res = true
		}

		if !res {
			break
		}
	}

	return errors.New("Serve terminated"), true
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
	return s.conn.WriteMessage(websocket.TextMessage, message)
}

func (s *WebSocketAdapter) WriteDataMessage(message []byte) error {
	s.writeLock.Lock()
	defer s.writeLock.Unlock()
	return s.conn.WriteMessage(websocket.BinaryMessage, message)
}

func (s *WebSocketAdapter) WritePingMessage() error {
	s.writeLock.Lock()
	defer s.writeLock.Unlock()
	return s.conn.WriteMessage(websocket.PingMessage, []byte{})
}

func (s *WebSocketAdapter) Name() string {
	return "WebSocket"
}

func (s *WebSocketAdapter) GetCommandSerializationType() commands.SerializationType {
	return s.serializationType
}
