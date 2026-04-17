package http

import (
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protocol"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protoutils"
)

const (
	HTTP  = "http"
	HTTPS = "https"
)

type Module struct{}

func (Module) CanMakeTransfer(*model.TransferContext) error { return nil }

func (Module) NewServer(db *database.DB, server *model.LocalAgent) protocol.Server {
	return &httpService{db: db, agent: server}
}

func (Module) NewClient(db *database.DB, cli *model.Client) protocol.Client {
	return &httpClient{db: db, client: cli}
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

type ModuleHTTPS struct{ Module }

func (ModuleHTTPS) CheckServerConfig(conf map[string]any) error {
	return protoutils.ValidateProtoConfig(conf, &httpsServerConfig{})
}

func (ModuleHTTPS) CheckClientConfig(conf map[string]any) error {
	return protoutils.ValidateProtoConfig(conf, &httpsClientConfig{})
}

func (ModuleHTTPS) CheckPartnerConfig(conf map[string]any) error {
	return protoutils.ValidateProtoConfig(conf, &httpsPartnerConfig{})
}
