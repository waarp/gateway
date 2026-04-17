package r66

import (
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/r66/r66auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protocol"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protoutils"
)

const (
	R66    = "r66"
	R66TLS = "r66-tls"

	AuthLegacyCertificate = r66auth.AuthLegacyCertificate
)

//nolint:gochecknoinits //init is used by design
func init() {
	authentication.AddInternalCredentialTypeForProtocol(auth.Password, R66, &r66auth.BcryptAuthHandler{})
	authentication.AddInternalCredentialTypeForProtocol(auth.Password, R66TLS, &r66auth.BcryptAuthHandler{})

	authentication.AddInternalCredentialTypeForProtocol(
		r66auth.AuthLegacyCertificate, R66TLS, &r66auth.LegacyCertificate{})
	authentication.AddExternalCredentialTypeForProtocol(
		r66auth.AuthLegacyCertificate, R66TLS, &r66auth.LegacyCertificate{})
}

type Module struct{}

func (Module) CanMakeTransfer(*model.TransferContext) error { return nil }

func (Module) NewServer(db *database.DB, server *model.LocalAgent) protocol.Server {
	return &service{db: db, agent: server}
}

func (Module) NewClient(db *database.DB, cli *model.Client) protocol.Client {
	return &Client{db: db, cli: cli}
}

func (Module) CheckServerConfig(conf map[string]any) error {
	return protoutils.ValidateProtoConfig(conf, &serverConfig{})
}

func (Module) CheckClientConfig(conf map[string]any) error {
	return protoutils.ValidateProtoConfig(conf, &clientConfig{})
}

func (Module) CheckPartnerConfig(conf map[string]any) error {
	return protoutils.ValidateProtoConfig(conf, &PartnerConfig{})
}

type ModuleTLS struct{ Module }

func (ModuleTLS) CheckServerConfig(conf map[string]any) error {
	return protoutils.ValidateProtoConfig(conf, &tlsServerConfig{})
}

func (ModuleTLS) CheckClientConfig(conf map[string]any) error {
	return protoutils.ValidateProtoConfig(conf, &tlsClientConfig{})
}

func (ModuleTLS) CheckPartnerConfig(conf map[string]any) error {
	return protoutils.ValidateProtoConfig(conf, &tlsPartnerConfig{})
}
