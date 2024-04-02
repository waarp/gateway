// Package protocols defines the interface which must be implemented by a
// protocol module in order to be usable by the gateway. Once a new protocol
// module is implemented, it must be registered in the list of protocols
// provided by this package (using the Register function).
package protocols

import (
	"errors"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protocol"
)

var ErrUnknownProtocol = errors.New("unknown protocol")

// Module is the interface that must be implemented by modules that wish to
// implement a transfer protocol.
type Module interface {
	// NewServer should return a new instance of a server of the protocol with
	// the given name and database server ID.
	NewServer(db *database.DB, server *model.LocalAgent) protocol.Server
	// NewClient should return a new instance of a client of the protocol with
	// the given name and database client ID.
	NewClient(db *database.DB, client *model.Client) protocol.Client

	// MakeServerConfig returns a new instance of a server configuration of the
	// protocol.
	MakeServerConfig() protocol.ServerConfig
	// MakeClientConfig returns a new instance of a client configuration of the
	// protocol.
	MakeClientConfig() protocol.ClientConfig
	// MakePartnerConfig returns a new instance of a partner configuration of the
	// protocol.
	MakePartnerConfig() protocol.PartnerConfig
}
