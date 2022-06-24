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

type Socket struct {
	connId                string
	adapter               adapters.SocketAdapter
	iface                 *water.Interface
	ifaceManaged          bool
	wg                    *sync.WaitGroup
	handlers              map[string]CommandHandler
	closechan             chan bool
	closechanopen         bool
	mac                   shared.MacAddr
	remoteProtocolVersion int
	packetBufferSize      int
	packetHandler         PacketHandler
}

func MakeSocket(connId string, adapter adapters.SocketAdapter, iface *water.Interface, ifaceManaged bool, packetHandler PacketHandler) *Socket {
	return &Socket{
		connId:                connId,
		adapter:               adapter,
		iface:                 iface,
		ifaceManaged:          ifaceManaged,
		wg:                    &sync.WaitGroup{},
		handlers:              make(map[commands.CommandName]CommandHandler),
		closechan:             make(chan bool),
		closechanopen:         true,
		mac:                   shared.DefaultMac,
		remoteProtocolVersion: 0,
		packetBufferSize:      2000,
		packetHandler:         packetHandler,
	}
}

func (s *Socket) GetConnectionID() string {
	return s.connId
}

func (s *Socket) Wait() {
	s.wg.Wait()
}

func (s *Socket) closeDone() {
	s.wg.Done()
	s.Close()
}

func (s *Socket) Close() {
	s.adapter.Close()
	if s.iface != nil && s.ifaceManaged {
		s.iface.Close()
	}

	if s.closechanopen {
		s.closechanopen = false
		close(s.closechan)
	}

	if s.packetHandler != nil {
		s.packetHandler.UnregisterSocket(s)
	}
}

func (s *Socket) Serve() {
	s.registerDefaultCommandHandlers()

	if s.packetHandler != nil {
		s.packetHandler.RegisterSocket(s)
	}

	s.registerDataHandler()

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
