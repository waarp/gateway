package http

import (
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protocol"
)

const (
	HTTP  = "http"
	HTTPS = "https"
)

type Module struct{}

func (Module) NewServer(db *database.DB, serv *model.LocalAgent) protocol.Server {
	return &httpService{db: db, agent: serv}
}

func (Module) NewClient(db *database.DB, cli *model.Client) protocol.Client {
	return &httpClient{db: db, client: cli}
}

func (Module) MakeServerConfig() protocol.ServerConfig   { return new(serverConfig) }
func (Module) MakeClientConfig() protocol.ClientConfig   { return new(clientConfig) }
func (Module) MakePartnerConfig() protocol.PartnerConfig { return new(partnerConfig) }

type ModuleHTTPS struct{ Module }

func (ModuleHTTPS) NewClient(db *database.DB, cli *model.Client) protocol.Client {
	return &httpsClient{httpClient: &httpClient{db: db, client: cli}}
}

func (ModuleHTTPS) MakeServerConfig() protocol.ServerConfig   { return new(httpsServerConfig) }
func (ModuleHTTPS) MakeClientConfig() protocol.ClientConfig   { return new(httpsClientConfig) }
func (ModuleHTTPS) MakePartnerConfig() protocol.PartnerConfig { return new(httpsPartnerConfig) }
