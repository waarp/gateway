// Package ftp provides the FTP and FTPS protocol implementation for the gateway
// to make transfers, both as client and server.
package ftp

import (
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protocol"
)

const (
	FTP  = "ftp"
	FTPS = "ftps"
)

type Module struct{}

func (Module) NewServer(db *database.DB, serv *model.LocalAgent) protocol.Server {
	return newServer(db, serv)
}

func (Module) NewClient(_ *database.DB, cli *model.Client) protocol.Client {
	return newClient(cli)
}

func (Module) MakeServerConfig() protocol.ServerConfig   { return new(ServerConfig) }
func (Module) MakeClientConfig() protocol.ClientConfig   { return new(ClientConfig) }
func (Module) MakePartnerConfig() protocol.PartnerConfig { return new(PartnerConfig) }

type ModuleFTPS struct{ Module }

func (ModuleFTPS) MakeServerConfig() protocol.ServerConfig   { return new(ServerConfigTLS) }
func (ModuleFTPS) MakeClientConfig() protocol.ClientConfig   { return new(ClientConfigTLS) }
func (ModuleFTPS) MakePartnerConfig() protocol.PartnerConfig { return new(PartnerConfigTLS) }
