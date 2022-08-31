package features

type Feature = string

const (
	Fragmentation Feature = "fragmentation"
	DatagramID0   Feature = "datagram_id_0"
	Compression   Feature = "compression"
)

type Config = map[Feature]bool

func IsFeatureSupported(feat Feature) bool {
	return feat == Fragmentation || feat == DatagramID0
}

type Container interface {
	IsFeatureEnabled(feat Feature) bool
}
