package webdav

import (
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protocol"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protoutils"
)

const (
	Webdav    = "webdav"
	WebdavTLS = "webdav-tls"
)

type Module struct{}

func (Module) CanMakeTransfer(*model.TransferContext) error { return nil }

func (Module) CheckServerConfig(conf map[string]any) error {
	return protoutils.ValidateProtoConfig(conf, &ServerConfig{})
}

func (Module) CheckClientConfig(conf map[string]any) error {
	return protoutils.ValidateProtoConfig(conf, &ClientConfig{})
}

func (Module) CheckPartnerConfig(conf map[string]any) error {
	return protoutils.ValidateProtoConfig(conf, &PartnerConfig{})
}

func (Module) NewServer(db *database.DB, serv *model.LocalAgent) protocol.Server {
	return NewServer(db, serv)
}

func (Module) NewClient(db *database.DB, cli *model.Client) protocol.Client {
	return NewClient(db, cli)
}

type ModuleTLS struct{ Module }

func (ModuleTLS) CheckServerConfig(conf map[string]any) error {
	return protoutils.ValidateProtoConfig(conf, &ServerConfigTLS{})
}

func (ModuleTLS) CheckClientConfig(conf map[string]any) error {
	return protoutils.ValidateProtoConfig(conf, &ClientConfigTLS{})
}

func (ModuleTLS) CheckPartnerConfig(conf map[string]any) error {
	return protoutils.ValidateProtoConfig(conf, &PartnerConfigTLS{})
}
