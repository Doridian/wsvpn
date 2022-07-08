package commands

import (
	"encoding/json"

	"github.com/google/uuid"
)

type InterfaceMode = string
type IpAddressWithCIDR = string

type CommandID = string
type CommandName = string

type CommandParameters interface {
	MakeCommand(id string) (*OutgoingCommand, error)
	MinProtocolVersion() int
	ServerCanIssue() bool
	ClientCanIssue() bool
}

type IncomingCommand struct {
	ID         CommandID       `json:"id"`
	Command    CommandName     `json:"command"`
	Parameters json.RawMessage `json:"parameters"`
}

type OutgoingCommand struct {
	ID         CommandID         `json:"id"`
	Command    CommandName       `json:"command"`
	Parameters CommandParameters `json:"parameters"`
}

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
