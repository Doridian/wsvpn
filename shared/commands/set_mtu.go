package commands

const SetMTUCommandName CommandName = "set_mtu"

type SetMTUParameters struct {
	MTU int `json:"mtu"`
}

func (c *SetMTUParameters) MakeCommand(id string) (*OutgoingCommand, error) {
	return makeCommand(SetMTUCommandName, id, c)
}

func (c *SetMTUParameters) MinProtocolVersion() int {
	return 7
}

func (c *SetMTUParameters) ServerCanIssue() bool {
	return true
}

func (c *SetMTUParameters) ClientCanIssue() bool {
	return false
}
