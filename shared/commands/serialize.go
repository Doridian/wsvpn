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
	SerializationTypeJson
	SerializationTypeProtobuf
)

var serializationTypeMap map[string]SerializationType
var serializationTypeReverseMap map[SerializationType]string
var serializationInit = &sync.Once{}

func initSerializationTypeMaps() {
	serializationTypeMap = make(map[string]int)
	serializationTypeReverseMap = make(map[int]string)

	serializationTypeMap["json"] = SerializationTypeJson
	serializationTypeMap["protobuf"] = SerializationTypeProtobuf

	for name, stype := range serializationTypeMap {
		serializationTypeReverseMap[stype] = name
	}
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

func SerializationTypeFromString(name string) SerializationType {
	initSerializationTypeMapsOnce()

	stype, ok := serializationTypeMap[strings.ToLower(name)]
	if !ok {
		return SerializationTypeInvalid
	}
	return stype
}

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
