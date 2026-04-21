package amqp091

import (
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protocol"
)

const AMQP091 = "amqp091"

type Module struct{}

func (Module) NewServer(db *database.DB, server *model.LocalAgent) protocol.Server {
	return newServer(db, server)
}

func (Module) NewClient(_ *database.DB, client *model.Client) protocol.Client {
	return newClient(client)
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
