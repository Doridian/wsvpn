package connectors

import (
	"net/http"
	"strings"

	"github.com/Doridian/wsvpn/shared/commands"
)

func addSupportedSerializationHeader(header http.Header) {
	header.Del(commands.SupportedCommandSerializationsHeaderName)
	header.Add(commands.SupportedCommandSerializationsHeaderName, strings.Join(commands.GetSupportedSerializationTypeNames(), ", "))
}

func readSerializationType(header http.Header) commands.SerializationType {
	res := header.Get(commands.CommandSerializationHeaderName)
	if res == "" {
		return commands.SerializationTypeJson
	}

	stype := commands.SerializationTypeFromString(res)
	if stype == commands.SerializationTypeInvalid {
		return commands.SerializationTypeJson
	}

	return stype
}
