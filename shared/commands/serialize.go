package commands

import "encoding/json"

func (c *OutgoingCommand) Serialize() ([]byte, error) {
	return json.Marshal(c)
}

func (c *IncomingCommand) DeserializeParameters(parameters CommandParameters) error {
	return json.Unmarshal(c.Parameters, parameters)
}

func DeserializeCommand(message []byte) (*IncomingCommand, error) {
	var command IncomingCommand
	err := json.Unmarshal(message, &command)
	if err != nil {
		return nil, err
	}
	return &command, nil
}
