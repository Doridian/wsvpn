package features

type Feature = string

const (
	FEATURE_FRAGMENTATION Feature = "fragmentation"
	FEATURE_DATAGRAM_ID_0 Feature = "datagram_id_0"
	FEATURE_COMPRESSION   Feature = "compression"
)

type FeaturesConfig = map[Feature]bool

func IsFeatureSupported(feat Feature) bool {
	return feat == FEATURE_FRAGMENTATION || feat == FEATURE_DATAGRAM_ID_0
}

type FeaturesContainer interface {
	IsFeatureEnabled(feat Feature) bool
}
