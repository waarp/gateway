package sftp

import (
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protocol"
)

const SFTP = "sftp"

type Module struct{}

func (m Module) NewClient(db *database.DB, cli *model.Client) protocol.Client {
	return &client{db: db, client: cli}
}

func (Module) NewServer(db *database.DB, serv *model.LocalAgent) protocol.Server {
	return &service{db: db, server: serv}
}

func (m Module) MakeServerConfig() protocol.ServerConfig   { return new(serverConfig) }
func (m Module) MakeClientConfig() protocol.ClientConfig   { return new(clientConfig) }
func (m Module) MakePartnerConfig() protocol.PartnerConfig { return new(partnerConfig) }
