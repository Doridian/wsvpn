package commands

import (
	"encoding/json"

	"github.com/google/uuid"
)

type InterfaceMode = string
type IpAddressWithCIDR = string

type CommandID = string
type CommandName = string

type baseCommand[T any] struct {
	ID         CommandID   `json:"id"`
	Command    CommandName `json:"command"`
	Parameters T           `json:"parameters"`
}

type CommandParameters interface {
	MakeCommand(id string) (*OutgoingCommand, error)
}

type IncomingCommand = baseCommand[json.RawMessage]
type OutgoingCommand = baseCommand[CommandParameters]

func makeCommand(name string, id string, parameters CommandParameters) (*OutgoingCommand, error) {
	cmd := &OutgoingCommand{Command: name, ID: id, Parameters: parameters}
	if cmd.ID == "" {
		uuid, err := uuid.NewRandom()
		if err != nil {
			return cmd, err
		}
		cmd.ID = uuid.String()
	}
	return cmd, nil
}
