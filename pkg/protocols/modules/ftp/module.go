// Package ftp provides the FTP and FTPS protocol implementation for the gateway
// to make transfers, both as client and server.
package ftp

import (
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/features"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protocol"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protoutils"
)

const (
	FTP  = "ftp"
	FTPS = "ftps"
)

type Module struct{}

func (Module) CanMakeTransfer(*model.TransferContext) error { return nil }

func (Module) NewServer(db *database.DB, server *model.LocalAgent) protocol.Server {
	return newServer(db, server)
}

func (Module) NewClient(db *database.DB, cli *model.Client) protocol.Client {
	return newClient(db, cli)
}

func (Module) CheckServerConfig(conf map[string]any) error {
	return protoutils.ValidateProtoConfig(conf, &ServerConfig{})
}

func (Module) CheckClientConfig(conf map[string]any) error {
	return protoutils.ValidateProtoConfig(conf, &ClientConfig{})
}

func (Module) CheckPartnerConfig(conf map[string]any) error {
	return protoutils.ValidateProtoConfig(conf, &PartnerConfig{})
}

func (m Module) OptionalFeatures() []features.Feature {
	return []features.Feature{
		features.Listing,
		features.Deletion,
		features.RecursiveDeletion,
	}
}

type ModuleFTPS struct{ Module }

func (ModuleFTPS) CheckServerConfig(conf map[string]any) error {
	return protoutils.ValidateProtoConfig(conf, &ServerConfigTLS{})
}

func (ModuleFTPS) CheckClientConfig(conf map[string]any) error {
	return protoutils.ValidateProtoConfig(conf, &ClientConfigTLS{})
}

func (ModuleFTPS) CheckPartnerConfig(conf map[string]any) error {
	return protoutils.ValidateProtoConfig(conf, &PartnerConfigTLS{})
}
