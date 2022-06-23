package commands

import "encoding/json"

func (c *baseCommand[T]) Serialize() ([]byte, error) {
	return json.Marshal(c)
}
