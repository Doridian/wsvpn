package commands

const AddRouteCommandName CommandName = "add_route"

type AddRouteParameters struct {
	Route IpAddressWithCIDR `json:"route"`
}

func (c *AddRouteParameters) MakeCommand(id string) (*OutgoingCommand, error) {
	return makeCommand(AddRouteCommandName, id, c)
}
