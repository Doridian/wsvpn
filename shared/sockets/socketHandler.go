package sockets

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

	"github.com/Doridian/wsvpn/shared"
	"github.com/songgao/water"
)

var pingIntervalPtr = flag.Duration("ping-interval", time.Second*time.Duration(30), "Send ping frames in this interval")
var pingTimeoutPtr = flag.Duration("ping-timeout", time.Second*time.Duration(5), "Disconnect if no ping response is received after timeout")

var defaultMac = [6]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}

var multiClientIface bool = false
var allowClientToClient bool = false
var macTable map[shared.MacAddr]*Socket = make(map[shared.MacAddr]*Socket)
var macLock sync.RWMutex
var allSockets = make(map[*Socket]*Socket)
var allSocketsLock sync.RWMutex

func FindSocketByMAC(mac shared.MacAddr) *Socket {
	macLock.RLock()
	defer macLock.RUnlock()

	return macTable[mac]
}

func BroadcastDataMessage(data []byte, skip *Socket) {
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
		v.adapter.WriteDataMessage(data)
	}
}

type CommandHandler func(args []string) error

type Socket struct {
	lastCommandId         uint64 // This MUST be the first element of the struct, see https://github.com/golang/go/issues/23345
	connId                string
	adapter               SocketAdapter
	iface                 *water.Interface
	noIfaceReader         bool
	wg                    *sync.WaitGroup
	handlers              map[string]CommandHandler
	closechan             chan bool
	closechanopen         bool
	mac                   shared.MacAddr
	remoteProtocolVersion int
}

func SetMultiClientIfaceMode(enable bool) {
	multiClientIface = enable
}

func SetClientToClient(enable bool) {
	allowClientToClient = enable
}

func MakeSocket(connId string, adapter SocketAdapter, iface *water.Interface, noIfaceReader bool) *Socket {
	return &Socket{
		connId:                connId,
		adapter:               adapter,
		iface:                 iface,
		noIfaceReader:         noIfaceReader,
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
	s.SendCommand("version", fmt.Sprintf("%d", shared.ProtocolVersion), shared.Version)
}

func (s *Socket) Wait() {
	s.wg.Wait()
}

func (s *Socket) rawSendCommand(commandId string, command string, args ...string) error {
	err := s.adapter.WriteControlMessage([]byte(fmt.Sprintf("%s|%s|%s", commandId, command, strings.Join(args, "|"))))
	if err != nil {
		log.Printf("[%s] Error writing control message: %v", s.connId, err)
		s.Close()
	}
	return err
}

func (s *Socket) WriteDataMessage(data []byte) error {
	return s.adapter.WriteDataMessage(data)
}

func (s *Socket) SendCommand(command string, args ...string) error {
	return s.rawSendCommand(fmt.Sprintf("%d", atomic.AddUint64(&s.lastCommandId, 1)), command, args...)
}

func (s *Socket) closeDone() {
	s.wg.Done()
	s.Close()
}

func (s *Socket) SetInterface(iface *water.Interface) error {
	if s.iface != nil {
		return errors.New("cannot re-define interface: Already set")
	}
	s.iface = iface
	s.tryServeIfaceRead()
	return nil
}

func (s *Socket) setMACFrom(msg []byte) {
	srcMac := shared.GetSrcMAC(msg)
	if !shared.MACIsUnicast(srcMac) || srcMac == s.mac {
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
	s.adapter.Close()
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

			err = s.adapter.WriteDataMessage(packet[:n])
			if err != nil {
				return
			}
		}
	}()
}

func (s *Socket) Serve() {
	s.registerDefaultCommandHandlers()

	s.tryServeIfaceRead()

	allSocketsLock.Lock()
	allSockets[s] = s
	allSocketsLock.Unlock()

	s.adapter.SetDataMessageHandler(func(message []byte) bool {
		if multiClientIface && len(message) >= 14 {
			s.setMACFrom(message)

			if allowClientToClient {
				dest := shared.GetDestMAC(message)
				isUnicast := shared.MACIsUnicast(dest)

				var sd *Socket
				if isUnicast {
					sd = FindSocketByMAC(dest)
					if sd != nil {
						sd.adapter.WriteDataMessage(message)
						return true
					}
				} else {
					BroadcastDataMessage(message, s)
				}
			}
		}

		if s.iface == nil {
			return true
		}
		s.iface.Write(message)
		return true
	})

	s.adapter.SetControlMessageHandler(func(message []byte) bool {
		str := strings.Split(string(message), "|")
		if len(str) < 2 {
			log.Printf("[%s] Invalid in-band command structure", s.connId)
			return false
		}

		commandId := str[0]
		commandName := str[1]
		if commandName == "reply" {
			commandResult := "N/A"
			if len(str) > 2 {
				commandResult = str[2]
			}
			log.Printf("[%s] Got command reply ID %s: %s", s.connId, commandId, commandResult)
			return true
		}

		var err error

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
		return err == nil
	})

	s.installPingPongHandlers(*pingIntervalPtr, *pingTimeoutPtr)

	s.wg.Add(1)
	go func() {
		defer s.closeDone()
		s.adapter.Serve()
	}()

	go s.sendDefaultWelcome()
}

func (s *Socket) installPingPongHandlers(pingInterval time.Duration, pingTimeout time.Duration) {
	if pingInterval <= 0 || pingTimeout <= 0 {
		log.Printf("[%s] Ping disabled", s.connId)
		return
	}

	// Create a dummy timer that won't ever run so we can wait for it
	pingTimeoutTimer := time.NewTimer(time.Hour)
	pingTimeoutTimer.Stop()

	s.adapter.SetPongHandler(func() {
		pingTimeoutTimer.Stop()
	})

	s.wg.Add(1)

	go func() {
		defer s.closeDone()
		defer pingTimeoutTimer.Stop()

		for {
			select {
			case <-time.After(pingInterval):
				pingTimeoutTimer.Stop()
				err := s.adapter.WritePingMessage()
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
