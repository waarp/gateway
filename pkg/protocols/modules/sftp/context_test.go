package sftp

import (
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline/pipelinetest"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

func makeAuths(ctx *pipelinetest.SelfContext) []*model.Credential {
	return []*model.Credential{makeServerKey(ctx.Server), makePartnerKey(ctx.Partner)}
}

func makeServerKey(serv *model.LocalAgent) *model.Credential {
	return &model.Credential{
		LocalAgentID: utils.NewNullInt64(serv.ID),
		Name:         "sftp_server_cert",
		Type:         AuthSSHPrivateKey,
		Value:        RSAPk,
	}
}

func makePartnerKey(part *model.RemoteAgent) *model.Credential {
	return &model.Credential{
		RemoteAgentID: utils.NewNullInt64(part.ID),
		Name:          "sftp_partner_cert",
		Type:          AuthSSHPublicKey,
		Value:         SSHPbk,
	}
}
