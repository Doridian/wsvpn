package sockets

import (
	"errors"
	"fmt"

	"github.com/Doridian/wsvpn/shared"
	"github.com/Doridian/wsvpn/shared/commands"
)

type CommandHandler func(command *commands.IncomingCommand) error

func (s *Socket) AddCommandHandler(command string, handler CommandHandler) {
	s.handlers[command] = handler
}

func (s *Socket) registerControlMessageHandler() {
	s.adapter.SetControlMessageHandler(func(message []byte) bool {
		var err error
		var command *commands.IncomingCommand

		command, err = commands.DeserializeCommand(message)
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
		s.log.Printf("Remote version is: %s (protocol %d)", parameters.Version, parameters.ProtocolVersion)
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
		s.CloseError(fmt.Sprintf("Error preparing command: %v", err))
		return err
	}

	cmdBytes, err := cmd.Serialize()
	if err != nil {
		s.CloseError(fmt.Sprintf("Error serializing command: %v", err))
		return err
	}

	err = s.adapter.WriteControlMessage(cmdBytes)
	if err != nil {
		s.CloseError(fmt.Sprintf("Error sending command: %v", err))
		return err
	}

	return nil
}
