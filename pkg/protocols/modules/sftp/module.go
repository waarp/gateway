package sftp

import (
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protocol"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protoutils"
)

const SFTP = "sftp"

type Module struct{}

func (Module) CanMakeTransfer(*model.TransferContext) error { return nil }

func (Module) NewServer(db *database.DB, server *model.LocalAgent) protocol.Server {
	return &service{db: db, server: server}
}

func (Module) NewClient(db *database.DB, cli *model.Client) protocol.Client {
	return &client{db: db, client: cli}
}

func (Module) CheckServerConfig(conf map[string]any) error {
	return protoutils.ValidateProtoConfig(conf, &serverConfig{})
}

func (Module) CheckClientConfig(conf map[string]any) error {
	return protoutils.ValidateProtoConfig(conf, &clientConfig{})
}

func (Module) CheckPartnerConfig(conf map[string]any) error {
	return protoutils.ValidateProtoConfig(conf, &partnerConfig{})
}
