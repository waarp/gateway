package r66

import (
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/services"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline/pipelinetest"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

const (
	serverLogin = "r66_login"
	serverPass  = "sesames"
)

//nolint:gochecknoglobals // these are test variables
var (
	cliConf  = &clientConfig{}
	servConf = &serverConfig{ServerLogin: serverLogin}
	partConf = &partnerConfig{ServerLogin: serverLogin}
)

func init() {
	pipelinetest.Protocols[R66] = pipelinetest.ProtoFeatures{
		MakeClient: func(db *database.DB, cli *model.Client) services.Client {
			return &Client{db: db, cli: cli, disableConnGrace: true}
		},
		MakeServer:        Module{}.NewServer,
		MakeServerConfig:  Module{}.MakeServerConfig,
		MakeClientConfig:  Module{}.MakeClientConfig,
		MakePartnerConfig: Module{}.MakePartnerConfig,
		TransID:           true,
		RuleName:          true,
		Size:              true,
		TransferInfo:      true,
	}
}

func serverPassword(server *model.LocalAgent) *model.Credential {
	return &model.Credential{
		LocalAgentID: utils.NewNullInt64(server.ID),
		Name:         "server_password",
		Type:         auth.Password,
		Value:        serverPass,
	}
}

func partnerPassword(partner *model.RemoteAgent) *model.Credential {
	return &model.Credential{
		RemoteAgentID: utils.NewNullInt64(partner.ID),
		Name:          "partner_password",
		Type:          auth.Password,
		Value:         serverPass,
	}
}
