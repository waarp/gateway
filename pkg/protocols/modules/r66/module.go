package r66

import (
	"fmt"
	"math/big"

	"github.com/bwmarrin/snowflake"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/features"
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
	return &service{db: db, dbAgent: server}
}

func (Module) NewClient(db *database.DB, cli *model.Client) protocol.Client {
	return &Client{db: db, cli: cli}
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
	}
}

func (m Module) IDGenerator() model.IDGenerator { return &IDGenerator{} }

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
	gen *snowflake.Node
}

func (i *IDGenerator) Init(db database.ReadAccess) error {
	const snowflakeMaxNode = 1024

	var nodeID, mod, machineID big.Int

	nodeID.SetBytes([]byte(db.GetConfig().GatewayName + db.GetConfig().NodeID))
	mod.SetInt64(snowflakeMaxNode)
	machineID.Mod(&nodeID, &mod)

	generator, err := snowflake.NewNode(machineID.Int64())
	if err != nil {
		return fmt.Errorf("failed to create the ID generator: %w", err)
	}

	i.gen = generator

	return nil
}

func (i *IDGenerator) GetNextID() (string, error) {
	return i.gen.Generate().String(), nil
}
