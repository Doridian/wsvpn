package adapters

import (
	"crypto/tls"
	"errors"
	"net"

	"github.com/Doridian/wsvpn/shared/commands"
	"github.com/Doridian/wsvpn/shared/features"
)

type MessageHandler = func(message []byte) bool

var ErrDataPayloadLimitReduced = errors.New("data payload limit reduced")

type SocketAdapter interface {
	Close() error

	// Boolean indicating whether the error was unexpected (true) or not (false)
	Serve() (bool, error)

	WaitReady()
	Name() string

	SetFeaturesContainer(ct features.Container)

	WriteControlMessage(message []byte) error
	SetControlMessageHandler(handler MessageHandler)

	WriteDataMessage(message []byte) error
	SetDataMessageHandler(handler MessageHandler)

	WritePingMessage() error
	SetPongHandler(handler func())

	GetTLSConnectionState() (tls.ConnectionState, bool)

	GetCommandSerializationType() commands.SerializationType

	RemoteAddr() net.Addr
	LocalAddr() net.Addr

	IsServer() bool
	IsClient() bool

	MaxDataPayloadLen() uint16

	RefreshFeatures()
}

type adapterBase struct {
	controlMessageHandler MessageHandler
	dataMessageHandler    MessageHandler
	pongHandler           func()
	featuresContainer     features.Container
}

func (s *adapterBase) SetControlMessageHandler(handler MessageHandler) {
	s.controlMessageHandler = handler
}

func (s *adapterBase) SetDataMessageHandler(handler MessageHandler) {
	s.dataMessageHandler = handler
}

func (s *adapterBase) SetPongHandler(handler func()) {
	s.pongHandler = handler
}

func (s *adapterBase) SetFeaturesContainer(featuresContainer features.Container) {
	s.featuresContainer = featuresContainer
}
