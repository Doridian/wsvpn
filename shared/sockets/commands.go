package sockets

import (
	"encoding/json"
	"log"

	"github.com/Doridian/wsvpn/shared"
	"github.com/Doridian/wsvpn/shared/commands"
)

type CommandHandler func(command *commands.IncomingCommand) error

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
