package commands

const AddRouteCommandName CommandName = "add_route"

type AddRouteParameters struct {
	Route IpAddressWithCIDR `json:"route"`
}

func (c *AddRouteParameters) MakeCommand(id string) (*OutgoingCommand, error) {
	return makeCommand(AddRouteCommandName, id, c)
}

func (c *AddRouteParameters) MinProtocolVersion() int {
	return 1
}

func (c *AddRouteParameters) ServerCanIssue() bool {
	return true
}

func (c *AddRouteParameters) ClientCanIssue() bool {
	return false
}
