package as2

import (
	"errors"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protocol"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protoutils"
)

const (
	AS2    = "as2"
	AS2TLS = "as2-tls"
)

var ErrTransferPull = errors.New(`"pull" transfers are not supported by the AS2 protocol`)

type Module struct{}

func (m Module) CanMakeTransfer(ctx *model.TransferContext) error {
	switch {
	case ctx.Transfer.IsServer() && !ctx.Rule.IsSend:
		return nil
	case !ctx.Transfer.IsServer() && ctx.Rule.IsSend:
		return nil
	default:
		return ErrTransferPull
	}
}

func (m Module) CheckServerConfig(conf map[string]any) error {
	return protoutils.ValidateProtoConfig(conf, &serverProtoConfig{})
}

func (m Module) CheckClientConfig(conf map[string]any) error {
	return protoutils.ValidateProtoConfig(conf, &clientProtoConfig{})
}

func (m Module) CheckPartnerConfig(conf map[string]any) error {
	return protoutils.ValidateProtoConfig(conf, &partnerProtoConfig{})
}

func (m Module) NewServer(db *database.DB, server *model.LocalAgent) protocol.Server {
	return NewServer(db, server)
}

func (m Module) NewClient(db *database.DB, client *model.Client) protocol.Client {
	return NewClient(db, client)
}

type ModuleTLS struct{ Module }

func (m ModuleTLS) CheckServerConfig(conf map[string]any) error {
	return protoutils.ValidateProtoConfig(conf, &serverProtoConfigTLS{})
}

func (m ModuleTLS) CheckClientConfig(conf map[string]any) error {
	return protoutils.ValidateProtoConfig(conf, &clientProtoConfigTLS{})
}

func (m ModuleTLS) CheckPartnerConfig(conf map[string]any) error {
	return protoutils.ValidateProtoConfig(conf, &partnerProtoConfigTLS{})
}
