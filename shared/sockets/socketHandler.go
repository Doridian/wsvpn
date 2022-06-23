package sockets

import (
	"encoding/json"
	"errors"
	"flag"
	"log"
	"sync"
	"time"

	"github.com/Doridian/wsvpn/shared"
	"github.com/Doridian/wsvpn/shared/commands"
	"github.com/Doridian/wsvpn/shared/sockets/adapters"
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

type CommandHandler func(command *commands.IncomingCommand) error

type Socket struct {
	connId                string
	adapter               adapters.SocketAdapter
	iface                 *water.Interface
	noIfaceReader         bool
	wg                    *sync.WaitGroup
	handlers              map[string]CommandHandler
	closechan             chan bool
	closechanopen         bool
	mac                   shared.MacAddr
	remoteProtocolVersion int
	packetBufferSize      int
}

func SetMultiClientIfaceMode(enable bool) {
	multiClientIface = enable
}

func SetClientToClient(enable bool) {
	allowClientToClient = enable
}

func MakeSocket(connId string, adapter adapters.SocketAdapter, iface *water.Interface, noIfaceReader bool) *Socket {
	return &Socket{
		connId:                connId,
		adapter:               adapter,
		iface:                 iface,
		noIfaceReader:         noIfaceReader,
		wg:                    &sync.WaitGroup{},
		handlers:              make(map[commands.CommandName]CommandHandler),
		closechan:             make(chan bool),
		closechanopen:         true,
		mac:                   defaultMac,
		remoteProtocolVersion: 0,
		packetBufferSize:      2000,
	}
}

func (s *Socket) Wait() {
	s.wg.Wait()
}

func (s *Socket) AddCommandHandler(command string, handler CommandHandler) {
	s.handlers[command] = handler
}

func (s *Socket) registerDefaultCommandHandlers() {
	s.AddCommandHandler(commands.VersionCommandName, func(command *commands.IncomingCommand) error {
		var parameters commands.VersionParameters
		err := json.Unmarshal(command.Parameters, &parameters)
		if err != nil {
			return err
		}
		log.Printf("[%s] Remote version is: %s (protocol %d)", s.connId, parameters.Version, parameters.ProtocolVersion)
		return nil
	})

	s.AddCommandHandler(commands.ReplyCommandName, func(command *commands.IncomingCommand) error {
		var parameters commands.ReplyParameters
		err := json.Unmarshal(command.Parameters, &parameters)
		if err != nil {
			return err
		}
		log.Printf("[%s] Got reply to command ID %s (%s): %s", s.connId, command.ID, shared.BoolToString(parameters.Ok, "ok", "error"), parameters.Message)
		return nil
	})
}

func (s *Socket) sendDefaultWelcome() {
	s.MakeAndSendCommand(&commands.VersionParameters{Version: shared.Version, ProtocolVersion: shared.ProtocolVersion})
}

func (s *Socket) MakeAndSendCommand(parameters commands.CommandParameters) error {
	return s.rawMakeAndSendCommand(parameters, "")
}

func (s *Socket) rawMakeAndSendCommand(parameters commands.CommandParameters, id string) error {
	cmd, err := parameters.MakeCommand(id)
	if err != nil {
		log.Printf("[%s] Error preparing command: %v", s.connId, err)
	}

	cmdBytes, err := cmd.Serialize()
	if err != nil {
		log.Printf("[%s] Error serializing command: %v", s.connId, err)
		s.Close()
	}

	err = s.adapter.WriteControlMessage(cmdBytes)
	if err != nil {
		log.Printf("[%s] Error sending command: %v", s.connId, err)
		s.Close()
	}

	return err
}

func (s *Socket) WriteDataMessage(data []byte) error {
	return s.adapter.WriteDataMessage(data)
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

		packet := make([]byte, s.packetBufferSize)

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

func (s *Socket) SetMTU(mtu int) {
	s.packetBufferSize = shared.GetPacketBufferSizeByMTU(mtu)
}

func (s *Socket) Serve() {
	s.registerDefaultCommandHandlers()

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
		var err error
		var command commands.IncomingCommand

		err = json.Unmarshal(message, &command)
		if err != nil {
			log.Printf("[%s] Error deserializing command: %v", s.connId, err)
			return false
		}

		handler := s.handlers[command.Command]
		if handler == nil {
			err = errors.New("unknown command")
		} else {
			err = handler(&command)
		}

		replyOk := true
		replyStr := "OK"
		if err != nil {
			replyOk = false
			replyStr = err.Error()
			log.Printf("[%s] Error in in-band command %s: %v", s.connId, command.Command, err)
		}

		if command.Command != commands.ReplyCommandName {
			s.rawMakeAndSendCommand(&commands.ReplyParameters{Message: replyStr, Ok: replyOk}, command.ID)
		}
		return replyOk
	})

	s.installPingPongHandlers(*pingIntervalPtr, *pingTimeoutPtr)

	s.wg.Add(1)
	go func() {
		defer s.closeDone()
		err, unexpected := s.adapter.Serve()
		if unexpected {
			log.Printf("[%s] Client ERROR: %v", s.connId, err)
		}
	}()

	s.adapter.WaitReady()

	s.tryServeIfaceRead()

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
					log.Printf("[%s] Error sending ping: %v", s.connId, err)
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
