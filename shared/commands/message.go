package commands

const MessageCommandName CommandName = "message"

type MessageParameters struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

func (c *MessageParameters) MakeCommand(id string) (*OutgoingCommand, error) {
	return makeCommand(MessageCommandName, id, c)
}

func (c *MessageParameters) MinProtocolVersion() int {
	return 8
}

func (c *MessageParameters) ServerCanIssue() bool {
	return true
}

func (c *MessageParameters) ClientCanIssue() bool {
	return true
}
