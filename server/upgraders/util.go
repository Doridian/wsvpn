package upgraders

import (
	"net/http"
	"strings"

	"github.com/Doridian/wsvpn/shared"
	"github.com/Doridian/wsvpn/shared/commands"
)

func determineBestSerialization(header http.Header) commands.SerializationType {
	res := header.Get(shared.SupportedCommandSerializationsHeaderName)
	if res == "" {
		return commands.SerializationTypeJson
	}

	bestSerializationType := commands.SerializationTypeInvalid
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
