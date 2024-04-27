package features

type Feature = string

const (
	Fragmentation Feature = "fragmentation"
	Compression   Feature = "compression"
)

type Config = map[Feature]bool

func IsFeatureSupported(feat Feature) bool {
	return feat == Fragmentation
}

type Container interface {
	IsFeatureEnabled(feat Feature) bool
}
