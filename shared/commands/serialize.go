package commands

import (
	"encoding/json"
	"errors"
)

type SerializationType = int

const (
	SerializationTypeJson SerializationType = iota
)

func (c *OutgoingCommand) Serialize(serializationType SerializationType) ([]byte, error) {
	switch serializationType {
	case SerializationTypeJson:
		return json.Marshal(c)
	}
	return []byte{}, errors.New("unknown serialization type")
}

func (c *IncomingCommand) DeserializeParameters(parameters CommandParameters) error {
	return json.Unmarshal(c.Parameters, parameters)
}

func DeserializeCommand(message []byte, serializationType SerializationType) (*IncomingCommand, error) {
	var command IncomingCommand
	var err error
	switch serializationType {
	case SerializationTypeJson:
		err = json.Unmarshal(message, &command)
	default:
		err = errors.New("unknown serialization type")
	}
	if err != nil {
		return nil, err
	}
	return &command, nil
}
