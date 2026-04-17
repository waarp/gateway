package gwtesting

import (
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protocol"
)

type Protocol interface {
	model.Protocol
	NewServer(db *database.DB, server *model.LocalAgent) protocol.Server
	NewClient(db *database.DB, client *model.Client) protocol.Client
}

type ProtoFeatures struct {
	Protocol

	TransID, RuleName, Size bool
}

//nolint:gochecknoglobals //global var is required here for more flexibility
var protocolsList = map[string]ProtoFeatures{}

func Register(
	proto string, features ProtoFeatures,
) {
	protocolsList[proto] = features
	model.Protocols[proto] = features.Protocol
}
