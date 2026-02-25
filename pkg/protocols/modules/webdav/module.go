package webdav

import (
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protocol"
)

const (
	Webdav    = "webdav"
	WebdavTLS = "webdav-tls"
)

type Module struct{}

func (Module) NewServer(db *database.DB, serv *model.LocalAgent) protocol.Server {
	return NewServer(db, serv)
}

func (Module) NewClient(db *database.DB, cli *model.Client) protocol.Client {
	return NewClient(db, cli)
}

func (Module) MakeServerConfig() protocol.ServerConfig   { return new(ServerConfig) }
func (Module) MakeClientConfig() protocol.ClientConfig   { return new(ClientConfig) }
func (Module) MakePartnerConfig() protocol.PartnerConfig { return new(PartnerConfig) }

type ModuleTLS struct{ Module }

func (ModuleTLS) MakeServerConfig() protocol.ServerConfig   { return new(ServerConfigTLS) }
func (ModuleTLS) MakeClientConfig() protocol.ClientConfig   { return new(ClientConfigTLS) }
func (ModuleTLS) MakePartnerConfig() protocol.PartnerConfig { return new(PartnerConfigTLS) }
