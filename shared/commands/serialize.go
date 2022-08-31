package commands

import (
	"encoding/json"
	"errors"
	"strings"
	"sync"
)

type SerializationType = int

const (
	SerializationTypeInvalid SerializationType = iota
	SerializationTypeJSON
)

var serializationTypeMap map[string]SerializationType
var serializationTypeReverseMap map[SerializationType]string
var serializationTypePriorityMap map[SerializationType]int
var serializationInit = &sync.Once{}

const SupportedCommandSerializationsHeaderName = "Supported-Command-Serializations"
const CommandSerializationHeaderName = "Command-Serialization"

func addSerializationType(stype SerializationType, name string, priority int) {
	serializationTypeMap[name] = stype
	serializationTypeReverseMap[stype] = name
	serializationTypePriorityMap[stype] = priority
}

func initSerializationTypeMaps() {
	serializationTypeMap = make(map[string]SerializationType)
	serializationTypeReverseMap = make(map[SerializationType]string)
	serializationTypePriorityMap = make(map[SerializationType]int)

	addSerializationType(SerializationTypeJSON, "json", 1)
}

func initSerializationTypeMapsOnce() {
	serializationInit.Do(initSerializationTypeMaps)
}

func SerializationTypeToString(stype SerializationType) string {
	initSerializationTypeMapsOnce()

	name, ok := serializationTypeReverseMap[stype]
	if !ok {
		return ""
	}
	return name
}

func SerializationTypePriority(stype SerializationType) int {
	initSerializationTypeMapsOnce()

	return serializationTypePriorityMap[stype]
}

func SerializationTypeFromString(name string) SerializationType {
	initSerializationTypeMapsOnce()

	stype, ok := serializationTypeMap[strings.ToLower(name)]
	if !ok {
		return SerializationTypeInvalid
	}
	return stype
}

func GetSupportedSerializationTypes() []SerializationType {
	res := make([]SerializationType, 0, len(serializationTypeMap))
	for _, stype := range serializationTypeMap {
		res = append(res, stype)
	}
	return res
}

func GetSupportedSerializationTypeNames() []string {
	res := make([]string, 0, len(serializationTypeMap))
	for name := range serializationTypeMap {
		res = append(res, name)
	}
	return res
}

func (c *OutgoingCommand) Serialize(serializationType SerializationType) ([]byte, error) {
	switch serializationType {
	case SerializationTypeJSON:
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
	case SerializationTypeJSON:
		err = json.Unmarshal(message, &command)
	default:
		err = errors.New("unknown serialization type")
	}
	if err != nil {
		return nil, err
	}
	return &command, nil
}
