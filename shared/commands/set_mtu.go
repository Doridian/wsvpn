package commands

const SetMtuCommandName CommandName = "set_mtu"

type SetMtuParameters struct {
	MTU int `json:"mtu"`
}

func (c *SetMtuParameters) MakeCommand(id string) (*OutgoingCommand, error) {
	return makeCommand(SetMtuCommandName, id, c)
}

func (c *SetMtuParameters) MinProtocolVersion() int {
	return 7
}

func (c *SetMtuParameters) ServerCanIssue() bool {
	return true
}

func (c *SetMtuParameters) ClientCanIssue() bool {
	return false
}
