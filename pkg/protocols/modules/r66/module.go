package r66

import (
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protocol"
)

const (
	R66    = "r66"
	R66TLS = "r66-tls"
)

type Module struct{}

func (Module) NewServer(db *database.DB, server *model.LocalAgent) protocol.Server {
	return &service{db: db, agent: server}
}

func (Module) NewClient(db *database.DB, cli *model.Client) protocol.Client {
	return &Client{db: db, cli: cli}
}

func (Module) MakeServerConfig() protocol.ServerConfig   { return new(serverConfig) }
func (Module) MakeClientConfig() protocol.ClientConfig   { return new(clientConfig) }
func (Module) MakePartnerConfig() protocol.PartnerConfig { return new(partnerConfig) }

type ModuleTLS struct{ Module }

func (ModuleTLS) MakeServerConfig() protocol.ServerConfig   { return new(tlsServerConfig) }
func (ModuleTLS) MakeClientConfig() protocol.ClientConfig   { return new(tlsClientConfig) }
func (ModuleTLS) MakePartnerConfig() protocol.PartnerConfig { return new(tlsPartnerConfig) }
