package shared

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

var defaultMac = [6]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}

var learnMac bool = false
var macTable map[MacAddr]*Socket = make(map[MacAddr]*Socket)
var macLock sync.RWMutex

func FindSocketByMAC(mac MacAddr) *Socket {
	macLock.RLock()
	defer macLock.RUnlock()

	return macTable[mac]
}

func BroadcastMessage(msgType int, data []byte, skip *Socket) {
	macLock.RLock()
	targetList := make([]*Socket, 0)
	for _, v := range macTable {
		if v == skip {
			continue
		}
		targetList = append(targetList, v)
	}
	macLock.RUnlock()

	for _, v := range targetList {
		v.WriteMessage(msgType, data)
	}
}

type CommandHandler func(args []string) error

type Socket struct {
	connId        string
	conn          *websocket.Conn
	iface         *water.Interface
	noIfaceReader bool
	writeLock     *sync.Mutex
	wg            *sync.WaitGroup
	handlers      map[string]CommandHandler
	closechan     chan bool
	closechanopen bool
	mac           MacAddr
}

func SetMACLearning(enable bool) {
	learnMac = enable
}

func MakeSocket(connId string, conn *websocket.Conn, iface *water.Interface, noIfaceReader bool) *Socket {
	return &Socket{
		connId:        connId,
		conn:          conn,
		iface:         iface,
		noIfaceReader: noIfaceReader,
		writeLock:     &sync.Mutex{},
		wg:            &sync.WaitGroup{},
		handlers:      make(map[string]CommandHandler),
		closechan:     make(chan bool),
		closechanopen: true,
		mac:           defaultMac,
	}
}

func (s *Socket) AddCommandHandler(command string, handler CommandHandler) {
	s.handlers[command] = handler
}

func (s *Socket) Wait() {
	s.wg.Wait()
}

func (s *Socket) rawSendCommand(commandId string, command string, args ...string) error {
	return s.WriteMessage(websocket.TextMessage,
		[]byte(fmt.Sprintf("%s|%s|%s", commandId, command, strings.Join(args, "|"))))
}

func (s *Socket) SendCommand(command string, args ...string) error {
	return s.rawSendCommand(fmt.Sprintf("%d", atomic.AddUint64(&lastCommandId, 1)), command, args...)
}

func (s *Socket) WriteMessage(msgType int, data []byte) error {
	s.writeLock.Lock()
	err := s.conn.WriteMessage(msgType, data)
	s.writeLock.Unlock()
	if err != nil {
		log.Printf("[%s] Error writing packet to WS: %v", s.connId, err)
		s.Close()
	}
	return err
}

func (s *Socket) closeDone() {
	s.wg.Done()
	s.Close()
}

func (s *Socket) SetInterface(iface *water.Interface) error {
	s.writeLock.Lock()
	defer s.writeLock.Unlock()

	if s.iface != nil {
		return errors.New("Cannot re-define interface. Already set.")
	}
	s.iface = iface
	s.tryServeIfaceRead()
	return nil
}

func (s *Socket) setMACFrom(msg []byte) {
	srcMac := GetSrcMAC(msg)
	if !MACIsUnicast(srcMac) || srcMac == s.mac {
		return
	}

	macLock.Lock()
	defer macLock.Unlock()
	if s.mac != defaultMac {
		delete(macTable, s.mac)
	}
	if macTable[srcMac] != nil {
		s.mac = defaultMac
		log.Printf("[%d] MAC collision. Killing.", s.connId)
		s.Close()
		return
	}
	s.mac = srcMac
	macTable[srcMac] = s
}

func (s *Socket) Close() {
	s.writeLock.Lock()
	defer s.writeLock.Unlock()
	s.conn.Close()
	if s.iface != nil && !learnMac {
		s.iface.Close()
	}
	if s.closechanopen {
		s.closechanopen = false
		close(s.closechan)
	}
	if s.mac != defaultMac {
		macLock.Lock()
		delete(macTable, s.mac)
		s.mac = defaultMac
		macLock.Unlock()
	}
}

func (s *Socket) tryServeIfaceRead() {
	if s.iface == nil || s.noIfaceReader {
		return
	}

	s.wg.Add(1)
	go func() {
		defer s.closeDone()

		packet := make([]byte, 2000)

		for {
			n, err := s.iface.Read(packet)
			if err != nil {
				log.Printf("[%s] Error reading packet from tun: %v", s.connId, err)
				return
			}

			err = s.WriteMessage(websocket.BinaryMessage, packet[:n])
			if err != nil {
				return
			}
		}
	}()
}

func (s *Socket) Serve() {
	s.writeLock.Lock()
	defer s.writeLock.Unlock()
	s.tryServeIfaceRead()

	s.wg.Add(1)
	go func() {
		defer s.closeDone()

		for {
			msgType, msg, err := s.conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
					log.Printf("[%s] Error reading packet from WS: %v", s.connId, err)
				}
				return
			}

			if msgType == websocket.BinaryMessage {
				if learnMac && len(msg) >= 14 {
					s.setMACFrom(msg)

					dest := GetDestMAC(msg)
					isUnicast := MACIsUnicast(dest)

					var sd *Socket
					if isUnicast {
						sd = FindSocketByMAC(dest)
						if sd != nil {
							sd.WriteMessage(websocket.BinaryMessage, msg)
							continue
						}
					} else {
						BroadcastMessage(websocket.BinaryMessage, msg, s)
					}
				}

				if s.iface == nil {
					continue
				}
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

				s.rawSendCommand(commandId, "reply", fmt.Sprintf("%v", err == nil))
			}
		}
	}()

	timeout := time.Duration(30) * time.Second

	lastResponse := time.Now()
	s.conn.SetPongHandler(func(msg string) error {
		lastResponse = time.Now()
		return nil
	})

	s.wg.Add(1)
	go func() {
		defer s.closeDone()

		for {
			select {
			case <-time.After(timeout / 2):
				if time.Now().Sub(lastResponse) > timeout {
					log.Printf("[%s] Ping timeout", s.connId)
					return
				}
				err := s.WriteMessage(websocket.PingMessage, []byte{})
				if err != nil {
					return
				}
			case <-s.closechan:
				return
			}
		}
	}()
}
