// Package pesit implements a connector for the Pesit protocol, allowing the
// gateway to perform transfers using that protocol. The module implements both
// a client and a server.
package pesit

import (
	"database/sql"
	"fmt"
	"sync/atomic"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/features"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protocol"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protoutils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

const (
	Pesit    = "pesit"
	PesitTLS = "pesit-tls"
)

type Module struct{}

func (Module) CanMakeTransfer(*model.TransferContext) error { return nil }

func (Module) NewServer(db *database.DB, server *model.LocalAgent) protocol.Server {
	return newService(db, server)
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

func (Module) OptionalFeatures() []features.Feature {
	return []features.Feature{}
}

func (Module) IDGenerator() model.IDGenerator { return &IDGenerator{} }

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

type IDGenerator struct {
	count atomic.Uint32
}

func (i *IDGenerator) Init(db database.ReadAccess) error {
	row := db.QueryRow(`SELECT MAX(remote_transfer_id) FROM normalized_transfers
		WHERE is_send=true AND protocol IN (?, ?)`, Pesit, PesitTLS)

	var val sql.NullString
	if err := row.Scan(&val); err != nil {
		return fmt.Errorf("failed to scan pesit counter: %w", err)
	}

	if val.Valid {
		lastID, convErr := utils.ParseUint[uint32](val.String)
		if convErr != nil {
			return fmt.Errorf("failed to parse pesit transfer ID: %w", convErr)
		}
		i.count.Store(lastID)
	}

	return nil
}

func (i *IDGenerator) GetNextID() (string, error) {
	const maxPesitID = 1<<24 - 1

	newID := i.count.Add(1)
	if newID > maxPesitID {
		newID = 1
		i.count.Store(newID)
	}

	return utils.FormatUint(newID), nil
}
