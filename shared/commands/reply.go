package commands

const ReplyCommandName CommandName = "reply"

type ReplyParameters struct {
	Ok      bool   `json:"ok"`
	Message string `json:"message"`
}

func (c *ReplyParameters) MakeCommand(id string) (*OutgoingCommand, error) {
	return makeCommand(ReplyCommandName, id, c)
}

func (c *ReplyParameters) MinProtocolVersion() int {
	return 0
}

func (c *ReplyParameters) ServerCanIssue() bool {
	return true
}

func (c *ReplyParameters) ClientCanIssue() bool {
	return true
}
