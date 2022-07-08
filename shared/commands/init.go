package commands

const InitCommandName CommandName = "init"

type InitParameters struct {
	Mode       InterfaceMode     `json:"mode"`
	DoIpConfig bool              `json:"do_ip_config"`
	IpAddress  IpAddressWithCIDR `json:"ip_address"`
	MTU        int               `json:"mtu"`
	ServerID   string            `json:"server_id"`
	ClientID   string            `json:"client_id"`
}

func (c *InitParameters) MakeCommand(id string) (*OutgoingCommand, error) {
	return makeCommand(InitCommandName, id, c)
}

func (c *InitParameters) MinProtocolVersion() int {
	return 1
}

func (c *InitParameters) ServerCanIssue() bool {
	return true
}

func (c *InitParameters) ClientCanIssue() bool {
	return false
}
