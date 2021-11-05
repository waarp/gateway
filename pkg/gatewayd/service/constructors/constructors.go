// Package constructors contains a list of all the
package constructors

import (
	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/service/proto"
	"code.waarp.fr/apps/gateway/gateway/pkg/http"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/config"
	"code.waarp.fr/apps/gateway/gateway/pkg/r66"
	"code.waarp.fr/apps/gateway/gateway/pkg/sftp"
)

// ServiceConstructors is a map associating each protocol with a constructor for
// a client of said protocol. In order for the gateway to be able to perform
// client transfer with a protocol, a constructor must be added to this map, to
// allow a client to be instantiated.
//
//nolint:gochecknoglobals // global var is used by design
var ServiceConstructors = map[string]serviceConstructor{}

type serviceConstructor func(db *database.DB, logger *log.Logger) proto.Service

//nolint:gochecknoinits // init is used by design
func init() {
	ServiceConstructors["sftp"] = sftp.NewService
	ServiceConstructors[config.ProtocolR66] = r66.NewService
	ServiceConstructors[config.ProtocolR66TLS] = r66.NewService
	ServiceConstructors["http"] = http.NewService
	ServiceConstructors["https"] = http.NewService
}
