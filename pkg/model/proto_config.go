package model

//nolint:gochecknoglobals //unfortunately needed to avoid an import cycle with the protocols package
var ConfigChecker ConfigValidator

type ConfigValidator interface {
	IsValidProtocol(proto string) bool
	CheckServerConfig(proto string, conf map[string]any) error
	CheckClientConfig(proto string, conf map[string]any) error
	CheckPartnerConfig(proto string, conf map[string]any) error
}

const (
	protoR66    = "r66"
	protoR66TLS = "r66-tls"
)
