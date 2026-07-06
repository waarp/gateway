package r66

import (
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline/pipelinetest"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protocol"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/gwtesting"
)

const (
	serverLogin = "r66_login"
	serverPass  = "sesames"
)

//nolint:gochecknoglobals // these are test variables
var (
	cliConf  = &ClientConfig{}
	servConf = &ServerConfig{SharedServerConfig: SharedServerConfig{ServerLogin: serverLogin}}
	partConf = &PartnerConfig{SharedPartnerConfig: SharedPartnerConfig{ServerLogin: serverLogin}}
)

func init() {
	pipelinetest.Register(R66, pipelinetest.ProtoFeatures{
		Protocol:     TestModule{},
		TransID:      true,
		RuleName:     true,
		Size:         true,
		TransferInfo: true,
	})
}

func init() {
	gwtesting.Register(R66, gwtesting.ProtoFeatures{
		Protocol: Module{},
		TransID:  true,
		RuleName: true,
		Size:     true,
	})

	gwtesting.Register(R66TLS, gwtesting.ProtoFeatures{
		Protocol: ModuleTLS{},
		TransID:  true,
		RuleName: true,
		Size:     true,
	})
}

type TestModule struct{ Module }

func (t TestModule) NewClient(db *database.DB, cli *model.Client) protocol.Client {
	return &Client{db: db, cli: cli, disableConnGrace: true}
}

func serverPassword(server *model.LocalAgent) *model.Credential {
	return &model.Credential{
		LocalAgentID: server.NullableID(),
		Name:         "server_password",
		Type:         auth.Password,
		Value:        serverPass,
	}
}

func partnerPassword(partner *model.RemoteAgent) *model.Credential {
	return &model.Credential{
		RemoteAgentID: partner.NullableID(),
		Name:          "partner_password",
		Type:          auth.Password,
		Value:         serverPass,
	}
}
