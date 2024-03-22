// Package constructors contains a list of all the
package constructors

import (
	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/service/proto"
	"code.waarp.fr/apps/gateway/gateway/pkg/http"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/config"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/r66"
	"code.waarp.fr/apps/gateway/gateway/pkg/sftp"
)

type (
	ServiceConstructor func(*database.DB, *log.Logger) proto.Service
	ClientConstructor  func(*model.Client) (pipeline.Client, error)
)

//nolint:gochecknoglobals // global var is used by design
var (
	// ServiceConstructors is a map associating each protocol with a constructor for
	// a client of said protocol. In order for the gateway to be able to perform
	// client transfer with a protocol, a constructor must be added to this map, to
	// allow a client to be instantiated.
	ServiceConstructors = map[string]ServiceConstructor{}

	// ClientConstructors is a map containing constructors for the various clients
	// supported by the gateway. It associates each protocol with the constructor for
	// its client. In order for the gateway to be able to execute a transfer in a
	// given protocol as a client, the constructor for the protocol's client must
	// be added to this map.
	ClientConstructors = map[string]ClientConstructor{}
)

//nolint:gochecknoinits // init is used by design
func init() {
	ClientConstructors["sftp"] = sftp.NewClient
	ClientConstructors[config.ProtocolR66] = r66.NewClient
	ClientConstructors["http"] = http.NewHTTPClient
	ClientConstructors["https"] = http.NewHTTPSClient

	ServiceConstructors["sftp"] = sftp.NewService
	ServiceConstructors[config.ProtocolR66] = r66.NewService
	ServiceConstructors[config.ProtocolR66TLS] = r66.NewService
	ServiceConstructors["http"] = http.NewService
	ServiceConstructors["https"] = http.NewService
}
