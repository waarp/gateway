package gatewayd

import (
	"code.waarp.fr/apps/gateway/gateway/pkg/http"
	"code.waarp.fr/apps/gateway/gateway/pkg/r66"
	"code.waarp.fr/apps/gateway/gateway/pkg/sftp"
)

// ServiceConstructors is a map associating each protocol with a constructor for
// a client of said protocol. In order for the gateway to be able to perform
// client transfer with a protocol, a constructor must be added to this map, to
// allow a client to be instantiated.
//nolint:gochecknoglobals // global var is used by design
var ServiceConstructors = map[string]serviceConstructor{}

//nolint:gochecknoinits // init is used by design
func init() {
	ServiceConstructors["sftp"] = sftp.NewService
	ServiceConstructors["r66"] = r66.NewService
	ServiceConstructors["http"] = http.NewService
	ServiceConstructors["https"] = http.NewService
}
