package upgraders

import (
	"net/http"
	"strings"

	"github.com/Doridian/wsvpn/shared/commands"
)

func handleHTTPSerializationHeaders(w http.ResponseWriter, r *http.Request) commands.SerializationType {
	serializationType := determineBestSerialization(r.Header)
	addSerializationHeader(w.Header(), serializationType)
	return serializationType
}

func determineBestSerialization(header http.Header) commands.SerializationType {
	res := header.Get(commands.SupportedCommandSerializationsHeaderName)
	if res == "" {
		return commands.SerializationTypeJSON
	}

	bestSerializationType := commands.SerializationTypeJSON
	bestSerializationTypePriority := -1

	serializations := strings.Split(res, ",")
	for _, serialization := range serializations {
		serialization = strings.Trim(serialization, " ")
		stype := commands.SerializationTypeFromString(serialization)
		if stype == commands.SerializationTypeInvalid {
			continue
		}

		priority := commands.SerializationTypePriority(stype)
		if priority > bestSerializationTypePriority {
			bestSerializationTypePriority = priority
			bestSerializationType = stype
		}
	}

	return bestSerializationType
}

func addSerializationHeader(headers http.Header, serializationType commands.SerializationType) {
	headers.Del(commands.CommandSerializationHeaderName)
	headers.Add(commands.CommandSerializationHeaderName, commands.SerializationTypeToString(serializationType))
}
