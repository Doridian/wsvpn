package commands

import "github.com/Doridian/wsvpn/shared/features"

const VersionCommandName CommandName = "version"

type VersionParameters struct {
	ProtocolVersion int                `json:"protocol_version"`
	Version         string             `json:"version"`
	EnabledFeatures []features.Feature `json:"enabled_features"`
}

func (c *VersionParameters) MakeCommand(id string) (*OutgoingCommand, error) {
	return makeCommand(VersionCommandName, id, c)
}

func (c *VersionParameters) MinProtocolVersion() int {
	return 0
}

func (c *VersionParameters) ServerCanIssue() bool {
	return true
}

func (c *VersionParameters) ClientCanIssue() bool {
	return true
}
