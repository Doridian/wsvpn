package shared

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/songgao/water"
)

var pingIntervalPtr = flag.Duration("ping-interval", time.Second*time.Duration(30), "Send ping frames in this interval")
var pingTimeoutPtr = flag.Duration("ping-timeout", time.Second*time.Duration(5), "Disconnect if no ping response is received after timeout")

var defaultMac = [6]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}

var multiClientIface bool = false
var allowClientToClient bool = false
var macTable map[MacAddr]*Socket = make(map[MacAddr]*Socket)
var macLock sync.RWMutex
var allSockets = make(map[*Socket]*Socket)
var allSocketsLock sync.RWMutex

func FindSocketByMAC(mac MacAddr) *Socket {
	macLock.RLock()
	defer macLock.RUnlock()

	return macTable[mac]
}

func BroadcastMessage(msgType int, data []byte, skip *Socket) {
	allSocketsLock.RLock()
	targetList := make([]*Socket, 0)
	for _, v := range allSockets {
		if v == skip {
			continue
		}
		targetList = append(targetList, v)
	}
	allSocketsLock.RUnlock()

	for _, v := range targetList {
		v.WriteMessage(msgType, data)
	}
}

type CommandHandler func(args []string) error

type Socket struct {
	lastCommandId         uint64 // This MUST be the first element of the struct, see https://github.com/golang/go/issues/23345
	connId                string
	conn                  *websocket.Conn
	iface                 *water.Interface
	noIfaceReader         bool
	writeLock             *sync.Mutex
	wg                    *sync.WaitGroup
	handlers              map[string]CommandHandler
	closechan             chan bool
	closechanopen         bool
	mac                   MacAddr
	remoteProtocolVersion int
}

func SetMultiClientIfaceMode(enable bool) {
	multiClientIface = enable
}

func SetClientToClient(enable bool) {
	allowClientToClient = enable
}

func MakeSocket(connId string, conn *websocket.Conn, iface *water.Interface, noIfaceReader bool) *Socket {
	return &Socket{
		connId:                connId,
		conn:                  conn,
		iface:                 iface,
		noIfaceReader:         noIfaceReader,
		writeLock:             &sync.Mutex{},
		wg:                    &sync.WaitGroup{},
		handlers:              make(map[string]CommandHandler),
		closechan:             make(chan bool),
		closechanopen:         true,
		mac:                   defaultMac,
		lastCommandId:         0,
		remoteProtocolVersion: 0,
	}
}

func (s *Socket) AddCommandHandler(command string, handler CommandHandler) {
	s.handlers[command] = handler
}

func (s *Socket) registerDefaultCommandHandlers() {
	s.AddCommandHandler("version", func(args []string) error {
		if len(args) != 2 {
			return errors.New("version command needs 2 arguments")
		}
		protocolVersion, err := strconv.Atoi(args[0])
		if err != nil {
			return err
		}
		s.remoteProtocolVersion = protocolVersion
		version := args[1]
		log.Printf("[%s] Remote version is: %s (protocol %d)", s.connId, version, protocolVersion)
		return nil
	})
}

func (s *Socket) sendDefaultWelcome() {
	s.SendCommand("version", fmt.Sprintf("%d", ProtocolVersion), Version)
}

func (s *Socket) Wait() {
	s.wg.Wait()
}

func (s *Socket) rawSendCommand(commandId string, command string, args ...string) error {
	return s.WriteMessage(websocket.TextMessage,
		[]byte(fmt.Sprintf("%s|%s|%s", commandId, command, strings.Join(args, "|"))))
}

func (s *Socket) SendCommand(command string, args ...string) error {
	return s.rawSendCommand(fmt.Sprintf("%d", atomic.AddUint64(&s.lastCommandId, 1)), command, args...)
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
		return errors.New("cannot re-define interface: Already set")
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
		log.Printf("[%s] MAC collision: Killing", s.connId)
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
	if s.iface != nil && !multiClientIface {
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

	allSocketsLock.Lock()
	delete(allSockets, s)
	allSocketsLock.Unlock()
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
	s.registerDefaultCommandHandlers()

	s.writeLock.Lock()
	defer s.writeLock.Unlock()
	s.tryServeIfaceRead()

	allSocketsLock.Lock()
	allSockets[s] = s
	allSocketsLock.Unlock()

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
				if multiClientIface && len(msg) >= 14 {
					s.setMACFrom(msg)

					if allowClientToClient {
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
					err = errors.New("unknown command")
				} else {
					err = handler(str[2:])
				}

				replyStr := "OK"
				if err != nil {
					replyStr = err.Error()
					log.Printf("[%s] Error in in-band command %s: %v", s.connId, commandName, err)
				}
				s.rawSendCommand(commandId, "reply", replyStr)
				if err != nil {
					return
				}
			}
		}
	}()

	s.installPingHandlers()

	s.sendDefaultWelcome()
}

func (s *Socket) installPingHandlers() {
	pingInterval := *pingIntervalPtr
	pingTimeout := *pingTimeoutPtr

	if pingInterval <= 0 || pingTimeout <= 0 {
		log.Printf("[%s] Ping disabled", s.connId)
		return
	}

	// Create a dummy timer that won't ever run so we can wait for it
	pingTimeoutTimer := time.NewTimer(time.Hour)
	pingTimeoutTimer.Stop()

	s.conn.SetPongHandler(func(msg string) error {
		pingTimeoutTimer.Stop()
		return nil
	})

	s.wg.Add(1)
	go func() {
		defer s.closeDone()
		defer pingTimeoutTimer.Stop()

		for {
			select {
			case <-time.After(pingInterval):
				pingTimeoutTimer.Stop()
				err := s.WriteMessage(websocket.PingMessage, []byte{})
				if err != nil {
					return
				}
				pingTimeoutTimer.Reset(pingTimeout)
			case <-pingTimeoutTimer.C:
				log.Printf("[%s] Ping timeout", s.connId)
				return
			case <-s.closechan:
				return
			}
		}
	}()

	log.Printf("[%s] Ping enabled with interval %v and timeout %v", s.connId, pingInterval, pingTimeout)
}
