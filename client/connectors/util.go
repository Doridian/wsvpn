package connectors

import (
	"net/http"
	"strings"

	"github.com/Doridian/wsvpn/shared"
	"github.com/Doridian/wsvpn/shared/commands"
)

func addSupportedSerializationHeader(header http.Header) {
	header.Del(shared.SupportedCommandSerializationsHeaderName)
	header.Add(shared.SupportedCommandSerializationsHeaderName, strings.Join(commands.GetSupportedSerializationTypeNames(), ", "))
}

func readSerializationType(header http.Header) commands.SerializationType {
	res := header.Get(shared.CommandSerializationHeaderName)
	if res == "" {
		return commands.SerializationTypeJson
	}
	return commands.SerializationTypeFromString(res)
}
