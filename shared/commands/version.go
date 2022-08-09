package commands

const VersionCommandName CommandName = "version"

type Feature = string

const (
	FEATURE_FRAGMENTATION Feature = "fragmentation"
	FEATURE_COMPRESSION   Feature = "compression"
)

type FeaturesConfig = []Feature

func IsFeatureSupported(feat Feature) bool {
	return feat == FEATURE_FRAGMENTATION
}

type VersionParameters struct {
	ProtocolVersion int            `json:"protocol_version"`
	Version         string         `json:"version"`
	EnabledFeatures FeaturesConfig `json:"enabled_features"`
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
