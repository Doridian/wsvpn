package commands

const VersionCommandName CommandName = "version"

type VersionParameters struct {
	ProtocolVersion int    `json:"protocol_version"`
	Version         string `json:"version"`
}

func (c *VersionParameters) MakeCommand(id string) (*OutgoingCommand, error) {
	return makeCommand(VersionCommandName, id, c)
}
