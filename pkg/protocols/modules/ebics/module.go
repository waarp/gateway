package ebics

import (
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protocol"
)

type Module struct{}

func (Module) NewServer(db *database.DB, server *model.LocalAgent) protocol.Server {
	return NewServer(db, server)
}

func (Module) NewClient(db *database.DB, client *model.Client) protocol.Client {
	return NewClient(db, client)
}

func (Module) MakeServerConfig() protocol.ServerConfig {
	return defaultServerConfig()
}

func (Module) MakeClientConfig() protocol.ClientConfig {
	return defaultClientConfig()
}

func (Module) MakePartnerConfig() protocol.PartnerConfig {
	return defaultPartnerConfig()
}
