package wstun_shared

import (
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/songgao/water"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var lastCommandId uint64 = 0

type CommandHandler func(args []string) error

type Socket struct {
	connId    string
	conn      *websocket.Conn
	iface     *water.Interface
	writeLock *sync.Mutex
	wg        *sync.WaitGroup
	handlers  map[string]CommandHandler
}

func MakeSocket(connId string, conn *websocket.Conn, iface *water.Interface) *Socket {
	return &Socket{
		connId:    connId,
		conn:      conn,
		iface:     iface,
		writeLock: &sync.Mutex{},
		wg:        &sync.WaitGroup{},
		handlers:  make(map[string]CommandHandler),
	}
}

func (s *Socket) AddCommandHandler(command string, handler CommandHandler) {
	s.handlers[command] = handler
}

func (s *Socket) Wait() {
	s.wg.Wait()
}

func (s *Socket) RawSendCommand(commandId string, command string, args ...string) error {
	data := []byte(fmt.Sprintf("%s|%s|%s", commandId, command, strings.Join(args, "|")))
	s.writeLock.Lock()
	err := s.conn.WriteMessage(websocket.TextMessage, data)
	s.writeLock.Unlock()
	return err
}

func (s *Socket) SendCommand(command string, args ...string) error {
	return s.RawSendCommand(fmt.Sprintf("%d", atomic.AddUint64(&lastCommandId, 1)), command, args...)
}

func (s *Socket) closeDone() {
	s.wg.Done()
	s.Close()
}

func (s *Socket) Close() {
	s.writeLock.Lock()
	s.conn.Close()
	s.writeLock.Unlock()
	s.iface.Close()
}

func (s *Socket) Serve() {
	s.wg.Add(3)

	go func() {
		defer s.closeDone()

		packet := make([]byte, 2000)

		for {
			n, err := s.iface.Read(packet)
			if err != nil {
				log.Printf("[%s] Error reading packet from tun: %v", s.connId, err)
				break
			}
			s.writeLock.Lock()
			err = s.conn.WriteMessage(websocket.BinaryMessage, packet[:n])
			s.writeLock.Unlock()
			if err != nil {
				log.Printf("[%s] Error writing packet to WS: %v", s.connId, err)
				break
			}
		}
	}()

	go func() {
		defer s.closeDone()

		for {
			msgType, msg, err := s.conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
					log.Printf("[%s] Error reading packet from WS: %v", s.connId, err)
				}
				break
			}
			if msgType == websocket.BinaryMessage {
				s.iface.Write(msg)
			} else if msgType == websocket.TextMessage {
				str := strings.Split(string(msg), "|")
				if len(str) < 2 {
					log.Printf("[%s] Invalid in-band command structure", s.connId)
					continue
				}

				commandId := str[0]
				commandName := str[1]
				if commandName == "reply" {
					commandResult := "N/A"
					if len(str) > 2 {
						commandResult = str[2]
					}
					log.Printf("[%s] Got command reply ID %s: %s", s.connId, commandId, commandResult)
					continue
				}

				handler := s.handlers[commandName]
				if handler == nil {
					err = errors.New("Unknown command")
				} else {
					err = handler(str[2:])
				}
				if err != nil {
					log.Printf("[%s] Error in in-band command %s: %v", s.connId, commandName, err)
				}

				s.RawSendCommand(commandId, "reply", fmt.Sprintf("%v", err == nil))
			}
		}
	}()

	timeout := time.Duration(30) * time.Second

	lastResponse := time.Now()
	s.conn.SetPongHandler(func(msg string) error {
		lastResponse = time.Now()
		return nil
	})

	go func() {
		defer s.closeDone()

		for {
			s.writeLock.Lock()
			err := s.conn.WriteMessage(websocket.PingMessage, []byte{})
			s.writeLock.Unlock()
			if err != nil {
				log.Printf("[%s] Error writing ping frame: %v", s.connId, err)
				break
			}
			time.Sleep(timeout / 2)
			if time.Now().Sub(lastResponse) > timeout {
				log.Printf("[%s] Ping timeout", s.connId)
				break
			}
		}
	}()
}
