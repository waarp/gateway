package protocols

import (
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protocol"
)

func MakeClient(db *database.DB, dbClient *model.Client) (protocol.Client, error) {
	mkClient := clientMakers[dbClient.Protocol]
	if mkClient == nil {
		return nil, ErrUnknownProtocol
	}

	return mkClient(db, dbClient), nil
}

func MakeServer(db *database.DB, dbServer *model.LocalAgent) (protocol.Server, error) {
	mkServer := serverMakers[dbServer.Protocol]
	if mkServer == nil {
		return nil, ErrUnknownProtocol
	}

	return mkServer(db, dbServer), nil
}

// IsValid returns whether the given protocol is implemented.
func IsValid(name string) bool { return model.Protocols[name] != nil }

//nolint:gochecknoglobals //global var is required here
var (
	clientMakers = map[string]func(*database.DB, *model.Client) protocol.Client{}
	serverMakers = map[string]func(*database.DB, *model.LocalAgent) protocol.Server{}
)
