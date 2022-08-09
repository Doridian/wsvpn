package sockets

import (
	"errors"
	"fmt"

	"github.com/Doridian/wsvpn/shared"
	"github.com/Doridian/wsvpn/shared/commands"
)

var ErrCommandNotSupported = errors.New("command not supported by peer")

type CommandHandler func(command *commands.IncomingCommand) error

func (s *Socket) AddCommandHandler(command string, handler CommandHandler) {
	s.handlers[command] = handler
}

func (s *Socket) registerControlMessageHandler() {
	s.adapter.SetControlMessageHandler(func(message []byte) bool {
		var err error
		var command *commands.IncomingCommand

		command, err = commands.DeserializeCommand(message, s.adapter.GetCommandSerializationType())
		if err != nil {
			s.log.Printf("Error deserializing command: %v", err)
			return false
		}

		handler := s.handlers[command.Command]
		if handler == nil {
			err = errors.New("unknown command")
		} else {
			err = handler(command)
		}

		replyOk := true
		replyStr := "OK"
		if err != nil {
			replyOk = false
			replyStr = err.Error()
			s.log.Printf("Error in in-band command %s: %v", command.Command, err)
		}

		if command.Command != commands.ReplyCommandName {
			s.rawMakeAndSendCommand(&commands.ReplyParameters{Message: replyStr, Ok: replyOk}, command.ID)
		}
		return replyOk
	})
}

func (s *Socket) registerDefaultCommandHandlers() {
	s.AddCommandHandler(commands.VersionCommandName, func(command *commands.IncomingCommand) error {
		var parameters commands.VersionParameters
		err := command.DeserializeParameters(&parameters)
		if err != nil {
			return err
		}

		s.remoteProtocolVersion = parameters.ProtocolVersion
		s.log.Printf("Remote version is: %s (protocol %d)", parameters.Version, parameters.ProtocolVersion)

		s.remoteFeatures = make(map[commands.Feature]bool)
		for _, v := range parameters.EnabledFeatures {
			s.remoteFeatures[v] = true
		}

		s.featureCheck()

		s.setReady()

		return nil
	})

	s.AddCommandHandler(commands.ReplyCommandName, func(command *commands.IncomingCommand) error {
		var parameters commands.ReplyParameters
		err := command.DeserializeParameters(&parameters)
		if err != nil {
			return err
		}
		s.log.Printf("Got reply to command ID %s (%s): %s", command.ID, shared.BoolToString(parameters.Ok, "ok", "error"), parameters.Message)
		return nil
	})

	s.AddCommandHandler(commands.MessageCommandName, func(command *commands.IncomingCommand) error {
		var parameters commands.MessageParameters
		err := command.DeserializeParameters(&parameters)
		if err != nil {
			return err
		}
		s.log.Printf("Got %s message from remote: %s", parameters.Type, parameters.Message)
		return nil
	})
}

func (s *Socket) sendDefaultWelcome() error {
	localFeaturesArray := make([]commands.Feature, len(s.localFeatures))

	for feat, en := range s.localFeatures {
		if !en {
			continue
		}
		localFeaturesArray = append(localFeaturesArray, feat)
	}

	return s.MakeAndSendCommand(&commands.VersionParameters{
		Version:         shared.Version,
		ProtocolVersion: shared.ProtocolVersion,
		EnabledFeatures: localFeaturesArray,
	})
}

func (s *Socket) SendMessage(msgType string, message string) error {
	return s.MakeAndSendCommand(&commands.MessageParameters{Type: msgType, Message: message})
}

func (s *Socket) MakeAndSendCommand(parameters commands.CommandParameters) error {
	return s.rawMakeAndSendCommand(parameters, "")
}

func (s *Socket) rawMakeAndSendCommand(parameters commands.CommandParameters, id string) error {
	if s.adapter.IsServer() {
		if !parameters.ServerCanIssue() {
			return ErrCommandNotSupported
		}
	} else {
		if !parameters.ClientCanIssue() {
			return ErrCommandNotSupported
		}
	}

	if s.remoteProtocolVersion < parameters.MinProtocolVersion() {
		return ErrCommandNotSupported
	}

	cmd, err := parameters.MakeCommand(id)
	if err != nil {
		s.CloseError(fmt.Errorf("error preparing command: %v", err))
		return err
	}

	cmdBytes, err := cmd.Serialize(s.adapter.GetCommandSerializationType())
	if err != nil {
		s.CloseError(fmt.Errorf("error serializing command: %v", err))
		return err
	}

	err = s.adapter.WriteControlMessage(cmdBytes)
	if err != nil {
		s.CloseError(fmt.Errorf("error sending command: %v", err))
		return err
	}

	return nil
}
