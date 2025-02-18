// Package pesit implements a connector for the Pesit protocol, allowing the
// gateway to perform transfers using that protocol. The module implements both
// a client and a server.
package pesit

import (
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protocol"
)

const (
	Pesit    = "pesit"
	PesitTLS = "pesit-tls"
)

type Module struct{}

func (Module) NewServer(db *database.DB, serv *model.LocalAgent) protocol.Server {
	return newService(db, serv)
}

func (Module) NewClient(_ *database.DB, client *model.Client) protocol.Client {
	return newClient(client)
}

func (Module) MakeServerConfig() protocol.ServerConfig   { return new(ServerConfig) }
func (Module) MakeClientConfig() protocol.ClientConfig   { return new(ClientConfig) }
func (Module) MakePartnerConfig() protocol.PartnerConfig { return new(PartnerConfig) }

type ModuleTLS struct{ Module }

func (ModuleTLS) MakeServerConfig() protocol.ServerConfig   { return new(ServerConfigTLS) }
func (ModuleTLS) MakeClientConfig() protocol.ClientConfig   { return new(ClientConfigTLS) }
func (ModuleTLS) MakePartnerConfig() protocol.PartnerConfig { return new(PartnerConfigTLS) }
